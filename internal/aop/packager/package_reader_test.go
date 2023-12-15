// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"testing"
	"u-control/uc-aom/internal/aop/manifest"
	"u-control/uc-aom/internal/aop/packager"
	"u-control/uc-aom/internal/aop/registry"
	model "u-control/uc-aom/internal/pkg/manifest"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/mock"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"

	orasRegistry "oras.land/oras-go/v2/registry"
)

type UnGzipTarballMock struct {
	mock.Mock
}

type nopCloser struct {
	io.Reader
}

func (nopCloser) Close() error {
	return nil
}

type mockOrasRegistry struct {
	mock.Mock
}

func (r *mockOrasRegistry) Repositories(ctx context.Context, last string, fn func(repos []string) error) error {
	args := r.Called(ctx, last, fn)
	return args.Error(0)
}

func (r *mockOrasRegistry) Repository(ctx context.Context, name string) (orasRegistry.Repository, error) {
	args := r.Called(ctx, name)
	return args.Get(0).(orasRegistry.Repository), args.Error(1)
}

type mockRepo struct {
	mock.Mock
}

func (r *mockRepo) Blobs() orasRegistry.BlobStore {
	args := r.Called()
	return args.Get(0).(orasRegistry.BlobStore)
}

func (r *mockRepo) Manifests() orasRegistry.BlobStore {
	args := r.Called()
	return args.Get(0).(orasRegistry.BlobStore)
}

func (r *mockRepo) Tags(ctx context.Context, last string, fn func(tags []string) error) error {
	args := r.Called(ctx, last, fn)
	return args.Error(0)
}

func (r *mockRepo) Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error) {
	args := r.Called(ctx, reference)
	return args.Get(0).(ocispec.Descriptor), args.Error(1)
}

func (r *mockRepo) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {
	args := r.Called(ctx, target)
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (r *mockRepo) Delete(ctx context.Context, target ocispec.Descriptor) error {
	args := r.Called(ctx, target)
	return args.Error(0)
}

func (r *mockRepo) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	args := r.Called(ctx, target)
	return args.Get(0).(bool), args.Error(1)
}

func (r *mockRepo) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	args := r.Called(ctx, expected, content)
	return args.Error(0)
}

func (r *mockRepo) PushTag(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	args := r.Called(ctx, expected, content, reference)
	return args.Error(0)
}

func (r *mockRepo) Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error {
	args := r.Called(ctx, desc, reference)
	return args.Error(0)
}

func (r *mockRepo) FetchReference(ctx context.Context, reference string) (ocispec.Descriptor, io.ReadCloser, error) {
	args := r.Called(ctx, reference)
	return args.Get(0).(ocispec.Descriptor), args.Get(1).(io.ReadCloser), args.Error(2)
}

func (r *mockRepo) PushReference(ctx context.Context, expected ocispec.Descriptor, content io.Reader, reference string) error {
	args := r.Called(ctx, expected, content, reference)
	return args.Error(0)
}

func (o *UnGzipTarballMock) unGzipTarballFuncMock(path string, fileReader io.Reader) error {
	args := o.Called(path, fileReader)
	return args.Error(0)
}

func TestPackagerReader_SuccessPull(t *testing.T) {
	// Arrange
	ctx := context.Background()

	addOnTarget := initAndGetAddOnTarget(t, ctx)

	tempDestDir := t.TempDir()
	unGzipTarballMock := UnGzipTarballMock{}
	unGzipTarballMock.On("unGzipTarballFuncMock", tempDestDir, mock.AnythingOfType("*os.File")).Return(nil)

	uut := packager.NewPackageReader(unGzipTarballMock.unGzipTarballFuncMock)

	// Act
	_, gotErr := uut.Pull(ctx, addOnTarget, &packager.PullOptions{DestDir: tempDestDir, Extract: true})

	// Assert
	if gotErr != nil {
		t.Errorf("packager.pull() = %v, want nil", gotErr)
	}

	result, gotErr := os.ReadDir(tempDestDir)
	if gotErr != nil {
		t.Errorf("packager.pull() = %v, want nil", gotErr)
	}

	if len(result) == 0 {
		t.Errorf("packager.pull(): Content of dest dir is %d", len(result))
	}
	unGzipTarballMock.AssertExpectations(t)

}

