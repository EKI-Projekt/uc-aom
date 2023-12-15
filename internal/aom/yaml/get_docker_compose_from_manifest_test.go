// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package yaml_test

import (
	"os"
	"reflect"
	"testing"
	"u-control/uc-aom/internal/aom/yaml"
	"u-control/uc-aom/internal/pkg/manifest"

	yaml3 "gopkg.in/yaml.v3"
)

func TestConvertDockerComposeWithVersion(t *testing.T) {
	services := make(map[string]*manifest.Service)

	manifestData := manifest.Root{
		Version:     "0.1",
		Title:       "AnyViz Cloud Adapter",
		Description: "The AnyViz cloud solution allows you to remotely monitor, control and analyse industrial PLCs, sensors and meters.",
		Logo:        "logoanyviz.png",
		Services:    services,
	}

	// act
	resultDockerComposeString, err := yaml.GetDockerComposeFromManifest(&manifestData)

	// assert
	if err != nil {
		t.Errorf("Failed creating docker compose from manifest file %v", err)
	}

	resultDockerComposeMap, errMap := createMapFrom(resultDockerComposeString)
	if errMap != nil {
		t.Errorf("Failed creating docker compose map %v", err)
	}

	expectedDockerCompose := make(map[interface{}]interface{})
	expectedDockerCompose["version"] = "2"

	eq := reflect.DeepEqual(resultDockerComposeMap, expectedDockerCompose)
	if !eq {
		t.Errorf("Expected \n%s but got \n%s", expectedDockerCompose, resultDockerComposeMap)
	}
}

func TestCreateDockerComposeFromManifest(t *testing.T) {
	// arrange
	serviceConfig := map[string]interface{}{"image": "anyviz/cloudadapter",
		"restart":       "always",
		"containerName": "anyviz",
		"ports":         []interface{}{"8888:8888"},
	}

	services := make(map[string]*manifest.Service)
	services["cloudadapter"] = &manifest.Service{
		Type:   "docker-compose",
		Config: serviceConfig,
	}

	manifestData := manifest.Root{
		Version:     "0.1",
		Title:       "AnyViz Cloud Adapter",
		Description: "The AnyViz cloud solution allows you to remotely monitor, control and analyse industrial PLCs, sensors and meters.",
		Logo:        "logoanyviz.png",
		Services:    services,
	}

	// act
	resultDockerComposeString, err := yaml.GetDockerComposeFromManifest(&manifestData)

	// assert
	if err != nil {
		t.Errorf("Failed creating docker compose from manifest file %v", err)
	}

	expectedDockerCompose := make(map[interface{}]interface{})
	expectedDockerCompose["version"] = "2"
	expectedServiceConfig := map[string]interface{}{"image": "anyviz/cloudadapter",
		"restart":        "always",
		"container_name": "anyviz",
		"ports":          []interface{}{"8888:8888"},
	}

	expectedDockerComposeService := make(map[string]interface{})
	expectedDockerComposeService["cloudadapter"] = expectedServiceConfig

	expectedDockerCompose["services"] = expectedDockerComposeService

	resultDockerComposeMap, errMap := createMapFrom(resultDockerComposeString)
	if errMap != nil {
		t.Errorf("Failed creating docker compose map %v", err)
	}

	eq := reflect.DeepEqual(resultDockerComposeMap, expectedDockerCompose)
	if !eq {
		t.Errorf("Expected \n%s but got \n%s", expectedDockerCompose, resultDockerComposeMap)
	}
}

func TestMultipleComposeServiceFromManifest(t *testing.T) {
	// arrange

	services := make(map[string]*manifest.Service)
	cloudAdapterServiceConfig := map[string]interface{}{
		"image":         "anyviz/cloudadapter",
		"restart":       "always",
		"containerName": "anyviz",
		"ports":         []interface{}{"8888:8888"},
	}
	services["cloudadapter"] = &manifest.Service{
		Type:   "docker-compose",
		Config: cloudAdapterServiceConfig,
	}

	nginxServiceConfig := map[string]interface{}{
		"image":         "nginx",
		"restart":       "always",
		"containerName": "nginx",
		"ports":         []interface{}{"8080:80"},
	}
	services["nginx"] = &manifest.Service{
		Type:   "docker-compose",
		Config: nginxServiceConfig,
	}

	manifestData := manifest.Root{
		Version:     "0.1",
		Title:       "AnyViz Cloud Adapter",
		Description: "The AnyViz cloud solution allows you to remotely monitor, control and analyse industrial PLCs, sensors and meters.",
		Logo:        "logoanyviz.png",
		Services:    services,
	}

	// act
	resultDockerComposeString, err := yaml.GetDockerComposeFromManifest(&manifestData)

	// assert
	if err != nil {
		t.Errorf("Failed creating docker compose from manifest file %v", err)
	}

	expectedDockerCompose := make(map[interface{}]interface{})
	expectedDockerCompose["version"] = "2"
	expectedCloudAdapterServiceConfig := map[string]interface{}{"image": "anyviz/cloudadapter",
		"restart":        "always",
		"container_name": "anyviz",
		"ports":          []interface{}{"8888:8888"},
	}

	expectedNginxServiceConfig := map[string]interface{}{"image": "nginx",
		"container_name": "nginx",
		"restart":        "always",
		"ports":          []interface{}{"8080:80"},
	}

	expectedDockerComposeService := make(map[string]interface{})
	expectedDockerComposeService["cloudadapter"] = expectedCloudAdapterServiceConfig
	expectedDockerComposeService["nginx"] = expectedNginxServiceConfig

	expectedDockerCompose["services"] = expectedDockerComposeService

	resultDockerComposeMap, errMap := createMapFrom(resultDockerComposeString)
	if errMap != nil {
		t.Errorf("Failed creating docker compose map %v", err)
	}

	eq := reflect.DeepEqual(resultDockerComposeMap, expectedDockerCompose)
	if !eq {
		t.Errorf("Expected \n%s but got \n%s", expectedDockerCompose, resultDockerComposeMap)
	}
}

