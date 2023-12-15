// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package migrate

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/utils"

	model "u-control/uc-aom/internal/pkg/manifest"
	modelV0_1 "u-control/uc-aom/internal/pkg/manifest/v0_1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func newMockLocalfsRegistry() *mockLocalfsRegistry {
	return &mockLocalfsRegistry{}
}

type mockLocalfsRegistry struct {
	mock.Mock
}

func (r *mockLocalfsRegistry) Repositories() ([]string, error) {
	args := r.Called()
	return args.Get(0).([]string), args.Error(1)

}

func (r *mockLocalfsRegistry) Repository(name string) (localFSRepository, error) {
	args := r.Called(name)

	return args.Get(0).(localFSRepository), args.Error(1)
}

func newMockLocalFSRepository() *mockLocalFSRepository {
	return &mockLocalFSRepository{}
}

type mockLocalFSRepository struct {
	mock.Mock
}

func (r *mockLocalFSRepository) Fetch() (io.Reader, error) {
	args := r.Called()
	data := args.Get(0).([]byte)
	reader := bytes.NewReader(data)
	return reader, args.Error(1)
}

func (r *mockLocalFSRepository) Push(content io.Reader) error {
	args := r.Called(content)
	return args.Error(0)
}

func newMockStackMigrator() *mockStackMigrator {
	return &mockStackMigrator{}
}

type mockStackMigrator struct {
	mock.Mock
}

func (r *mockStackMigrator) MigrateStack(name string, version string, manifest *model.Root, settings ...*model.Setting) error {
	args := r.Called(name, version, manifest, settings)
	return args.Error(0)
}

func newMockVersionResolver() *mockVersionResolver {
	return &mockVersionResolver{}
}

type mockVersionResolver struct {
	mock.Mock
}

func (r *mockVersionResolver) getVersion() (string, error) {
	args := r.Called()
	return args.String(0), args.Error(1)
}

func (r *mockVersionResolver) updateVersion(version string) error {
	args := r.Called(version)
	return args.Error(0)
}

type mockAddOnEnvironmentResolver struct {
	mock.Mock
}

func (r *mockAddOnEnvironmentResolver) GetAddOnEnvironment(name string) (map[string]string, error) {
	args := r.Called(name)
	return args.Get(0).(map[string]string), args.Error(1)
}

type mockReverseProxyMigrator struct {
	mock.Mock
}

func (m *mockReverseProxyMigrator) Migrate(name string, versionToMigrate string, title string, permissionId string, proxyRoute map[string]*model.ProxyRoute) error {
	args := m.Called(name, versionToMigrate, title, permissionId, proxyRoute)
	return args.Error(0)
}

