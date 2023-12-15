// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package system_test

import (
	"encoding/json"
	"os"
	"testing"
	"u-control/uc-aom/internal/aom/system"

	"github.com/stretchr/testify/assert"
)

func TestIsSshRootAccessEnabled(t *testing.T) {
	// Arrange
	type configFile struct {
		RootAccessEnabled int `json:"rootAccessEnabled"`
	}
	configFilePath := os.Getenv("ROOT_ACCESS_CONFIG_FILE")
	assert.NotEmpty(t, configFilePath)

	preTestContent, err := os.ReadFile(configFilePath)
	assert.Nil(t, err)

	writeConfigFileContent := func(contentToWrite []byte) error {
		return os.WriteFile(configFilePath, contentToWrite, os.ModePerm)
	}

	defer writeConfigFileContent(preTestContent)

	tests := []struct {
		name           string
		isEnabled      int
		expectedResult bool
	}{
		{
			name:           "SSH Root Access is enabled",
			isEnabled:      1,
			expectedResult: true,
		},
		{
			name:           "SSH Root Access is disabled",
			isEnabled:      0,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			configFile := configFile{RootAccessEnabled: tt.isEnabled}
			configFileForTestCase, err := json.Marshal(&configFile)
			assert.Nil(t, err)
			err = writeConfigFileContent(configFileForTestCase)
			assert.Nil(t, err)

			uut := system.NewuOSSystem("")

			// Act
			result, err := uut.IsSshRootAccessEnabled()

			// Assert
			if err != nil {
				t.Fatalf("Expected nil but got %v", err)
			}

			if result != tt.expectedResult {
				t.Fatalf("Expected result '%t' but got '%t'", tt.expectedResult, result)
			}
		})
	}
}

func TestAvailableSpaceInBytesPass(t *testing.T) {
	// Arrange
	uut := system.NewuOSSystem("/go/src/uc-aom/volatile")

	// Act
	sizeInBytes, err := uut.AvailableSpaceInBytes()

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if sizeInBytes <= 0 {
		t.Fatalf("Expected to be greater than zero")
	}
}

func TestAvailabeSpaceInBytesFail(t *testing.T) {
	// Arrange
	uut := system.NewuOSSystem("/does/not/exsit")

	// Act
	sizeInBytes, err := uut.AvailableSpaceInBytes()

	// Assert
	if err == nil {
		t.Errorf("Expected error, none received.")
	}

	if sizeInBytes > 0 {
		t.Errorf("Mismatching size. Want 0, Got %v", sizeInBytes)
	}
}
