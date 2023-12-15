// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package utils_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"u-control/uc-aom/internal/aom/utils"

	"github.com/stretchr/testify/assert"
)

func TestMkDirAll(t *testing.T) {
	// Arrange
	type args struct {
		name               string
		expectedPermission os.FileMode
	}

	testCases := []args{
		{
			name:               "abc",
			expectedPermission: os.ModeDir | os.FileMode(0775),
		},
		{
			name:               "xyz",
			expectedPermission: os.ModeDir | os.FileMode(0666),
		},
	}

	testPath := t.TempDir()

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Test created dir permission of %s", tc.name), func(t *testing.T) {
			// Act
			path := filepath.Join(testPath, tc.name)
			err := utils.MkDirAll(path, tc.expectedPermission)

			// Assert
			assert.Nil(t, err)
			info, err := os.Stat(path)
			assert.Nil(t, err)
			assert.Equal(t, tc.expectedPermission, info.Mode())
		})
	}

	t.Cleanup(func() {
		os.RemoveAll(testPath)
	})
}

func TestWriteFileToDestination(t *testing.T) {
	type args struct {
		name string
	}

	testCases := []args{
		{
			name: "abc",
		},
		{
			name: "xyz",
		},
	}

	testPath := t.TempDir()

	for _, tc := range testCases {
		// Act
		err := utils.WriteFileToDestination(tc.name, []byte{0}, testPath)

		// Assert
		assert.Nil(t, err)
		target := filepath.Join(testPath, tc.name)
		info, err := os.Stat(target)
		assert.Nil(t, err)
		assert.Equal(t, os.FileMode(0644), info.Mode())
	}

	t.Cleanup(func() {
		os.RemoveAll(testPath)
	})
}
