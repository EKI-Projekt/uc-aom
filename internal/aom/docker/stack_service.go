// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"u-control/uc-aom/internal/aom/config"

	"github.com/compose-spec/compose-go/loader"
	composeTypes "github.com/compose-spec/compose-go/types"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	log "github.com/sirupsen/logrus"
)

const (
	UcAomStackVersionLabel = "com.weidmueller.uc.aom.stack.version"
)

// StackService is a service to deploy, retrieve status and remove addOns
// using the docker client and compose service from docker-compose
type StackService struct {
	cli            DockerClient
	composeService Compose
}

// Stack service API for the stack service implementation
type StackServiceAPI interface {
	CreateStackWithDockerCompose(stackName string, composeAsString string) error
	CreateStackWithoutStartWithDockerCompose(stackName string, dockerCompose string) error
	ListAllStackContainers(stackName string) ([]types.Container, error)
	DeleteAddOnStack(stackName string) error
	DeleteDockerImages(images ...string) error
	ImportDockerImage(image io.Reader) error
	RemoveUnusedVolumes(stackName string, volumeNames ...string) error

	// Return low-level information about a container.
	InspectContainer(ContainerName string) (*ContainerInfo, error)
	StartupStackNonBlocking(stackName string) error
	StopStack(stackName string) error
	VolumeInspect(volumeID string) (types.Volume, error)
}

// Interface for the docker client that is passed into the stack service
type DockerClient interface {
	ContainerList(context context.Context, options types.ContainerListOptions) ([]types.Container, error)
	ContainerInspect(context context.Context, containerName string) (types.ContainerJSON, error)
	ImageRemove(context context.Context, imageName string, options types.ImageRemoveOptions) ([]types.ImageDeleteResponseItem, error)
	VolumeRemove(context context.Context, volumeName string, force bool) error
	VolumeInspect(ctx context.Context, volumeID string) (types.Volume, error)
	ImageLoad(context context.Context, image io.Reader, quiet bool) (types.ImageLoadResponse, error)
}

// Interface for the compose service that is passed into the stack service
type Compose interface {
	Create(ctx context.Context, project *composeTypes.Project, options api.CreateOptions) error
	Up(context context.Context, project *composeTypes.Project, options api.UpOptions) error
	Down(context context.Context, projectName string, options api.DownOptions) error
	Start(ctx context.Context, projectName string, options api.StartOptions) error
	Stop(ctx context.Context, projectName string, options api.StopOptions) error
}

type ContainerInfo struct {
	Config *Config `json:"Config"`
	State  *State  `json:"State"`
}

type State struct {
	Dead       bool    `json:"Dead"`
	Error      string  `json:"Error"`
	ExitCode   int     `json:"ExitCode"`
	FinishedAt string  `json:"FinishedAt"`
	OOMKilled  bool    `json:"OOMKilled"`
	Paused     bool    `json:"Paused"`
	Pid        int     `json:"Pid"`
	Restarting bool    `json:"Restarting"`
	Running    bool    `json:"Running"`
	StartedAt  string  `json:"StartedAt"`
	Status     string  `json:"Status"`
	Health     *Health `json:"Health,omitempty"`
}

type Health struct {
	Status        string `json:"Status"`
	FailingStreak int    `json:"FailingStreak"`
}

type Config struct {
	Env []string `json:"Env"`
}

// NewStackService creates a new service to deploy, retrieve status and delete addOns
// using the docker client and compose service from docker-compose
func NewStackService(dockerClient DockerClient, compose Compose) *StackService {
	service := &StackService{
		cli:            dockerClient,
		composeService: compose,
	}
	return service
}

// Create stack with the given stackName and docker compose content.
func (s *StackService) CreateStackWithDockerCompose(stackName string, dockerCompose string) error {
	project, err := getProjectFromConfig(stackName, dockerCompose)
	if err != nil {
		return err
	}

	options := getUpOptions(project)
	return s.composeService.Up(context.Background(), project, options)
}

