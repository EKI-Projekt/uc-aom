// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package status

import (
	"u-control/uc-aom/internal/aom/docker"
)

// Creates a new instance of AddOnStatusResolver with docker client
func AdaptAddOnStatusResolverToDocker(stackService docker.StackServiceAPI) *AddOnStatusResolver {
	var listAddOnContainersFunc ListAddOnContainersFunc
	listAddOnContainersFunc = func(name string) ([]*ListAddOnContainersFuncReturnType, error) {
		stackContainers, err := stackService.ListAllStackContainers(name)
		if err != nil {
			return nil, err
		}
		addContainers := make([]*ListAddOnContainersFuncReturnType, len(stackContainers))

		for index, stackContainer := range stackContainers {
			addContainers[index] = &ListAddOnContainersFuncReturnType{Status: stackContainer.Status}
		}

		return addContainers, nil
	}
	return NewAddOnStatusResolver(listAddOnContainersFunc)
}
