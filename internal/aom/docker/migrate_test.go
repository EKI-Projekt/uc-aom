// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"os"
	"path"
	"testing"
	"u-control/uc-aom/internal/aom/docker/v0_1"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer"
	"u-control/uc-aom/internal/aom/yaml"
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newMockPortainerClientService() (portainer.PortainerClientService, error) {
	return &mockPortainerClientService{}, nil
}

type mockPortainerClientService struct {
	mock.Mock
}

func (m *mockPortainerClientService) DeleteAddOnStack(stackname string) error {
	args := m.Called(stackname)
	return args.Error(0)
}

func (m *mockPortainerClientService) Logout() error {
	args := m.Called()
	return args.Error(0)
}

func Test_stackMigrator_MigrateStack(t *testing.T) {
	// arrange
	stackName := "uc-addon-test"
	normalizedPortainerStackName := portainer.NormalizeName(stackName)
	volumeName := "data"
	manifest := &model.Root{
		ManifestVersion: model.ValidManifestVersion,
		Version:         "1.0.0-1",
		Title:           "title",
		Description:     "Description",
		Environments: map[string]*model.Environment{
			"uc-addon-test": model.NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{
				volumeName: {},
			}),
		},
	}

	mockStackService := &MockStackService{}
	portainerVolumeName := normalizedPortainerStackName + "_" + volumeName
	newStackName := stackName + "_" + volumeName

	portainerMountPoint := t.TempDir()

	os.Mkdir(path.Join(portainerMountPoint, "logDir"), 0666)
	os.Create(path.Join(portainerMountPoint, "logDir", "tmpfile.txt"))
	os.Create(path.Join(portainerMountPoint, "tmpfile.txt"))

	mockStackService.On("VolumeInspect", portainerVolumeName).Return(types.Volume{
		Mountpoint: portainerMountPoint,
	}, nil)
	newStackMountPoint := t.TempDir()
	mockStackService.On("VolumeInspect", newStackName).Return(types.Volume{
		Mountpoint: newStackMountPoint,
	}, nil)

	mockStackService.On("CreateStackWithoutStartWithDockerCompose", stackName, mock.AnythingOfType("string")).Return(nil)
	mockStackService.On("RemoveUnusedVolumes", normalizedPortainerStackName, []string{volumeName}).Return(nil)

	mockPortainerClient := &mockPortainerClientService{}
	mockPortainerClient.On("DeleteAddOnStack", stackName).Return(nil)
	mockPortainerClient.On("Logout").Return(nil)

	m := &stackMigrator{
		stackService: mockStackService,
		connectToPortainer: func() (portainer.PortainerClientService, error) {
			return mockPortainerClient, nil
		},
	}

	// act
	err := m.MigrateStack(stackName, v0_1.StackVersion, manifest)
	assert.NoError(t, err)

	// assert

	_, err = os.Stat(portainerMountPoint)
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(portainerMountPoint, "logDir"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(path.Join(portainerMountPoint, "tmpfile.txt"))
	assert.ErrorIs(t, err, os.ErrNotExist)

	_, err = os.Stat(newStackMountPoint)
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(newStackMountPoint, "logDir"))
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(newStackMountPoint, "tmpfile.txt"))
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(newStackMountPoint, "logDir", "tmpfile.txt"))
	assert.NoError(t, err)

	mockStackService.AssertExpectations(t)
	mockPortainerClient.AssertExpectations(t)

}

