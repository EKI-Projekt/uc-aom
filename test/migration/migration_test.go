// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package migration

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/aom/docker"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	sharedConfig "u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/pkg/manifest"
	testhelpers "u-control/uc-aom/test/test-helpers"

	"github.com/stretchr/testify/assert"
)

func TestMigrationv0_3_2_to_v0_5_2(t *testing.T) {
	dockerCli, err := docker.NewDockerCli()
	assert.NoError(t, err)
	dockerClient := dockerCli.Client()
	v0_3_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_3_2/docker-compose.yml"
	dockerComposeUp(t, v0_3_2_composeFilePath)
	testAddOnRepositoryName := "test-uc-addon-status-running"
	containerName := "uc-addon-status-running"
	manifestPath := path.Join(config.UC_AOM_STATE_DIRECTORY, testAddOnRepositoryName, sharedConfig.UcImageManifestFilename)

	ucAopContainerName := "uc-aom-mit-uc-aop-1"
	ucAomContainerName := "uc-aom-mit-uc-aom-1"

	pushTestAddOnToRegistry(t, ucAopContainerName, testAddOnRepositoryName)

	ucAomGrpcAddress := fmt.Sprintf("%s:3800", ucAomContainerName)
	ucAomTestenv, err := testhelpers.NewTestEnvironmentWithGrpcAddr(context.Background(), ucAomGrpcAddress)
	assert.NoError(t, err)
	t.Cleanup(func() {
		ucAomTestenv.CloseConnection()
	})
	bootupWaitTime := 5 * time.Second
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_3_2InstalledAddon := ucAomTestenv.CreateTestAddOnRoutine(t, testAddOnRepositoryName, "0.1.0-1")
	v0_5_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_2/docker-compose.yml"
	dockerComposeUp(t, v0_5_2_composeFilePath)

	v0_5_2AddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_3_2InstalledAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_2AddOnStatus, grpc_api.AddOnStatus_RUNNING)

	t.Cleanup(func() {
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_3_2InstalledAddon)
		dir, _ := os.ReadDir(config.UC_AOM_STATE_DIRECTORY)
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{config.UC_AOM_STATE_DIRECTORY, d.Name()}...))
		}
	})

	// check labels of migrated docker container
	containerInfo, err := dockerClient.ContainerInspect(ucAomTestenv.Ctx, containerName)
	if assert.Contains(t, containerInfo.Config.Labels, config.UcAomVersionLabel) {
		assert.Equal(t, containerInfo.Config.Labels[config.UcAomVersionLabel], "0.5.2")
	}
	if assert.Contains(t, containerInfo.Config.Labels, docker.UcAomStackVersionLabel) {
		assert.Equal(t, containerInfo.Config.Labels[docker.UcAomStackVersionLabel], docker.StackVersion)
	}

	// check migrated manifest version
	manifestAsBytes, err := os.ReadFile(manifestPath)
	assert.NoError(t, err)
	migratedManifestVerserion, err := manifest.UnmarshalManifestVersionFrom(manifestAsBytes)
	assert.NoError(t, err)
	assert.Equal(t, manifest.ValidManifestVersion, migratedManifestVerserion)

}

