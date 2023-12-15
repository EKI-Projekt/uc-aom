// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"regexp"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/aom/env"
	"u-control/uc-aom/internal/aom/iam"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/system"
	"u-control/uc-aom/internal/aom/utils"
	"u-control/uc-aom/internal/aom/yaml"
	"u-control/uc-aom/internal/pkg/manifest"

	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"

	log "github.com/sirupsen/logrus"
)

var (
	ErrorAddOnAlreadyInstalled = errors.New("Already installed.")
	ErrorCodesys               = grpcStatus.Error(codes.Unimplemented, "Not supported for the codesys app")
)

// Abstraction layer over the individual steps
// of (un)installing/upgrading AddOns.
type Service struct {
	stackService             docker.StackServiceAPI
	reverseProxy             *routes.ReverseProxy
	iamPermissionWriter      *iam.IamPermissionWriter
	localCatalogue           catalogue.LocalAddOnCatalogue
	Validator                manifest.Validator
	addOnEnvironmentResolver env.EnvResolver
	system                   system.System
}

// Create a new instance of the Service.
func NewService(
	stackService docker.StackServiceAPI,
	reverseProxy *routes.ReverseProxy,
	iamPermissionWriter *iam.IamPermissionWriter,
	localCatalogue catalogue.LocalAddOnCatalogue,
	validator manifest.Validator,
	addOnEnvironmentResolver env.EnvResolver,
	system system.System) *Service {
	return &Service{stackService, reverseProxy, iamPermissionWriter, localCatalogue, validator, addOnEnvironmentResolver, system}
}

// Create an AddOn.
func (tx *Tx) CreateAddOnRoutine(name string, version string, settings ...*manifest.Setting) error {
	log.Tracef("CreateAddOnRoutine('%s', '%s', '%v')", name, version, settings)
	isInstalled, err := tx.service.isAddOnInstalled(name)
	if err != nil {
		return err
	}

	if isInstalled {
		return ErrorAddOnAlreadyInstalled
	}

	tx.SubscribeRollbackHook(func() {
		tx.service.localCatalogue.DeleteAddOn(name)
	})
	catalogueAddOn, err := tx.service.localCatalogue.PullAddOn(name, version)
	if err != nil {
		return err
	}

	if err := CheckDiskSpace(tx.service.system, catalogueAddOn.EstimatedInstallSize); err != nil {
		return err
	}

	if err := tx.service.Validator.Validate(&catalogueAddOn.AddOn.Manifest); err != nil {
		return err
	}

	manifestVersionValidator := NewManifestVersionValidator(catalogueAddOn.AddOn.Manifest.ManifestVersion)
	if err := manifestVersionValidator.Validate(); err != nil {
		return err
	}

	tx.setAddOnContext(catalogueAddOn.AddOn.Name, catalogueAddOn.AddOn.Manifest.Title, Installing)

	if err := tx.service.checkCapabilities(&catalogueAddOn.AddOn.Manifest); err != nil {
		return err
	}

	if len(settings) != 0 {
		catalogueAddOn.AddOn.Manifest.Settings["environmentVariables"] = settings
	}

	tx.SubscribeRollbackHook(func() {
		imageReferences := manifest.GetDockerImageReferences(catalogueAddOn.AddOn.Manifest.Services)
		tx.service.stackService.DeleteDockerImages(imageReferences...)
	})
	for _, image := range catalogueAddOn.DockerImageData {
		err = tx.service.stackService.ImportDockerImage(image)
		if err != nil {
			return err
		}
	}

	manifestAdapter := newManifestFeatureToSystemAdapter(tx.service.system)
	manifestToDeploy, err := manifestAdapter.adaptFeaturesToSystem(&catalogueAddOn.AddOn.Manifest)
	if err != nil {
		return err
	}

	dockerCompose, err := yaml.GetDockerComposeFromManifest(manifestToDeploy)
	if err != nil {
		return err
	}

	tx.SubscribeRollbackHook(func() {
		tx.service.stackService.DeleteAddOnStack(catalogueAddOn.AddOn.Name)
		tx.service.removeUnusedVolumes(catalogueAddOn.AddOn)
	})
	if err := tx.service.stackService.CreateStackWithDockerCompose(catalogueAddOn.AddOn.Name, dockerCompose); err != nil {
		return err
	}

	tx.SubscribeRollbackHook(func() {
		tx.service.deleteIamPermission(catalogueAddOn.AddOn.Name)
	})
	if err := tx.service.createIamPermission(catalogueAddOn.AddOn.Name, catalogueAddOn.AddOn.Manifest.Title); err != nil {
		return err
	}

	tx.SubscribeRollbackHook(func() {
		tx.service.deleteProxyRoutes(catalogueAddOn.AddOn.Name, catalogueAddOn.AddOn.Manifest.Publish)
	})
	return tx.service.createProxyRoutes(catalogueAddOn.AddOn.Name, catalogueAddOn.AddOn.Manifest.Title, catalogueAddOn.AddOn.Name, catalogueAddOn.AddOn.Manifest.Publish)
}

