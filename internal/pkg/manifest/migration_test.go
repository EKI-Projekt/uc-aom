// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"encoding/json"
	"reflect"
	"testing"
	"u-control/uc-aom/internal/pkg/manifest/v0_1"

	"github.com/stretchr/testify/assert"
)

func TestMigrateUcManifest(t *testing.T) {
	type args struct {
		manifestVersion   string
		manifestToMigrate interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *Root
		wantErr bool
	}{
		{
			name: "migrate from v0_1 to v0_2",
			args: args{
				manifestVersion: v0_1.ValidManifestVersion,
				manifestToMigrate: &v0_1.Root{
					ManifestVersion: v0_1.ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Vendor: &v0_1.Vendor{
						Name: "name",
						Url:  "abcd@vendor.com",
					},
					Services: map[string]*v0_1.Service{
						"ucaomtest-running": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"image":         "test/uc-aom-running:0.1",
								"stdinOpen":     true,
								"tty":           true,
								"containerName": "uc-addon-status-running",
								"command":       []string{"/bin/ash"},
								"restart":       "always",
							},
						},
					},
				},
			},
			want: &Root{
				ManifestVersion: ValidManifestVersion,
				Version:         "1.0.0-1",
				Title:           "title",
				Description:     "Description",
				Logo:            "logo.png",
				Platform:        []string{"ucg"},
				Vendor: &Vendor{
					Name: "name",
					Url:  "abcd@vendor.com",
				},
				Services: map[string]*Service{
					"ucaomtest-running": {
						Type: "docker-compose",
						Config: map[string]interface{}{
							"image":         "test/uc-aom-running:0.1",
							"stdinOpen":     true,
							"tty":           true,
							"containerName": "uc-addon-status-running",
							"command":       []string{"/bin/ash"},
							"restart":       "no",
						},
					},
				},
				Features: []Feature{},
			},
			wantErr: false,
		},
		{
			name: "migrate from v0_1 to v0_2 shall return error if vendor was not set",
			args: args{
				manifestVersion: v0_1.ValidManifestVersion,
				manifestToMigrate: &v0_1.Root{
					ManifestVersion: v0_1.ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Services: map[string]*v0_1.Service{
						"ucaomtest-running": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"image":         "test/uc-aom-running:0.1",
								"stdinOpen":     true,
								"tty":           true,
								"containerName": "uc-addon-status-running",
								"command":       []string{"/bin/ash"},
							},
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "migrate from v0_1 to v0_2 shall return error if vendor was set but empty",
			args: args{
				manifestVersion: v0_1.ValidManifestVersion,
				manifestToMigrate: &v0_1.Root{
					ManifestVersion: v0_1.ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Services: map[string]*v0_1.Service{
						"ucaomtest-running": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"image":         "test/uc-aom-running:0.1",
								"stdinOpen":     true,
								"tty":           true,
								"containerName": "uc-addon-status-running",
								"command":       []string{"/bin/ash"},
							},
						},
					},
					Vendor: &v0_1.Vendor{},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "migrate not migrate, call with latest manifest",
			args: args{
				manifestVersion: ValidManifestVersion,
				manifestToMigrate: &Root{
					ManifestVersion: ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Vendor: &Vendor{
						Name: "name",
						Url:  "abcd@vendor.com",
					},
					Services: map[string]*Service{
						"ucaomtest-running": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"image":         "test/uc-aom-running:0.1",
								"stdinOpen":     true,
								"tty":           true,
								"containerName": "uc-addon-status-running",
								"command":       []string{"/bin/ash"},
							},
						},
					},
					Features: []Feature{},
				},
			},
			want: &Root{
				ManifestVersion: ValidManifestVersion,
				Version:         "1.0.0-1",
				Title:           "title",
				Description:     "Description",
				Logo:            "logo.png",
				Platform:        []string{"ucg"},
				Vendor: &Vendor{
					Name: "name",
					Url:  "abcd@vendor.com",
				},
				Services: map[string]*Service{
					"ucaomtest-running": {
						Type: "docker-compose",
						Config: map[string]interface{}{
							"image":         "test/uc-aom-running:0.1",
							"stdinOpen":     true,
							"tty":           true,
							"containerName": "uc-addon-status-running",
							"command":       []string{"/bin/ash"},
						},
					},
				},
				Features: []Feature{},
			},
			wantErr: false,
		},
		{
			name: "shall return error if type can not be converted",
			args: args{
				manifestVersion: v0_1.ValidManifestVersion,
				manifestToMigrate: struct {
					Name string
				}{
					Name: "test",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "shall return error if version is unknown",
			args: args{
				manifestVersion: "abcd",
				manifestToMigrate: struct {
					Name string
				}{
					Name: "test",
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "migrate from v0_1 to v0_2 shall convert internal-bridge networkMode to an external network",
			args: args{
				manifestVersion: v0_1.ValidManifestVersion,
				manifestToMigrate: &v0_1.Root{
					ManifestVersion: v0_1.ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Vendor: &v0_1.Vendor{
						Name: "name",
						Url:  "abcd@vendor.com",
					},
					Services: map[string]*v0_1.Service{
						"ucaomtest-running": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"image":         "test/uc-aom-running:0.1",
								"stdinOpen":     true,
								"tty":           true,
								"containerName": "uc-addon-status-running",
								"command":       []string{"/bin/ash"},
								"restart":       "always",
								"networkMode":   v0_1.InternalAddOnNetworkName,
							},
						},
					},
				},
			},
			want: &Root{
				ManifestVersion: ValidManifestVersion,
				Version:         "1.0.0-1",
				Title:           "title",
				Description:     "Description",
				Logo:            "logo.png",
				Platform:        []string{"ucg"},
				Vendor: &Vendor{
					Name: "name",
					Url:  "abcd@vendor.com",
				},
				Services: map[string]*Service{
					"ucaomtest-running": {
						Type: "docker-compose",
						Config: map[string]interface{}{
							"image":         "test/uc-aom-running:0.1",
							"stdinOpen":     true,
							"tty":           true,
							"containerName": "uc-addon-status-running",
							"command":       []string{"/bin/ash"},
							"restart":       "no",
							"networks":      []string{externalAddOnNetworkConfigName},
						},
					},
				},
				Features: []Feature{},
				Environments: map[string]*Environment{
					"migration-environment-v0-1-to-v0-2": NewEnvironment("docker-compose").WithNetworks(
						map[string]map[string]interface{}{externalAddOnNetworkConfigName: {"external": true, "name": InternalAddOnNetworkName}},
					),
				},
			},
			wantErr: false,
		},
		{
			name: "migrate from v0_1 to v0_2 shall convert internal-bridge networkMode to an external network and not touch existing enviroment settings",
			args: args{
				manifestVersion: v0_1.ValidManifestVersion,
				manifestToMigrate: &v0_1.Root{
					ManifestVersion: v0_1.ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Vendor: &v0_1.Vendor{
						Name: "name",
						Url:  "abcd@vendor.com",
					},
					Services: map[string]*v0_1.Service{
						"ucaomtest-running": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"image":         "test/uc-aom-running:0.1",
								"stdinOpen":     true,
								"tty":           true,
								"containerName": "uc-addon-status-running",
								"command":       []string{"/bin/ash"},
								"restart":       "always",
								"networkMode":   v0_1.InternalAddOnNetworkName,
								"volumes":       []string{"data:/data"},
							},
						},
					},
					Environments: map[string]*v0_1.Environment{
						"ucaomtest-running": {
							Type: "docker-compose",
							Config: v0_1.EnvironmentConfig{
								Volumes: map[string]map[string]interface{}{
									"data": {},
								},
							},
						},
					},
				},
			},
			want: &Root{
				ManifestVersion: ValidManifestVersion,
				Version:         "1.0.0-1",
				Title:           "title",
				Description:     "Description",
				Logo:            "logo.png",
				Platform:        []string{"ucg"},
				Vendor: &Vendor{
					Name: "name",
					Url:  "abcd@vendor.com",
				},
				Services: map[string]*Service{
					"ucaomtest-running": {
						Type: "docker-compose",
						Config: map[string]interface{}{
							"image":         "test/uc-aom-running:0.1",
							"stdinOpen":     true,
							"tty":           true,
							"containerName": "uc-addon-status-running",
							"command":       []string{"/bin/ash"},
							"restart":       "no",
							"networks":      []string{externalAddOnNetworkConfigName},
							"volumes":       []string{"data:/data"},
						},
					},
				},
				Features: []Feature{},
				Environments: map[string]*Environment{
					"migration-environment-v0-1-to-v0-2": NewEnvironment("docker-compose").WithNetworks(
						map[string]map[string]interface{}{externalAddOnNetworkConfigName: {"external": true, "name": InternalAddOnNetworkName}},
					),
					"ucaomtest-running": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{
						"data": {},
					}),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifestAsBytes, err := json.Marshal(tt.args.manifestToMigrate)
			assert.Nil(t, err)
			got, err := MigrateUcManifest(tt.args.manifestVersion, manifestAsBytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("MigrateUcManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			var wantContent []byte
			if tt.want != nil {
				wantContent, err = json.Marshal(tt.want)
				assert.Nil(t, err)
			}

			if !reflect.DeepEqual(got, wantContent) {
				t.Errorf("MigrateUcManifest() = %v, want %v", string(got), string(wantContent))
			}
		})
	}
}