func TestPackagerReader_UnzipManifestAndLogo(t *testing.T) {
	// Arrange
	ctx := context.Background()

	addOnTarget := initAndGetAddOnTarget(t, ctx)

	tempDestDir := t.TempDir()
	unGzipTarballMock := UnGzipTarballMock{}
	unGzipTarballMock.On("unGzipTarballFuncMock", tempDestDir, mock.AnythingOfType("*os.File")).Return(nil)
	uut := packager.NewPackageReader(unGzipTarballMock.unGzipTarballFuncMock)

	// Act
	_, gotErr := uut.Pull(ctx, addOnTarget, &packager.PullOptions{DestDir: tempDestDir, Extract: true})

	// Assert
	if gotErr != nil {
		t.Errorf("packager.pull() = %v, want nil", gotErr)
	}

	unGzipTarballMock.AssertExpectations(t)

}

func TestPackagerReader_NotUnzipManifestAndLogo(t *testing.T) {
	// Arrange
	ctx := context.Background()

	addOnTarget := initAndGetAddOnTarget(t, ctx)

	tempDestDir := t.TempDir()
	unGzipTarballMock := UnGzipTarballMock{}
	unGzipTarballMock.On("unGzipTarballFuncMock", mock.Anything, mock.Anything).Return(nil)
	uut := packager.NewPackageReader(unGzipTarballMock.unGzipTarballFuncMock)

	// Act
	_, gotErr := uut.Pull(ctx, addOnTarget, &packager.PullOptions{DestDir: tempDestDir, Extract: false})

	// Assert
	if gotErr != nil {
		t.Errorf("packager.pull() = %v, want nil", gotErr)
	}

	unGzipTarballMock.AssertNotCalled(t, "unGzipTarballFuncMock", mock.Anything, mock.Anything)

}

func TestPackagerReader_RemoveManifestAndLogoAfterUnzip(t *testing.T) {
	// Arrange
	ctx := context.Background()

	addOnTarget := initAndGetAddOnTarget(t, ctx)

	tempDestDir := t.TempDir()
	unGzipTarballMock := UnGzipTarballMock{}
	unGzipTarballMock.On("unGzipTarballFuncMock", tempDestDir, mock.AnythingOfType("*os.File")).Return(nil)
	uut := packager.NewPackageReader(unGzipTarballMock.unGzipTarballFuncMock)

	// Act
	_, gotErr := uut.Pull(ctx, addOnTarget, &packager.PullOptions{DestDir: tempDestDir, Extract: true})

	// Assert
	if gotErr != nil {
		t.Errorf("packager.pull() = %v, want nil", gotErr)
	}

	_, gotErr = os.Stat(tempDestDir + "/manifest.json and logo.png")

	if !errors.Is(gotErr, os.ErrNotExist) {
		t.Errorf("packager.pull() = %v, want os.ErrNotExist", gotErr)
	}

	unGzipTarballMock.AssertExpectations(t)

}

func TestPackagerReader_FailedUnzipManifestAndLogo(t *testing.T) {
	// Arrange
	ctx := context.Background()

	addOnTarget := initAndGetAddOnTarget(t, ctx)

	tempDestDir := t.TempDir()
	unGzipTarballMock := UnGzipTarballMock{}
	unGzipTarballMock.On("unGzipTarballFuncMock", tempDestDir, mock.AnythingOfType("*os.File")).Return(errors.New("test"))
	uut := packager.NewPackageReader(unGzipTarballMock.unGzipTarballFuncMock)

	// Act
	_, gotErr := uut.Pull(ctx, addOnTarget, &packager.PullOptions{DestDir: tempDestDir, Extract: true})

	// Assert
	if gotErr == nil {
		t.Errorf("packager.pull(): expect error")
	}

	unGzipTarballMock.AssertExpectations(t)

}

func TestPackageReader_InvalidAddOnPackage(t *testing.T) {
	// Arrange
	ctx := context.Background()
	tag := "0.42"

	mockOrasRegistry := &mockOrasRegistry{}
	mockRepo := &mockRepo{}
	desc := ocispec.Descriptor{}
	mockOrasRegistry.On("Repository", mock.Anything, mock.AnythingOfType("string")).Return(mockRepo, nil)
	mockRepo.On("Resolve", mock.Anything, mock.AnythingOfType("string")).Return(desc, fs.ErrInvalid)

	addOnTarget := registry.NewOciTargetDecorator(mockRepo, tag)
	targetRepository := registry.NewOciRepositoryTargetDecorator(addOnTarget, addOnTarget.AddOnVersion(), "MOCKREPO")

	tempDestDir := t.TempDir()
	unGzipTarballMock := UnGzipTarballMock{}
	uut := packager.NewPackageReader(unGzipTarballMock.unGzipTarballFuncMock)

	// Act
	_, gotErr := uut.Pull(ctx, targetRepository, &packager.PullOptions{DestDir: tempDestDir, Extract: true})

	// Assert
	if gotErr == nil {
		t.Errorf("packager.pull(): expect error")
	}

	unGzipTarballMock.AssertExpectations(t)
}

