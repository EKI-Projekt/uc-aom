// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package env_test

import (
	"reflect"
	"testing"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/aom/env"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/mock"
)

func TestGetAddOnEnvironment(t *testing.T) {
	// arrange
	tests := []struct {
		testCase string
		input    []string
		expected map[string]string
	}{
		{
			"Empty",
			[]string{""},
			map[string]string{},
		},
		{
			"Non-empty",
			[]string{"KEY_0=Value", "KEY_1=sOmEpAsS=wOrD"},
			map[string]string{"KEY_0": "Value", "KEY_1": "sOmEpAsS=wOrD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			uut, mockStackService := createUut(tt.input)

			// act
			actual, err := uut.GetAddOnEnvironment("test")

			// assert
			if err != nil {
				t.Errorf("Received unexpected error: %v", err)
			}

			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("Expected: %#v \t Actual: %#v", tt.expected, actual)
			}

			mockStackService.AssertExpectations(t)
		})
	}

}

func createUut(input []string) (*env.AddOnEnvironmentResolver, *docker.MockStackService) {
	container := types.Container{
		ID: "id-test-case",
	}

	containerInfo := &docker.ContainerInfo{
		Config: &docker.Config{Env: input},
	}

	mockStackService := &docker.MockStackService{}
	mockStackService.On("ListAllStackContainers", mock.AnythingOfType("string")).Return([]types.Container{container}, nil)
	mockStackService.On("InspectContainer", "id-test-case").Return(containerInfo, nil)

	return env.NewAddOnEnvironmentResolver(mockStackService), mockStackService
}
