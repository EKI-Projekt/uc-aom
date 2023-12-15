// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"reflect"
	"testing"
)

func TestHasLocalPublicVolumes(t *testing.T) {
	type args struct {
		environments map[string]*Environment
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "shall return true if driver is local-public",
			args: args{
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": LocalPublicVolumeDriverName}}),
				},
			},
			want: true,
		},
		{
			name: "shall return true if driver is local-public-access",
			args: args{
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": LocalPublicVolumeAccessDriverName}}),
				},
			},
			want: true,
		},
		{
			name: "shall return false if driver is local",
			args: args{
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": "local"}}),
				},
			},
			want: false,
		},
		{
			name: "shall return false if driver is not set (default local)",
			args: args{
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {}}),
				},
			},
			want: false,
		},
		{
			name: "shall return false if volumes not set",
			args: args{
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose"),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasLocalPublicVolumes(tt.args.environments); got != tt.want {
				t.Errorf("HasLocalPublicVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUsesLocalPublicVolume(t *testing.T) {
	type args struct {
		service      *Service
		environments map[string]*Environment
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "shall return true if local-public volume is used",
			args: args{
				service: &Service{
					Config: map[string]interface{}{
						"volumes": []interface{}{"test-volume:/data"},
					},
				},
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": LocalPublicVolumeDriverName}}),
				},
			},
			want: true,
		},
		{
			name: "shall return false if is defined local-public volume but not used",
			args: args{
				service: &Service{
					Config: map[string]interface{}{},
				},
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": LocalPublicVolumeDriverName}}),
				},
			},
			want: false,
		},
		{
			name: "shall return false if local volume is used",
			args: args{
				service: &Service{
					Config: map[string]interface{}{},
				},
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {}}),
				},
			},
			want: false,
		},
		{
			name: "shall return false if volumes isn't a string type",
			args: args{
				service: &Service{
					Config: map[string]interface{}{
						"volumes": []interface{}{1},
					},
				},
				environments: map[string]*Environment{
					"test": NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": LocalPublicVolumeDriverName}}),
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := UsesLocalPublicVolume(tt.args.service, tt.args.environments); got != tt.want {
				t.Errorf("UsesLocalPublicVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDockerImageReferences(t *testing.T) {
	type args struct {
		services map[string]*Service
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDockerImageReferences(tt.args.services); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDockerImageReferences() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetVolumeNames(t *testing.T) {
	type args struct {
		environments map[string]*Environment
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetVolumeNames(tt.args.environments); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetVolumeNames() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_filterVolumes(t *testing.T) {
	type args struct {
		environments map[string]*Environment
		filter       func(volumesSetting map[string]interface{}) bool
	}
	tests := []struct {
		name string
		args args
		want map[string]map[string]interface{}
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterVolumes(tt.args.environments, tt.args.filter); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isLocalPublicVolume(t *testing.T) {
	type args struct {
		volumeSettings map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLocalPublicVolume(tt.args.volumeSettings); got != tt.want {
				t.Errorf("isLocalPublicVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_returnTopLevelNameOrFromSettings(t *testing.T) {
	type args struct {
		topLevelName string
		settings     map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := returnTopLevelNameOrFromSettings(tt.args.topLevelName, tt.args.settings); got != tt.want {
				t.Errorf("returnTopLevelNameOrFromSettings() = %v, want %v", got, tt.want)
			}
		})
	}
}