func TestPackageReader_TestDestinationDirectoryExists(t *testing.T) {
	// Arrange
	ctx := context.Background()

	addOnTarget := initAndGetAddOnTarget(t, ctx)
	targetRepository := registry.NewOciRepositoryTargetDecorator(addOnTarget, addOnTarget.AddOnVersion(), "TODOMOCKREPO")

	wrongDestDir := "not-exists"
	unGzipTarballMock := UnGzipTarballMock{}
	uut := packager.NewPackageReader(unGzipTarballMock.unGzipTarballFuncMock)

	// Act
	_, gotErr := uut.Pull(ctx, targetRepository, &packager.PullOptions{DestDir: wrongDestDir, Extract: true})

	if !errors.Is(gotErr, os.ErrNotExist) {
		t.Errorf("packager.pull() = %v, want os.ErrNotExist", gotErr)
	}

}

func initAndGetAddOnTarget(t *testing.T, ctx context.Context) *registry.RepositoryTargetDecorator {
	memoryStore := memory.New()
	manifestLogoDir := "manifest-logo-dir"
	dockerImagePath := "docker-image-path"
	tag := "0.42.0-1"

	ioStubPackageCreator := ioStubPackageCreator{
		dockerTarballPayload: []byte(dockerImagePath),
		gzipTarballPayload:   []byte(manifestLogoDir),
	}
	pkg := packager.NewPackageCreator(ioStubPackageCreator.gzipTarballFuncMock, ioStubPackageCreator.exportDockerImageFuncMock)

	root := &model.Root{
		ManifestVersion: "0.1",
		Version:         tag,
		Logo:            "logo.png",
		Title:           "add-on's title",
		Description:     "Describes this. test add-on",
		Services: map[string]*model.Service{
			"ucAddonTestService": {
				Type:   "docker-compose",
				Config: map[string]interface{}{"image": "test/docker-image:v1.1.1-alpha"},
			},
		},
		Platform: []string{"ucm", "ucg"},
		Vendor: &model.Vendor{
			Name:    "abc",
			Url:     "https://www.abc.de",
			Email:   "email@abc.de",
			Street:  "street",
			Zip:     "12345",
			City:    "City",
			Country: "Country",
		},
	}

	mockManifest := &mockManifestReader{logoErr: nil}
	mockManifest.On("ReadManifestFrom", mock.AnythingOfType("string")).Return(root, nil)

	rc := nopCloser{bytes.NewBufferString(ImageIndexWithSingleImageManifestJSON)}
	desc := ocispec.Descriptor{}

	mockOrasRegistry := &mockOrasRegistry{}
	mockRepo := &mockRepo{}
	mockOrasRegistry.On("Repository", mock.Anything, mock.AnythingOfType("string")).Return(mockRepo, nil)
	mockRepo.On("Resolve", mock.Anything, mock.AnythingOfType("string")).Return(desc, nil)
	mockRepo.On("Fetch", mock.Anything, mock.Anything).Return(rc, nil)

	addOnManifest, err := manifest.ParseAndValidate(mockManifest, mockManifest.fileExistsFunc, "")
	if err != nil {
		t.Fatal(err)
	}

	addOnTarget, err := pkg.CreateOciArtifact(ctx, addOnManifest, mockOrasRegistry)
	if err != nil {
		t.Fatal("[CreateOciArtifact] Unexpected error = ", err)
	}

	_, err = registry.Copy(ctx, addOnTarget, memoryStore, oras.DefaultCopyOptions)
	if err != nil {
		t.Fatal("[Push] Unexpected error = ", err)
	}

	return registry.NewOciRepositoryTargetDecorator(addOnTarget, addOnTarget.AddOnVersion(), "TODOMOCKREPO")
}

const ImageIndexWithSingleImageManifestJSON string = `{
		"mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
		"schemaVersion": 2,
		"manifests": [
		  {
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"digest": "sha256:d8a6c150cc9ec2dbc8b3b6c676d90046d6d3e820a6cb1f2ff93a24cc6273207e",
			"size": 528,
			"platform": {
			  "architecture": "arm",
			  "os": "linux",
			  "variant": "v7"
			}
		  }
		]
	  }`
