// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/pkg/config"

	"github.com/cavaliergopher/cpio"
	"github.com/stretchr/testify/assert"
	"u-control/uc-aom/internal/aop/service"
)

var expectedFilesPath = config.CACHE_DROP_IN_PATH

func TestExportCmdCreatesValidSwuFile(t *testing.T) {
	// arrange
	addOnRepositoryName := "test-uc-addon-status-running-addon-pkg"
	addOnVersion := "0.1.0-1"
	expectedSwuName := addOnRepositoryName + "_" + addOnVersion + ".swu"
	tempOutputPath := t.TempDir()
	uut := NewExportCmd()
	targetCredentialsFilepath := createTargetCredentials(t, addOnRepositoryName)
	uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", tempOutputPath, "--version", addOnVersion})

	// act
	if got := uut.Execute(); got != nil {
		t.Errorf("uut.Execute() = %v, unexpected error", got)
	}

	// assert
	expectedCreatedFiles := []string{expectedSwuName, expectedSwuName, expectedSwuName}
	expectedSwuContent := []string{"sw-description", config.UcSwuFilePayloadName}
	expectedPayloadContent := [][]string{
		{addOnRepositoryName},
		{addOnRepositoryName, addOnVersion},
		{addOnRepositoryName, addOnVersion, config.UcConfigAnnotationTitle},
		{addOnRepositoryName, addOnVersion, config.UcImageManifestDescriptorFilename},
		{addOnRepositoryName, addOnVersion, config.UcImageLayerAnnotationTitle},
		{addOnRepositoryName, addOnVersion, "test"},
		{addOnRepositoryName, addOnVersion, "test", "uc-aom-running:0.1"},
	}
	assertExpectedFilesWereCreated(t, tempOutputPath, expectedCreatedFiles)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-amd64", expectedSwuContent, expectedPayloadContent, expectedSwuName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm", expectedSwuContent, expectedPayloadContent, expectedSwuName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm64", expectedSwuContent, expectedPayloadContent, expectedSwuName)
}

func TestExportCmdCreatesValidSwuFile_Codename(t *testing.T) {
	// arrange
	repositoryName := "posuma/test-uc-addon-posuma-addon-pkg"
	normalizedName := "test-uc-addon-posuma-addon-pkg"
	version := "0.1.0-1"

	tempOutputPath := t.TempDir()
	uut := NewExportCmd()
	targetCredentialsFilepath := createTargetCredentials(t, repositoryName)
	uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", tempOutputPath, "--version", version})

	expectedSwuFileName := normalizedName + "_" + version + ".swu"

	// act
	if got := uut.Execute(); got != nil {
		t.Errorf("uut.Execute() = %v, unexpected error", got)
	}

	// assert
	expectedCreatedFiles := []string{expectedSwuFileName, expectedSwuFileName, expectedSwuFileName}
	expectedSwuContent := []string{"sw-description", config.UcSwuFilePayloadName}
	expectedPayloadContent := [][]string{
		{normalizedName},
		{normalizedName, version},
		{normalizedName, version, config.UcConfigAnnotationTitle},
		{normalizedName, version, config.UcImageManifestDescriptorFilename},
		{normalizedName, version, config.UcImageLayerAnnotationTitle},
		{normalizedName, version, "test"},
		{normalizedName, version, "test", "uc-aom-posuma:0.1"},
	}
	assertExpectedFilesWereCreated(t, tempOutputPath, expectedCreatedFiles)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-amd64", expectedSwuContent, expectedPayloadContent, expectedSwuFileName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm", expectedSwuContent, expectedPayloadContent, expectedSwuFileName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm64", expectedSwuContent, expectedPayloadContent, expectedSwuFileName)
}

func TestExportCmdCreatesValidSwuFile_MultiService(t *testing.T) {
	repositoryName := "test-uc-addon-multi-service-addon-pkg"
	version := "0.1.0-1"

	tempOutputPath := t.TempDir()
	uut := NewExportCmd()
	targetCredentialsFilepath := createTargetCredentials(t, repositoryName)
	uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", tempOutputPath, "--version", version})

	expectedSwuName := repositoryName + "_" + version + ".swu"

	// act
	if got := uut.Execute(); got != nil {
		t.Errorf("uut.Execute() = %v, unexpected error", got)
	}

	// assert
	expectedCreatedFiles := []string{expectedSwuName, expectedSwuName, expectedSwuName}
	expectedSwuContent := []string{"sw-description", config.UcSwuFilePayloadName}
	expectedPayloadContent := [][]string{
		{repositoryName},
		{repositoryName, version},
		{repositoryName, version, config.UcConfigAnnotationTitle},
		{repositoryName, version, config.UcImageManifestDescriptorFilename},
		{repositoryName, version, config.UcImageLayerAnnotationTitle},
		{repositoryName, version, "test"},
		{repositoryName, version, "test", "uc-aom-multi-service-a:0.1"},
		{repositoryName, version, "test", "uc-aom-multi-service-b:0.1"},
	}
	assertExpectedFilesWereCreated(t, tempOutputPath, expectedCreatedFiles)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-amd64", expectedSwuContent, expectedPayloadContent, expectedSwuName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm", expectedSwuContent, expectedPayloadContent, expectedSwuName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm64", expectedSwuContent, expectedPayloadContent, expectedSwuName)
}

