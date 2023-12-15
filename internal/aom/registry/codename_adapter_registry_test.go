// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry_test

import (
	"fmt"
	"testing"
	"u-control/uc-aom/internal/aom/registry"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRepositories_WithCodeNameRepository(t *testing.T) {
	type args struct {
		codeName       string
		normalizedName string
	}

	testCases := []args{
		{
			codeName:       "abc/test-uc-addon-01",
			normalizedName: "test-uc-addon-01",
		},
		{
			codeName:       "v2/test-uc-addon-02",
			normalizedName: "test-uc-addon-02",
		},
		{
			codeName:       "1/test-uc-addon-03",
			normalizedName: "test-uc-addon-03",
		},
		{
			codeName:       "test-uc-addon-04",
			normalizedName: "test-uc-addon-04",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Normalizing codeName %s", tc.codeName), func(t *testing.T) {
			// Arrange
			mockRegistry := &registry.MockRegistry{}
			uut := createUutWithRegistry(mockRegistry)
			mockRegistry.On("Repositories").Return([]string{tc.codeName}, nil)

			// Act
			got, err := uut.Repositories()

			// Assert
			assert.Nil(t, err)
			gotName := got[0]
			if gotName != tc.normalizedName {
				t.Errorf("Expected name to be %s but got %s", tc.normalizedName, gotName)
			}
		})

	}
}

func TestTags_WithCodeNameRepository(t *testing.T) {
	type args struct {
		codeName       string
		normalizedName string
		versions       []string
	}

	testCases := []args{
		{
			codeName:       "abc/test-uc-addon-01",
			normalizedName: "test-uc-addon-01",
			versions:       []string{"0.1.0-1"},
		},
		{
			codeName:       "v2/test-uc-addon-02",
			normalizedName: "test-uc-addon-02",
			versions:       []string{"0.1.0-1"},
		},
		{
			codeName:       "1/test-uc-addon-03",
			normalizedName: "test-uc-addon-03",
			versions:       []string{"0.1.0-1"},
		},
		{
			codeName:       "test-uc-addon-04",
			normalizedName: "test-uc-addon-04",
			versions:       []string{"0.1.0-1"},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("'%s' should get tags using the code name '%s'", tc.normalizedName, tc.codeName), func(t *testing.T) {
			// Arrange
			mockRegistry := &registry.MockRegistry{}
			mockRegistry.On("Repositories").Return([]string{tc.codeName}, nil)
			mockRegistry.On("Tags", tc.codeName).Return(tc.versions, nil)
			uut := createUutWithRegistry(mockRegistry)

			// Act
			_, err := uut.Tags(tc.normalizedName)

			// Assert
			assert.Nil(t, err)
			mockRegistry.AssertExpectations(t)
		})
	}
}

func TestPull_WithCodeNameRepository(t *testing.T) {
	type args struct {
		codeName       string
		normalizedName string
		versions       []string
	}

	testCases := []args{
		{
			codeName:       "abc/test-uc-addon-01",
			normalizedName: "test-uc-addon-01",
			versions:       []string{"0.1.0-1"},
		},
		{
			codeName:       "v2/test-uc-addon-02",
			normalizedName: "test-uc-addon-02",
			versions:       []string{"0.1.0-1"},
		},
		{
			codeName:       "1/test-uc-addon-03",
			normalizedName: "test-uc-addon-03",
			versions:       []string{"0.1.0-1"},
		},
		{
			codeName:       "test-uc-addon-04",
			normalizedName: "test-uc-addon-04",
			versions:       []string{"0.1.0-1"},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("'%s' should pull using the code name '%s'", tc.normalizedName, tc.codeName), func(t *testing.T) {
			// Arrange
			mockRegistry := &registry.MockRegistry{}
			uut := createUutWithRegistry(mockRegistry)

			mockRegistry.On("Repositories").Return([]string{tc.codeName}, nil)
			mockRegistry.On(
				"Pull",
				tc.codeName,
				"0.1.0-1",
				mock.AnythingOfType("*registry.ucImageLayerProcessor")).Return(uint64(0), nil)

			// Act
			ilp := registry.NewUcImageLayerProcessor(nil)
			_, err := uut.Pull(tc.normalizedName, tc.versions[0], ilp)

			// Assert
			assert.Nil(t, err)
			mockRegistry.AssertExpectations(t)
		})
	}
}

func TestDelete_WithCodeNameRepository(t *testing.T) {
	type args struct {
		codeName       string
		normalizedName string
		version        string
	}

	testCases := []args{
		{
			codeName:       "abc/test-uc-addon-01",
			normalizedName: "test-uc-addon-01",
			version:        "0.1.0-1",
		},
		{
			codeName:       "v2/test-uc-addon-02",
			normalizedName: "test-uc-addon-02",
			version:        "0.1.0-1",
		},
		{
			codeName:       "1/test-uc-addon-03",
			normalizedName: "test-uc-addon-03",
			version:        "0.1.0-1",
		},
		{
			codeName:       "test-uc-addon-04",
			normalizedName: "test-uc-addon-04",
			version:        "0.1.0-1",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("'%s' should get tags using the code name '%s'", tc.normalizedName, tc.codeName), func(t *testing.T) {
			// Arrange
			mockRegistry := &registry.MockRegistry{}
			mockRegistry.On("Repositories").Return([]string{tc.codeName}, nil)
			mockRegistry.On("Delete", tc.codeName, tc.version).Return(nil)
			uut := createUutWithRegistry(mockRegistry)

			// Act
			err := uut.Delete(tc.normalizedName, tc.version)

			// Assert
			assert.Nil(t, err)
			mockRegistry.AssertExpectations(t)
		})
	}
}

func TestShallReturnNotFoundIfRepositoryCanNotBeFound(t *testing.T) {
	// Arrange
	repository := "test"
	mockRegistry := &registry.MockRegistry{}
	mockRegistry.On("Repositories").Return([]string{"test1", "test2", "codename/test3"}, nil)
	uut := createUutWithRegistry(mockRegistry)

	// Act
	result, err := uut.Tags(repository)

	// Assert
	assert.Empty(t, result)
	assert.Error(t, err)
	mockRegistry.AssertExpectations(t)
}

func createUutWithRegistry(mockRegistry registry.AddOnRegistry) registry.AddOnRegistry {
	return registry.NewCodeNameAdapterRegistry(mockRegistry)
}
