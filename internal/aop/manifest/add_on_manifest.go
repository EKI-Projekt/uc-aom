// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/pkg/config"
	model "u-control/uc-aom/internal/pkg/manifest"
)

type AddOnManifest struct {
	*model.Root
	manifestDirPath string
	fileExistsFunc  fileio.FileExistsFunc
}

// Represents manifest validation errors.
type AddOnManifestValidationError struct {
	message string
}

func (r *AddOnManifestValidationError) Error() string {
	return r.message
}

// Migrate the manifest file under the given dirpath.
func MigrateManifestFile(dirpath string) error {
	rawManifest, err := getRawManifest(dirpath)
	if err != nil {
		return err
	}

	manifestVersion, err := model.UnmarshalManifestVersionFrom(rawManifest)
	if err != nil {
		return err
	}

	migratedManifest, err := model.MigrateUcManifest(manifestVersion, rawManifest)
	if err != nil {
		return err
	}

	return updateManifestFileContent(dirpath, migratedManifest)
}

// Parse, validate and return the validated AddOn manifest under the given dirpath.
func ParseAndValidate(reader model.ManifestFileReader, fileExistsFunc fileio.FileExistsFunc, dirpath string) (*AddOnManifest, error) {
	root, err := reader.ReadManifestFrom(dirpath)
	if err != nil {
		return nil, err
	}

	manifestValidator, err := model.NewValidator()
	if err != nil {
		return nil, err
	}

	if err := manifestValidator.Validate(root); err != nil {
		return nil, err
	}

	addOnManifest := AddOnManifest{root, dirpath, fileExistsFunc}
	return &addOnManifest, addOnManifest.validateAssets()
}

// Returns the directory which contains the manifest.json of the add-on.
func (r *AddOnManifest) ManifestBaseDirectory() string {
	return r.manifestDirPath
}

func getRawManifest(baseManifestPath string) ([]byte, error) {
	manifestPath := path.Join(baseManifestPath, config.UcImageManifestFilename)
	manifestContent, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}
	return manifestContent, nil
}

func updateManifestFileContent(manifestPath string, manifestContent []byte) error {
	pathToManifest := path.Join(manifestPath, config.UcImageManifestFilename)

	err := os.WriteFile(pathToManifest, manifestContent, os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}

// Performs manifests assets validation returning an error should it fail.
func (r *AddOnManifest) validateAssets() error {
	logoPath := filepath.Join(r.manifestDirPath, r.Logo)
	if _, err := r.fileExistsFunc(logoPath); errors.Is(err, os.ErrNotExist) {
		return &AddOnManifestValidationError{message: fmt.Sprintf("Logo file '%s' does not exist", logoPath)}
	}

	return nil
}