func Test_stackMigrator_MigrateStackWithFilesFromDockerImage(t *testing.T) {
	// arrange
	stackName := "uc-addon-test"
	normalizedPortainerStackName := portainer.NormalizeName(stackName)
	volumeName := "data"
	manifest := &model.Root{
		ManifestVersion: model.ValidManifestVersion,
		Version:         "1.0.0-1",
		Title:           "title",
		Description:     "Description",
		Environments: map[string]*model.Environment{
			"uc-addon-test": model.NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{
				volumeName: {},
			}),
		},
	}

	mockStackService := &MockStackService{}
	portainerVolumeName := normalizedPortainerStackName + "_" + volumeName
	newStackName := stackName + "_" + volumeName

	portainerMountPoint := t.TempDir()

	os.Mkdir(path.Join(portainerMountPoint, "logDir"), 0666)
	os.Create(path.Join(portainerMountPoint, "logDir", "tmpfile.txt"))
	os.Create(path.Join(portainerMountPoint, "tmpfile.txt"))

	mockStackService.On("VolumeInspect", portainerVolumeName).Return(types.Volume{
		Mountpoint: portainerMountPoint,
	}, nil)

	newStackMountPoint := t.TempDir()
	// Create a file in the new mount point
	// This can happen if there is a file in the Docker image in the same place where the volume is mapped.
	os.Create(path.Join(newStackMountPoint, "tmpfile.txt"))

	mockStackService.On("VolumeInspect", newStackName).Return(types.Volume{
		Mountpoint: newStackMountPoint,
	}, nil)

	mockStackService.On("CreateStackWithoutStartWithDockerCompose", stackName, mock.AnythingOfType("string")).Return(nil)
	mockStackService.On("RemoveUnusedVolumes", normalizedPortainerStackName, []string{volumeName}).Return(nil)

	mockPortainerClient := &mockPortainerClientService{}
	mockPortainerClient.On("DeleteAddOnStack", stackName).Return(nil)
	mockPortainerClient.On("Logout").Return(nil)

	m := &stackMigrator{
		stackService: mockStackService,
		connectToPortainer: func() (portainer.PortainerClientService, error) {
			return mockPortainerClient, nil
		},
	}

	// act
	err := m.MigrateStack(stackName, v0_1.StackVersion, manifest)
	assert.NoError(t, err)

	// assert

	_, err = os.Stat(portainerMountPoint)
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(portainerMountPoint, "logDir"))
	assert.ErrorIs(t, err, os.ErrNotExist)
	_, err = os.Stat(path.Join(portainerMountPoint, "tmpfile.txt"))
	assert.ErrorIs(t, err, os.ErrNotExist)

	_, err = os.Stat(newStackMountPoint)
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(newStackMountPoint, "logDir"))
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(newStackMountPoint, "tmpfile.txt"))
	assert.NoError(t, err)
	_, err = os.Stat(path.Join(newStackMountPoint, "logDir", "tmpfile.txt"))
	assert.NoError(t, err)

	mockStackService.AssertExpectations(t)
	mockPortainerClient.AssertExpectations(t)

}

func Test_stackMigrator_MigrateStackWithSettings(t *testing.T) {
	// Arrange
	stackName := "uc-addon-test"
	normalizedPortainerStackName := portainer.NormalizeName(stackName)

	migratedManifest := &model.Root{
		ManifestVersion: model.ValidManifestVersion,
		Version:         "1.0.0-1",
		Title:           "title",
		Description:     "Description",
		Services: map[string]*model.Service{
			"testservice": {
				Type: "docker-compose",
				Config: map[string]interface{}{
					"environment": map[string]interface{}{
						"param1": "aaa",
					},
				},
			},
		},
	}

	settingsForMigration := []*model.Setting{
		model.NewSettings("param1", "param1", false).WithTextBoxValue("bbb"),
	}

	mockStackService := &MockStackService{}
	var gotDockerCompose string
	mockStackService.On("CreateStackWithoutStartWithDockerCompose", stackName, mock.AnythingOfType("string")).Run(func(args mock.Arguments) {
		gotDockerCompose = args.String(1)
	}).Return(nil)

	mockPortainerClient := &mockPortainerClientService{}
	mockPortainerClient.On("DeleteAddOnStack", stackName).Return(nil)
	mockPortainerClient.On("Logout").Return(nil)

	mockStackService.On("RemoveUnusedVolumes", normalizedPortainerStackName, []string{}).Return(nil)

	m := &stackMigrator{
		stackService: mockStackService,
		connectToPortainer: func() (portainer.PortainerClientService, error) {
			return mockPortainerClient, nil
		},
	}

	// Act
	err := m.MigrateStack(stackName, v0_1.StackVersion, migratedManifest, settingsForMigration...)
	assert.NoError(t, err)

	// Assert
	wantManifest := &model.Root{
		ManifestVersion: model.ValidManifestVersion,
		Version:         "1.0.0-1",
		Title:           "title",
		Description:     "Description",
		Services: map[string]*model.Service{
			"testservice": {
				Type: "docker-compose",
				Config: map[string]interface{}{
					"environment": map[string]interface{}{
						"param1": "bbb",
					},
				},
			},
		},
	}

	wantdockerCompose, err := yaml.GetDockerComposeFromManifest(wantManifest)
	assert.NoError(t, err)
	assert.Equal(t, wantdockerCompose, gotDockerCompose)

	mockStackService.AssertExpectations(t)
	mockPortainerClient.AssertExpectations(t)
}
