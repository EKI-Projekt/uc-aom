// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package status_test

import (
	"errors"
	"testing"
	"u-control/uc-aom/internal/aom/status"
)

func TestGetAddOnStatusRunning(t *testing.T) {
	// arrange

	tests := []struct {
		testCase string
		Status   []string
		want     status.AddOnStatus
	}{
		{
			"Shall return Running",
			[]string{"Up 3 days"},
			status.Running,
		},
		{
			"Shall return Error if status is exited",
			[]string{"Exited (127) 3 seconds ago"},
			status.Error,
		},
		{
			"Shall return Starting if status is (health: starting)",
			[]string{"Up 3 seconds (health: starting)"},
			status.Starting,
		},
		{
			"Shall return unhealthy if status is unhealthy",
			[]string{"Up 2 minutes (unhealthy)"},
			status.Unhealthy,
		},
		{
			"Shall return Running if all container are running",
			[]string{"Up 3 days", "Up 3 days"},
			status.Running,
		},
		{
			"Shall return Error if one of the container is in error",
			[]string{"Up 3 days", "Exited (127) 3 seconds ago"},
			status.Error,
		},
		{
			"Shall return Starting if one of the container is starting ",
			[]string{"Up 3 days", "Up 3 seconds (health: starting)"},
			status.Starting,
		},
		{
			"Shall return unhealthy if one of the container is unhealthy ",
			[]string{"Up 3 days", "Up 2 minutes (unhealthy)"},
			status.Unhealthy,
		},
		{
			"Shall return Error if status is empty",
			[]string{},
			status.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			result := make([]*status.ListAddOnContainersFuncReturnType, len(tt.Status))
			for index, statusString := range tt.Status {
				result[index] = &status.ListAddOnContainersFuncReturnType{Status: statusString}
			}
			uut := createUut(result)

			// act
			got, _ := uut.GetAddOnStatus("test")

			// assert
			if got != tt.want {
				t.Errorf("GetAddOnStatus('test') = %d; want %d", got, tt.want)
			}
		})
	}

}

func TestGetAddOnShallReturnErrorFromListAddOnContainers(t *testing.T) {
	// arrange
	want := errors.New("Test Error")
	mockFunction := func(name string) ([]*status.ListAddOnContainersFuncReturnType, error) {
		return nil, want
	}

	uut := status.NewAddOnStatusResolver(mockFunction)

	// act
	_, got := uut.GetAddOnStatus("test")

	// assert
	if got == nil {
		t.Errorf("GetAddOnStatus error is nil; want error '%s'", want.Error())
	}

}

func createUut(listAddOnContainersFuncResult []*status.ListAddOnContainersFuncReturnType) *status.AddOnStatusResolver {

	mockFunction := func(name string) ([]*status.ListAddOnContainersFuncReturnType, error) {
		return listAddOnContainersFuncResult, nil
	}

	return status.NewAddOnStatusResolver(mockFunction)
}
