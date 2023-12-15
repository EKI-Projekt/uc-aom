// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package oraswrapper_test

import (
	"fmt"
	"testing"
	oraswrapper "u-control/uc-aom/internal/pkg/oras-wrapper"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
)

func TestFromOCIPlatform(t *testing.T) {
	type args struct {
		os           string
		architecture string
		variant      string
		expectedKey  oraswrapper.PlatformKey
	}

	testCases := []args{
		{
			os:           "linux",
			architecture: "arm",
			variant:      "7",
			expectedKey:  "linux-arm-7",
		},
		{
			os:           "linux",
			architecture: "amd64",
			variant:      "",
			expectedKey:  "linux-amd64",
		},
		{
			os:           "linux",
			architecture: "arm64",
			variant:      "",
			expectedKey:  "linux-arm64",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Running test for the platform key %s", tc.expectedKey), func(t *testing.T) {
			// Arrange
			platform := &ocispec.Platform{
				OS:           tc.os,
				Architecture: tc.architecture,
				Variant:      tc.variant,
			}

			// Act
			platformKey := oraswrapper.FromOCIPlatform(platform)

			// Assert
			if platformKey != tc.expectedKey {
				t.Errorf("Expected platform key to be '%s' but got '%s'", tc.expectedKey, platformKey)
			}
		})
	}
}

func TestToOCIPlatform(t *testing.T) {
	type args struct {
		platformKey     oraswrapper.PlatformKey
		expectedOs      string
		expectedArch    string
		expectedVariant string
	}

	testCases := []args{
		{
			platformKey:     "linux-amd64",
			expectedOs:      "linux",
			expectedArch:    "amd64",
			expectedVariant: "",
		},
		{
			platformKey:     "linux-arm-v7",
			expectedOs:      "linux",
			expectedArch:    "arm",
			expectedVariant: "v7",
		},
		{
			platformKey:     "linux-arm64",
			expectedOs:      "linux",
			expectedArch:    "arm64",
			expectedVariant: "",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Running test for %s", tc.platformKey), func(t *testing.T) {
			// Act

			platform, err := tc.platformKey.ToOCIPlatform()

			// Assert
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedOs, platform.OS)
			assert.Equal(t, tc.expectedArch, platform.Architecture)
			assert.Equal(t, tc.expectedVariant, platform.Variant)
		})
	}
}

func TestPlatformAsMapKey(t *testing.T) {
	// Arrange
	var p1 oraswrapper.PlatformKey = "linux-amd64"
	var p2 oraswrapper.PlatformKey = "linux-amd64"

	m := make(map[oraswrapper.PlatformKey]string)

	// Act
	m[p1] = "abc"
	m[p2] = "xyz"

	// Assert
	expectedLength := 1
	gotLength := len(m)
	if gotLength > 1 {
		t.Errorf("Expected map length to be %d but got %d", expectedLength, gotLength)
	}

	expectedValue := "xyz"
	gotValue := m["linux-amd64"]
	if gotValue != expectedValue {
		t.Errorf("Expected value to be %s but got %s", expectedValue, gotValue)
	}
}