func TestExportCmdExportsNothingOnFailure(t *testing.T) {
	// arrange
	tempOutputPath := t.TempDir()
	uut := NewExportCmd()
	wrongRepoName := "a-wrong-repo-name"
	addOnVersion := "0.1.0-1"
	targetCredentialsFilepath := createTargetCredentials(t, wrongRepoName)
	uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", tempOutputPath, "--version", addOnVersion})
	uut.SilenceUsage = true
	uut.SilenceErrors = true

	// act
	got := uut.Execute()

	// assert
	assert.Error(t, got)
	assertNoFilesWereCreated(t, tempOutputPath)
}

func TestExportCreatesExpectedSwDescription(t *testing.T) {
	// Arrange
	addOnRepositoryName := "test-uc-addon-status-running-addon-pkg"
	addOnVersion := "0.1.0-1"
	tempOutputPath := t.TempDir()
	uut := NewExportCmd()
	targetCredentialsFilepath := createTargetCredentials(t, addOnRepositoryName)
	uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", tempOutputPath, "--version", addOnVersion})

	// Act
	if got := uut.Execute(); got != nil {
		t.Errorf("uut.Execute() = %v, unexpected error", got)
	}

	// Assert
	swuName := addOnRepositoryName + "_" + addOnVersion + ".swu"
	platformSubDirectory, err := os.ReadDir(tempOutputPath)
	assert.Nil(t, err)
	for _, platform := range platformSubDirectory {
		swuFilePath := path.Join(tempOutputPath, platform.Name(), swuName)
		swuFile, err := os.Open(swuFilePath)
		t.Cleanup(func() {
			swuFile.Close()
		})
		assert.Nil(t, err)

		swuReader := cpio.NewReader(swuFile)
		swuHeader, err := swuReader.Next()
		assert.Nil(t, err)
		if !assert.Equal(t, "sw-description", swuHeader.Name, "Expected 'sw-description' as first file in swu archive, but got '%s'", swuHeader.Name) {
			t.FailNow()
		}

		swDescriptionContent := make([]byte, swuHeader.Size)
		_, err = swuReader.Read(swDescriptionContent)
		if !assert.Nil(t, err, "Error reading sw-description from swu archive %v", err) {
			t.FailNow()
		}

		expectedSwDescriptionFilePath := "testdata/sw-description"
		expectedFileContent, err := os.ReadFile(expectedSwDescriptionFilePath)
		assert.Nil(t, err)
		assert.Equal(t, expectedFileContent, swDescriptionContent)
	}
}

func assertSwuHasExpectedContent(t *testing.T, outputDirectory string, expectedSwuContent []string, expectedPayloadContent [][]string, expectedSwuName string) {
	expectedSwuFilePath := filepath.Join(outputDirectory, expectedSwuName)
	filesInSwuFile := getAllFilenamesInSwu(t, expectedSwuFilePath)
	if !contentEqual(filesInSwuFile, expectedSwuContent) {
		t.Fatalf("These files are in the swu-File: %v, but want: %v ", filesInSwuFile, expectedSwuContent)
	}

	assertPayloadHasExpectedContent(t, expectedSwuFilePath, expectedPayloadContent)
}

func assertPayloadHasExpectedContent(t *testing.T, expectedSwuFilePath string, expectedPayloadContent [][]string) {
	payloadExtractionDirectory := t.TempDir()
	swuExtractionDirectory := extractSwuAndPayload(t, expectedSwuFilePath, payloadExtractionDirectory)

	filepathesInExtractedPayload, err := getAllPathesRecursiveFrom(payloadExtractionDirectory)
	fullPathExpectedPayload := transformFilenamesToPath(payloadExtractionDirectory, expectedPayloadContent)
	if err != nil {
		t.Fatalf("Could not read payload-content: %v", err)
	}
	if !contentEqual(filepathesInExtractedPayload, fullPathExpectedPayload) {
		t.Fatalf("These files are in the payload: %v, but want: %v ", filepathesInExtractedPayload, fullPathExpectedPayload)
	}
	assertSwDescriptionHasExpectedFilesPath(t, swuExtractionDirectory, expectedFilesPath)
}

func assertSwDescriptionHasExpectedFilesPath(t *testing.T, swuExtractionDirectory string, expextedFilesPath string) {
	b, err := ioutil.ReadFile(filepath.Join(swuExtractionDirectory, "sw-description"))
	if err != nil {
		t.Log(err)
	}
	swDescriptionContent := string(b)
	if !strings.Contains(swDescriptionContent, expectedFilesPath) {
		t.Errorf("Expected sw-description has the expected files path '%s'", expectedFilesPath)
	}
}