// Can upgrade or reconfigure an installed add-on.
func (tx *Tx) ReplaceAddOnRoutine(name string, version string, settings ...*manifest.Setting) error {
	if isCodesys(name) {
		return ErrorCodesys
	}
	addOn, err := tx.service.localCatalogue.GetAddOn(name)
	if err != nil {
		return err
	}

	if addOn.Version == version {
		tx.setAddOnContext(addOn.Name, addOn.Manifest.Title, Configuring)
		return tx.configureAction(addOn, settings...)
	}

	tx.setAddOnContext(addOn.Name, addOn.Manifest.Title, Updating)
	return tx.updateAction(addOn, version, settings...)
}

// Delete an installed AddOn.
func (tx *Tx) DeleteAddOnRoutine(name string) error {
	if isCodesys(name) {
		return ErrorCodesys
	}
	addOn, err := tx.service.localCatalogue.GetAddOn(name)
	if err != nil {
		return err
	}
	tx.setAddOnContext(addOn.Name, addOn.Manifest.Title, Deleting)
	return tx.service.deleteAddOnWithVolumes(addOn)
}

func (s *Service) checkCapabilities(manifest *manifest.Root) error {
	capabilities := NewCapabilities(s.system, manifest.Platform...)

	if len(manifest.Features) > 0 {
		capabilities.WithFeatures(manifest.Features)
	}

	err := capabilities.Validate()
	return err
}

func (s *Service) isAddOnInstalled(addOnName string) (bool, error) {
	_, err := s.localCatalogue.GetAddOn(addOnName)
	if err != nil {
		if errors.Is(err, catalogue.ErrorAddOnNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (s *Service) deleteAddOnExceptVolumes(addOn catalogue.CatalogueAddOn) error {
	noop := func(catalogue.CatalogueAddOn) error {
		return nil
	}
	return s.deleteAddOnResources(addOn, noop)
}

func (s *Service) deleteAddOnWithVolumes(addOn catalogue.CatalogueAddOn) error {
	return s.deleteAddOnResources(addOn, s.removeUnusedVolumes)
}

func (s *Service) removeUnusedVolumes(addOn catalogue.CatalogueAddOn) error {
	volumes := manifest.GetVolumeNames(addOn.Manifest.Environments)
	return s.stackService.RemoveUnusedVolumes(addOn.Name, volumes...)
}

func (s *Service) deleteAddOnResources(addOn catalogue.CatalogueAddOn, action func(catalogue.CatalogueAddOn) error) error {
	// Should we try and delete as much or as little as possible?
	if err := s.stackService.DeleteAddOnStack(addOn.Name); err != nil {
		return err
	}

	imageReferences := manifest.GetDockerImageReferences(addOn.Manifest.Services)
	if err := s.stackService.DeleteDockerImages(imageReferences...); err != nil {
		return err
	}

	if err := action(addOn); err != nil {
		return err
	}

	if err := s.deleteProxyRoutes(addOn.Name, addOn.Manifest.Publish); err != nil {
		return err
	}

	if err := s.deleteIamPermission(addOn.Name); err != nil {
		return err
	}

	return s.localCatalogue.DeleteAddOn(addOn.Name)
}

func (s *Service) createProxyRoutes(name string, title string, permissionId string, proxyRoute map[string]*manifest.ProxyRoute) error {
	convertedPermissionId := utils.ReplaceSlashesWithDashes(permissionId)
	for id, location := range proxyRoute {
		reverseProxyHttpConf := routes.NewReverseProxyHttpConf(name, location)
		reverseProxyMap := &routes.ReverseProxyMap{AddOnName: name, AddOnTitle: title, To: location.To, Id: convertedPermissionId}
		prefixedId := routes.CreatePrefixedRouteFilenameId(name, id)
		err := s.reverseProxy.Create(prefixedId, reverseProxyMap, reverseProxyHttpConf)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) deleteProxyRoutes(name string, proxyRoute map[string]*manifest.ProxyRoute) error {
	for id := range proxyRoute {
		prefixedId := routes.CreatePrefixedRouteFilenameId(name, id)
		err := s.reverseProxy.Delete(prefixedId)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) createIamPermission(permissionId string, addOnTitle string) error {
	permissionId = utils.ReplaceSlashesWithDashes(permissionId)
	permission := &iam.IamPermission{AddOnTitle: addOnTitle, Id: permissionId, NoAuthOpt: iam.IAM_AUTH_NO_AUTH_OPT}
	return s.iamPermissionWriter.Create(permission)
}

func (s *Service) deleteIamPermission(permissionId string) error {
	permissionId = utils.ReplaceSlashesWithDashes(permissionId)
	return s.iamPermissionWriter.Delete(permissionId)
}

func isCodesys(appName string) bool {
	r := regexp.MustCompile(`(?i)\bcodesys\b`)
	return r.MatchString(appName)
}
