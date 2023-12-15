// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service_test

import (
	"testing"
	"u-control/uc-aom/internal/aom/service"
	"u-control/uc-aom/internal/pkg/manifest"
	"u-control/uc-aom/internal/pkg/manifest/v0_1"
)

func TestValidManifestVersion(t *testing.T) {
	// arrange
	manifestVersion := manifest.ValidManifestVersion
	validator := service.NewManifestVersionValidator(manifestVersion)

	// act
	err := validator.Validate()

	// assert
	if err != nil {
		t.Errorf("Expected manifest version to be valid. Actual %v", err)
	}
}

func TestInvalidManifestVersion(t *testing.T) {
	// arrange
	manifestVersion := v0_1.ValidManifestVersion
	validator := service.NewManifestVersionValidator(manifestVersion)

	// act
	err := validator.Validate()

	// assert
	if err == nil {
		t.Error("Expected manifest version to be invalid")
	}
}
