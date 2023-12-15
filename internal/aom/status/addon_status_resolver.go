// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package status

import "strings"

// type of the add-on status
type AddOnStatus int

const (
	// Starting status while healthcheck is executing
	Starting AddOnStatus = iota

	// Normal running status if all is ok
	Running AddOnStatus = iota

	// Unhealthy status if healthcheck doesn't complete successfully
	Unhealthy AddOnStatus = iota

	// Error status if add-on exited
	Error AddOnStatus = iota
)

// Callback function to get all add-on containers
type ListAddOnContainersFunc func(name string) ([]*ListAddOnContainersFuncReturnType, error)

type ListAddOnContainersFuncReturnType struct {
	// Status of the add-on container, needs to be parsed by the resolver
	Status string
}

// AddOnStatusResolver resolves the status of an add-on
type AddOnStatusResolver struct {
	listAddOnContainersFunc ListAddOnContainersFunc
}

// creates a new instance of AddOnStatusResolver
func NewAddOnStatusResolver(listAddOnContainersFunc ListAddOnContainersFunc) *AddOnStatusResolver {
	return &AddOnStatusResolver{
		listAddOnContainersFunc,
	}
}

// return the add-on status combined of all add-on containers
func (s *AddOnStatusResolver) GetAddOnStatus(name string) (AddOnStatus, error) {

	addOnContainers, err := s.listAddOnContainersFunc(name)
	if err != nil {
		return Error, err
	}

	addOnStatus := Error
	for _, addOnContainer := range addOnContainers {
		addOnStatus = s.convertContainerStatusToAddOnStatus(addOnContainer.Status)
		if addOnStatus != Running {
			return addOnStatus, nil
		}
	}
	return addOnStatus, nil
}

func (s *AddOnStatusResolver) convertContainerStatusToAddOnStatus(status string) AddOnStatus {
	statusConverted := strings.ToLower(status)

	if strings.HasPrefix(statusConverted, "exited") {
		return Error
	} else if strings.Contains(statusConverted, "(healthy)") {
		return Running
	} else if strings.Contains(statusConverted, "(unhealthy)") {
		return Unhealthy
	} else if strings.Contains(statusConverted, "(health: starting)") {
		return Starting
	}
	return Running
}