func transformFilenamesToPath(payloadExtractionDirectory string, expectedPayloadContent [][]string) []string {
	fullPathes := make([]string, 0, len(expectedPayloadContent))

	for _, expectedPathInPayload := range expectedPayloadContent {
		fullPath := filepath.Join(append([]string{payloadExtractionDirectory}, expectedPathInPayload...)...)
		fullPathes = append(fullPathes, fullPath)
	}
	return fullPathes
}

func TestExportCmdCreatesValidSwuFile_CustomRegistryAddress(t *testing.T) {
	// arrange
	service.REGISTRY_ADDRESS = "xyz"
	registryAddress := "registry:5000"
	addOnRepositoryName := "test-uc-addon-status-running-addon-pkg"
	addOnVersion := "0.1.0-1"
	expectedSwuName := addOnRepositoryName + "_" + addOnVersion + ".swu"
	tempOutputPath := t.TempDir()
	uut := NewExportCmd()
	targetCredentialsFilepath := createTargetCredentialsWithServerAddress(t, addOnRepositoryName, registryAddress)
	uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", tempOutputPath, "--version", addOnVersion})

	// act
	if got := uut.Execute(); got != nil {
		t.Errorf("uut.Execute() = %v, unexpected error", got)
	}

	// assert
	expectedCreatedFiles := []string{expectedSwuName, expectedSwuName, expectedSwuName}
	expectedSwuContent := []string{"sw-description", config.UcSwuFilePayloadName}
	expectedPayloadContent := [][]string{
		{addOnRepositoryName},
		{addOnRepositoryName, addOnVersion},
		{addOnRepositoryName, addOnVersion, config.UcConfigAnnotationTitle},
		{addOnRepositoryName, addOnVersion, config.UcImageManifestDescriptorFilename},
		{addOnRepositoryName, addOnVersion, config.UcImageLayerAnnotationTitle},
		{addOnRepositoryName, addOnVersion, "test"},
		{addOnRepositoryName, addOnVersion, "test", "uc-aom-running:0.1"},
	}
	assertExpectedFilesWereCreated(t, tempOutputPath, expectedCreatedFiles)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-amd64", expectedSwuContent, expectedPayloadContent, expectedSwuName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm", expectedSwuContent, expectedPayloadContent, expectedSwuName)
	assertSwuHasExpectedContent(t, tempOutputPath+"/linux-arm64", expectedSwuContent, expectedPayloadContent, expectedSwuName)
}

func extractSwuAndPayload(t *testing.T, expectedSwuFilePath string, payloadExtractionDirectory string) string {
	swuExtractionDirectory := t.TempDir()
	err := extractSwuToDirectory(t, expectedSwuFilePath, swuExtractionDirectory)
	if err != nil {
		t.Fatalf("Could not extract swu file: %v", err)
	}

	expectedTarFilePath := filepath.Join(swuExtractionDirectory, config.UcSwuFilePayloadName)
	tarFile, err := os.Open(expectedTarFilePath)
	if err != nil {
		t.Fatalf("Could not open extracted tar file: %v", err)
	}

	err = fileio.UnTarball(payloadExtractionDirectory, tarFile)
	if err != nil {
		t.Fatalf("Could not extract tar file: %v", err)
	}

	return swuExtractionDirectory

}

func extractSwuToDirectory(t *testing.T, swuFilePath string, targetDirectory string) error {
	return visitSwuContent(swuFilePath, func(hdr *cpio.Header, cpioReader *cpio.Reader) error {
		fileContent, err := ioutil.ReadAll(cpioReader)
		if err != nil {
			return err
		}

		extractFileTo := filepath.Join(targetDirectory, hdr.Name)
		if err = os.WriteFile(extractFileTo, fileContent, 0744); err != nil {
			return err
		}

		return nil
	})
}

func getAllFilenamesInSwu(t *testing.T, swuFilePath string) []string {
	filesInSwuFile := []string{}

	err := visitSwuContent(swuFilePath, func(hdr *cpio.Header, cpioReader *cpio.Reader) error {
		filesInSwuFile = append(filesInSwuFile, hdr.Name)
		return nil
	})
	if err != nil {
		t.Fatalf("Could not verfify content of swu: %v", err)
	}

	return filesInSwuFile
}

func visitSwuContent(swuFilePath string, visit func(*cpio.Header, *cpio.Reader) error) error {
	swuFile, err := os.Open(swuFilePath)
	if err != nil {
		return err
	}

	cpioReader := cpio.NewReader(swuFile)

	for {
		hdr, err := cpioReader.Next()
		if err == io.EOF {
			// end of cpio archive
			break
		}
		if err != nil {
			return err
		}

		err = visit(hdr, cpioReader)
		if err != nil {
			return err
		}
	}

	return nil
}

func assertNoFilesWereCreated(t *testing.T, testDirectory string) {
	assertExpectedFilesWereCreated(t, testDirectory, []string{})
}
