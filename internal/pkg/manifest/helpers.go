// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"fmt"
	"strings"
)

// Returns the docker image references from the manifest.
func GetDockerImageReferences(services map[string]*Service) []string {
	images := make([]string, 0, len(services))
	for _, service := range services {
		if val, ok := service.Config["image"]; ok {
			images = append(images, fmt.Sprintf("%v", val))
		}
	}

	return images
}

// Returns any used volume names from the manifest.
func GetVolumeNames(environments map[string]*Environment) []string {
	volumeNames := make([]string, 0)
	for _, environment := range environments {
		for volumeName, volumeSettings := range environment.Config.Volumes {
			name := returnTopLevelNameOrFromSettings(volumeName, volumeSettings)
			volumeNames = append(volumeNames, name)
		}
	}
	return volumeNames
}

// Return True if one of the volumes uses the local-public driver otherwise false
func HasLocalPublicVolumes(environments map[string]*Environment) bool {
	for _, environment := range environments {
		for _, volumeSettings := range environment.Config.Volumes {
			if isLocalPublicVolume(volumeSettings) {
				return true
			}
		}
	}
	return false
}

// Return True if the service uses a local public volumes otherwise false
func UsesLocalPublicVolume(service *Service, environments map[string]*Environment) bool {
	localPublicVolumes := filterVolumes(environments, isLocalPublicVolume)

	if serviceVolumes, ok := service.Config["volumes"].([]interface{}); ok {
		for topLevelName, volumesSettings := range localPublicVolumes {
			name := returnTopLevelNameOrFromSettings(topLevelName, volumesSettings)
			for _, serviceVolume := range serviceVolumes {

				if serviceVolumeString, ok := serviceVolume.(string); ok {
					if strings.Contains(serviceVolumeString, name) {
						return true
					}
				}

			}
		}
	}
	return false

}

func filterVolumes(environments map[string]*Environment, filter func(volumesSetting map[string]interface{}) bool) map[string]map[string]interface{} {
	filterVolumes := make(map[string]map[string]interface{})

	for _, environment := range environments {
		for name, settings := range environment.Config.Volumes {
			if filter(settings) {
				filterVolumes[name] = settings
			}
		}
	}
	return filterVolumes
}

func isLocalPublicVolume(volumeSettings map[string]interface{}) bool {
	if value, hasDriver := volumeSettings["driver"].(string); hasDriver {
		return strings.Contains(value, LocalPublicVolumeDriverName)
	}
	return false
}

func returnTopLevelNameOrFromSettings(topLevelName string, settings map[string]interface{}) string {
	if name, exists := settings["name"].(string); exists {
		return name
	}
	return topLevelName
}
