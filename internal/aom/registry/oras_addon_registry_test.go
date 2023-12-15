// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package registry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"reflect"
	"testing"
	"u-control/uc-aom/internal/pkg/config"
	oraswrapper "u-control/uc-aom/internal/pkg/oras-wrapper"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/mock"
	"oras.land/oras-go/v2/registry"
)

type mockOrasRegistry struct {
	mock.Mock
}

func (r *mockOrasRegistry) Repositories(ctx context.Context, last string, fn func(repos []string) error) error {
	args := r.Called(ctx, last, fn)
	return args.Error(0)
}

func (r *mockOrasRegistry) Repository(ctx context.Context, name string) (registry.Repository, error) {
	args := r.Called(ctx, name)
	return args.Get(0).(registry.Repository), args.Error(1)
}

type mockRepo struct {
	mock.Mock
	SupportPlatform platform
}

func (r *mockRepo) IsSupportedPlatform(platform *v1.Platform) bool {
	return platform.Architecture == r.SupportPlatform.architecture && platform.OS == r.SupportPlatform.os
}

func (r *mockRepo) Blobs() registry.BlobStore {
	args := r.Called()
	return args.Get(0).(registry.BlobStore)
}

func (r *mockRepo) Manifests() registry.BlobStore {
	args := r.Called()
	return args.Get(0).(registry.BlobStore)
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

func NopResetCloser(r *bytes.Reader) io.ReadCloser {
	return &nopResetCloser{r}
}

type nopResetCloser struct {
	*bytes.Reader
}

func (r nopResetCloser) Close() error {
	r.Seek(0, 0)
	return nil
}

type platform struct {
	architecture string
	os           string
}

func GetRepositoryTestFn() GetRepositoryFn {

	return func(ctx context.Context, registry registry.Registry, name string) (Repository, error) {
		return registry.Repository(ctx, name)
	}
}

func TestORASAddOnRegistry_Tags(t *testing.T) {
	type fields struct {
		registry              mockOrasRegistry
		getRepositoryCallback GetRepositoryFn
		architecture          string
		os                    string
	}

	type args struct {
		repository string
		tags       []string
		platforms  []platform
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:   "WithRightPlatform",
			fields: fields{registry: mockOrasRegistry{}, getRepositoryCallback: GetRepositoryTestFn(), architecture: "arm", os: "linux"},
			args: args{
				repository: "test-repository",
				tags:       []string{"1.0.0"},
				platforms:  []platform{{architecture: "arm", os: "linux"}},
			},
			want:    []string{"1.0.0"},
			wantErr: false,
		},
		{
			name:   "WithWrongPlatform",
			fields: fields{registry: mockOrasRegistry{}, getRepositoryCallback: GetRepositoryTestFn(), architecture: "arm", os: "linux"},
			args: args{
				repository: "test-repository",
				tags:       []string{"1.0.0"},
				platforms:  []platform{{architecture: "amd64", os: "linux"}},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "WithMultiPlatforms",
			fields: fields{registry: mockOrasRegistry{}, getRepositoryCallback: GetRepositoryTestFn(), architecture: "arm", os: "linux"},
			args: args{
				repository: "test-repository",
				tags:       []string{"1.0.0"},
				platforms:  []platform{{architecture: "amd64", os: "linux"}, {architecture: "arm", os: "linux"}},
			},
			want:    []string{"1.0.0"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			r := &ORASAddOnRegistry{
				registry:              &tt.fields.registry,
				getRepositoryCallback: tt.fields.getRepositoryCallback,
				architecture:          tt.fields.architecture,
				os:                    tt.fields.os,
			}
			supportedPlatform := platform{tt.fields.architecture, tt.fields.os}
			mockRepo := createMockRepository(supportedPlatform, tt.args.tags)
			setupMockRepository(mockRepo, tt.args.tags, tt.args.platforms)

			tt.fields.registry.On("Repository", mock.Anything, tt.args.repository).Return(mockRepo, nil)

			// act
			got, err := r.Tags(tt.args.repository)

			// assert
			if (err != nil) != tt.wantErr {
				t.Errorf("ORASAddOnRegistry.Tags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ORASAddOnRegistry.Tags() = %v, want %v", got, tt.want)
			}

			tt.fields.registry.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestORASAddOnRegistry_TagsMediaTypes(t *testing.T) {
	dockerMediaType := "application/vnd.docker.distribution.manifest.v2+json"

	supportedPlatform := platform{architecture: "arm", os: "linux"}

	type fields struct {
		registry              mockOrasRegistry
		getRepositoryCallback GetRepositoryFn
		architecture          string
		os                    string
	}

	type args struct {
		repository string
		tags       []string
		mediaTypes []string
		fetchError error
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:   "all-docker-media-type",
			fields: fields{registry: mockOrasRegistry{}, getRepositoryCallback: GetRepositoryTestFn(), architecture: supportedPlatform.architecture, os: supportedPlatform.os},
			args: args{
				repository: "test-repository",
				tags:       []string{"1.1.0-rc1", "1.1.0-rc2", "1.1.0-rc3"},
				mediaTypes: []string{dockerMediaType, dockerMediaType, dockerMediaType},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "docker-and-correct-media-type",
			fields: fields{registry: mockOrasRegistry{}, getRepositoryCallback: GetRepositoryTestFn(), architecture: supportedPlatform.architecture, os: supportedPlatform.os},
			args: args{
				repository: "test-repository",
				tags:       []string{"2.1.0-rc1", "2.1.0-rc2", "2.1.0-rc3"},
				mediaTypes: []string{dockerMediaType, config.UcImageLayerMediaType, config.UcImageLayerMediaType},
			},
			want:    []string{"2.1.0-rc2", "2.1.0-rc3"},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			r := &ORASAddOnRegistry{
				registry:              &tt.fields.registry,
				getRepositoryCallback: tt.fields.getRepositoryCallback,
				architecture:          tt.fields.architecture,
				os:                    tt.fields.os,
			}
			mockRepo := createMockRepository(supportedPlatform, tt.args.tags)
			setupMockRepositoryWithMediaTypes(mockRepo, tt.args.tags, tt.args.mediaTypes)

			tt.fields.registry.On("Repository", mock.Anything, tt.args.repository).Return(mockRepo, nil)

			// act
			got, err := r.Tags(tt.args.repository)

			// assert
			if (err != nil) != tt.wantErr {
				t.Errorf("ORASAddOnRegistry.Tags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ORASAddOnRegistry.Tags() = %v, want %v", got, tt.want)
			}

			tt.fields.registry.AssertExpectations(t)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestORASAddOnRegistry_TagsFetchError(t *testing.T) {

	supportedPlatform := platform{architecture: "arm", os: "linux"}

	// arrange
	tags := []string{"1.0.0-1"}
	wantErr := fs.ErrInvalid
	var want []string
	registry := &mockOrasRegistry{}
	r := &ORASAddOnRegistry{registry: registry, getRepositoryCallback: GetRepositoryTestFn(), architecture: supportedPlatform.architecture, os: supportedPlatform.os}
	mockRepo := createMockRepository(supportedPlatform, tags)
	registry.On("Repository", mock.Anything, "test-repository").Return(mockRepo, nil)
	manifestTuples := createImageManifestsWithPlatformsAndUcManifestLayer([]platform{mockRepo.SupportPlatform})
	imageIndexTuple, _ := oraswrapper.CreateImageIndexTuple(manifestTuples)
	mockRepo.On("Resolve", mock.Anything, tags[0]).Return(*imageIndexTuple.Desc, nil)
	mockRepo.On("Fetch", mock.Anything, *imageIndexTuple.Desc).Return(createReaderCloserAsFetchResultFrom(imageIndexTuple.Blob), nil)
	manifestTuple := manifestTuples[0]
	mockRepo.On("Fetch", mock.Anything, convertToJSONDesc(manifestTuple.Desc)).Return(createReaderCloserAsFetchResultFrom(manifestTuple.Blob), wantErr)

	// act
	got, err := r.Tags("test-repository")

	// assert
	if err == nil {
		t.Errorf("ORASAddOnRegistry.Tags() error = %v, wantErr %v", err, wantErr)
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ORASAddOnRegistry.Tags() = %v, want %v", got, want)
	}

	registry.AssertExpectations(t)
	mockRepo.AssertExpectations(t)

}

func createMockRepository(supportPlatform platform, providedTags []string) *mockRepo {
	mockRepo := &mockRepo{SupportPlatform: supportPlatform}
	tagPaginationCallbackFn := func(args mock.Arguments) {
		arg := args.Get(2).(func(tags []string) error)
		arg(providedTags)
	}
	mockRepo.On("Tags", mock.Anything, mock.Anything, mock.Anything).Run(tagPaginationCallbackFn).Return(nil)

	return mockRepo
}

func setupMockRepository(mockRepo *mockRepo, providedTags []string, platforms []platform) {
	for _, providedTag := range providedTags {
		manifestTuples := createImageManifestsWithPlatformsAndUcManifestLayer(platforms)
		imageIndexTuple, _ := oraswrapper.CreateImageIndexTuple(manifestTuples)

		mockRepo.On("Resolve", mock.Anything, providedTag).Return(*imageIndexTuple.Desc, nil)
		mockRepo.On("Fetch", mock.Anything, *imageIndexTuple.Desc).Return(createReaderCloserAsFetchResultFrom(imageIndexTuple.Blob), nil)

		for _, manifestTuple := range manifestTuples {
			if mockRepo.IsSupportedPlatform(manifestTuple.Desc.Platform) {
				mockRepo.On("Fetch", mock.Anything, convertToJSONDesc(manifestTuple.Desc)).Return(createReaderCloserAsFetchResultFrom(manifestTuple.Blob), nil)
			}
		}

	}
}

func setupMockRepositoryWithMediaTypes(mockRepo *mockRepo, providedTags []string, mediaTypes []string) {
	for tagIndex, providedTag := range providedTags {
		manifestTuple := createImageManifestsWithPlatformAndMediaType(mockRepo.SupportPlatform, mediaTypes[tagIndex])
		imageIndexTuple, _ := oraswrapper.CreateImageIndexTuple([]*oraswrapper.DescriptorBlobTuple{manifestTuple})

		mockRepo.On("Resolve", mock.Anything, providedTag).Return(*imageIndexTuple.Desc, nil)
		mockRepo.On("Fetch", mock.Anything, *imageIndexTuple.Desc).Return(createReaderCloserAsFetchResultFrom(imageIndexTuple.Blob), nil)

		mockRepo.On("Fetch", mock.Anything, convertToJSONDesc(manifestTuple.Desc)).Return(createReaderCloserAsFetchResultFrom(manifestTuple.Blob), nil)
	}
}

func createImageManifestsWithPlatformsAndUcManifestLayer(platforms []platform) []*oraswrapper.DescriptorBlobTuple {

	result := make([]*oraswrapper.DescriptorBlobTuple, 0, len(platforms))

	for _, platform := range platforms {
		ociPlatform := &ocispec.Platform{
			Architecture: platform.architecture,
			OS:           platform.os,
		}
		ucManifest := &oraswrapper.DescriptorBlobTuple{
			Desc: &ocispec.Descriptor{
				MediaType: config.UcImageLayerMediaType,
			},
		}
		platformKey := oraswrapper.FromOCIPlatform(ociPlatform)
		_, manifestTuple, _ := oraswrapper.CreateImageConfigTupleAndImageManifestTuple(platformKey, "", ucManifest, []*oraswrapper.DescriptorBlobTuple{})
		result = append(result, manifestTuple)
	}

	return result

}

func createImageManifestsWithPlatformAndMediaType(platform platform, mediaType string) *oraswrapper.DescriptorBlobTuple {

	ociPlatform := &ocispec.Platform{
		Architecture: platform.architecture,
		OS:           platform.os,
	}
	ucManifest := &oraswrapper.DescriptorBlobTuple{
		Desc: &ocispec.Descriptor{
			MediaType: mediaType,
		},
	}
	platformKey := oraswrapper.FromOCIPlatform(ociPlatform)
	_, manifestTuple, _ := oraswrapper.CreateImageConfigTupleAndImageManifestTuple(platformKey, "", ucManifest, []*oraswrapper.DescriptorBlobTuple{})

	return manifestTuple
}

func createReaderCloserAsFetchResultFrom(blob []byte) io.ReadCloser {
	fetchResult := NopResetCloser(bytes.NewReader(blob))
	return fetchResult
}

func convertToJSONDesc(content *ocispec.Descriptor) ocispec.Descriptor {
	contentBytes, _ := json.Marshal(content)
	desc := ocispec.Descriptor{}
	json.Unmarshal(contentBytes, &desc)
	return desc
}
