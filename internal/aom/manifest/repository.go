// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"io/fs"
	"path/filepath"
	"strings"
	"u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
)

// In LocalFSRepository the callbacks for file- and dirReader are defined
type LocalFSRepository struct {
	fileReader FileReaderCallback
	walkDir    WalkDirCallback
}

type FileReaderCallback func(string) ([]byte, error)

type WalkDirCallback func(string, fs.WalkDirFunc) error

// NewRepository returns the file- and walkDir callbacks for the filesystem
func NewRepository(fileReader FileReaderCallback, walkDir WalkDirCallback) *LocalFSRepository {
	return &LocalFSRepository{fileReader: fileReader, walkDir: walkDir}
}

// GetManifestsDirectories returns addOn directories from the given parent directory.
// An addOn directory may be nested and in guaranteed to contain a manifest.json file.
func (r *LocalFSRepository) GetManifestsDirectories(basepath string) ([]string, error) {
	manifestDirs := make([]string, 0)
	err := r.walkDir(basepath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, config.DROP_IN_FOLDER_NAME) {
			return nil
		}

		if !d.IsDir() && d.Name() == config.UcImageManifestFilename {
			rel, err := filepath.Rel(basepath, path)
			if err != nil {
				return err
			}
			manifestDirs = append(manifestDirs, filepath.Dir(rel))
		}
		return nil
	})
	return manifestDirs, err
}

// ReadManifestFrom returns the deserialized manifest from the given directory
func (r *LocalFSRepository) ReadManifestFrom(directoryOfManifest string) (*manifest.Root, error) {
	file := filepath.Join(directoryOfManifest, config.UcImageManifestFilename)
	bytes, err := r.fileReader(file)
	if err != nil {
		log.Printf("Failed to read manifest: %v", err)
		return nil, err
	}

	manifest, err := manifest.NewFromBytes(bytes)
	if err != nil {
		log.Printf("Failed to load manifest: %v", err)
		return nil, err
	}
	return manifest, nil
}
