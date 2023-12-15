// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"u-control/uc-aom/internal/aom/docker"

	"github.com/docker/docker/errdefs"
)

var dockerClient *docker.DockerClientMock
var dockerCompose *docker.ComposeMock

func createUut() *docker.StackService {
	dockerClient = &docker.DockerClientMock{}
	dockerCompose = &docker.ComposeMock{}
	dockerWrapper := docker.NewStackService(dockerClient, dockerCompose)
	return dockerWrapper
}

func TestListAllStackContainers(t *testing.T) {
	// arrange
	uut := createUut()

	// act
	uut.ListAllStackContainers("Test AddOn")

	// assert
	expectedLabelFilter := "com.docker.compose.project=testaddon"
	filters := dockerClient.CalledListContainerOptions.Filters
	actualLabelFilter := filters.Get("label")[0]
	if actualLabelFilter != expectedLabelFilter {
		t.Errorf("Expected filter label to be %s but got %s", expectedLabelFilter, actualLabelFilter)
	}
}

func TestInspectContainer(t *testing.T) {
	// arrange
	uut := createUut()

	// act
	info, err := uut.InspectContainer("abc")
	if err != nil {
		t.Logf("failed inspect container: %v", err)
	}

	// assert
	expectedStatus := "running"
	if info.State.Status != expectedStatus {
		t.Errorf("expected status to be '%s' but got '%s'", expectedStatus, info.State.Status)
	}

	expectedHealthStatus := "healthy"
	if info.State.Health.Status != expectedHealthStatus {
		t.Errorf("expected health status to be '%s' but got '%s'", expectedHealthStatus, info.State.Health.Status)
	}

	expectedEnvParams := []string{"param1=abc", "param2=xyz"}
	if len(info.Config.Env) != len(expectedEnvParams) {
		t.Errorf("expected '%d' env params but got '%d'", len(expectedEnvParams), len(info.Config.Env))
	}

	for i, param := range expectedEnvParams {
		envParam := info.Config.Env[i]
		if param != envParam {
			t.Errorf("expected param to be '%s' but got '%s'", param, info.Config.Env[i])
		}
	}
}

func TestDeleteDockerImageOneRef(t *testing.T) {
	// arrange
	uut := createUut()
	ref := "test:1.0.0"

	callImageRemove := dockerClient.On("ImageRemove", ref)
	callImageRemove.Return(nil, nil)

	// act
	err := uut.DeleteDockerImages(ref)

	// assert
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	dockerClient.AssertExpectations(t)
}

func TestDeleteDockerImageTwoRefs(t *testing.T) {
	// arrange
	uut := createUut()
	refs := []string{"test:1.0.0", "test2:1.0.0"}

	for _, ref := range refs {
		callImageRemove := dockerClient.On("ImageRemove", ref)
		callImageRemove.Return(nil, nil)
	}

	// act
	err := uut.DeleteDockerImages(refs[0], refs[1])

	// assert
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	dockerClient.AssertExpectations(t)
}

func TestDeleteDockerImageShallContinueIfImageNotFound(t *testing.T) {
	// arrange
	uut := createUut()
	refs := []string{"test:1.0.0", "test2:1.0.0"}

	for _, ref := range refs {
		callImageRemove := dockerClient.On("ImageRemove", ref)
		callImageRemove.Return(nil, errdefs.NotFound(errors.New("image not found")))
	}

	// act
	err := uut.DeleteDockerImages(refs[0], refs[1])
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	// asssert
	dockerClient.AssertExpectations(t)
}

func TestDeleteDockerImageShallBreakDirectly(t *testing.T) {
	// arrange
	uut := createUut()
	refs := []string{"test:1.0.0", "test2:1.0.0"}

	callImageRemove := dockerClient.On("ImageRemove", refs[0])
	expectedError := errors.New("error in image remove")
	callImageRemove.Return(nil, expectedError)

	// act
	err := uut.DeleteDockerImages(refs[0], refs[1])

	// assert
	if err == nil {
		t.Fatalf("Error is nil, want: %v", expectedError)
	}

	dockerClient.AssertExpectations(t)
}