func TestMigrationv0_3_2_to_v0_5_2_VolumeWithExistingFile(t *testing.T) {
	v0_3_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_3_2/docker-compose.yml"
	dockerComposeUp(t, v0_3_2_composeFilePath)
	testAddOnRepositoryName := "test-uc-addon-update-with-volume"

	ucAopContainerName := "uc-aom-mit-uc-aop-1"
	ucAomContainerName := "uc-aom-mit-uc-aom-1"

	pushTestAddOnToRegistry(t, ucAopContainerName, testAddOnRepositoryName)

	ucAomGrpcAddress := fmt.Sprintf("%s:3800", ucAomContainerName)
	ucAomTestenv, err := testhelpers.NewTestEnvironmentWithGrpcAddr(context.Background(), ucAomGrpcAddress)
	assert.NoError(t, err)
	bootupWaitTime := 5 * time.Second
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_3_2InstalledAddon := ucAomTestenv.CreateTestAddOnRoutine(t, testAddOnRepositoryName, "0.1.0-1")
	v0_5_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_2/docker-compose.yml"
	dockerComposeUp(t, v0_5_2_composeFilePath)

	v0_5_2AddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_3_2InstalledAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_2AddOnStatus, grpc_api.AddOnStatus_RUNNING)

	t.Cleanup(func() {
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_3_2InstalledAddon)
		dir, _ := os.ReadDir(config.UC_AOM_STATE_DIRECTORY)
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{config.UC_AOM_STATE_DIRECTORY, d.Name()}...))
		}
	})
}
func TestMigrationv0_3_2_to_v0_5_0_to_v0_5_2_VolumeWithExistingFile(t *testing.T) {
	// Due to a bug in v0_5_0 the migration crashes of docker stacks created by portainer.
	// This test verifies that v0_5_2 restarts and successfully completes the migration after the crash in v0_5_0.
	v0_3_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_3_2/docker-compose.yml"
	dockerComposeUp(t, v0_3_2_composeFilePath)
	testAddOnRepositoryName := "test-uc-addon-update-with-volume"

	ucAopContainerName := "uc-aom-mit-uc-aop-1"
	ucAomContainerName := "uc-aom-mit-uc-aom-1"

	pushTestAddOnToRegistry(t, ucAopContainerName, testAddOnRepositoryName)

	ucAomGrpcAddress := fmt.Sprintf("%s:3800", ucAomContainerName)
	ucAomTestenv, err := testhelpers.NewTestEnvironmentWithGrpcAddr(context.Background(), ucAomGrpcAddress)
	assert.NoError(t, err)
	bootupWaitTime := 5 * time.Second
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_3_2InstalledAddon := ucAomTestenv.CreateTestAddOnRoutine(t, testAddOnRepositoryName, "0.1.0-1")
	v0_5_0_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_0/docker-compose.yml"
	dockerComposeUp(t, v0_5_0_composeFilePath)

	t.Logf("Wait %s for bootup v0_5_0 to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_5_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_2/docker-compose.yml"
	dockerComposeUp(t, v0_5_2_composeFilePath)

	t.Logf("Wait %s for bootup v0_5_2 to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_5_2AddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_3_2InstalledAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_2AddOnStatus, grpc_api.AddOnStatus_RUNNING)

	t.Cleanup(func() {
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_3_2InstalledAddon)
		dir, _ := os.ReadDir(config.UC_AOM_STATE_DIRECTORY)
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{config.UC_AOM_STATE_DIRECTORY, d.Name()}...))
		}
	})
}

func TestMigrationv0_3_2_to_v0_5_2_WithSettings(t *testing.T) {
	// Arrange
	v0_3_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_3_2/docker-compose.yml"
	dockerComposeUp(t, v0_3_2_composeFilePath)
	testAddOnRepositoryName := "test-uc-addon-settings"

	ucAopContainerName := "uc-aom-mit-uc-aop-1"
	ucAomContainerName := "uc-aom-mit-uc-aom-1"

	pushTestAddOnToRegistry(t, ucAopContainerName, testAddOnRepositoryName)

	ucAomGrpcAddress := fmt.Sprintf("%s:3800", ucAomContainerName)
	ucAomTestenv, err := testhelpers.NewTestEnvironmentWithGrpcAddr(context.Background(), ucAomGrpcAddress)
	assert.NoError(t, err)
	bootupWaitTime := 5 * time.Second
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	param1InstallValue := "p1 install"
	param2InstallValue := "abc"
	//param3 is only in 0.2.0-1 version available
	param4InstallValue := "p4 install"

	settings := []*grpc_api.Setting{
		{
			Name: "param1", SettingOneof: &grpc_api.Setting_TextBox{
				TextBox: &grpc_api.TextBox{
					Value: param1InstallValue,
				},
			},
		},
		{
			Name: "param2", SettingOneof: &grpc_api.Setting_DropDownList{
				DropDownList: &grpc_api.DropDownList{
					Elements: []*grpc_api.DropDownItem{
						{
							Value: param2InstallValue, Selected: true,
						},
					},
				},
			},
		},
		{
			Name: "param4", SettingOneof: &grpc_api.Setting_TextBox{
				TextBox: &grpc_api.TextBox{
					Value: param4InstallValue,
				},
			},
		},
	}

	v0_3_2InstalledAddon := ucAomTestenv.CreateTestAddOnRoutine(t, testAddOnRepositoryName, "0.1.0-1", settings...)

	// Act
	v0_5_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_2/docker-compose.yml"
	dockerComposeUp(t, v0_5_2_composeFilePath)

	// Assert
	t.Cleanup(func() {
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_3_2InstalledAddon)
		dir, _ := os.ReadDir(config.UC_AOM_STATE_DIRECTORY)
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{config.UC_AOM_STATE_DIRECTORY, d.Name()}...))
		}
	})

	v0_5_2AddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_3_2InstalledAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_2AddOnStatus, grpc_api.AddOnStatus_RUNNING)

	v0_5_2MigratedInstalledAddon, err := ucAomTestenv.GetInstalledAddOn(v0_3_2InstalledAddon.Name)
	assert.NoError(t, err)

	assert.ElementsMatch(t, v0_3_2InstalledAddon.Settings, v0_5_2MigratedInstalledAddon.Settings)

}

