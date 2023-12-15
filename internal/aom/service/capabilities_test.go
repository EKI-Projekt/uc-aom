// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service_test

import (
	"testing"
	"u-control/uc-aom/internal/aom/service"
	"u-control/uc-aom/internal/aom/system"
	"u-control/uc-aom/internal/pkg/manifest"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name                 string
		platforms            string
		features             []manifest.Feature
		SshRootAccessEnabled bool
		expectError          bool
	}{
		{
			name:      "Validate platform and feature success",
			platforms: "ucm",
			features: []manifest.Feature{
				{
					Name:     "ucontrol.software.root_access",
					Required: nil,
				},
			},
			SshRootAccessEnabled: true,
			expectError:          false,
		},
		{
			name:                 "Platform fail",
			platforms:            "wrongPlatform",
			features:             []manifest.Feature{},
			SshRootAccessEnabled: false,
			expectError:          true,
		},
		{
			name:      "Feature SSH root access fail",
			platforms: "ucm",
			features: []manifest.Feature{
				{
					Name:     "ucontrol.software.root_access",
					Required: nil,
				},
			},
			SshRootAccessEnabled: false,
			expectError:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockSystem := &system.MockSystem{}

			mockSystem.On("IsSshRootAccessEnabled").Return(tt.SshRootAccessEnabled, nil).Maybe()

			uut := service.NewCapabilities(mockSystem, tt.platforms)
			uut.WithFeatures(tt.features)

			// Act
			result := uut.Validate()

			// Assert
			if (result != nil) != tt.expectError {
				t.Errorf("Unexpected result '%v'", result)
			}
			mockSystem.AssertExpectations(t)

		})
	}
}