func (s *StackService) CreateStackWithoutStartWithDockerCompose(stackName string, dockerCompose string) error {
	project, err := getProjectFromConfig(stackName, dockerCompose)
	if err != nil {
		return err
	}

	options := getCreateOptions()
	return s.composeService.Create(context.Background(), project, options)
}

func buildConfigDetails(yaml string, env map[string]string) composeTypes.ConfigDetails {
	return buildConfigDetailsMultipleFiles(env, yaml)
}

func buildConfigDetailsMultipleFiles(env map[string]string, yamls ...string) composeTypes.ConfigDetails {
	return composeTypes.ConfigDetails{
		ConfigFiles: buildConfigFiles(yamls),
		Environment: env,
	}
}

func buildConfigFiles(yamls []string) []composeTypes.ConfigFile {
	configFiles := make([]composeTypes.ConfigFile, len(yamls))
	for i, yaml := range yamls {
		configFiles[i] = composeTypes.ConfigFile{
			Content: []byte(yaml),
		}
	}
	return configFiles
}

func getProjectFromConfig(projectName string, yaml string) (*composeTypes.Project, error) {
	project, err := loader.Load(buildConfigDetails(yaml, make(map[string]string)), func(options *loader.Options) {

		// Skip consistency check and validation on the device because of performance reasons
		options.SkipConsistencyCheck = true
		options.SkipValidation = true

		// skip conversion because the device use always linux based OS
		options.ConvertWindowsPaths = false

		options.SetProjectName(projectName, true)
	})

	if err != nil {
		return &composeTypes.Project{}, err
	}

	for i, s := range project.Services {
		s.CustomLabels = map[string]string{
			api.ProjectLabel:     project.Name,
			api.ServiceLabel:     s.Name,
			api.VersionLabel:     api.ComposeVersion,
			api.WorkingDirLabel:  project.WorkingDir,
			api.ConfigFilesLabel: strings.Join(project.ComposeFiles, ","),
			api.OneoffLabel:      "False", // default, will be overridden by `run` command
		}
		project.Services[i] = s
	}
	addUcAomVersionLabel(project)

	return project, nil
}

func getUpOptions(project *composeTypes.Project) api.UpOptions {
	options := api.UpOptions{
		Create: getCreateOptions(),
		Start:  getStartOptionsWithProject(project),
	}
	return options
}

func getCreateOptions() api.CreateOptions {
	timeout := 10 * time.Second
	opt := api.CreateOptions{
		Services:             make([]string, 0),
		RemoveOrphans:        true,
		IgnoreOrphans:        false,
		Recreate:             "diverged",
		RecreateDependencies: "diverged",
		Inherit:              false,
		Timeout:              &timeout,
		QuietPull:            false,
	}
	return opt
}

func getStartOptionsWithProject(project *composeTypes.Project) api.StartOptions {
	opt := api.StartOptions{
		Project:      project,
		Attach:       nil,
		AttachTo:     nil,
		CascadeStop:  false,
		ExitCodeFrom: "",
		Wait:         false,
	}
	return opt
}

// Delete the docker stack by the given stackName.
// Note: stackName is normalized based on NormalizeName function in this package.
func (s *StackService) DeleteAddOnStack(stackName string) error {
	projectName := normalizeStackName(stackName)
	err := s.composeService.Down(context.Background(), projectName, api.DownOptions{})
	return err
}

// List all containers of the corresponding stackName.
// Note: stackName is normalized based on NormalizeName function in this package.
func (s *StackService) ListAllStackContainers(stackName string) ([]types.Container, error) {
	projectName := normalizeStackName(stackName)
	containers, err := s.cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filters.NewArgs(projectFilter(projectName)),
		All:     true,
	})

	return containers, err
}