func TestMigrationv0_3_2_to_v0_5_2_Communication(t *testing.T) {
	// Arrange
	v0_3_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_3_2/docker-compose.yml"
	dockerComposeUp(t, v0_3_2_composeFilePath)
	receiverAddOnRepositoryName := "test-uc-addon-communication-receiver"
	senderAddOnRepositoryName := "test-uc-addon-communication-sender"

	ucAopContainerName := "uc-aom-mit-uc-aop-1"
	ucAomContainerName := "uc-aom-mit-uc-aom-1"

	pushTestAddOnToRegistry(t, ucAopContainerName, receiverAddOnRepositoryName)
	pushTestAddOnToRegistry(t, ucAopContainerName, senderAddOnRepositoryName)

	ucAomGrpcAddress := fmt.Sprintf("%s:3800", ucAomContainerName)
	ucAomTestenv, err := testhelpers.NewTestEnvironmentWithGrpcAddr(context.Background(), ucAomGrpcAddress)
	assert.NoError(t, err)
	bootupWaitTime := 5 * time.Second
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_3_2InstalledReceiverAddon := ucAomTestenv.CreateTestAddOnRoutine(t, receiverAddOnRepositoryName, "0.1.0-1")
	v0_3_2InstalledSenderAddon := ucAomTestenv.CreateTestAddOnRoutine(t, senderAddOnRepositoryName, "0.1.0-1")

	// Act
	v0_5_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_2/docker-compose.yml"
	dockerComposeUp(t, v0_5_2_composeFilePath)

	// Assert
	t.Cleanup(func() {
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_3_2InstalledReceiverAddon)
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_3_2InstalledSenderAddon)
		dir, _ := os.ReadDir(config.UC_AOM_STATE_DIRECTORY)
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{config.UC_AOM_STATE_DIRECTORY, d.Name()}...))
		}
	})
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_5_2ReceiverAddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_3_2InstalledReceiverAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_2ReceiverAddOnStatus, grpc_api.AddOnStatus_RUNNING)

	v0_5_2SenderAddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_3_2InstalledSenderAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_2SenderAddOnStatus, grpc_api.AddOnStatus_RUNNING)

}

func TestMigrationv0_4_0_to_v0_5_3(t *testing.T) {
	// The migration should test if file larger than 1MB can be upload via post.
	// Arrange
	v0_4_0_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_4_0/docker-compose.yml"
	dockerComposeUp(t, v0_4_0_composeFilePath)
	testAddOnRepositoryName := "test-uc-addon-provide-ui"

	ucAopContainerName := "uc-aom-mit-uc-aop-1"
	ucAomContainerName := "uc-aom-mit-uc-aom-1"

	pushTestAddOnToRegistry(t, ucAopContainerName, testAddOnRepositoryName)

	ucAomGrpcAddress := fmt.Sprintf("%s:3800", ucAomContainerName)
	ucAomTestenv, err := testhelpers.NewTestEnvironmentWithGrpcAddr(context.Background(), ucAomGrpcAddress)
	assert.NoError(t, err)
	bootupWaitTime := 5 * time.Second
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_4_0_InstalledAddon := ucAomTestenv.CreateTestAddOnRoutine(t, testAddOnRepositoryName, "0.1.0-1")

	f, err := os.CreateTemp(t.TempDir(), "upload-file")
	assert.NoError(t, err)

	size10MB := int64(10 * 1024 * 1024)
	err = f.Truncate(size10MB)
	assert.NoError(t, err)

	// Act
	v0_5_3_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_3/docker-compose.yml"
	dockerComposeUp(t, v0_5_3_composeFilePath)

	// Assert
	t.Cleanup(func() {
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_4_0_InstalledAddon)
		dir, _ := os.ReadDir(config.UC_AOM_STATE_DIRECTORY)
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{config.UC_AOM_STATE_DIRECTORY, d.Name()}...))
		}
	})
	v0_5_3AddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_4_0_InstalledAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_3AddOnStatus, grpc_api.AddOnStatus_RUNNING)

	// Act
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("myFile", f.Name())
	assert.NoError(t, err)
	_, err = io.Copy(fw, f)
	assert.NoError(t, err)

	w.Close()

	resp, err := http.Post(fmt.Sprintf("http://nginx%s/upload", v0_4_0_InstalledAddon.Location), w.FormDataContentType(), &b)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

}

