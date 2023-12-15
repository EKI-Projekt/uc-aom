// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"fmt"
	"os"
	"path"
	sharedConfig "u-control/uc-aom/internal/aom/config"
	v0_1_stack "u-control/uc-aom/internal/aom/docker/v0_1"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer"
	"u-control/uc-aom/internal/aom/yaml"
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/docker/daemon/graphdriver/copy"

	log "github.com/sirupsen/logrus"
)

type StackMigrator interface {
	// MigrateStack starts a new stack migratine.
	MigrateStack(name string, version string, manifest *model.Root, settings ...*model.Setting) error
}

func NewStackMigrator(stackService StackServiceAPI) StackMigrator {
	return &stackMigrator{stackService: stackService,
		connectToPortainer: v0_1_stack.ConnectToPortainer}
}

type stackMigrator struct {
	stackService       StackServiceAPI
	connectToPortainer func() (portainer.PortainerClientService, error)
}

func (m *stackMigrator) MigrateStack(name string, versionToMigrate string, manifest *model.Root, settings ...*model.Setting) error {

	switch versionToMigrate {
	case v0_1_stack.StackVersion:
		portainerClient, err := m.connectToPortainer()
		if err != nil {
			return err
		}

		defer func() {
			log.Trace("Logout from portainer...")
			err := portainerClient.Logout()
			if err != nil {
				log.Errorf("Logout error from Portainer: %v", err)
			}
			log.Trace("Logout from portainer was successful")
		}()

		err = portainerClient.DeleteAddOnStack(name)
		if err != nil {
			return err
		}

		if len(settings) != 0 {
			if manifest.Settings == nil {
				manifest.Settings = make(map[string][]*model.Setting)
			}
			manifest.Settings["environmentVariables"] = settings
		}

		dockerCompose, err := yaml.GetDockerComposeFromManifest(manifest)
		if err != nil {
			return err
		}

		err = m.stackService.CreateStackWithoutStartWithDockerCompose(name, dockerCompose)
		if err != nil {
			return err
		}

		volumeNames := model.GetVolumeNames(manifest.Environments)
		normalizedPortainerStackName := portainer.NormalizeName(name)
		err = m.migratePortainerVolumeData(name, normalizedPortainerStackName, volumeNames)
		if err != nil {
			return err
		}
		return m.stackService.RemoveUnusedVolumes(normalizedPortainerStackName, volumeNames...)

	case StackVersion:
		// nothing to do if version is the current stack version

	default:
		return fmt.Errorf("Stack version %s is unknown", versionToMigrate)
	}

	return nil
}

func (m stackMigrator) migratePortainerVolumeData(stackName string, portainerStackName string, volumeNames []string) error {
	log.Tracef("Migrating volumes between %s and %s ...", portainerStackName, stackName)
	for _, volumeName := range volumeNames {
		log.Tracef("Migrating volume %s", volumeName)

		portainerVolumeInfo, err := m.stackService.VolumeInspect(
			createStackScopedVolumeName(portainerStackName, volumeName),
		)
		if err != nil {
			return err
		}

		netStackVolumeName := createStackScopedVolumeName(stackName, volumeName)
		volumeInfo, err := m.stackService.VolumeInspect(netStackVolumeName)
		if err != nil {
			return err
		}

		cacheDirPattern := fmt.Sprintf("%s_*", netStackVolumeName)
		cacheDir, err := os.MkdirTemp(sharedConfig.UC_AOM_CACHE_DIRECTORY, cacheDirPattern)
		if err != nil {
			cacheDirName := path.Join(sharedConfig.UC_AOM_CACHE_DIRECTORY, cacheDirPattern)
			return fmt.Errorf("Unexpected Error while create cache directory %s: %v ", cacheDirName, err)
		}
		err = copy.DirCopy(portainerVolumeInfo.Mountpoint, cacheDir, copy.Content, false)
		if err != nil {
			return fmt.Errorf("Unexpected Error copy from %s to %s: %v ", portainerVolumeInfo.Mountpoint, cacheDir, err)
		}

		log.Tracef("Remove content of %s", portainerVolumeInfo.Mountpoint)
		err = removeContentOf(portainerVolumeInfo.Mountpoint)
		if err != nil {
			return fmt.Errorf("Unexpected removeContentOf %s: %v ", portainerVolumeInfo.Mountpoint, err)
		}

		// make sure that the destination volume is empty because otherwise DirCopy will return an error.
		log.Tracef("Remove content of %s", volumeInfo.Mountpoint)
		err = removeContentOf(volumeInfo.Mountpoint)
		if err != nil {
			return fmt.Errorf("Unexpected removeContentOf %s: %v ", volumeInfo.Mountpoint, err)
		}

		err = copy.DirCopy(cacheDir, volumeInfo.Mountpoint, copy.Content, false)
		if err != nil {
			return fmt.Errorf("Unexpected Error copy from %s to %s: %v ", cacheDir, volumeInfo.Mountpoint, err)
		}

		err = os.RemoveAll(cacheDir)
		if err != nil {
			return fmt.Errorf("Unexpected RemoveAll(%s): %v ", cacheDir, err)
		}
	}
	return nil
}

func removeContentOf(parentDir string) error {
	dir, err := os.ReadDir(parentDir)
	if err != nil {
		return err
	}
	for _, d := range dir {
		err := os.RemoveAll(path.Join([]string{parentDir, d.Name()}...))
		if err != nil {
			return err
		}
	}
	return nil
}