func projectFilter(projectName string) filters.KeyValuePair {
	return filters.Arg("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))
}

// InspectContainer returns docker container inspect information
func (s *StackService) InspectContainer(containerId string) (*ContainerInfo, error) {
	containerJSONStruct, err := s.cli.ContainerInspect(context.Background(), containerId)
	if err != nil {
		return nil, err
	}

	containerJSON, err := json.Marshal(containerJSONStruct)
	if err != nil {
		return nil, err
	}

	containerInfo := &ContainerInfo{}
	err = json.Unmarshal(containerJSON, &containerInfo)
	if err != nil {
		return nil, err
	}
	return containerInfo, nil
}

// Start the referenced stack.
// Method is not waiting until the stack is started.
func (s *StackService) StartupStackNonBlocking(stackName string) error {
	projectName := normalizeStackName(stackName)
	startOption := api.StartOptions{
		Project:      nil,
		Attach:       nil,
		AttachTo:     nil,
		CascadeStop:  false,
		ExitCodeFrom: "",
		Wait:         false,
	}
	return s.composeService.Start(context.Background(), projectName, startOption)
}

// Stop the referenced stack.
func (s *StackService) StopStack(stackName string) error {
	projectName := normalizeStackName(stackName)
	return s.composeService.Stop(context.Background(), projectName, api.StopOptions{})
}

// VolumeInspect returns docker volume inspect information
func (s *StackService) VolumeInspect(volumeId string) (types.Volume, error) {
	return s.cli.VolumeInspect(context.Background(), volumeId)
}

// Delete the docker images via their full reference (<repository>:<tag>) returns an error on the first failed delete.
func (s *StackService) DeleteDockerImages(references ...string) error {
	for _, ref := range references {
		log.Debugf("Removing Image: %s", ref)
		options := types.ImageRemoveOptions{
			Force:         false,
			PruneChildren: true,
		}
		if _, err := s.cli.ImageRemove(context.Background(), ref, options); err != nil {
			if client.IsErrNotFound(err) {
				continue
			}
			return err
		}
	}
	return nil
}

// Removes any unused volumes referenced by the stack identified by stackName and volume name.
// Returns an error on the first failed delete.
// NOTE: In use volumes are *not* removed.
func (s *StackService) RemoveUnusedVolumes(stackName string, volumeNames ...string) error {
	projectName := normalizeStackName(stackName)
	for _, name := range volumeNames {
		volumeName := createStackScopedVolumeName(projectName, name)
		if err := s.cli.VolumeRemove(context.Background(), volumeName, false); err != nil {
			if errdefs.IsNotFound(err) {
				continue
			}

			if errdefs.IsConflict(err) {
				continue
			}

			return err
		}
	}
	return nil
}

// Import docker image tarball represented by image.
func (s *StackService) ImportDockerImage(input io.Reader) error {
	_, err := s.cli.ImageLoad(context.Background(), input, true)
	return err
}

// Normalize the stackname based on docker compose spec
func normalizeStackName(stackName string) string {
	return loader.NormalizeProjectName(stackName)
}

func addUcAomVersionLabel(project *composeTypes.Project) {

	for i, service := range project.Services {
		service.Labels = service.Labels.Add(config.UcAomVersionLabel, config.UcAomVersion)
		service.Labels = service.Labels.Add(UcAomStackVersionLabel, StackVersion)
		project.Services[i] = service
	}

	for i, volume := range project.Volumes {
		volume.Labels = volume.Labels.Add(config.UcAomVersionLabel, config.UcAomVersion)
		volume.Labels = volume.Labels.Add(UcAomStackVersionLabel, StackVersion)
		project.Volumes[i] = volume
	}

	for i, network := range project.Networks {
		network.Labels = network.Labels.Add(config.UcAomVersionLabel, config.UcAomVersion)
		network.Labels = network.Labels.Add(UcAomStackVersionLabel, StackVersion)
		project.Networks[i] = network
	}
}

func createStackScopedVolumeName(projectName string, volumename string) string {
	volumeName := fmt.Sprintf("%s_%s", projectName, volumename)
	return volumeName
}
