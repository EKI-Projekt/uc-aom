// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package migrate

import (
	"bytes"
	"fmt"
	"io"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/aom/docker"
	v0_1_stack "u-control/uc-aom/internal/aom/docker/v0_1"
	v0_1_stack_portainer "u-control/uc-aom/internal/aom/docker/v0_1/portainer"
	"u-control/uc-aom/internal/aom/env"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/service"
	"u-control/uc-aom/internal/aom/utils"

	log "github.com/sirupsen/logrus"

	model "u-control/uc-aom/internal/pkg/manifest"
)

const currentVersion = config.UcAomVersion

type Migrator interface {
	Migrate() error
}

type installAddonMigrator struct {
	localfsRegistry      localFSRegistry
	stackService         docker.StackServiceAPI
	stackMigrator        docker.StackMigrator
	routesMigrator       routes.ReverseProxyMigrator
	transactionScheduler *service.TransactionScheduler
	versionResolver      versionResolver
	envResolver          env.EnvResolver
}

// NewInstallAddOnMigrator returns a new instance of Migrator
func NewInstallAddOnMigrator(root string,
	localfs *manifest.LocalFSRepository,
	transactionScheduler *service.TransactionScheduler,
	service *service.Service,
	stackService docker.StackServiceAPI,
	envResolver env.EnvResolver,
	reverseProxy routes.ReverseProxyCreater) Migrator {
	localFSAdapter := &localFSRegistryAdapter{root: root, localfs: localfs}
	stackMigrator := docker.NewStackMigrator(stackService)
	routesMigrator := routes.NewReverseProxyMigrator(reverseProxy)

	versionResolver := &aomVersionResolver{
		localStateDir:   config.UC_AOM_STATE_DIRECTORY,
		localFSRegistry: localFSAdapter,
	}
	return &installAddonMigrator{
		localfsRegistry:      localFSAdapter,
		envResolver:          envResolver,
		stackMigrator:        stackMigrator,
		routesMigrator:       routesMigrator,
		stackService:         stackService,
		transactionScheduler: transactionScheduler,
		versionResolver:      versionResolver,
	}
}

func (m *installAddonMigrator) Migrate() error {

	previousVersion, err := m.versionResolver.getVersion()
	if err != nil {
		return fmt.Errorf("Error while getVersion: %v", err)
	}

	if previousVersion == currentVersion {
		log.Tracef("No migration needed. Current version %s is running", currentVersion)
		return nil
	}

	switch previousVersion {
	case "0.3.2":
		err := m.migrateFromV0_3_2()
		if err != nil {
			return err
		}
		fallthrough

	case "0.4.0":
		// Migration from 0.4.0 to 0.5.0 isn't needed since the migration is only required due to a portainer related bug
		// and portainer was exchanged with docker compose in 0.4.0
		fallthrough

	case "0.5.0", "0.5.1":
		// Migration from 0.5.0 and 0.5.1 to 0.5.2 isn't needed since the firmware release has been retired.
		// We thought that volumes might always be empty but that is not the case if the docker image has default configuration.
		// That configuration can be accessed and changed via a volume mount.
		// 0.5.1 still has a migration bug if settings are changed.
		fallthrough

	case "0.5.2":
		err := m.migrateFromV0_5_2()
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("No migration step for: %s, current version is %s.", previousVersion, currentVersion)

	}

	m.versionResolver.updateVersion(currentVersion)

	return nil
}

func (m *installAddonMigrator) migrateFromV0_5_2() error {
	log.Trace("Migrate all installed add-ons from v0.5.2...")
	installedAddOnRepositories, err := m.localfsRegistry.Repositories()
	if err != nil {
		log.Errorf("Repositories(): %v", err)
		return err
	}

	for _, repositoryName := range installedAddOnRepositories {
		err := m.migrateAddOnFromV0_5_2(repositoryName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *installAddonMigrator) migrateAddOnFromV0_5_2(repositoryName string) error {
	log.Tracef("Migrate add-on %s...", repositoryName)
	repository, err := m.localfsRegistry.Repository(repositoryName)
	if err != nil {
		return err
	}

	_, manifestAsBytes, err := fetchManifestFrom(repository)
	if err != nil {
		return err
	}

	manifest, err := model.NewFromBytes(manifestAsBytes)
	if err != nil {
		return err
	}

	permissionId := utils.ReplaceSlashesWithDashes(repositoryName)
	err = m.routesMigrator.Migrate(repositoryName, routes.TemplateVersionV0_1_0, manifest.Title, permissionId, manifest.Publish)

	return err
}

func (m *installAddonMigrator) migrateFromV0_3_2() error {
	log.Trace("Migrate all installed add-ons from v0.3.2...")
	installedAddOnRepositories, err := m.localfsRegistry.Repositories()
	if err != nil {
		log.Errorf("Repositories(): %v", err)
		return err
	}

	for _, repositoryName := range installedAddOnRepositories {
		err := m.migrateAddOnFromV0_3_2(repositoryName)
		if err != nil {
			log.Errorf("Add-on %s, migrateAddOnFromV0_3_2(): %v", repositoryName, err)
			return err
		}
	}
	return nil

}

func (m *installAddonMigrator) migrateAddOnFromV0_3_2(repositoryName string) error {
	log.Tracef("Migrate add-on %s...", repositoryName)
	repository, err := m.localfsRegistry.Repository(repositoryName)
	if err != nil {
		return err
	}

	manifestVersion, manifestAsBytes, err := fetchManifestFrom(repository)
	if err != nil {
		return err
	}

	migratedManifestAsBytes, err := model.MigrateUcManifest(manifestVersion, manifestAsBytes)
	if err != nil {
		return err
	}

	migratedManifest, err := model.NewFromBytes(migratedManifestAsBytes)
	if err != nil {
		return err
	}
	normalizedName := v0_1_stack_portainer.NormalizeName(repositoryName)
	settings, err := m.getSettingsForReplaceRoutine(migratedManifest, normalizedName)
	if err != nil {
		return err
	}

	err = m.stackMigrator.MigrateStack(repositoryName, v0_1_stack.StackVersion, migratedManifest, settings...)
	if err != nil {
		return err
	}
	return repository.Push(bytes.NewBuffer(migratedManifestAsBytes))
}

func (m *installAddonMigrator) getSettingsForReplaceRoutine(migratedManifest *model.Root, repositoryName string) ([]*model.Setting, error) {
	manifestSettings := migratedManifest.Settings["environmentVariables"]
	if len(manifestSettings) == 0 {
		return manifestSettings, nil
	}

	currentSettings, err := m.envResolver.GetAddOnEnvironment(repositoryName)
	if err != nil {
		return nil, err
	}
	settings := manifest.CombineManifestSettingsWithSettingsMap(manifestSettings, currentSettings)
	return settings, nil

}

func fetchManifestFrom(repository localFSRepository) (string, []byte, error) {

	content, err := repository.Fetch()
	if err != nil {
		return "", nil, err
	}

	manifestAsBytes, err := io.ReadAll(content)
	if err != nil {
		return "", nil, err
	}

	manifestVersion, err := model.UnmarshalManifestVersionFrom(manifestAsBytes)
	return manifestVersion, manifestAsBytes, err
}
