// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/user"
	"strconv"
	"syscall"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

type saveData struct {
	State map[string]string `json:"state"`
}

func serveDockerVolumesDriver(driver volume.Driver, driverName string) error {
	// error is used in goroutine
	var err error = nil
	const rootUsername = "root"

	h := volume.NewHandler(driver)
	u, err := user.Lookup(rootUsername)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return err
	}
	go func() {
		err = h.ServeUnix(driverName, gid)
		if err != nil {
			log.Errorf("ServeUnix(): Unexpected error %v", err)
		}
	}()
	return err
}

func createLocalCapabilitiesResponse() *volume.CapabilitiesResponse {
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{Scope: "local"},
	}
}

func changeOwnerOfPath(owner *user.User, path string) error {
	uid, _ := strconv.Atoi(owner.Uid)
	gid, _ := strconv.Atoi(owner.Gid)

	return syscall.Chown(path, uid, gid)
}

func saveState(path string, volumes map[string]string) error {
	data := saveData{
		State: volumes,
	}

	fileData, err := json.Marshal(data)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to save volumes state file: %s", path))
	}

	return ioutil.WriteFile(path, fileData, 0600)
}

func findExistingVolumesFromStateFile(path string) (error, map[string]string) {
	fileData, err := ioutil.ReadFile(path)
	if err != nil {
		return err, map[string]string{}
	}

	var data saveData
	e := json.Unmarshal(fileData, &data)
	if e != nil {
		return e, map[string]string{}
	}

	return nil, data.State
}

type driverVolumes map[string]string

func (s driverVolumes) exists(name string) bool {
	return s[name] != ""
}

func (s driverVolumes) getVolume(name string) *volume.Volume {
	return &volume.Volume{
		Name:       name,
		Mountpoint: s[name],
	}
}

func (s driverVolumes) getVolumes() []*volume.Volume {
	volumes := make([]*volume.Volume, 0, len(s))
	for name := range s {
		volumes = append(volumes, s.getVolume(name))
	}
	return volumes
}
