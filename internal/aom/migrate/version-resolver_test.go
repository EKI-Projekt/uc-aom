package migrate

import (
	"encoding/json"
	"os"
	"path"
	"testing"
	model "u-control/uc-aom/internal/pkg/manifest"
	modelV0_1 "u-control/uc-aom/internal/pkg/manifest/v0_1"

	"github.com/stretchr/testify/assert"
)

func Test_aomVersionResolver_updateVersion(t *testing.T) {
	type fields struct {
		localStateDir string
	}
	type args struct {
		version string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Shall update the release file",
			fields: fields{
				localStateDir: t.TempDir(),
			},
			args: args{
				version: "1.0.0",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &aomVersionResolver{
				localStateDir: tt.fields.localStateDir,
			}
			if err := r.updateVersion(tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("aomVersionResolver.updateVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
			equalUpdateVersionWithReleasefile(t, tt.fields.localStateDir, tt.args.version)
		})
	}
}

func equalUpdateVersionWithReleasefile(t *testing.T, localStateDir string, wantVersion string) {
	tempFilePath := path.Join(localStateDir, releaseFilename)
	content, err := os.ReadFile(tempFilePath)
	assert.NoError(t, err)
	releaseFile := &releaseFile{}
	err = json.Unmarshal(content, releaseFile)
	assert.NoError(t, err)
	assert.Equal(t, releaseFile.Version, wantVersion)
}

func Test_aomVersionResolver_getVersionAfterUpdate(t *testing.T) {
	// arrange
	want := "1.0.0"
	wantErr := false
	r := &aomVersionResolver{
		localStateDir: t.TempDir(),
	}
	err := r.updateVersion(want)
	assert.NoError(t, err)

	// act & assert
	got, err := r.getVersion()
	if (err != nil) != wantErr {
		t.Errorf("Failed to getVersion error!  %v", err)
	}

	assert.Equal(t, want, got)
}

func Test_aomVersionResolver_getVersionWithCurrentVersion(t *testing.T) {
	// arrange
	want := currentVersion
	wantErr := false
	newMockLocalfsRegistry := newMockLocalfsRegistry()
	newMockLocalfsRegistry.On("Repositories").Return([]string{}, nil)

	r := &aomVersionResolver{
		localStateDir:   t.TempDir(),
		localFSRegistry: newMockLocalfsRegistry,
	}

	// act & assert
	got, err := r.getVersion()
	if (err != nil) != wantErr {
		t.Errorf("Failed to getVersion error!  %v", err)
	}

	assert.Equal(t, want, got)
	newMockLocalfsRegistry.AssertExpectations(t)
}

func Test_aomVersionResolver_getVersionWithV0_3_2(t *testing.T) {
	// arrange
	want := "0.3.2"
	wantErr := false
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

	r := &aomVersionResolver{
		localStateDir:   t.TempDir(),
		localFSRegistry: mockRegistry,
	}

	// act & assert
	got, err := r.getVersion()
	if (err != nil) != wantErr {
		t.Errorf("Failed to getVersion error!  %v", err)
	}

	assert.Equal(t, want, got)
	mockRegistry.AssertExpectations(t)
}

func Test_aomVersionResolver_getVersionWithV0_4_0(t *testing.T) {
	// arrange
	want := "0.4.0"
	wantErr := false
	mockRegistry := newMockLocalfsRegistry()
	mockRegistry.On("Repositories").Return([]string{"test"}, nil)
	mockRepository := newMockLocalFSRepository()
	mockRegistry.On("Repository", "test").Return(mockRepository, nil)
	mockManifestV0_1 := &model.Root{
		Version:         "1.0.0-1",
		ManifestVersion: model.ValidManifestVersion,
		Vendor: &model.Vendor{
			Name: "Test",
		},
	}
	mockManifestV0_1AsByte, err := json.Marshal(mockManifestV0_1)
	mockRepository.On("Fetch").Return(mockManifestV0_1AsByte, nil)
	assert.NoError(t, err)

	r := &aomVersionResolver{
		localStateDir:   t.TempDir(),
		localFSRegistry: mockRegistry,
	}

	// act & assert
	got, err := r.getVersion()
	if (err != nil) != wantErr {
		t.Errorf("Failed to getVersion error!  %v", err)
	}

	assert.Equal(t, want, got)
	mockRegistry.AssertExpectations(t)
}
func Test_aomVersionResolver_getVersionWithUnknownVersion(t *testing.T) {
	// arrange
	want := ""
	wantErr := true
	mockRegistry := newMockLocalfsRegistry()
	mockRegistry.On("Repositories").Return([]string{"test"}, nil)
	mockRepository := newMockLocalFSRepository()
	mockRegistry.On("Repository", "test").Return(mockRepository, nil)
	unknownManifestVersion := "0.0"
	mockManifestV0_1 := &model.Root{
		Version:         "1.0.0-1",
		ManifestVersion: unknownManifestVersion,
		Vendor: &model.Vendor{
			Name: "Test",
		},
	}
	mockManifestV0_1AsByte, err := json.Marshal(mockManifestV0_1)
	mockRepository.On("Fetch").Return(mockManifestV0_1AsByte, nil)
	assert.NoError(t, err)

	r := &aomVersionResolver{
		localStateDir:   t.TempDir(),
		localFSRegistry: mockRegistry,
	}

	// act & assert
	got, err := r.getVersion()
	if (err != nil) != wantErr {
		t.Errorf("Failed to getVersion error!  %v", err)
	}

	assert.Equal(t, want, got)
	mockRegistry.AssertExpectations(t)
}
