// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/stretchr/testify/assert"
)

func Test_localPublicAccessDriver_ShallCreateStateDir(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	os.RemoveAll(stateDir)
	stateDirUser := createTestStateDirUser()

	// Act
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)

	// Assert
	assert.NotNil(t, uut)
	gotFileInfo, err := os.Stat(stateDir)
	assert.Nil(t, err)

	assertUserSettings(t, gotFileInfo, stateDirUser)

}

func Test_localPublicAccessDriver_Get_ShallReturnErrorIfNotExist(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	req := &volume.GetRequest{Name: "test"}

	// Act
	res, err := uut.Get(req)

	// Assert
	assert.Nil(t, res)
	assert.Error(t, err)

}

func Test_localPublicAccessDriver_Get_ShallReturnIfExist(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	name := "test"
	createReq := &volume.CreateRequest{Name: name}

	// Act
	err := uut.Create(createReq)

	req := &volume.GetRequest{Name: name}
	gotRes, err := uut.Get(req)

	// Assert
	assert.Nil(t, err)
	wantRes := &volume.GetResponse{Volume: &volume.Volume{Name: name, Mountpoint: stateDir}}
	assert.Equal(t, *wantRes.Volume, *gotRes.Volume)
}

func Test_localPublicAccessDriver_List_ShallReturnTheStateDir(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	name := "test"
	createReq := &volume.CreateRequest{Name: name}

	// Act
	err := uut.Create(createReq)
	gotRes, err := uut.List()

	// Assert
	assert.Nil(t, err)
	wantVolumes := []*volume.Volume{{
		Name:       name,
		Mountpoint: stateDir,
	}}
	wantRes := &volume.ListResponse{Volumes: wantVolumes}
	assert.EqualValues(t, wantRes.Volumes, gotRes.Volumes)
}

func Test_localPublicAccessDriver_ShallNotCreate(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	req := &volume.CreateRequest{Name: "test"}

	// Act
	err := uut.Create(req)

	// Assert
	assert.Nil(t, err)

	wantDir := filepath.Join(stateDir, req.Name)
	_, err = os.Stat(wantDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func Test_localPublicAccessDriver_ShallNotRemoveStateDir(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	name := filepath.Base(stateDir)

	removeReq := &volume.RemoveRequest{Name: name}

	// Act
	err := uut.Remove(removeReq)
	assert.Nil(t, err)

	// Assert
	_, err = os.Stat(stateDir)
	assert.Nil(t, err)
}

func Test_localPublicAccessDriver_ShallRemoveInternal(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	name := "test"
	createReq := &volume.CreateRequest{Name: name}
	err := uut.Create(createReq)
	removeReq := &volume.RemoveRequest{Name: name}

	// Act
	err = uut.Remove(removeReq)
	assert.Nil(t, err)
	gotList, err := uut.List()
	assert.Nil(t, err)

	// Assert
	wantVolumes := []*volume.Volume{}
	wantRes := &volume.ListResponse{Volumes: wantVolumes}
	assert.EqualValues(t, wantRes.Volumes, gotList.Volumes)

}

func Test_localPublicAccessDriver_Capabilities(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)

	// Act
	got := uut.Capabilities()

	// Assert
	want := createLocalCapabilitiesResponse()
	assert.EqualValues(t, want, got)
}

func Test_localPublicAccessDriver_Create(t *testing.T) {
	// Arrange
	name := "test"
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	req := &volume.CreateRequest{Name: name}

	// Act
	err := uut.Create(req)

	// Assert
	assert.Nil(t, err)
	gotFileInfo, err := os.Stat(stateDir)
	assert.Nil(t, err)
	assertUserSettings(t, gotFileInfo, stateDirUser)
	stateFilePath := path.Join(stateDir, publicAccessStateFile)
	assertStateFile(t, stateFilePath, name, stateDir)
}

func Test_localPublicAccessDriver_List_FromState(t *testing.T) {
	// Arrange
	name := "test"
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uutCreate := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	req := &volume.CreateRequest{Name: name}
	err := uutCreate.Create(req)
	if err != nil {
		t.Errorf("Failed to create a new driver: %v", err)
	}

	// Act
	uut := newLocalPublicAccessDriver(stateDir, stateDir, stateDirUser)
	res, err := uut.List()

	// Assert
	assert.Nil(t, err)
	wantVolumes := []*volume.Volume{{
		Name:       name,
		Mountpoint: stateDir,
	}}
	wantRes := &volume.ListResponse{Volumes: wantVolumes}
	assert.EqualValues(t, wantRes.Volumes, res.Volumes)
}

func assertStateFile(t *testing.T, stateFilePath string, name string, stateDir string) {
	_, err := os.Stat(stateFilePath)
	if os.IsNotExist(err) {
		t.Error("Expected to have a state file but got none")
	}
	content, err := ioutil.ReadFile(stateFilePath)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	var s saveData
	err = json.Unmarshal(content, &s)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	mountPoint := s.State[name]
	if mountPoint != stateDir {
		t.Errorf("Expected state of '%s' to be '%s' but got '%s'", name, stateDir, mountPoint)
	}
}
