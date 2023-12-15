// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import "u-control/uc-aom/internal/pkg/manifest"

// Represents an invalid manifest version.
type InvalidManifestError struct {
	message string
}

func (r *InvalidManifestError) Error() string {
	return r.message
}

type ManifestVersionValidator struct {
	manifestVersion string
}

func NewManifestVersionValidator(manifestVersion string) *ManifestVersionValidator {
	return &ManifestVersionValidator{
		manifestVersion: manifestVersion,
	}
}

// Returns an error if the manifest version validation fails, otherwise nil.
func (r *ManifestVersionValidator) Validate() error {
	if r.manifestVersion != manifest.ValidManifestVersion {
		return &InvalidManifestError{"Invalid manifest version"}
	}
	return nil
}