func TestMigrationv0_5_2_to_v0_5_3(t *testing.T) {
	// The migration should test if file larger than 1MB can be upload via post.
	// Arrange
	v0_5_2_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_2/docker-compose-without-portainer.yml"
	dockerComposeUp(t, v0_5_2_composeFilePath)
	testAddOnRepositoryName := "test-uc-addon-provide-ui"

	ucAopContainerName := "uc-aom-mit-uc-aop-1"
	ucAomContainerName := "uc-aom-mit-uc-aom-1"

	pushTestAddOnToRegistry(t, ucAopContainerName, testAddOnRepositoryName)

	ucAomGrpcAddress := fmt.Sprintf("%s:3800", ucAomContainerName)
	ucAomTestenv, err := testhelpers.NewTestEnvironmentWithGrpcAddr(context.Background(), ucAomGrpcAddress)
	assert.NoError(t, err)
	bootupWaitTime := 5 * time.Second
	t.Logf("Wait %s for bootup to complete...", bootupWaitTime)
	time.Sleep(bootupWaitTime)

	v0_5_2_InstalledAddon := ucAomTestenv.CreateTestAddOnRoutine(t, testAddOnRepositoryName, "0.1.0-1")

	f, err := os.CreateTemp(t.TempDir(), "upload-file")
	assert.NoError(t, err)

	size10MB := int64(10 * 1024 * 1024)
	err = f.Truncate(size10MB)
	assert.NoError(t, err)

	// Act
	v0_5_3_composeFilePath := "/go/src/uc-aom/scripts/docker/v0_5_3/docker-compose.yml"
	dockerComposeUp(t, v0_5_3_composeFilePath)

	// Assert
	t.Cleanup(func() {
		ucAomTestenv.DeleteTestAddOnRoutine(t, v0_5_2_InstalledAddon)
		dir, _ := os.ReadDir(config.UC_AOM_STATE_DIRECTORY)
		for _, d := range dir {
			os.RemoveAll(path.Join([]string{config.UC_AOM_STATE_DIRECTORY, d.Name()}...))
		}
	})
	v0_5_3AddOnStatus, err := ucAomTestenv.GetInstalledAddOnStatus(v0_5_2_InstalledAddon)
	assert.NoError(t, err)
	assert.Equal(t, v0_5_3AddOnStatus, grpc_api.AddOnStatus_RUNNING)

	// Act
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("myFile", f.Name())
	assert.NoError(t, err)
	_, err = io.Copy(fw, f)
	assert.NoError(t, err)

	w.Close()

	resp, err := http.Post(fmt.Sprintf("http://nginx%s/upload", v0_5_2_InstalledAddon.Location), w.FormDataContentType(), &b)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

}

func dockerComposeUp(t *testing.T, composeFilepath string) {

	dockerComposeCommandArgs := []string{
		"compose",
		"--project-name",
		"uc-aom-mit",
		"-f",
		"/go/src/uc-aom/scripts/docker/docker-compose-migration-test.yml",
		"-f",
		composeFilepath,
		"up",
		"-d",
		"--build",
		"--remove-orphans",
	}
	dockerExecCommand := exec.Command("docker", dockerComposeCommandArgs...)

	stdout, err := dockerExecCommand.StdoutPipe()
	assert.NoError(t, err)
	stderr, err := dockerExecCommand.StderrPipe()
	assert.NoError(t, err)
	err = dockerExecCommand.Start()
	assert.NoError(t, err)

	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))
	for scanner.Scan() {
		m := scanner.Text()
		t.Log(m)
	}

	err = dockerExecCommand.Wait()
	assert.NoError(t, err)
}

func pushTestAddOnToRegistry(t *testing.T, ucAopContainerName string, repositoryName string) {
	// TODO: we need to build the docker images in the aop container otherwise we will get error with newer version
	targetCredentialsPath := "/tmp/target-credentials.json"
	dockerContainerExec(t, ucAopContainerName, "cp", "/tmp/target-credentials-template.json", targetCredentialsPath)

	sedCommand := []string{
		"sed",
		"-e",
		fmt.Sprintf("s,@REPOSITORY_NAME@,%s,g", repositoryName),
		"-i",
		targetCredentialsPath,
	}

	dockerContainerExec(t, ucAopContainerName, sedCommand...)

	addOnRepository := fmt.Sprintf("/testdata/%s/@0.1.0", repositoryName)

	createManifestCommand := []string{
		"/tmp/test-add-on-manifest-create.sh",
		addOnRepository,
		"0.0.0.0",
		"host.docker.internal",
	}

	dockerContainerExec(t, ucAopContainerName, createManifestCommand...)
	generatedManifestPath := dockerContainerExec(t, ucAopContainerName, createManifestCommand...)
	pushCommand := []string{
		"uc-aom-packager",
		"push",
		"-m",
		generatedManifestPath,
		"-s",
		"/tmp/source-credentials.json",
		"-t",
		targetCredentialsPath,
	}

	dockerContainerExec(t, ucAopContainerName, pushCommand...)
}

func dockerContainerExec(t *testing.T, container string, containerCommand ...string) string {
	dockerExecCommandArgs := append([]string{
		"container",
		"exec",
		container,
	}, containerCommand...)

	dockerExecCommand := exec.Command("docker", dockerExecCommandArgs...)
	output, err := dockerExecCommand.Output()
	if !assert.NoError(t, err) {
		t.Error(string(output))
	}

	return string(output)
}
