// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"encoding/json"
	"errors"
	"fmt"
	"u-control/uc-aom/internal/pkg/manifest/v0_1"
)

var (
	TypeConversionError = errors.New("Can not convert to manifest struct type")
)

const (
	externalAddOnNetworkConfigName = "uc-aom-network"
)

// Migrate the manifest from the provided version to the latest version
func MigrateUcManifest(manifestVersion string, manifestToMigrate []byte) ([]byte, error) {
	var currentManifest *Root
	var err error = nil

	switch manifestVersion {
	case v0_1.ValidManifestVersion:

		manifestV0_1 := &v0_1.Root{}
		err = json.Unmarshal(manifestToMigrate, manifestV0_1)
		if err != nil {
			return nil, err
		}
		currentManifest, err = migrateFromV0_1ToV0_2(manifestV0_1)
		if err != nil {
			return nil, err
		}

		// fallthrough for further migration
		fallthrough
	case ValidManifestVersion:

		// has been migrated in the previous case
		if currentManifest != nil {
			return json.Marshal(currentManifest)
		}

		// manifest has the lastet version
		return manifestToMigrate, nil

	default:
		{
			return nil, fmt.Errorf("Manifest version %s is unknown", manifestVersion)
		}
	}
}

func migrateFromV0_1ToV0_2(manifestToMigrate *v0_1.Root) (*Root, error) {
	contentv0_1, err := json.Marshal(manifestToMigrate)
	if err != nil {
		return nil, err
	}

	migratedManifest, err := NewFromBytes(contentv0_1)
	if err != nil {
		return nil, err
	}

	// In v0.1 the vendor property was optional, but it is required now.
	if migratedManifest.Vendor == nil || *migratedManifest.Vendor == (Vendor{}) {
		return nil, errors.New("Vendor is required")
	}

	migratedManifest.Features = make([]Feature, 0)

	// In v0.1 the "restart" property allowed a value of "always", but this value if prohibit now and will be set to the default value "no".
	// This restriction prevents a endless restart loop.
	for _, service := range migratedManifest.Services {
		if service.Config["restart"] == "always" {
			service.Config["restart"] = "no"
		}

	}

	// In v0.1 we added add-on services to the internal-bridge network by using the networkMode property
	//
	// service: {
	// 	...
	// 	"networkMode": "internal-bridge"
	// 	...
	// }
	//
	// However, this is a limitation because the service is only connected to the internal-bridge and not to the stack network.
	// This workaround was needed because of an issue in portainer 2.0.0 (see for details https://github.com/portainer/portainer/issues/2041).
	//
	// Now, due to the replacement of portainer by the native Docker GO library, we can remove this workaround and switch to the supported configuration.
	// The supported docker-compose configuration should have an external network entry which the services can use.
	//
	//  services:
	//   networks:
	// 		- network1
	//    ...
	//  networks:
	//   network1:
	//     name: my-pre-existing-network
	//     external: true
	//
	// See for more details https://docs.docker.com/compose/networking/#use-a-pre-existing-network
	if useInternalBridge(migratedManifest.Services) {
		addEnvironmentforExternalAddOnNetwork(migratedManifest)
		attachServicesToExternalAddOnNetwork(migratedManifest)
	}

	migratedManifest.ManifestVersion = ValidManifestVersion
	return migratedManifest, nil
}

func useInternalBridge(services map[string]*Service) bool {
	for _, service := range services {
		if service.Config["networkMode"] == v0_1.InternalAddOnNetworkName {
			return true
		}
	}
	return false
}

func addEnvironmentforExternalAddOnNetwork(migratedManifest *Root) {
	if len(migratedManifest.Environments) == 0 {
		migratedManifest.Environments = make(map[string]*Environment)
	}
	ucAomNetworkConfig := map[string]map[string]interface{}{externalAddOnNetworkConfigName: {"external": true, "name": InternalAddOnNetworkName}}

	// The name of the environment isn't use to generate a compose file.
	// Therefore, we add a new configuration instead of changing an existing one.
	migratedManifest.Environments["migration-environment-v0-1-to-v0-2"] = NewEnvironment("docker-compose").WithNetworks(ucAomNetworkConfig)
}

func attachServicesToExternalAddOnNetwork(migratedManifest *Root) {
	for _, service := range migratedManifest.Services {
		if service.Config["networkMode"] == v0_1.InternalAddOnNetworkName {
			delete(service.Config, "networkMode")
			service.Config["networks"] = []string{externalAddOnNetworkConfigName}
		}
	}
}
