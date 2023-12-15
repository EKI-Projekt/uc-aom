// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"
	"os/user"
	"reflect"
	"testing"
	"u-control/uc-aom/internal/aom/system"
	"u-control/uc-aom/internal/pkg/manifest"
)

func Test_manifestFeatureToSystemAdapter_adaptFeaturesToSystem(t *testing.T) {
	type fields struct {
		system          *system.MockSystem
		adaptedManifest *manifest.Root
	}
	type args struct {
		sourceManifest *manifest.Root
	}

	mockAdminUser := &user.User{
		Uid: "1000",
		Gid: "1000",
	}

	expectedUsersValue := fmt.Sprintf("%s:%s", mockAdminUser.Uid, mockAdminUser.Gid)

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *manifest.Root
		wantErr bool
	}{
		{
			name: "shall add admin user if services uses a public volume",
			fields: fields{
				system: createMockSystemWith(mockAdminUser),
			},
			args: args{
				sourceManifest: &manifest.Root{
					Services: map[string]*manifest.Service{
						"test": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"volumes": []interface{}{"test-volume:/data"},
							},
						},
					},
					Environments: map[string]*manifest.Environment{
						"test": manifest.NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": manifest.LocalPublicVolumeDriverName}}),
					},
				},
			},
			want: &manifest.Root{
				Services: map[string]*manifest.Service{
					"test": {
						Type: "docker-compose",
						Config: map[string]interface{}{
							"volumes": []interface{}{"test-volume:/data"},
							"user":    expectedUsersValue,
						},
					},
				},
				Environments: map[string]*manifest.Environment{
					"test": manifest.NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": manifest.LocalPublicVolumeDriverName}}),
				},
			},
		},
		{
			name: "shall not add admin user if services doesn't use a public volume",
			fields: fields{
				system: createMockSystemWith(mockAdminUser),
			},
			args: args{
				sourceManifest: &manifest.Root{
					Services: map[string]*manifest.Service{
						"test": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"volumes": []interface{}{"test-none-public:/data"},
							},
						},
					},
					Environments: map[string]*manifest.Environment{
						"test": manifest.NewEnvironment("docker-compose").WithVolumes(map[string]map[string]interface{}{"test-volume": {"driver": manifest.LocalPublicVolumeDriverName}, "test-none-public": {}}),
					},
				},
			},
		},
		{
			name: "shall not add admin user if services doesn't use a volume",
			fields: fields{
				system: createMockSystem(),
			},
			args: args{
				sourceManifest: &manifest.Root{
					Services: map[string]*manifest.Service{
						"test": {
							Type:   "docker-compose",
							Config: map[string]interface{}{},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &manifestFeatureToSystemAdapter{
				system:          tt.fields.system,
				adaptedManifest: tt.fields.adaptedManifest,
			}
			got, err := c.adaptFeaturesToSystem(tt.args.sourceManifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("manifestFeatureToSystemAdapter.adaptFeaturesToSystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// if want is nil, we expect that the result will be the sourceManifest
			if tt.want == nil {
				tt.want = tt.args.sourceManifest
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("manifestFeatureToSystemAdapter.adaptFeaturesToSystem() = %v, want %v", got, tt.want)
			}

			tt.fields.system.AssertExpectations(t)

		})
	}
}

func createMockSystemWith(mockAdminUser *user.User) *system.MockSystem {
	mockSystemWithAdminUser := &system.MockSystem{}
	mockSystemWithAdminUser.On("LookupAdminUser").Return(mockAdminUser, nil)
	return mockSystemWithAdminUser
}
func createMockSystem() *system.MockSystem {
	return &system.MockSystem{}
}
