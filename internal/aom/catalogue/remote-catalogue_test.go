// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package catalogue_test

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/registry"
	"u-control/uc-aom/internal/pkg/manifest"
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	architecture = "arm"
	OS           = "linux"
)

type mockManifestReader struct {
	mock.Mock
}

func (r *mockManifestReader) ReadManifestFrom(directoryOfManifest string) (*manifest.Root, error) {
	args := r.Called(directoryOfManifest)
	return args.Get(0).(*manifest.Root), args.Error(1)
}

func TestVersion(t *testing.T) {
	type testCaseData struct {
		tagsFromRepository   []string
		expectedFilteredTags []string
	}
	testCases := []testCaseData{
		{
			[]string{"0.9.9.3-alpha.1-1", "0.8.9.1-1", "0.9.9.3-1", "0.9.9.3-beta.1-1", "0.9.9.9-1", "1.0.0-rc.4-1"},
			[]string{"0.8.9.1-1", "0.9.9.3-alpha.1-1", "0.9.9.3-beta.1-1", "0.9.9.3-1", "0.9.9.9-1", "1.0.0-rc.4-1"},
		},
		{
			[]string{"0.9.9.3-1-alpha.1", "0.8.9.1-1", "0.9.9.3-1", "0.9.9.3-1-beta.1", "0.9.9.9-1", "1.0.0-1-rc.4"},
			[]string{"0.8.9.1-1", "0.9.9.3-1-alpha.1", "0.9.9.3-1-beta.1", "0.9.9.3-1", "0.9.9.9-1", "1.0.0-1-rc.4"},
		},
		{
			[]string{"1.0.0-1", "0.9.9-1", "1.0.0-rc.9-1", "0.1.1-1"},
			[]string{"0.1.1-1", "0.9.9-1", "1.0.0-rc.9-1", "1.0.0-1"},
		},
		{
			[]string{"1.0.0-1", "1.0.0-1"},
			[]string{"1.0.0-1", "1.0.0-1"},
		},
		{
			[]string{"1.0.0", "1.0.0-", "1.0.0-1", "-1"},
			[]string{"1.0.0", "1.0.0-", "1.0.0-1", "-1"},
		},
		{
			[]string{"0.4-rc.5-1", "0.2-beta.1-1", "0.1-1", "0.5-1", "0.2-1"},
			[]string{"0.1-1", "0.2-beta.1-1", "0.2-1", "0.4-rc.5-1", "0.5-1"},
		},
		{
			[]string{"HKO_2.3-45", "HKO_1.2-34", "HKO_3.4-56"},
			[]string{"HKO_2.3-45", "HKO_1.2-34", "HKO_3.4-56"},
		},
		{
			[]string{"1.9.9.3-alpha.1-1", "2.8.9.1-1", "1.9.9.3-1", "0.9.9.3-beta.1-1", "0.9.9.9-1", "3.0.0-rc.4-1"},
			[]string{"0.9.9.3-beta.1-1", "0.9.9.9-1", "1.9.9.3-alpha.1-1", "1.9.9.3-1", "2.8.9.1-1", "3.0.0-rc.4-1"},
		},
		{
			[]string{"1.9.9.3-1-alpha.1", "2.8.9.1-1", "1.9.9.3-1", "0.9.9.3-1-beta.1", "0.9.9.9-1", "3.0.0-1-rc.4"},
			[]string{"0.9.9.3-1-beta.1", "0.9.9.9-1", "1.9.9.3-1-alpha.1", "1.9.9.3-1", "2.8.9.1-1", "3.0.0-1-rc.4"},
		},
	}

	for _, testCase := range testCases {
		t.Run(fmt.Sprintf("Sorting #[%s]", strings.Join(testCase.tagsFromRepository, "|")), func(t *testing.T) {
			// Arrange
			mockRegistry := &registry.MockRegistry{}
			uut := createUut(mockRegistry)

			addOnRepositoryName := "addOnRepoName"
			mockRegistry.On("Tags", addOnRepositoryName).Return(testCase.tagsFromRepository, nil)

			// Act
			actual, err := uut.GetAddOnVersions(addOnRepositoryName)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Assert
			if !reflect.DeepEqual(actual, testCase.expectedFilteredTags) {
				t.Errorf("Sort failed.\n\tExpected: %#v.\n\tActual: %#v", testCase.expectedFilteredTags, actual)
			}
		})
	}
}