type TypeTestRemoveUnusedVolumes struct {
	testCaseName    string
	volumeNames     []string
	returnArguments error
}

func TestRemoveUnusedVolumes(t *testing.T) {
	testCases := []TypeTestRemoveUnusedVolumes{
		{
			testCaseName:    "Single Volume",
			volumeNames:     []string{"test-volume"},
			returnArguments: nil,
		},
		{
			testCaseName:    "Shall Not Return Err If Not Found",
			volumeNames:     []string{"test-volume"},
			returnArguments: errdefs.NotFound(errors.New("volume not found")),
		},
		{
			testCaseName:    "Shall Not Return Err If Still In Use",
			volumeNames:     []string{"test-volume"},
			returnArguments: errdefs.Conflict(errors.New("volume is in use")),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.testCaseName, func(t *testing.T) {
			// arrange
			uut := createUut()
			stackName := "test"
			for _, volumeName := range testCase.volumeNames {
				expectedVolumeName := fmt.Sprintf("%s_%s", stackName, volumeName)
				callVolumesRemove := dockerClient.On("VolumeRemove", expectedVolumeName)
				callVolumesRemove.Return(testCase.returnArguments)
			}

			// act
			err := uut.RemoveUnusedVolumes(stackName, testCase.volumeNames...)

			// assert
			if err != nil {
				t.Fatalf("Error %v", err)
			}

			dockerClient.AssertExpectations(t)
		})
	}
}

func TestImportDockerImage(t *testing.T) {
	// arrange
	uut := createUut()
	image := []byte("testimage")

	expectedInput := bytes.NewReader(image)

	callImagesImport := dockerClient.On("ImageLoad", expectedInput)
	callImagesImport.Return(nil, nil)

	// act
	input := bytes.NewReader(image)
	err := uut.ImportDockerImage(input)

	// assert
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	dockerClient.AssertExpectations(t)
}

var composeYaml = `
version: "2"
services:
  ucaomtest-running:
    container_name: uc-addon-status-running
    image: alpine
    stdin_open: true
    tty: true
`

func TestCreateStack(t *testing.T) {
	// arrange
	uut := createUut()
	name := "MyTest Stack Name"
	expectedName := "myteststackname"
	expectedServiceName := "ucaomtest-running"

	// act
	err := uut.CreateStackWithDockerCompose(name, composeYaml)

	// assert
	if err != nil {
		t.Fatalf("Error %v", err)
	}

	if dockerCompose.Project.Name != expectedName {
		t.Errorf("Expected project name to be %s but got %s", expectedName, dockerCompose.Project.Name)
	}

	serviceNames := dockerCompose.Project.ServiceNames()
	if serviceNames[0] != expectedServiceName {
		t.Errorf("Expected service name to be %s but got %s", expectedServiceName, serviceNames[0])
	}
}

func TestDeleteStack(t *testing.T) {
	// arrange
	uut := createUut()
	name := "MyTest Stack Name"
	expectedName := "myteststackname"
	callKill := dockerCompose.On("Down", expectedName)
	callKill.Return(nil)

	// act
	err := uut.DeleteAddOnStack(name)

	// assert
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	dockerCompose.AssertExpectations(t)
}

func TestStartStack(t *testing.T) {
	// Arrange
	uut := createUut()
	name := "Test Start Stack Name"
	expectedName := "teststartstackname"
	dockerCompose.On("Start", expectedName).Return(nil)

	// Act
	err := uut.StartupStackNonBlocking(name)

	// Assert
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	dockerCompose.AssertExpectations(t)
}

func TestStopStack(t *testing.T) {
	// Arrange
	uut := createUut()
	name := "Test Start Stack Name"
	expectedName := "teststartstackname"
	dockerCompose.On("Stop", expectedName).Return(nil)

	// Act
	err := uut.StopStack(name)

	// Assert
	if err != nil {
		t.Fatalf("Error %v", err)
	}
	dockerCompose.AssertExpectations(t)
}