func Test_installAddonMigrator_Migrate(t *testing.T) {
	type fields struct {
		localfsRegistry localFSRegistry
		versionResolver versionResolver
		stackMigrator   docker.StackMigrator
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Shall not migrate if version is the current version",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return(currentVersion, nil)
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					return mockStackMigrator
				}(),
			},
			wantErr: false,
		},
		{
			name: "Shall not migrate if no local repositories exists",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					mockRegistry.On("Repositories").Return([]string{}, nil)
					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return("0.3.2", nil)
					mockVersionResolver.On("updateVersion", currentVersion).Return(nil)
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					return mockStackMigrator
				}(),
			},
			wantErr: false,
		},
		{
			name: "Shall not migrate error is returned by 'getVersion'",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return("", errors.New("Version error"))
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					return mockStackMigrator
				}(),
			},
			wantErr: true,
		},
		{
			name: "Shall not migrate if error is returned by 'Repositories'",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					mockRegistry.On("Repositories").Return([]string{"test"}, errors.New("Error Repositories"))
					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return("0.3.2", nil)
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					return mockStackMigrator
				}(),
			},
			wantErr: true,
		},
		{
			name: "Shall not migrate if error is returned by 'Repository' (checkForMigration)",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					mockRegistry.On("Repositories").Return([]string{"test"}, nil)
					mockRepository := newMockLocalFSRepository()
					mockRegistry.On("Repository", "test").Return(mockRepository, errors.New("Error Repository"))
					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return("0.3.2", nil)
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					return mockStackMigrator
				}(),
			},
			wantErr: true,
		},
		{
			name: "Shall not migrate if error is returned by 'Fetch' (checkForMigration)",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					mockRegistry.On("Repositories").Return([]string{"test"}, nil)
					mockRepository := newMockLocalFSRepository()
					mockRegistry.On("Repository", "test").Return(mockRepository, nil)
					mockRepository.On("Fetch").Return([]byte{}, errors.New("fetch error"))
					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return("0.3.2", nil)
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					return mockStackMigrator
				}(),
			},
			wantErr: true,
		},
		{
			name: "Shall not migrate if error is returned by 'MigrateStack'",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					mockRegistry.On("Repositories").Return([]string{"test"}, nil)
					mockRepository := newMockLocalFSRepository()
					mockRegistry.On("Repository", "test").Return(mockRepository, nil)
					mockManifestV0_1 := &modelV0_1.Root{
						Version:         "1.0.0-1",
						ManifestVersion: modelV0_1.ValidManifestVersion,
						Vendor: &modelV0_1.Vendor{
							Name: "Test",
						},
					}
					mockManifestV0_1AsByte, err := json.Marshal(mockManifestV0_1)
					mockRepository.On("Fetch").Return(mockManifestV0_1AsByte, nil)
					assert.NoError(t, err)
					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return("0.3.2", nil)
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					mockStackMigrator.On("MigrateStack", "test", "0.1", mock.Anything, mock.Anything).Return(errors.New("MigrateStack error"))
					return mockStackMigrator
				}(),
			},
			wantErr: true,
		},
		{
			name: "Shall not migrate if error is returned by 'Push' (migrateAndReplace)",
			fields: fields{
				localfsRegistry: func() localFSRegistry {
					mockRegistry := newMockLocalfsRegistry()
					mockRegistry.On("Repositories").Return([]string{"test"}, nil)
					mockRepository := newMockLocalFSRepository()
					mockRegistry.On("Repository", "test").Return(mockRepository, nil)
					mockManifestV0_1 := &modelV0_1.Root{
						Version:         "1.0.0-1",
						ManifestVersion: modelV0_1.ValidManifestVersion,
						Vendor: &modelV0_1.Vendor{
							Name: "Test",
						},
					}
					mockManifestV0_1AsByte, err := json.Marshal(mockManifestV0_1)
					assert.NoError(t, err)

					mockRepository.On("Fetch").Return(mockManifestV0_1AsByte, nil)
					mockRepository.On("Push", mock.Anything).Return(errors.New("Push error"))

					return mockRegistry
				}(),
				versionResolver: func() versionResolver {
					mockVersionResolver := newMockVersionResolver()
					mockVersionResolver.On("getVersion").Return("0.3.2", nil)
					return mockVersionResolver
				}(),
				stackMigrator: func() docker.StackMigrator {
					mockStackMigrator := newMockStackMigrator()
					mockStackMigrator.On("MigrateStack", "test", "0.1", mock.Anything, mock.Anything).Return(nil)
					return mockStackMigrator
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &installAddonMigrator{
				localfsRegistry: tt.fields.localfsRegistry,
				versionResolver: tt.fields.versionResolver,
				stackMigrator:   tt.fields.stackMigrator,
			}
			err := m.Migrate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Failed to Migrate error!  %v", err)
			}

			mock.AssertExpectationsForObjects(t, tt.fields.localfsRegistry)
			mock.AssertExpectationsForObjects(t, tt.fields.versionResolver)
			mock.AssertExpectationsForObjects(t, tt.fields.stackMigrator)
		})
	}
}

func Test_installAddonMigrator_Migrate_0_3_2(t *testing.T) {
	// Arrange
	mockRegistry := newMockLocalfsRegistry()
	repositoryName := "test"
	mockRegistry.On("Repositories").Return([]string{repositoryName}, nil)

	mockRepository := newMockLocalFSRepository()

	mockManifestV0_1 := &modelV0_1.Root{
		Version:         "1.0.0-1",
		ManifestVersion: modelV0_1.ValidManifestVersion,
		Vendor: &modelV0_1.Vendor{
			Name: "Test",
		},
	}

	mockManifestV0_1AsByte, err := json.Marshal(mockManifestV0_1)
	assert.NoError(t, err)

	mockRepository.On("Fetch").Return(mockManifestV0_1AsByte, nil)
	var gotFromPushCall io.Reader
	mockRepository.On("Push", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		gotFromPushCall = args.Get(0).(io.Reader)
	})

	mockRegistry.On("Repository", repositoryName).Return(mockRepository, nil)

	mockAddOnEnvironmentResolver := &mockAddOnEnvironmentResolver{}

	mockVersionResolver := newMockVersionResolver()
	mockVersionResolver.On("getVersion").Return("0.3.2", nil)
	mockVersionResolver.On("updateVersion", currentVersion).Return(nil)

	mockStackMigrator := newMockStackMigrator()
	mockStackMigrator.On("MigrateStack", "test", "0.1", mock.AnythingOfType("*manifest.Root"), mock.AnythingOfType("[]*manifest.Setting")).Return(nil)

	routesMigrator := &mockReverseProxyMigrator{}
	routesMigrator.On("Migrate", repositoryName, routes.TemplateVersionV0_1_0, mockManifestV0_1.Title, utils.ReplaceSlashesWithDashes(repositoryName), mock.AnythingOfType("map[string]*manifest.ProxyRoute")).Return(nil)

	// Act
	m := &installAddonMigrator{
		localfsRegistry: mockRegistry,
		envResolver:     mockAddOnEnvironmentResolver,
		versionResolver: mockVersionResolver,
		stackMigrator:   mockStackMigrator,
		routesMigrator:  routesMigrator,
	}
	result := m.Migrate()
	assert.NoError(t, result)

	// Assert
	assert.NotNil(t, gotFromPushCall)

	gotBuffer := &bytes.Buffer{}
	_, err = io.Copy(gotBuffer, gotFromPushCall)
	assert.NoError(t, err)

	gotManifest, err := model.NewFromBytes(gotBuffer.Bytes())
	assert.NoError(t, err)

	wantManifest := createWantManifestFromBytes(t, mockManifestV0_1.ManifestVersion, mockManifestV0_1AsByte)

	if !reflect.DeepEqual(gotManifest, wantManifest) {
		t.Errorf("Expected: %#v \t Actual: %#v", wantManifest, gotManifest)
	}

	mockRegistry.AssertExpectations(t)
	mockRepository.AssertExpectations(t)
	mockAddOnEnvironmentResolver.AssertExpectations(t)
	mockVersionResolver.AssertExpectations(t)
	mockStackMigrator.AssertExpectations(t)
}

