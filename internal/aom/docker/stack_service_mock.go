// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package docker

import (
	"context"
	"io"

	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/mock"
)

type MockStackService struct {
	mock.Mock
}

func (r *MockStackService) CreateStackWithDockerCompose(stackName string, dockerCompose string) error {
	args := r.Called(stackName, dockerCompose)
	return args.Error(0)
}
func (r *MockStackService) CreateStackWithoutStartWithDockerCompose(stackName string, dockerCompose string) error {
	args := r.Called(stackName, dockerCompose)
	return args.Error(0)
}

func (r *MockStackService) ListAllStackContainers(stackName string) ([]types.Container, error) {
	args := r.Called(stackName)
	return args.Get(0).([]types.Container), args.Error(1)
}

func (r *MockStackService) DeleteAddOnStack(stackName string) error {
	args := r.Called(stackName)
	return args.Error(0)
}

func (r *MockStackService) DeleteDockerImages(images ...string) error {
	args := r.Called(images)
	return args.Error(0)
}

func (r *MockStackService) ImportDockerImage(dockerImage io.Reader) error {
	args := r.Called(dockerImage)
	return args.Error(0)
}

func (r *MockStackService) RemoveUnusedVolumes(stackName string, volumeNames ...string) error {
	args := r.Called(stackName, volumeNames)
	return args.Error(0)
}

func (r MockStackService) StartupStackNonBlocking(stackName string) error {
	args := r.Called(stackName)
	return args.Error(0)
}

func (r MockStackService) StopStack(stackName string) error {
	args := r.Called(stackName)
	return args.Error(0)
}

func (r *MockStackService) InspectContainer(containerId string) (*ContainerInfo, error) {
	args := r.Called(containerId)
	return args.Get(0).(*ContainerInfo), args.Error(1)
}

func (r *MockStackService) VolumeInspect(volumeID string) (types.Volume, error) {
	args := r.Called(volumeID)
	return args.Get(0).(types.Volume), args.Error(1)
}

type DockerClientMock struct {
	mock.Mock
	CalledListContainerOptions types.ContainerListOptions
}

func (d *DockerClientMock) ContainerList(ctx context.Context, options types.ContainerListOptions) ([]types.Container, error) {
	d.CalledListContainerOptions = options
	return nil, nil
}

func (d *DockerClientMock) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	config := container.Config{
		Env: []string{"param1=abc", "param2=xyz"},
	}
	state := types.ContainerState{
		Status:     "running",
		Running:    false,
		Paused:     false,
		Restarting: false,
		OOMKilled:  false,
		Dead:       false,
		Pid:        0,
		ExitCode:   0,
		Error:      "",
		StartedAt:  "",
		FinishedAt: "",
		Health: &types.Health{
			Status:        "healthy",
			FailingStreak: 0,
		},
	}
	info := types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{},
		Config:            &config,
	}
	info.State = &state
	return info, nil
}

func (d *DockerClientMock) ImageRemove(ctx context.Context, imageID string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error) {
	args := d.Called(imageID)
	return nil, args.Error(1)
}

func (d *DockerClientMock) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	args := d.Called(volumeID)
	return args.Error(0)
}

func (d *DockerClientMock) VolumeInspect(ctx context.Context, volumeID string) (types.Volume, error) {
	args := d.Called(volumeID)
	return types.Volume{}, args.Error(1)
}

func (d *DockerClientMock) ImageLoad(ctx context.Context, input io.Reader, quiet bool) (types.ImageLoadResponse, error) {
	args := d.Called(input)
	return types.ImageLoadResponse{}, args.Error(1)
}

type ComposeMock struct {
	mock.Mock
	Project *composeTypes.Project
}

func (c *ComposeMock) Create(ctx context.Context, project *composeTypes.Project, options api.CreateOptions) error {
	c.Project = project
	return nil
}

func (c *ComposeMock) Up(ctx context.Context, project *composeTypes.Project, options api.UpOptions) error {
	c.Project = project
	return nil
}

func (c *ComposeMock) Down(ctx context.Context, projectName string, options api.DownOptions) error {
	args := c.Called(projectName)
	return args.Error(0)
}

func (c *ComposeMock) Start(ctx context.Context, projectName string, options api.StartOptions) error {
	args := c.Called(projectName)
	return args.Error(0)
}

func (c *ComposeMock) Stop(ctx context.Context, projectName string, options api.StopOptions) error {
	args := c.Called(projectName)
	return args.Error(0)
}
