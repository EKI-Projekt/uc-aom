// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"reflect"
	"testing"
	"u-control/uc-aom/internal/aop/manifest"
	"u-control/uc-aom/internal/pkg/config"
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

type mockManifestReader struct {
	mock.Mock

	logoErr error
}

func (r *mockManifestReader) ReadManifestFrom(directoryOfManifest string) (*model.Root, error) {
	args := r.Called(directoryOfManifest)
	return args.Get(0).(*model.Root), args.Error(1)
}

func (r *mockManifestReader) fileExistsFunc(path string) (fs.FileInfo, error) {
	return nil, r.logoErr
}

func TestAddOnManifestPass(t *testing.T) {
	// Arrange
	root := &model.Root{
		ManifestVersion: "0.1",
		Version:         "0.1-1",
		Logo:            "logo.png",
		Services: map[string]*model.Service{
			"ucAddonTestService": {
				Type:   "docker-compose",
				Config: map[string]interface{}{"image": "test/image:0.42"},
			},
		},
		Platform: []string{"ucm", "ucg"},
		Vendor: &model.Vendor{
			Name:    "abc",
			Url:     "https://www.abc.de",
			Email:   "email@abc.de",
			Street:  "street",
			Zip:     "12345",
			City:    "City",
			Country: "Country",
		},
	}

	mockManifest := &mockManifestReader{logoErr: nil}
	mockManifest.On("ReadManifestFrom", mock.AnythingOfType("string")).Return(root, nil)

	// Act
	manifest, err := manifest.ParseAndValidate(mockManifest, mockManifest.fileExistsFunc, "manifest-directory-base")

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if manifest.Version != "0.1-1" {
		t.Errorf("Unexpected manifest Version. Expected '0.1', Actual '%s'", manifest.Version)
	}

	if manifest.Logo != "logo.png" {
		t.Errorf("Unexpected logo path. Expected 'logo.png', Actual '%s'", manifest.Logo)
	}

	if manifest.ManifestBaseDirectory() != "manifest-directory-base" {
		t.Errorf("Unexpected manifest base directory. Expected 'manifest-directory-base', Actual '%s'", manifest.ManifestBaseDirectory())
	}

	refs := model.GetDockerImageReferences(manifest.Services)

	if len(refs) == 0 {
		t.Fatal("Expected GetDockerImageReferences to be not empty")
	}

	if refs[0] != "test/image:0.42" {
		t.Errorf("Unexpected docker image. Expected 'test/image:0.42', Actual '%s'", refs[0])
	}
}

func TestAddOnManifestLogoDoesNotExistFail(t *testing.T) {
	// Arrange
	baseDirectory := "manifest-directory-base"
	root := &model.Root{
		ManifestVersion: "0.1",
		Version:         "0.1-1",
		Logo:            "logo.png",
		Services: map[string]*model.Service{
			"ucAddonTestService": {
				Type:   "docker-compose",
				Config: map[string]interface{}{"image": "test/image:0.42"},
			},
		},
		Platform: []string{"ucm", "ucg"},
		Vendor: &model.Vendor{
			Name:    "abc",
			Url:     "https://www.abc.de",
			Email:   "email@abc.de",
			Street:  "street",
			Zip:     "12345",
			City:    "City",
			Country: "Country",
		},
	}

	mockManifest := &mockManifestReader{logoErr: fs.ErrNotExist}
	mockManifest.On("ReadManifestFrom", mock.AnythingOfType("string")).Return(root, nil)
	mockManifest.On("ReadManifestAndValidateFrom", mock.AnythingOfType("string")).Return(root, nil)

	// Act
	manifest, err := manifest.ParseAndValidate(mockManifest, mockManifest.fileExistsFunc, baseDirectory)

	// Assert
	if err == nil || err.Error() != "Logo file 'manifest-directory-base/logo.png' does not exist" {
		t.Fatal("Expected error not thrown.")
	}

	if manifest.Logo != "logo.png" {
		t.Errorf("Unexpected manifest Logo. Expected 'logo.png', Actual '%s'", manifest.Logo)
	}

	if manifest.ManifestBaseDirectory() != baseDirectory {
		t.Errorf("Unexpected manifest base directory. Expected '%s', Actual '%s'", baseDirectory, manifest.ManifestBaseDirectory())
	}
}

func TestAddOnMigrateManifest(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	pathToTestManifest := "testdata/manifest-migration-0.1.json"

	manifestPath := copyManifestToTestDir(t, pathToTestManifest, testDir)

	// Act
	err := manifest.MigrateManifestFile(testDir)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert
	pathToExpectedManifest := "testdata/manifest-migration-0.2.json"

	assertCompareManifestFiles(t, manifestPath, pathToExpectedManifest)
}

func TestAddOnMigrateManifestMissingManifest(t *testing.T) {
	// Arrange
	testDir := t.TempDir()

	// Act
	err := manifest.MigrateManifestFile(testDir)

	// Assert
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Expected error '%v' but got %v.", fs.ErrExist, err)
	}
}

func TestAddOnMigrateManifestMigrationFail(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	pathToTestManifest := "testdata/manifest-migration-missing-vendor-0.1.json"

	copyManifestToTestDir(t, pathToTestManifest, testDir)

	// Act
	err := manifest.MigrateManifestFile(testDir)
	// Assert
	if err == nil {
		t.Fatalf("Expected error but got nil.")
	}
}

func copyManifestToTestDir(t *testing.T, pathToTestManifest string, testDir string) string {
	manifestData, err := os.ReadFile(pathToTestManifest)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	manifestPath := path.Join(testDir, config.UcImageManifestFilename)
	os.WriteFile(manifestPath, manifestData, os.ModePerm)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	return manifestPath
}

func assertCompareManifestFiles(t *testing.T, actualManifestPath string, expectedManifestPath string) {
	actualManifestData, err := os.ReadFile(actualManifestPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	actualRoot, err := model.NewFromBytes(actualManifestData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedManifestData, err := os.ReadFile(expectedManifestPath)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedRoot, err := model.NewFromBytes(expectedManifestData)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(actualRoot, expectedRoot) {
		t.Fatalf("Actual manifest mismatch expected manifest.")
	}
}
