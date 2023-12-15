// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"sync"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

const (
	stateFile = manifest.LocalPublicVolumeDriverName + ".json"
)

type localPublicDriver struct {
	stateDir       string
	volumesDir     string
	volumesDirUser *user.User
	volumes        driverVolumes
	mutex          *sync.Mutex
	name           string
	stateFilePath  string
}

func CreateAndServeLocalPublicVolumesDriver(publicVolumesUser *user.User) error {
	driver := newLocalPublicDriver(config.UC_AOM_STATE_DIRECTORY, PUBLIC_VOLUMES_PATH, publicVolumesUser)
	return serveDockerVolumesDriver(driver, driver.name)
}

func newLocalPublicDriver(stateDir string, volumesDir string, volumesDirUser *user.User) *localPublicDriver {
	driver := &localPublicDriver{
		stateDir:       stateDir,
		volumesDir:     volumesDir,
		volumesDirUser: volumesDirUser,
		volumes:        driverVolumes{},
		mutex:          &sync.Mutex{},
		name:           manifest.LocalPublicVolumeDriverName,
		stateFilePath:  path.Join(stateDir, stateFile),
	}

	log.Debugf("%-18s %s", "Starting driver...", driver.name)

	os.Mkdir(volumesDir, 0700)
	changeOwnerOfPath(driver.volumesDirUser, volumesDir)

	_, driver.volumes = findExistingVolumesFromStateFile(driver.stateFilePath)
	log.Debugf("Found %s volumes on startup", strconv.Itoa(len(driver.volumes)))

	return driver
}

func (driver localPublicDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	log.Debugf("%-18s", "Get called... ")

	if driver.volumes.exists(req.Name) {
		log.Debugf("Found %s", req.Name)
		return &volume.GetResponse{
			Volume: driver.volumes.getVolume(req.Name),
		}, nil
	}

	log.Debugf("Couldn't find %s", req.Name)
	return nil, fmt.Errorf("No volume found with the name %s", req.Name)
}

func (driver localPublicDriver) List() (*volume.ListResponse, error) {
	volumes := driver.volumes.getVolumes()
	log.Debugf("List called: Found %s volumes", strconv.Itoa(len(volumes)))

	return &volume.ListResponse{
		Volumes: volumes,
	}, nil
}

func (driver localPublicDriver) Create(req *volume.CreateRequest) error {
	log.Debugf("%-18s", "Create called... ")

	mountpoint := filepath.Join(driver.volumesDir, req.Name)

	if mountpoint == "" {
		log.Debugf("No %s option provided", "mountpoint")
		return errors.New("The `mountpoint` option is required")
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	if driver.volumes.exists(req.Name) {
		return fmt.Errorf("The volume %s already exists", req.Name)
	}

	err := os.MkdirAll(mountpoint, 0755)
	log.Debugf("Ensuring directory %s exists on host...", mountpoint)

	if err != nil {
		log.Debugf("%17s Could not create directory %s", " ", mountpoint)
		return err
	}

	err = changeOwnerOfPath(driver.volumesDirUser, mountpoint)
	if err != nil {
		log.Debugf("%17s Could not change owner of directory %s", " ", mountpoint)
		return err
	}

	driver.volumes[req.Name] = mountpoint

	e := saveState(driver.stateFilePath, driver.volumes)
	if e != nil {
		log.Errorln(e.Error())
	}

	log.Debugf("%17s Created volume %s with mountpoint %s", " ", req.Name, mountpoint)
	return nil
}

func (driver localPublicDriver) Remove(req *volume.RemoveRequest) error {
	log.Debugf("%-18s", "Remove called... ")

	if !driver.volumes.exists(req.Name) {
		log.Debugf("The volume %s doesn't exists", req.Name)
		return nil
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	err := os.RemoveAll(driver.volumes[req.Name])
	if err != nil {
		log.Errorln(err.Error())
	}

	delete(driver.volumes, req.Name)

	err = saveState(driver.stateFilePath, driver.volumes)
	if err != nil {
		log.Debugln(err.Error())
		return err
	}

	log.Debugf("Removed %s", req.Name)

	return nil
}

func (driver localPublicDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	log.Debugf("Mounted called on %s", req.Name)
	return &volume.MountResponse{Mountpoint: driver.volumes[req.Name]}, nil
}

func (driver localPublicDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	log.Debugf("Path called: returned path %s", driver.volumes[req.Name])
	return &volume.PathResponse{Mountpoint: driver.volumes[req.Name]}, nil
}

func (driver localPublicDriver) Unmount(req *volume.UnmountRequest) error {
	log.Debugf("Unmount called on %s", req.Name)
	return nil
}

func (driver localPublicDriver) Capabilities() *volume.CapabilitiesResponse {
	log.Debugf("%-18s", "Capabilities called... ")

	return createLocalCapabilitiesResponse()
}
