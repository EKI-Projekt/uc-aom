// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	model "u-control/uc-aom/internal/pkg/manifest"
)

// CombineManifestSettingsWithSettingsMap combine the manifest settings with values from the map
func CombineManifestSettingsWithSettingsMap(manifestSettings []*model.Setting, SettingsMap map[string]string) []*model.Setting {

	combinedSettings := cloneSettings(manifestSettings)

	for _, s := range combinedSettings {
		if currentValue, hasValueForSetting := SettingsMap[s.Name]; hasValueForSetting {
			if s.IsTextBox() {
				s.Value = currentValue
			} else {
				s.SelectValue(currentValue)
			}
		}
	}

	return combinedSettings
}

func cloneSettings(settings []*model.Setting) []*model.Setting {
	clonedSettings := make([]*model.Setting, len(settings))
	copy(clonedSettings, settings)
	return clonedSettings
}
