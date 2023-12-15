// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package utils_test

import (
	"errors"
	"os"
	"path"
	"reflect"
	"testing"
	"u-control/uc-aom/internal/aop/utils"
	"u-control/uc-aom/internal/pkg/config"
)

func TestCopyToWorkDirSuccess(t *testing.T) {
	// Arrange
	testFiles := make(map[string]string)
	testFiles[config.UcImageManifestFilename] = "manifestContent"
	testFiles["logo.png"] = "logoContent"

	source := t.TempDir()
	for fileName, fileContent := range testFiles {
		testFilePath := path.Join(source, fileName)
		err := os.WriteFile(testFilePath, []byte(fileContent), os.ModePerm)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
	}

	destination := t.TempDir()

	// Act
	err := utils.CopyFiles(source, destination, func(fileName string) bool {
		return true
	})
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	// Assert
	returnFiles, err := os.ReadDir(destination)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	if len(returnFiles) != len(testFiles) {
		t.Errorf("Unexpected number of files. Expected %d but got %d", len(testFiles), len(returnFiles))
	}
	for _, returnFile := range returnFiles {
		destinationFilePath := path.Join(destination, returnFile.Name())
		fileInfo, err := os.ReadFile(destinationFilePath)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}

		if !reflect.DeepEqual(testFiles[returnFile.Name()], string(fileInfo)) {
			t.Errorf("Unexpected file content")
		}
	}
}

type copyFilesWithFilterTests struct {
	description           string
	filterFuncReturnValue bool
	testFiles             []string
	expectedFiles         []string
}

var testCases = []copyFilesWithFilterTests{
	{
		description:           "Should have an empty destination folder",
		filterFuncReturnValue: false,
		testFiles:             []string{"manifest.json.license"},
		expectedFiles:         []string{},
	},
	{
		description:           "Should copy the file into the destination folder",
		filterFuncReturnValue: true,
		testFiles:             []string{"manifest.json"},
		expectedFiles:         []string{"manifest.json"},
	},
}

func TestCopyToWorkDir_WithFilesFilter(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Arrange
			sourcePath := initSource(t, tc.testFiles)

			// Act
			destinationPath := copyToDestination(t, tc, sourcePath)

			// Assert
			got := assertCopiedFiles(t, destinationPath, tc.expectedFiles)
			if got != nil {
				t.Errorf("Expected not to have '%v' in the destination folder", got)
			}
		})
	}
}

func initSource(t *testing.T, testFiles []string) string {
	sourcePath := t.TempDir()
	for _, fileName := range testFiles {
		dir := path.Dir(fileName)
		dirPath := path.Join(sourcePath, dir)
		err := os.MkdirAll(dirPath, 0755)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		testFilePath := path.Join(dirPath, path.Base(fileName))
		err = os.WriteFile(testFilePath, []byte(""), os.ModePerm)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
	}
	return sourcePath
}

func copyToDestination(t *testing.T, testCase copyFilesWithFilterTests, source string) string {
	destination := t.TempDir()
	err := utils.CopyFiles(source, destination, func(fileName string) bool {
		return testCase.filterFuncReturnValue
	})
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	return destination
}

func assertCopiedFiles(t *testing.T, destinationPath string, expectedFiles []string) error {
	returnFiles, err := os.ReadDir(destinationPath)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	for _, file := range returnFiles {
		fileName := file.Name()
		if !contains(expectedFiles, fileName) {
			return errors.New(fileName)
		}
	}
	return nil
}

func contains(array []string, value string) bool {
	for _, v := range array {
		if value == v {
			return true
		}
	}
	return false
}