func Test_installAddonMigrator_Migrate_0_3_2_and_settings(t *testing.T) {
	// Arrange
	mockRegistry := newMockLocalfsRegistry()
	repositoryName := "test"
	mockRegistry.On("Repositories").Return([]string{repositoryName}, nil)

	mockRepository := newMockLocalFSRepository()

	mockManifestV0_1 := &modelV0_1.Root{
		Version:         "1.0.0-1",
		ManifestVersion: modelV0_1.ValidManifestVersion,
		Vendor: &modelV0_1.Vendor{
			Name: "Test",
		},
		Settings: map[string][]*modelV0_1.Setting{
			"environmentVariables": {
				{
					Name:     "Test",
					Value:    "1",
					Label:    "Test",
					Select:   []*modelV0_1.Item{},
					Required: true,
				},
			},
		},
	}

	mockManifestV0_1AsByte, err := json.Marshal(mockManifestV0_1)
	assert.NoError(t, err)

	mockRepository.On("Fetch").Return(mockManifestV0_1AsByte, nil)
	var gotFromPushCall io.Reader
	mockRepository.On("Push", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		gotFromPushCall = args.Get(0).(io.Reader)
	})

	mockRegistry.On("Repository", repositoryName).Return(mockRepository, nil)

	mockAddOnEnvironmentResolver := &mockAddOnEnvironmentResolver{}
	mockEnvironments := map[string]string{"Test": "2"}
	mockAddOnEnvironmentResolver.On("GetAddOnEnvironment", repositoryName).Return(mockEnvironments, nil)

	mockVersionResolver := newMockVersionResolver()
	mockVersionResolver.On("getVersion").Return("0.3.2", nil)
	mockVersionResolver.On("updateVersion", currentVersion).Return(nil)

	mockStackMigrator := newMockStackMigrator()
	mockStackMigrator.On("MigrateStack", "test", "0.1", mock.AnythingOfType("*manifest.Root"), mock.AnythingOfType("[]*manifest.Setting")).Return(nil)

	routesMigrator := &mockReverseProxyMigrator{}
	routesMigrator.On("Migrate", repositoryName, routes.TemplateVersionV0_1_0, mockManifestV0_1.Title, utils.ReplaceSlashesWithDashes(repositoryName), mock.AnythingOfType("map[string]*manifest.ProxyRoute")).Return(nil)

	// Act
	m := &installAddonMigrator{
		localfsRegistry: mockRegistry,
		envResolver:     mockAddOnEnvironmentResolver,
		stackMigrator:   mockStackMigrator,
		versionResolver: mockVersionResolver,
		routesMigrator:  routesMigrator,
	}
	result := m.Migrate()
	assert.NoError(t, result)

	// Assert
	assert.NotNil(t, gotFromPushCall)

	gotBuffer := &bytes.Buffer{}
	_, err = io.Copy(gotBuffer, gotFromPushCall)
	assert.NoError(t, err)

	gotManifest, err := model.NewFromBytes(gotBuffer.Bytes())
	assert.NoError(t, err)

	wantManifest := createWantManifestFromBytes(t, mockManifestV0_1.ManifestVersion, mockManifestV0_1AsByte)

	if !reflect.DeepEqual(gotManifest, wantManifest) {
		t.Errorf("Expected: %#v \t Actual: %#v", wantManifest, gotManifest)
	}

	mockRegistry.AssertExpectations(t)
	mockRepository.AssertExpectations(t)
	mockAddOnEnvironmentResolver.AssertExpectations(t)
	mockVersionResolver.AssertExpectations(t)
	mockStackMigrator.AssertExpectations(t)
}

