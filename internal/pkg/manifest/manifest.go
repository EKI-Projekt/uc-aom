// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"log"
	"os"
	"path/filepath"
	"u-control/uc-aom/internal/pkg/config"
)

type ManifestFileReader interface {

	// ReadManifestFrom returns the manifest from the given directory
	ReadManifestFrom(directoryOfManifest string) (*Root, error)
}

type OsManifestFileReader struct {
}

func NewOsManifestFileReader() *OsManifestFileReader {
	return &OsManifestFileReader{}
}

// Read the manifest from the directory.
func (r *OsManifestFileReader) ReadManifestFrom(directoryOfManifest string) (*Root, error) {
	file := filepath.Join(directoryOfManifest, config.UcImageManifestFilename)
	bytes, err := os.ReadFile(file)
	if err != nil {
		log.Printf("Failed to read manifest: %v", err)
		return nil, err
	}

	return NewFromBytes(bytes)
}
