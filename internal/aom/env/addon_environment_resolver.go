// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package env

import (
	"errors"
	"fmt"
	"strings"
	"u-control/uc-aom/internal/aom/docker"

	log "github.com/sirupsen/logrus"
)

type EnvResolver interface {
	GetAddOnEnvironment(name string) (map[string]string, error)
}

// AddOnEnvironmentResolver resolves the currently used environment of an add-on.
type AddOnEnvironmentResolver struct {
	stackService docker.StackServiceAPI
}

// Creates a new instance of AddOnEnvironmentResolver with the given stackService.
func NewAddOnEnvironmentResolver(stackService docker.StackServiceAPI) *AddOnEnvironmentResolver {
	return &AddOnEnvironmentResolver{stackService: stackService}
}

// From the total set of containers that an add-On has,
// returns the first successful retrieval of the environment container.
func (s *AddOnEnvironmentResolver) GetAddOnEnvironment(name string) (map[string]string, error) {
	envPayload, err := s.getEnvironmentFromFirstContainer(name)
	if err != nil {
		log.Warningf("Failed to retrieve environment for stack (%s): %s.", name, err.Error())
		return nil, err
	}

	return s.convertContainerEnvToMap(envPayload...), nil
}

func (s *AddOnEnvironmentResolver) getEnvironmentFromFirstContainer(name string) ([]string, error) {
	stackContainers, err := s.stackService.ListAllStackContainers(name)
	if err != nil {
		return nil, err
	}

	var combinedErrors strings.Builder
	for _, container := range stackContainers {
		containerInfo, err := s.stackService.InspectContainer(container.ID)
		if err != nil {
			fmt.Fprintf(&combinedErrors, "%s\n", err.Error())
			continue
		}

		return containerInfo.Config.Env, nil
	}

	if combinedErrors.Len() != 0 {
		return make([]string, 0), errors.New(combinedErrors.String())
	}

	return make([]string, 0), nil
}

func (s *AddOnEnvironmentResolver) convertContainerEnvToMap(environment ...string) map[string]string {
	result := make(map[string]string, len(environment))
	for _, env := range environment {
		split := strings.SplitN(env, "=", 2)
		if len(split) != 2 {
			continue
		}

		result[split[0]] = split[1]
	}
	return result
}
