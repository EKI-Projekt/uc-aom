// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"reflect"
	"testing"
	model "u-control/uc-aom/internal/pkg/manifest"
)

func Test_combineUpdateSettingsWithCurrentSettings(t *testing.T) {
	type args struct {
		updateSettings  []*model.Setting
		currentSettings map[string]string
	}
	tests := []struct {
		name string
		args args
		want []*model.Setting
	}{
		{
			name: "Shall create settings without current settings",
			args: args{
				currentSettings: make(map[string]string),
				updateSettings: []*model.Setting{
					model.NewSettings("test", "test", true).WithTextBoxValue("abcd"),
				},
			},
			want: []*model.Setting{
				model.NewSettings("test", "test", true).WithTextBoxValue("abcd"),
			},
		},
		{
			name: "Shall update settings current settings",
			args: args{
				currentSettings: map[string]string{
					"test": "currentABCD",
				},
				updateSettings: []*model.Setting{
					model.NewSettings("test", "test", true).WithTextBoxValue("abcd"),
				},
			},
			want: []*model.Setting{
				model.NewSettings("test", "test", true).WithTextBoxValue("currentABCD"),
			},
		},
		{
			name: "Shall not update settings if setting is not included",
			args: args{
				currentSettings: map[string]string{
					"notIncludedInUpdate": "1234",
				},
				updateSettings: []*model.Setting{
					model.NewSettings("test", "test", true).WithTextBoxValue("abcd"),
				},
			},
			want: []*model.Setting{
				model.NewSettings("test", "test", true).WithTextBoxValue("abcd"),
			},
		},
		{
			name: "Shall update settings current if setting is dropdown list",
			args: args{
				currentSettings: map[string]string{
					"test": "def",
				},
				updateSettings: []*model.Setting{
					model.NewSettings("test", "test", true).WithSelectItems(
						&model.Item{Label: "ABC", Value: "abc", Selected: true},
						&model.Item{Label: "DEF", Value: "def"},
					),
				},
			},
			want: []*model.Setting{
				model.NewSettings("test", "test", true).WithSelectItems(
					&model.Item{Label: "ABC", Value: "abc"},
					&model.Item{Label: "DEF", Value: "def", Selected: true},
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CombineManifestSettingsWithSettingsMap(tt.args.updateSettings, tt.args.currentSettings)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("combineUpdateSettingsWithCurrentSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}
