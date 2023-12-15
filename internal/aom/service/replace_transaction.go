// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/aom/yaml"
	model "u-control/uc-aom/internal/pkg/manifest"
)

func (tx *Tx) updateAction(addOn catalogue.CatalogueAddOn, version string, settings ...*model.Setting) error {
	futureManifest, err := tx.service.localCatalogue.FetchManifest(addOn.Name, version)
	if err != nil {
		return err
	}

	manifestVersionValidator := NewManifestVersionValidator(futureManifest.ManifestVersion)
	if err := manifestVersionValidator.Validate(); err != nil {
		return err
	}

	if err := tx.service.checkCapabilities(futureManifest); err != nil {
		return err
	}

	if len(settings) == 0 && futureManifest.Settings != nil {
		currentSettings, err := tx.service.addOnEnvironmentResolver.GetAddOnEnvironment(addOn.Name)
		if err != nil {
			return err
		}

		settingsOfUpdate := futureManifest.Settings["environmentVariables"]
		settings = manifest.CombineManifestSettingsWithSettingsMap(settingsOfUpdate, currentSettings)
	}

	if err = tx.service.deleteAddOnExceptVolumes(addOn); err != nil {
		return err
	}

	if err = tx.CreateAddOnRoutine(addOn.Name, version, settings...); err != nil {
		return err
	}

	return tx.service.removeUnusedVolumes(addOn)
}

func (tx *Tx) configureAction(addOn catalogue.CatalogueAddOn, settings ...*model.Setting) error {
	if len(settings) != 0 {
		addOn.Manifest.Settings["environmentVariables"] = settings
	}

	manifestAdapter := newManifestFeatureToSystemAdapter(tx.service.system)
	manifestToDeploy, err := manifestAdapter.adaptFeaturesToSystem(&addOn.Manifest)
	if err != nil {
		return err
	}

	dockerCompose, err := yaml.GetDockerComposeFromManifest(manifestToDeploy)
	if err != nil {
		return err
	}

	if err := tx.service.stackService.DeleteAddOnStack(addOn.Name); err != nil {
		return err
	}

	tx.SubscribeRollbackHook(func() {
		tx.service.stackService.DeleteAddOnStack(addOn.Name)
		tx.service.removeUnusedVolumes(addOn)
	})
	return tx.service.stackService.CreateStackWithDockerCompose(addOn.Name, dockerCompose)
}