func TestCreateDockerComposeFromManifests(t *testing.T) {
	tests := []struct {
		testCase                string
		manifestFilepath        string
		expectedComposeFilepath string
	}{
		{
			"Testcase with service build from manifest",
			"testdata/manifest-with-service-build.json",
			"testdata/manifest-with-service-build-expected-compose.yml",
		},
		{
			"Testcase with volume environments from manifest",
			"testdata/manifest-with-volume-environment.json",
			"testdata/manifest-with-volume-environment-expected-compose.yml",
		},
		{
			"Testcase with volume driver opts from manifest",
			"testdata/manifest-with-volume-driver-opts.json",
			"testdata/manifest-with-volume-driver-opts-expected-compose.yml",
		},
		{
			"Testcase with volume external from manifest",
			"testdata/manifest-with-volume-external.json",
			"testdata/manifest-with-volume-external-expected-compose.yml",
		},
		{
			"Testcase with volume name from manifest",
			"testdata/manifest-with-volume-name.json",
			"testdata/manifest-with-volume-name-expected-compose.yml",
		},
		{
			"Testcase with volume driver from manifest",
			"testdata/manifest-with-volume-driver.json",
			"testdata/manifest-with-volume-driver-expected-compose.yml",
		},
		{
			"Testcase with network external from manifest",
			"testdata/manifest-with-network-external.json",
			"testdata/manifest-with-network-external-expected-compose.yml",
		},
		{
			"Testcase with network driver opts from manifest",
			"testdata/manifest-with-network-driver-opts.json",
			"testdata/manifest-with-network-driver-opts-expected-compose.yml",
		},
		{
			"Testcase service with environment array and with environmentVariables from manifest",
			"testdata/manifest-with-settings-array.json",
			"testdata/manifest-with-settings-expected-compose.yml",
		},
		{
			"Testcase service with environment dictionary and with environmentVariables from manifest",
			"testdata/manifest-with-settings-dictionary.json",
			"testdata/manifest-with-settings-expected-compose.yml",
		},
		{
			"Testcase service with conflicting environment and environmentVariables from manifest",
			"testdata/manifest-with-settings-duplicate-array.json",
			"testdata/manifest-with-settings-expected-compose.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.testCase, func(t *testing.T) {
			// arrange
			manifestFileContent, err := os.ReadFile(tt.manifestFilepath)
			if err != nil {
				t.Errorf("Can not read test file %s", tt.manifestFilepath)
			}
			manifestData, err := manifest.NewFromBytes(manifestFileContent)
			if err != nil {
				t.Fatal(err)
			}

			// act
			resultDockerComposeString, err := yaml.GetDockerComposeFromManifest(manifestData)

			// assert
			if err != nil {
				t.Errorf("Failed creating docker compose from manifest file %v", err)
			}

			expectedDockerComposeFileContent, err := os.ReadFile(tt.expectedComposeFilepath)
			if err != nil {
				t.Errorf("Can not read test file %s", tt.expectedComposeFilepath)
			}

			expectedDockerComposeMap := make(map[interface{}]interface{})

			yaml3.Unmarshal(expectedDockerComposeFileContent, &expectedDockerComposeMap)

			resultDockerComposeMap, errMap := createMapFrom(resultDockerComposeString)
			if errMap != nil {
				t.Errorf("Failed creating docker compose map %v", err)
			}

			eq := reflect.DeepEqual(resultDockerComposeMap, expectedDockerComposeMap)
			if !eq {
				t.Errorf("Generated compose string \n%s", resultDockerComposeString)
				t.Errorf("Expected \n%s but got \n%s", expectedDockerComposeMap, resultDockerComposeMap)
			}
		})
	}
}

func createMapFrom(resultDockerComposeString string) (map[interface{}]interface{}, error) {
	resultDockerComposeMap := make(map[interface{}]interface{})
	err := yaml3.Unmarshal([]byte(resultDockerComposeString), resultDockerComposeMap)
	return resultDockerComposeMap, err
}
