// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"u-control/uc-aom/internal/aop/packager"
	"u-control/uc-aom/internal/aop/service"
)

type pushTestCase struct {
	name           string
	repositoryName string
	manifestDir    string
	version        string
	serverAddress  string
}

func TestNewPushCmd_SingleArchImage(t *testing.T) {
	tests := []pushTestCase{
		{
			name:           "Push an add-on with a single arch image",
			repositoryName: "test-uc-addon-single-arch-image-pkg",
			manifestDir:    "test-uc-addon-single-arch-image",
			version:        "0.1.0-1",
			serverAddress:  "registry:5000",
		},
	}
	for _, currentTestCase := range tests {

		t.Run(currentTestCase.name, func(t *testing.T) {

			// arrange
			uut := NewPushCommand()
			sourceCredentials := createSourceCredentials(t, currentTestCase.serverAddress)
			targetCredentials := createTargetCredentials(t, currentTestCase.repositoryName)
			filesPath := createTestAppFiles(t, fmt.Sprintf("../../../testdata/%s/@%s", currentTestCase.manifestDir, currentTestCase.version))
			uut.SetArgs([]string{"--target-credentials", targetCredentials, "--source-credentials", sourceCredentials, "--manifest", filesPath})

			// act
			got := uut.Execute()

			// assert
			if got != nil && !errors.Is(got, packager.SingleArchImageError) {
				t.Errorf("Expected a single arch image error but got %v", got)
			}

			if got == nil {
				t.Error("Expected a single arch image error but got none")
			}
		})
	}
}

func TestNewPushCmd_MultiService(t *testing.T) {
	tests := []pushTestCase{
		{
			name:           "Push an add-on with multi service",
			repositoryName: "test-uc-addon-multi-service-addon-pkg",
			manifestDir:    "test-uc-addon-multi-service",
			version:        "0.1.0",
			serverAddress:  "registry:5000",
		},
	}
	for _, currentTestCase := range tests {

		t.Run(currentTestCase.name, func(t *testing.T) {

			// arrange
			uut := NewPushCommand()
			sourceCredentials := createSourceCredentials(t, currentTestCase.serverAddress)
			targetCredentials := createTargetCredentials(t, currentTestCase.repositoryName)
			filesPath := createTestAppFiles(t, fmt.Sprintf("../../../testdata/%s/@%s", currentTestCase.manifestDir, currentTestCase.version))
			uut.SetArgs([]string{"--target-credentials", targetCredentials, "--source-credentials", sourceCredentials, "--manifest", filesPath})

			// act
			got := uut.Execute()

			// assert
			if got != nil {
				t.Errorf("Unexpected error: %v", got)
			}
		})
	}
}

func TestNewPushCmd_NoService(t *testing.T) {
	tests := []pushTestCase{
		{
			name:           "Push an add-on with no service",
			repositoryName: "test-uc-addon-no-service-addon-pkg",
			manifestDir:    "test-uc-addon-no-service",
			version:        "0.1.0",
			serverAddress:  "registry:5000",
		},
	}
	for _, currentTestCase := range tests {

		t.Run(currentTestCase.name, func(t *testing.T) {

			// arrange
			uut := NewPushCommand()
			sourceCredentials := createSourceCredentials(t, currentTestCase.serverAddress)
			targetCredentials := createTargetCredentials(t, currentTestCase.repositoryName)
			filesPath := createTestAppFiles(t, fmt.Sprintf("../../../testdata/%s/@%s", currentTestCase.manifestDir, currentTestCase.version))
			uut.SetArgs([]string{"--target-credentials", targetCredentials, "--source-credentials", sourceCredentials, "--manifest", filesPath})

			// act
			got := uut.Execute()

			// assert
			if got == nil {
				t.Error("Expected an error but got none")
			}
		})
	}
}

func TestNewPushCmd_CustomRegistryAddress(t *testing.T) {
	tests := []pushTestCase{
		{
			name:           "Push an add-on to a custom registry",
			repositoryName: "test-uc-addon-status-running-addon-pkg",
			manifestDir:    "test-uc-addon-status-running",
			version:        "0.1.0",
			serverAddress:  "registry:5000",
		},
	}
	for _, currentTestCase := range tests {

		t.Run(currentTestCase.name, func(t *testing.T) {

			// arrange
			service.REGISTRY_ADDRESS = "xyz"
			uut := NewPushCommand()
			sourceCredentials := createSourceCredentials(t, currentTestCase.serverAddress)
			targetCredentials := createTargetCredentialsWithServerAddress(t, currentTestCase.repositoryName, currentTestCase.serverAddress)
			filesPath := createTestAppFiles(t, fmt.Sprintf("../../../testdata/%s/@%s", currentTestCase.manifestDir, currentTestCase.version))
			uut.SetArgs([]string{"--target-credentials", targetCredentials, "--source-credentials", sourceCredentials, "--manifest", filesPath})

			// act
			got := uut.Execute()

			// assert
			if got != nil {
				t.Errorf("unexpected error: %v", got)
			}
		})
	}
}

func createTestAppFiles(t *testing.T, manifestPath string) string {
	testDir := t.TempDir()
	source, err := os.Open(fmt.Sprintf("%s/manifest.json", manifestPath))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer source.Close()

	target, err := os.Create(fmt.Sprintf("%s/manifest.json", testDir))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	defer target.Close()

	_, err = os.Create(filepath.Join(testDir, "logo.png"))
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	_, err = io.Copy(target, source)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	return testDir
}

func TestFilterAppFilesFunc(t *testing.T) {
	// Arrange
	type testcase struct {
		fileName       string
		expectedResult bool
	}

	testcases := []testcase{
		{
			"manifest.json",
			true,
		},
		{
			"logo.png",
			true,
		},
		{
			"icon.jpg",
			true,
		},
		{
			"manifest.json.license",
			false,
		},
		{
			"info.txt",
			false,
		},
		{
			"icon.gif",
			false,
		},
	}

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("%s", tc.fileName), func(t *testing.T) {
			// Act
			res := filterAppFilesFunc(tc.fileName)

			// Assert
			if res != tc.expectedResult {
				t.Errorf("Expected result to be %t but got %t", tc.expectedResult, res)
			}
		})
	}
}
