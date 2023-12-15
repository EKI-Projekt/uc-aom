// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"u-control/uc-aom/internal/aop/credentials"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func getFilenamesRecursive(outputPath string) ([]string, error) {
	fileNames := make([]string, 0)

	err := filepath.WalkDir(outputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Do not include root directory
		if path == outputPath {
			return nil
		}

		if !d.IsDir() {
			fileNames = append(fileNames, d.Name())
		}

		return nil
	})

	return fileNames, err
}

func getAllPathesRecursiveFrom(outputPath string) ([]string, error) {
	pathes := make([]string, 0)

	err := filepath.WalkDir(outputPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Do not include root directory
		if path == outputPath {
			return nil
		}

		pathes = append(pathes, path)

		return nil
	})

	return pathes, err
}

func contentEqual(actual, expected []string) bool {
	less := func(a, b string) bool { return a < b }
	difference := cmp.Diff(actual, expected, cmpopts.SortSlices(less))
	return difference == ""
}

func assertExpectedFilesWereCreated(t *testing.T, testDirectory string, expectedFiles []string) {
	filenames, err := getFilenamesRecursive(testDirectory)
	if err != nil {
		t.Fatalf("Could not check created files: %v", err)
	}

	if !contentEqual(filenames, expectedFiles) {
		t.Fatalf("These files were created: %v, but want: %v ", filenames, expectedFiles)
	}
}

func createTargetCredentials(t *testing.T, repositoryname string) string {
	t.Helper()
	c := credentials.Credentials{}
	c.RepositoryName = repositoryname
	data, _ := json.Marshal(&c)
	targetCredentialsFilepath := filepath.Join(t.TempDir(), "target-credentials.json")
	os.WriteFile(targetCredentialsFilepath, data, 0644)
	return targetCredentialsFilepath
}

func createTargetCredentialsWithServerAddress(t *testing.T, repositoryname string, serverAddress string) string {
	t.Helper()
	c := credentials.Credentials{}
	c.RepositoryName = repositoryname
	c.ServerAddress = serverAddress
	data, _ := json.Marshal(&c)
	targetCredentialsFilepath := filepath.Join(t.TempDir(), "target-credentials.json")
	os.WriteFile(targetCredentialsFilepath, data, 0644)
	return targetCredentialsFilepath
}

func createSourceCredentials(t *testing.T, serverAddress string) string {
	t.Helper()
	c := credentials.Credentials{}
	c.ServerAddress = serverAddress
	data, _ := json.Marshal(&c)
	sourceCredentialsFilepath := filepath.Join(t.TempDir(), "source-credentials.json")
	os.WriteFile(sourceCredentialsFilepath, data, 0644)
	return sourceCredentialsFilepath
}
