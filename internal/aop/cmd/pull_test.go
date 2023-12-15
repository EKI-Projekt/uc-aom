// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/aop/service"
)

type testCase struct {
	name              string
	repositoryname    string
	version           string
	extract           bool
	expectedFilenames []string
}

func TestNewPullCmd(t *testing.T) {
	tests := []testCase{
		{
			name:           "Pull an add-on",
			repositoryname: "test-uc-addon-status-running-addon-pkg",
			version:        "0.1.0-1",
			extract:        true,
			expectedFilenames: []string{
				"logo.png",
				config.UcImageManifestFilename,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
			},
		},
		{
			name:           "Pull an add-on without extract",
			repositoryname: "test-uc-addon-status-running-addon-pkg",
			version:        "0.1.0-1",
			extract:        false,
			expectedFilenames: []string{
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
				config.UcImageLayerAnnotationTitle,
				config.UcImageLayerAnnotationTitle,
				config.UcImageLayerAnnotationTitle,
			},
		},
		{
			name:           "Pull an add-on with multi service",
			repositoryname: "test-uc-addon-multi-service-addon-pkg",
			version:        "0.1.0-1",
			extract:        true,
			expectedFilenames: []string{
				"logo.png",
				config.UcImageManifestFilename,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				"uc-aom-multi-service-a:0.1",
				"uc-aom-multi-service-a:0.1",
				"uc-aom-multi-service-a:0.1",
				"uc-aom-multi-service-b:0.1",
				"uc-aom-multi-service-b:0.1",
				"uc-aom-multi-service-b:0.1",
			},
		},
	}
	for _, currentTestCase := range tests {

		t.Run(currentTestCase.name, func(t *testing.T) {

			// arrange
			uut := NewPullCmd()
			targetCredentialsFilepath := createTargetCredentials(t, currentTestCase.repositoryname)
			outputPath := t.TempDir()
			extractArg := fmt.Sprintf("-x=%t", currentTestCase.extract)
			uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", outputPath, "--version", currentTestCase.version, extractArg})

			// act
			if got := uut.Execute(); got != nil {
				t.Errorf("NewPullCmd() = %v, unexpected error", got)
			}

			// assert
			assertFilesCreated(t, &currentTestCase, outputPath)
		})
	}
}

func TestNewPullCmd_CustomRegistryAddress(t *testing.T) {
	tests := []testCase{
		{
			name:           "Pull an add-on",
			repositoryname: "test-uc-addon-status-running-addon-pkg",
			version:        "0.1.0-1",
			extract:        true,
			expectedFilenames: []string{
				"logo.png",
				config.UcImageManifestFilename,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
			},
		},
		{
			name:           "Pull an add-on without extract",
			repositoryname: "test-uc-addon-status-running-addon-pkg",
			version:        "0.1.0-1",
			extract:        false,
			expectedFilenames: []string{
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
				"uc-aom-running:0.1",
				config.UcImageLayerAnnotationTitle,
				config.UcImageLayerAnnotationTitle,
				config.UcImageLayerAnnotationTitle,
			},
		},
		{
			name:           "Pull an add-on with multi service",
			repositoryname: "test-uc-addon-multi-service-addon-pkg",
			version:        "0.1.0-1",
			extract:        true,
			expectedFilenames: []string{
				"logo.png",
				config.UcImageManifestFilename,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcConfigAnnotationTitle,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				config.UcImageManifestDescriptorFilename,
				"uc-aom-multi-service-a:0.1",
				"uc-aom-multi-service-a:0.1",
				"uc-aom-multi-service-a:0.1",
				"uc-aom-multi-service-b:0.1",
				"uc-aom-multi-service-b:0.1",
				"uc-aom-multi-service-b:0.1",
			},
		},
	}
	for _, currentTestCase := range tests {

		t.Run(currentTestCase.name, func(t *testing.T) {

			// arrange
			service.REGISTRY_ADDRESS = "xyz"
			registryAddress := "registry:5000"
			uut := NewPullCmd()
			targetCredentialsFilepath := createTargetCredentialsWithServerAddress(t, currentTestCase.repositoryname, registryAddress)
			outputPath := t.TempDir()
			extractArg := fmt.Sprintf("-x=%t", currentTestCase.extract)
			uut.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", outputPath, "--version", currentTestCase.version, extractArg})

			// act
			if got := uut.Execute(); got != nil {
				t.Errorf("NewPullCmd() = %v, unexpected error", got)
			}

			// assert
			assertFilesCreated(t, &currentTestCase, outputPath)
		})
	}
}

func assertFilesCreated(t *testing.T, currentTestCase *testCase, outputPath string) {
	expectedCreatedSubfolder := filepath.Join(outputPath, currentTestCase.repositoryname, currentTestCase.version)
	_, err := os.Stat(expectedCreatedSubfolder)
	if err != nil {
		t.Errorf("os.Stat = %v, unexpected error ", err)
	}

	assertExpectedFilesWereCreated(t, outputPath, currentTestCase.expectedFilenames)
}