func TestRemoteRegistryConnectionProblem(t *testing.T) {
	// Arrange
	mockRegistry := &registry.MockRegistry{}
	uut := createUut(mockRegistry)
	networkError := errors.New("NETWORK_ERROR")

	mockRegistry.On("Repositories").Return([]string{}, networkError)

	// Act
	payload, err := uut.GetLatestAddOns()

	// Assert
	if payload != nil {
		t.Errorf("Unexpected return value: %#v", payload)
	}

	if err == nil {
		t.Errorf("Expected error, none received.")
	}

	if _, ok := err.(*catalogue.RemoteRegistryConnectionError); !ok {
		t.Errorf("Expected error of type catalogue.RemoteRegistryConnectionError, Actual: %#v .", err)
	}

}

func TestGetLatestAddOns(t *testing.T) {
	mockRegistry := &registry.MockRegistry{}
	mockManifestReader := &mockManifestReader{}
	uut := createUutWithManifestReader(mockRegistry, mockManifestReader)

	type args struct {
		repository string
		tags       []string
		pullError  error
	}

	registryArgs := []args{
		{
			"passRepo",
			[]string{"0.1.0-1", "0.1.0-2", "0.2.0-1"},
			nil,
		},
		{
			"failedRepo",
			[]string{"1.1.0-1", "1.1.0-2", "1.2.0-1"},
			errors.New("Pull-Failed"),
		},
	}

	type want struct {
		repository string
		tag        string
	}

	wantLatestAddOns := []want{{
		"passRepo",
		"0.2.0-1",
	}}

	wantRepositories := make([]string, 0, len(registryArgs))
	for _, registryArg := range registryArgs {
		wantRepositories = append(wantRepositories, registryArg.repository)
	}

	mockRegistry.On("Repositories").Return(wantRepositories, nil)

	for _, registryArg := range registryArgs {
		mockRegistry.On("Tags", registryArg.repository).Return(registryArg.tags, nil)

		latestTag := registryArg.tags[len(registryArg.tags)-1]
		mockRegistry.On("Pull",
			registryArg.repository,
			latestTag,
			mock.AnythingOfType("*registry.ucImageLayerProcessor")).Return(uint64(0), registryArg.pullError)

		if registryArg.pullError == nil {
			mockManifestReader.On("ReadManifestFrom", registryArg.repository).Return(&manifest.Root{Version: latestTag, Title: registryArg.repository}, nil)
		}
	}

	// act
	got, err := uut.GetLatestAddOns()

	// assert
	assert.Nil(t, err)

	gotLatestAddOns := make([]want, 0, len(got))
	for _, gotElement := range got {
		gotLatestAddOns = append(gotLatestAddOns, want{gotElement.Name, gotElement.Version})
	}

	assert.Equal(t, wantLatestAddOns, gotLatestAddOns)
	mockRegistry.AssertExpectations(t)

}

func createUut(mockRegistry registry.AddOnRegistry) *catalogue.ORASRemoteAddOnCatalogue {
	uut := catalogue.NewORASRemoteAddOnCatalogue("", mockRegistry, nil)
	return uut
}

func createUutWithManifestReader(mockRegistry registry.AddOnRegistry, mockManifestReader model.ManifestFileReader) *catalogue.ORASRemoteAddOnCatalogue {
	uut := catalogue.NewORASRemoteAddOnCatalogue("", mockRegistry, mockManifestReader)
	return uut
}