func Test_installAddonMigrator_Migrate_0_4_0(t *testing.T) {
	// Arrange
	mockRegistry := newMockLocalfsRegistry()
	repositoryName := "test"
	mockRegistry.On("Repositories").Return([]string{repositoryName}, nil)

	mockRepository := newMockLocalFSRepository()

	mockManifest := &model.Root{
		Version:         "1.0.0-1",
		Title:           "app-title",
		ManifestVersion: model.ValidManifestVersion,
		Vendor: &model.Vendor{
			Name: "Test",
		},
		Publish: map[string]*model.ProxyRoute{
			"ui": {
				From: "/web",
				To:   "localhost:5000",
			},
		},
	}

	mockManifestAsByte, err := json.Marshal(mockManifest)
	assert.NoError(t, err)

	mockRepository.On("Fetch").Return(mockManifestAsByte, nil)
	mockRegistry.On("Repository", repositoryName).Return(mockRepository, nil)

	mockAddOnEnvironmentResolver := &mockAddOnEnvironmentResolver{}

	mockVersionResolver := newMockVersionResolver()
	mockVersionResolver.On("getVersion").Return("0.4.0", nil)
	mockVersionResolver.On("updateVersion", currentVersion).Return(nil)

	routesMigrator := &mockReverseProxyMigrator{}
	routesMigrator.On("Migrate", repositoryName, routes.TemplateVersionV0_1_0, mockManifest.Title, utils.ReplaceSlashesWithDashes(repositoryName), mockManifest.Publish).Return(nil)

	// Act
	m := &installAddonMigrator{
		localfsRegistry: mockRegistry,
		envResolver:     mockAddOnEnvironmentResolver,
		versionResolver: mockVersionResolver,
		routesMigrator:  routesMigrator,
	}
	result := m.Migrate()
	assert.NoError(t, result)

	// Assert
	mockRegistry.AssertExpectations(t)
	mockRepository.AssertExpectations(t)
	mockAddOnEnvironmentResolver.AssertExpectations(t)
	mockVersionResolver.AssertExpectations(t)
	routesMigrator.AssertExpectations(t)
}

func Test_installAddonMigrator_Migrate_0_5_2(t *testing.T) {
	// Arrange
	mockRegistry := newMockLocalfsRegistry()
	repositoryName := "test"
	mockRegistry.On("Repositories").Return([]string{repositoryName}, nil)

	mockRepository := newMockLocalFSRepository()

	mockManifest := &model.Root{
		Version:         "1.0.0-1",
		Title:           "app-title",
		ManifestVersion: model.ValidManifestVersion,
		Vendor: &model.Vendor{
			Name: "Test",
		},
		Publish: map[string]*model.ProxyRoute{
			"ui": {
				From: "/web",
				To:   "localhost:5000",
			},
		},
	}

	mockManifestAsByte, err := json.Marshal(mockManifest)
	assert.NoError(t, err)

	mockRepository.On("Fetch").Return(mockManifestAsByte, nil)
	mockRegistry.On("Repository", repositoryName).Return(mockRepository, nil)

	mockAddOnEnvironmentResolver := &mockAddOnEnvironmentResolver{}

	mockVersionResolver := newMockVersionResolver()
	mockVersionResolver.On("getVersion").Return("0.5.2", nil)
	mockVersionResolver.On("updateVersion", currentVersion).Return(nil)

	routesMigrator := &mockReverseProxyMigrator{}
	routesMigrator.On("Migrate", repositoryName, routes.TemplateVersionV0_1_0, mockManifest.Title, utils.ReplaceSlashesWithDashes(repositoryName), mockManifest.Publish).Return(nil)

	// Act
	m := &installAddonMigrator{
		localfsRegistry: mockRegistry,
		envResolver:     mockAddOnEnvironmentResolver,
		versionResolver: mockVersionResolver,
		routesMigrator:  routesMigrator,
	}
	result := m.Migrate()
	assert.NoError(t, result)

	// Assert
	mockRegistry.AssertExpectations(t)
	mockRepository.AssertExpectations(t)
	mockAddOnEnvironmentResolver.AssertExpectations(t)
	mockVersionResolver.AssertExpectations(t)
	routesMigrator.AssertExpectations(t)
}

func createWantManifestFromBytes(t *testing.T, manifestVersion string, data []byte) *model.Root {
	wantManifestAsBytes, err := model.MigrateUcManifest(manifestVersion, data)
	assert.NoError(t, err)
	wantManifest, err := model.NewFromBytes(wantManifestAsBytes)
	assert.NoError(t, err)
	return wantManifest
}
