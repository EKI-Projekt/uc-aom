// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package test

import (
	"os"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/cmd"
	"u-control/uc-aom/internal/pkg/config"
	testhelpers "u-control/uc-aom/test/test-helpers"
)

func TestUcAomSetup(t *testing.T) {
	// Assert
	testhelpers.PrepareEnvironment(t)
	uut := cmd.NewUcAom(nil)

	// Act
	err := uut.Setup()

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	_, err = os.Stat(config.CACHE_DROP_IN_PATH)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestUcAomSetupMissingDirectory(t *testing.T) {
	type testCaseData struct {
		caseName          string
		expectedErrorPath *string
	}
	testCases := []testCaseData{
		{
			"ASSETS_INSTALL_PATH",
			&catalogue.ASSETS_INSTALL_PATH,
		},
		{
			"ASSETS_TMP_PATH",
			&catalogue.ASSETS_TMP_PATH,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.caseName, func(t *testing.T) {
			// Arrange
			testhelpers.PrepareEnvironment(t)
			uut := cmd.NewUcAom(nil)
			os.Remove(*testCase.expectedErrorPath)

			// Act
			err := uut.Setup()

			// Assert
			if err == nil {
				t.Fatalf("Expected error but got nil.")
			}

			if !strings.Contains(err.Error(), *testCase.expectedErrorPath) {
				t.Fatalf("Unexpected error: %v", err)
			}

		})
	}
}
