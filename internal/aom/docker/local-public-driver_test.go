// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/stretchr/testify/assert"
)

func Test_localPublicDriver_ShallCreateStateDir(t *testing.T) {
	stateDir := t.TempDir()
	os.RemoveAll(stateDir)
	stateDirUser := createTestStateDirUser()
	// Act
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)

	// Assert
	assert.NotNil(t, uut)
	gotFileInfo, err := os.Stat(stateDir)
	assert.Nil(t, err)

	assertUserSettings(t, gotFileInfo, stateDirUser)

}

func Test_localPublicDriver_Get_ShallReturnErrorIfNotExist(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)
	req := &volume.GetRequest{Name: "test"}

	// Act
	res, err := uut.Get(req)

	// Assert
	assert.Nil(t, res)
	assert.Error(t, err)

}

func Test_localPublicDriver_Get_ShallReturnIfExist(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)
	createReq := &volume.CreateRequest{Name: "test"}
	err := uut.Create(createReq)
	assert.Nil(t, err)

	// Act
	req := &volume.GetRequest{Name: createReq.Name}
	gotRes, err := uut.Get(req)

	// Assert
	assert.Nil(t, err)
	wantRes := &volume.GetResponse{Volume: &volume.Volume{Name: createReq.Name, Mountpoint: filepath.Join(stateDir, createReq.Name)}}
	assert.Equal(t, *wantRes.Volume, *gotRes.Volume)
}

func Test_localPublicDriver_List_ShallReturnEmptyListIfNoneCreated(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)

	// Act
	gotRes, err := uut.List()

	// Assert
	assert.Nil(t, err)

	wantRes := &volume.ListResponse{Volumes: make([]*volume.Volume, 0)}
	assert.Equal(t, *wantRes, *gotRes)
}

func Test_localPublicDriver_List_ShallReturnIfExist(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)
	createReq := &volume.CreateRequest{Name: "test"}
	err := uut.Create(createReq)
	assert.Nil(t, err)

	// Act
	gotRes, err := uut.List()

	// Assert
	assert.Nil(t, err)
	wantVolumes := []*volume.Volume{{
		Name:       createReq.Name,
		Mountpoint: filepath.Join(stateDir, createReq.Name),
	}}
	wantRes := &volume.ListResponse{Volumes: wantVolumes}
	assert.EqualValues(t, wantRes.Volumes, gotRes.Volumes)
}

func Test_localPublicDriver_Create(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)
	req := &volume.CreateRequest{Name: "test"}

	// Act
	err := uut.Create(req)

	// Assert
	assert.Nil(t, err)

	wantDir := filepath.Join(stateDir, req.Name)
	gotFileInfo, err := os.Stat(wantDir)
	assert.Nil(t, err)

	assertUserSettings(t, gotFileInfo, stateDirUser)
}

func Test_localPublicDriver_Create_NotCreateTwice(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)
	req := &volume.CreateRequest{Name: "test"}

	// Act
	err := uut.Create(req)
	assert.Nil(t, err)
	err = uut.Create(req)

	// Assert
	assert.Error(t, err)
}

func Test_localPublicDriver_Remove(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)
	createReq := &volume.CreateRequest{Name: "test"}
	removeReq := &volume.RemoveRequest{Name: createReq.Name}

	// Act
	err := uut.Create(createReq)
	assert.Nil(t, err)
	err = uut.Remove(removeReq)
	assert.Nil(t, err)

	// Assert
	wantDir := filepath.Join(stateDir, createReq.Name)
	_, err = os.Stat(wantDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func Test_localPublicDriver_Remove_Twice(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)
	createReq := &volume.CreateRequest{Name: "test"}
	removeReq := &volume.RemoveRequest{Name: createReq.Name}
	err := uut.Create(createReq)
	assert.Nil(t, err)

	// Act
	err = uut.Remove(removeReq)
	assert.Nil(t, err)
	err = uut.Remove(removeReq)
	assert.Nil(t, err)

	// Assert
	wantDir := filepath.Join(stateDir, createReq.Name)
	_, err = os.Stat(wantDir)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func Test_localPublicDriver_Capabilities(t *testing.T) {
	// Arrange
	stateDir := t.TempDir()
	stateDirUser := createTestStateDirUser()
	uut := newLocalPublicDriver(stateDir, stateDir, stateDirUser)

	// Act
	got := uut.Capabilities()

	// Assert
	want := createLocalCapabilitiesResponse()
	assert.EqualValues(t, want, got)
}

func createTestStateDirUser() *user.User {
	return &user.User{Uid: "1000", Gid: "1000", Username: "TestUser", Name: "TestUser"}
}

func assertUserSettings(t *testing.T, gotFileInfo fs.FileInfo, stateDirUser *user.User) {
	gotStateDirSys := gotFileInfo.Sys().(*syscall.Stat_t)
	assert.EqualValues(t, stateDirUser.Uid, strconv.FormatUint(uint64(gotStateDirSys.Uid), 10))
	assert.EqualValues(t, stateDirUser.Gid, strconv.FormatUint(uint64(gotStateDirSys.Gid), 10))
}
