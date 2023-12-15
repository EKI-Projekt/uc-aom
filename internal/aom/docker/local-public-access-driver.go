// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"
	"sync"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

const (
	publicAccessStateFile = manifest.LocalPublicVolumeAccessDriverName + ".json"
)

type localPublicAccessDriver struct {
	volumesDir string
	volumes    driverVolumes
	// prevent concurrent access to volumes
	mutex         *sync.Mutex
	name          string
	stateFilePath string
}

func CreateAndServelocalPublicAccessVolumesDriver(publicVolumesUser *user.User) error {
	driver := newLocalPublicAccessDriver(config.UC_AOM_STATE_DIRECTORY, PUBLIC_VOLUMES_PATH, publicVolumesUser)
	return serveDockerVolumesDriver(driver, driver.name)
}

func newLocalPublicAccessDriver(stateDir string, volumesDir string, volumesDirUser *user.User) *localPublicAccessDriver {
	driver := &localPublicAccessDriver{
		volumesDir:    volumesDir,
		volumes:       driverVolumes{},
		name:          manifest.LocalPublicVolumeAccessDriverName,
		mutex:         &sync.Mutex{},
		stateFilePath: path.Join(stateDir, publicAccessStateFile),
	}
	log.Debugf("%-18s %s", "Starting driver...", driver.name)
	os.Mkdir(volumesDir, 0700)
	changeOwnerOfPath(volumesDirUser, volumesDir)

	_, driver.volumes = findExistingVolumesFromStateFile(driver.stateFilePath)

	log.Debugf("Found %s volumes on startup", strconv.Itoa(len(driver.volumes)))

	return driver
}

func (driver localPublicAccessDriver) Get(req *volume.GetRequest) (*volume.GetResponse, error) {
	log.Debugf("%-18s", "Get Called... ")

	if driver.volumes.exists(req.Name) {
		log.Debugf("Found %s", req.Name)
		return &volume.GetResponse{
			Volume: driver.volumes.getVolume(req.Name),
		}, nil
	}

	log.Debugf("Couldn't find %s", req.Name)
	return nil, fmt.Errorf("No volume found with the name %s", req.Name)

}

func (driver localPublicAccessDriver) List() (*volume.ListResponse, error) {
	log.Debugf("%-18s", "List Called... ")

	volumes := driver.volumes.getVolumes()

	log.Debugf("Found %s volumes", strconv.Itoa(len(volumes)))

	return &volume.ListResponse{
		Volumes: volumes,
	}, nil
}

func (driver localPublicAccessDriver) Create(req *volume.CreateRequest) error {
	log.Debugf("%-18s", "Create Called...")

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	if driver.volumes.exists(req.Name) {
		return fmt.Errorf("The volume %s already exists", req.Name)
	}

	driver.volumes[req.Name] = driver.volumesDir

	e := saveState(driver.stateFilePath, driver.volumes)
	if e != nil {
		log.Errorln(e.Error())
	}
	return nil
}

func (driver localPublicAccessDriver) Remove(req *volume.RemoveRequest) error {
	log.Debugf("%-18s", "Remove Called... ")

	if !driver.volumes.exists(req.Name) {
		log.Debugf("The volume %s doesn't exists", req.Name)
		return nil
	}

	driver.mutex.Lock()
	defer driver.mutex.Unlock()

	delete(driver.volumes, req.Name)

	err := saveState(driver.stateFilePath, driver.volumes)
	if err != nil {
		log.Errorln(err.Error())
	}

	return nil
}

func (driver localPublicAccessDriver) Mount(req *volume.MountRequest) (*volume.MountResponse, error) {
	log.Debugf("Mount called on %s", req.Name)
	return &volume.MountResponse{Mountpoint: driver.volumes[req.Name]}, nil
}

func (driver localPublicAccessDriver) Path(req *volume.PathRequest) (*volume.PathResponse, error) {
	log.Debugf("Path called: Returned path %s", driver.volumes[req.Name])

	return &volume.PathResponse{Mountpoint: driver.volumes[req.Name]}, nil
}

func (driver localPublicAccessDriver) Unmount(req *volume.UnmountRequest) error {
	log.Debugf("Unmount Called on %s", req.Name)
	return nil
}

func (driver localPublicAccessDriver) Capabilities() *volume.CapabilitiesResponse {
	log.Debugf("Capabilities Called... ")
	return createLocalCapabilitiesResponse()
}
