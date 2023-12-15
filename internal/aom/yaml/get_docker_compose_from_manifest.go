// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package yaml

import (
	"fmt"
	"regexp"
	"strings"
	"u-control/uc-aom/internal/pkg/manifest"

	yaml3 "gopkg.in/yaml.v3"
)

const (
	// SUPPORTED_COMPOSE_FILE_VERSION the version of docker-compose that is supported by uc-aom
	SUPPORTED_COMPOSE_FILE_VERSION = "2"
)

// GetDockerComposeFromManifest returns a docker-compose string from the property values
// found in the manifest
func GetDockerComposeFromManifest(manifestRoot *manifest.Root) (string, error) {

	if manifestRoot == nil {
		return "", &yaml3.TypeError{Errors: []string{"Argument nil"}}
	}

	dockerComposeServices := getDockerComposeServicesFrom(manifestRoot.Services, manifestRoot.Settings)
	dockerComposeVolumes := getDockerComposeVolumesFrom(manifestRoot.Environments)
	dockerComposeNetworks := getDockerComposeNetworksFrom(manifestRoot.Environments)

	compose := manifestToDockerCompose{
		version:  SUPPORTED_COMPOSE_FILE_VERSION,
		services: dockerComposeServices,
		volumes:  dockerComposeVolumes,
		networks: dockerComposeNetworks,
	}
	dockerComposeYAML, err := yaml3.Marshal(&compose)

	return string(dockerComposeYAML), err
}

func getDockerComposeServicesFrom(manifestServices map[string]*manifest.Service, manifestSettings map[string][]*manifest.Setting) map[string]interface{} {
	dockerComposeServices := make(map[string]interface{})

	for name, service := range manifestServices {
		if service.Type != "docker-compose" {
			continue
		}

		config := service.Config
		mergeEnvironmentVariables(config, manifestSettings)
		dockerComposeServices[name] = config
	}
	return dockerComposeServices
}

func getDockerComposeEnvironmentsFrom(manifestEnvironments map[string]*manifest.Environment) map[string]manifest.EnvironmentConfig {
	dockerComposeEnvironments := make(map[string]manifest.EnvironmentConfig)

	for name, environment := range manifestEnvironments {
		if environment.Type != "docker-compose" {
			continue
		}
		dockerComposeEnvironments[name] = environment.Config
	}
	return dockerComposeEnvironments
}

func getDockerComposeVolumesFrom(manifestEnvironments map[string]*manifest.Environment) map[string]interface{} {
	dockerComposeEnvironments := getDockerComposeEnvironmentsFrom(manifestEnvironments)
	dockerComposeVolumes := make(map[string]interface{})
	for _, environmentConfig := range dockerComposeEnvironments {
		for volumeName, volumeSettings := range environmentConfig.Volumes {
			if dockerComposeVolumes[volumeName] = nil; len(volumeSettings) > 0 {
				dockerComposeVolumes[volumeName] = volumeSettings
			}
		}
	}

	return dockerComposeVolumes
}

func getDockerComposeNetworksFrom(manifestEnvironments map[string]*manifest.Environment) map[string]interface{} {
	dockerComposeEnvironments := getDockerComposeEnvironmentsFrom(manifestEnvironments)
	dockerComposeNetworks := make(map[string]interface{})
	for _, environmentConfig := range dockerComposeEnvironments {
		for networkName, networkSettings := range environmentConfig.Networks {
			if dockerComposeNetworks[networkName] = nil; len(networkSettings) > 0 {
				dockerComposeNetworks[networkName] = networkSettings
			}
		}
	}

	return dockerComposeNetworks
}

func mergeEnvironmentVariables(config map[string]interface{}, manifestSettings map[string][]*manifest.Setting) {
	if settings, ok := manifestSettings["environmentVariables"]; ok {
		environment := getEnvironmentAsMapFrom(config)
		settingsEnvironment := getDockerComposeEnvironmentMapFrom(settings)
		for key := range settingsEnvironment {
			environment[key] = settingsEnvironment[key]
		}

		config["environment"] = environment
	}
}

func getEnvironmentAsMapFrom(config map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	if environment, ok := config["environment"]; ok {
		if environmentAsArray, ok := environment.([]interface{}); ok {
			for _, keyValue := range environmentAsArray {
				// The cast cannot fail, since it was encoded as JSON
				key, value := toKeyValue(keyValue.(string))
				result[key] = value
			}
		} else if environmentAsMap, ok := environment.(map[string]interface{}); ok {
			for key, value := range environmentAsMap {
				result[key] = fmt.Sprintf("%v", value)
			}
		}
	}

	return result
}

func toKeyValue(keyValue string) (string, interface{}) {
	result := strings.SplitN(keyValue, "=", 2)
	return result[0], result[1]
}

func getDockerComposeEnvironmentMapFrom(settings []*manifest.Setting) map[string]interface{} {
	transformed := make(map[string]interface{}, len(settings))
	for _, setting := range settings {
		if setting.Select == nil {
			transformed[setting.Name] = setting.Value
		} else {
			for _, selectItem := range setting.Select {
				if selectItem.Selected {
					transformed[setting.Name] = selectItem.Value
					break
				}
			}
		}
	}

	return transformed
}

type manifestToDockerCompose struct {
	version  string
	services map[string]interface{}
	volumes  map[string]interface{}
	networks map[string]interface{}
}

func (compose *manifestToDockerCompose) MarshalYAML() (interface{}, error) {
	yaml := make(map[string]interface{})
	yaml["version"] = compose.version

	if len(compose.services) != 0 {
		yaml["services"] = convertManifestSettings(compose.services)
	}

	if len(compose.volumes) != 0 {
		yaml["volumes"] = convertManifestSettings(compose.volumes)
	}

	if len(compose.networks) != 0 {
		yaml["networks"] = convertManifestSettings(compose.networks)
	}

	return yaml, nil
}

func convertManifestSettings(composeSetting map[string]interface{}) map[string]interface{} {
	convertedSettings := make(map[string]interface{})
	for settingsName, settingsValue := range composeSetting {
		switch configSettingsTyped := settingsValue.(type) {
		case map[string]interface{}:
			convertedSettings[settingsName] = convertDockerSettingWithSnakeCasedKeys(configSettingsTyped)

		default:
			convertedSettings[settingsName] = settingsValue
		}
	}
	return convertedSettings
}

func convertDockerSettingWithSnakeCasedKeys(dockerConfig map[string]interface{}) map[string]interface{} {
	composeService := make(map[string]interface{})
	for configKey, configSettings := range dockerConfig {
		// Don't convert the environment variables to snake case!
		if configKey == "environment" {
			composeService[configKey] = configSettings
			continue
		}

		convertedConfigKey := toSnakeCase(configKey)
		switch configSettingsTyped := configSettings.(type) {
		case map[string]interface{}:
			composeService[convertedConfigKey] = convertDockerSettingWithSnakeCasedKeys(configSettingsTyped)
		default:
			composeService[convertedConfigKey] = configSettings
		}
	}
	return composeService
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}
