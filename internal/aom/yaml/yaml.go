// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package yaml

import (
	yaml3 "gopkg.in/yaml.v3"
	"strings"
)

type compose struct {
	Services map[string]*Service `yaml:"services"`
}

type Service struct {
	Ports []string `yaml:"ports"`
}

func getStackServices(stack *compose) (ret []*Service) {
	for _, value := range stack.Services {
		ret = append(ret, value)
	}
	return
}

func arrfilter(services []*Service, test func(service *Service) bool) (ret []*Service) {
	for _, s := range services {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func arrMap(services []*Service, new func(service *Service) *Service) (ret []*Service) {
	for _, s := range services {
		ret = append(ret, new(s))
	}
	return
}

// The GetComposeServicesPublicPortsFrom returns all host ports found in the given compose data
func GetComposeServicesPublicPortsFrom(composeData string) ([]*Service, error) {
	byteData := []byte(composeData)
	stackCompose := compose{}
	err := yaml3.Unmarshal(byteData, &stackCompose)
	if err != nil {
		return nil, err
	}

	services := getStackServices(&stackCompose)

	servicesWithHostPorts := arrfilter(services, func(service *Service) bool {
		for _, port := range service.Ports {
			if strings.Contains(port, ":") {
				return true
			}
		}
		return false
	})

	servicesHostPorts := arrMap(servicesWithHostPorts, func(service *Service) *Service {
		return &Service{
			Ports: parseHostPorts(service.Ports),
		}
	})

	return servicesHostPorts, nil
}

func parseHostPorts(portMaps []string) []string {
	var hostPorts []string
	for _, p := range portMaps {
		portMapSplit := strings.Split(p, ":")
		if len(portMapSplit) == 2 {
			hostPorts = append(hostPorts, portMapSplit[0])
		}
	}
	return hostPorts
}
