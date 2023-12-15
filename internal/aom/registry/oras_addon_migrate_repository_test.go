// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package registry

import (
	"context"
	"io"
	"reflect"
	"testing"
	aom_manifest "u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/pkg/manifest"
	"u-control/uc-aom/internal/pkg/manifest/v0_1"
	oraswrapper "u-control/uc-aom/internal/pkg/oras-wrapper"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
)

func Test_orasAddonMigration_ResolveAndMigrate_v1_0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	builder := oraswrapper.NewGraphBuilder(tag)

	manifestContent := []byte("manifestContent")
	builder.AppendUcManifest(manifestContent)

	dockerImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	builder.AppendDockerImage(dockerImageContent, dockerPlatform)

	source, err := builder.BuildAndTag(ctx)
	assert.Nil(t, err)

	want, err := source.Resolve(ctx, tag)
	assert.Nil(t, err)

	migrationStorage := memory.New()

	uut := &orasAddonMigrateRepository{
		source:                         source,
		migrationStorage:               migrationStorage,
		dockerImageSourceDescriptorMap: make(map[string]ocispec.Descriptor),
	}

	// arrange
	got, err := uut.Resolve(ctx, tag)

	// assert
	assert.Nil(t, err)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("orasAddonMigration.ResolveAndMigrate() = %v, want %v", got, want)
	}

	exists, err := migrationStorage.Exists(ctx, got)

	assert.False(t, exists)
	assert.Nil(t, err)

}

func Test_orasAddonMigration_ResolveAndMigrate_v0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	v0Builder := oraswrapper.NewGraphBuilder(tag)

	v0_1Manifest := &v0_1.Root{
		ManifestVersion: v0_1.ValidManifestVersion,
		Title:           "test",
		Version:         tag,
		Vendor: &v0_1.Vendor{
			Name: "Vendor",
		},
	}
	decompressor := NewMockDecompressor()
	decompressor.WithManifestV0_1(v0_1Manifest)
	v0Builder.AppendUcManifest(decompressor.UcManifestLayerBlob)

	dockerImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	v0Builder.AppendDockerImage(dockerImageContent, dockerPlatform)

	v0Source, err := BuildAndTagV0(ctx, v0Builder, dockerPlatform)
	assert.Nil(t, err)

	v1Builder := oraswrapper.NewGraphBuilder(tag)
	ucImageLayerAnnotations := manifest.CreateUcManifestAnnotationsV1_0(tag, manifest.ValidManifestVersion)
	v1ManifestContent, err := manifest.MigrateUcManifest(v0_1Manifest.ManifestVersion, decompressor.UcManifestBlob)
	assert.Nil(t, err)
	v1Builder.AppendUcManifest(v1ManifestContent, ucImageLayerAnnotations...)
	v1Builder.AppendDockerImage(dockerImageContent, dockerPlatform)
	v1Source, err := v1Builder.BuildAndTag(ctx)
	assert.Nil(t, err)
	want, err := v1Source.Resolve(ctx, tag)

	migrationStorage := memory.New()

	uut := &orasAddonMigrateRepository{
		source:                         v0Source,
		migrationStorage:               migrationStorage,
		dockerImageSourceDescriptorMap: make(map[string]ocispec.Descriptor),
		decompressor:                   decompressor,
	}

	// arrange
	got, err := uut.Resolve(ctx, tag)

	// assert
	assert.Nil(t, err)

	assertDescriptors(t, want, got)

	exists, err := migrationStorage.Exists(ctx, got)

	assert.True(t, exists)
	assert.Nil(t, err)

	imageIndex, err := oraswrapper.FetchImageIndex(ctx, migrationStorage, got)
	assert.Nil(t, err)

	v1ImageIndex, err := oraswrapper.FetchImageIndex(ctx, v1Source, want)
	assert.Nil(t, err)

	imageManifest, err := oraswrapper.FetchImageManifest(ctx, migrationStorage, imageIndex.Manifests[0])
	assert.Nil(t, err)

	v1ImageManifest, err := oraswrapper.FetchImageManifest(ctx, v1Source, v1ImageIndex.Manifests[0])
	assert.Nil(t, err)

	for i, wantDesc := range v1ImageManifest.Layers {
		gotDesc := imageManifest.Layers[i]
		assertDescriptors(t, wantDesc, gotDesc)
	}
}

func Test_orasAddonMigration_ResolveAndMigrate_v0_onlyOnes(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	v0Builder := oraswrapper.NewGraphBuilder(tag)

	v0_1Manifest := &v0_1.Root{
		ManifestVersion: v0_1.ValidManifestVersion,
		Title:           "test",
		Version:         tag,
		Vendor: &v0_1.Vendor{
			Name: "Vendor",
		},
	}
	decompressor := NewMockDecompressor()
	decompressor.WithManifestV0_1(v0_1Manifest)
	v0Builder.AppendUcManifest(decompressor.UcManifestLayerBlob)

	dockerImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	v0Builder.AppendDockerImage(dockerImageContent, dockerPlatform)

	v0Source, err := BuildAndTagV0(ctx, v0Builder, dockerPlatform)
	assert.Nil(t, err)

	migrationStorage := memory.New()
	uut := &orasAddonMigrateRepository{
		source:                         v0Source,
		migrationStorage:               migrationStorage,
		dockerImageSourceDescriptorMap: make(map[string]ocispec.Descriptor),
		decompressor:                   decompressor,
	}

	// arrange
	got1, err := uut.Resolve(ctx, tag)
	got2, err := uut.Resolve(ctx, tag)

	equalDesc := EqualOCI(got1, got2)
	assert.True(t, equalDesc)

}

func Test_orasAddonMigration_ResolveAndMigrate_Fetch_dockerImageFromSource_v0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	v0Builder := oraswrapper.NewGraphBuilder(tag)

	v0_1Manifest := &v0_1.Root{
		ManifestVersion: v0_1.ValidManifestVersion,
		Title:           "test",
		Version:         tag,
		Vendor: &v0_1.Vendor{
			Name: "Vendor",
		},
	}
	decompressor := NewMockDecompressor()
	decompressor.WithManifestV0_1(v0_1Manifest)
	v0Builder.AppendUcManifest(decompressor.UcManifestLayerBlob)

	wantImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	v0Builder.AppendDockerImage(wantImageContent, dockerPlatform)

	v0Source, err := BuildAndTagV0(ctx, v0Builder, dockerPlatform)
	assert.Nil(t, err)

	migrationStorage := memory.New()
	uut := &orasAddonMigrateRepository{
		source:                         v0Source,
		migrationStorage:               migrationStorage,
		dockerImageSourceDescriptorMap: make(map[string]ocispec.Descriptor),
		decompressor:                   decompressor,
	}

	// act
	got, err := uut.Resolve(ctx, tag)

	// assert
	assert.Nil(t, err)
	imageIndex, err := oraswrapper.FetchImageIndex(ctx, migrationStorage, got)
	assert.Nil(t, err)

	imageManifest, err := oraswrapper.FetchImageManifest(ctx, migrationStorage, imageIndex.Manifests[0])
	assert.Nil(t, err)
	dockerImageLayers := imageManifest.Layers[1:]
	for _, dockerImageLayer := range dockerImageLayers {
		rc, err := uut.Fetch(ctx, dockerImageLayer)
		assert.Nil(t, err)
		gotImageContent, err := io.ReadAll(rc)
		assert.Nil(t, err)
		assert.Equal(t, wantImageContent, gotImageContent)
	}
}

func Test_orasAddonMigration_ResolveAndMigrate_Fetch_dockerImageFromSource_v1_0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	builder := oraswrapper.NewGraphBuilder(tag)

	manifestContent := []byte("manifestContent")
	ucImageLayerAnnotations := manifest.CreateUcManifestAnnotationsV1_0(tag, manifest.ValidManifestVersion)
	builder.AppendUcManifest(manifestContent, ucImageLayerAnnotations...)

	wantImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	builder.AppendDockerImage(wantImageContent, dockerPlatform)

	source, err := builder.BuildAndTag(ctx)
	assert.Nil(t, err)

	_, err = source.Resolve(ctx, tag)
	assert.Nil(t, err)

	migrationStorage := memory.New()

	uut := &orasAddonMigrateRepository{
		source:           source,
		migrationStorage: migrationStorage,
	}

	// arrange
	got, err := uut.Resolve(ctx, tag)

	// assert

	assert.Nil(t, err)
	imageIndex, err := oraswrapper.FetchImageIndex(ctx, uut, got)
	assert.Nil(t, err)

	imageManifest, err := oraswrapper.FetchImageManifest(ctx, uut, imageIndex.Manifests[0])
	assert.Nil(t, err)
	dockerImageLayers := imageManifest.Layers[1:]
	for _, dockerImageLayer := range dockerImageLayers {
		rc, err := uut.Fetch(ctx, dockerImageLayer)
		assert.Nil(t, err)
		gotImageContent, err := io.ReadAll(rc)
		assert.Nil(t, err)
		assert.Equal(t, wantImageContent, gotImageContent)
	}

}

func Test_orasAddonMigration_ResolveAndMigrate_Fetch_UcManifest_v1_0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	builder := oraswrapper.NewGraphBuilder(tag)

	wantManifestContent := []byte("manifestContent")
	annotations := manifest.CreateUcManifestAnnotationsV1_0(tag, manifest.ValidManifestVersion)
	builder.AppendUcManifest(wantManifestContent, annotations...)

	imageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	builder.AppendDockerImage(imageContent, dockerPlatform)

	source, err := builder.BuildAndTag(ctx)
	assert.Nil(t, err)

	_, err = source.Resolve(ctx, tag)
	assert.Nil(t, err)

	migrationStorage := memory.New()
	decompressor := NewMockDecompressor()
	decompressor.WithFetchContent(wantManifestContent)
	uut := &orasAddonMigrateRepository{
		source:           source,
		migrationStorage: migrationStorage,
		decompressor:     decompressor,
	}

	// arrange
	got, err := uut.Resolve(ctx, tag)

	// assert
	assert.Nil(t, err)
	imageIndex, err := oraswrapper.FetchImageIndex(ctx, uut, got)
	assert.Nil(t, err)

	imageManifest, err := oraswrapper.FetchImageManifest(ctx, uut, imageIndex.Manifests[0])
	assert.Nil(t, err)

	ucManifestLayer, err := getUcImageLayer(imageManifest)
	assert.Nil(t, err)

	rc, err := uut.Fetch(ctx, ucManifestLayer)
	assert.Nil(t, err)
	gotUcManifestContent, err := io.ReadAll(rc)
	assert.Nil(t, err)
	assert.Equal(t, gotUcManifestContent, wantManifestContent)
	decompressor.AssertExpectations(t)

}

func Test_orasAddonMigration_ResolveAndMigrate_Fetch_UcManifest_v0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	v0Builder := oraswrapper.NewGraphBuilder(tag)

	v0_1Manifest := &v0_1.Root{
		ManifestVersion: v0_1.ValidManifestVersion,
		Title:           "test",
		Version:         tag,
		Vendor: &v0_1.Vendor{
			Name: "Vendor",
		},
	}
	decompressor := NewMockDecompressor()
	decompressor.WithManifestV0_1(v0_1Manifest)
	wantManifestContent, err := manifest.MigrateUcManifest(v0_1Manifest.ManifestVersion, decompressor.UcManifestBlob)
	assert.Nil(t, err)
	decompressor.WithFetchContent(wantManifestContent)
	v0Builder.AppendUcManifest(decompressor.UcManifestLayerBlob)

	wantImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	v0Builder.AppendDockerImage(wantImageContent, dockerPlatform)

	v0Source, err := BuildAndTagV0(ctx, v0Builder, dockerPlatform)
	assert.Nil(t, err)

	migrationStorage := memory.New()
	uut := &orasAddonMigrateRepository{
		source:                         v0Source,
		migrationStorage:               migrationStorage,
		dockerImageSourceDescriptorMap: make(map[string]ocispec.Descriptor),
		decompressor:                   decompressor,
	}

	// act
	got, err := uut.Resolve(ctx, tag)

	// assert
	assert.Nil(t, err)
	imageIndex, err := oraswrapper.FetchImageIndex(ctx, uut, got)
	assert.Nil(t, err)

	imageManifest, err := oraswrapper.FetchImageManifest(ctx, uut, imageIndex.Manifests[0])
	assert.Nil(t, err)

	ucManifestLayer, err := getUcImageLayer(imageManifest)
	assert.Nil(t, err)

	rc, err := uut.Fetch(ctx, ucManifestLayer)
	gotUcManifestContent, err := aom_manifest.UnmarshalDecompressedContent(rc)
	assert.Nil(t, err)
	assert.Equal(t, wantManifestContent, gotUcManifestContent.Manifest)

	decompressor.AssertExpectations(t)
}

func Test_orasAddonMigration_ResolveAndMigrate_MapDescriptorSize_dockerImageFromSource_v0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	v0Builder := oraswrapper.NewGraphBuilder(tag)

	v0_1Manifest := &v0_1.Root{
		ManifestVersion: v0_1.ValidManifestVersion,
		Title:           "test",
		Version:         tag,
		Vendor: &v0_1.Vendor{
			Name: "Vendor",
		},
	}
	decompressor := NewMockDecompressor()
	decompressor.WithManifestV0_1(v0_1Manifest)
	v0Builder.AppendUcManifest(decompressor.UcManifestLayerBlob)

	wantImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}

	v0Builder.AppendDockerImage(wantImageContent, dockerPlatform)

	v0Source, err := BuildAndTagV0(ctx, v0Builder, dockerPlatform)
	assert.Nil(t, err)

	migrationStorage := memory.New()
	uut := &orasAddonMigrateRepository{
		source:                         v0Source,
		migrationStorage:               migrationStorage,
		dockerImageSourceDescriptorMap: make(map[string]ocispec.Descriptor),
		decompressor:                   decompressor,
	}

	// act
	got, err := uut.Resolve(ctx, tag)

	// assert
	assert.Nil(t, err)
	imageIndex, err := oraswrapper.FetchImageIndex(ctx, migrationStorage, got)
	assert.Nil(t, err)

	imageManifest, err := oraswrapper.FetchImageManifest(ctx, migrationStorage, imageIndex.Manifests[0])
	assert.Nil(t, err)
	platformKey := oraswrapper.FromOCIPlatform(dockerPlatform)
	originalDockerImageLayerTuple := v0Builder.DockerImages[platformKey]
	dockerImageLayers := imageManifest.Layers[1:]

	assert.Len(t, dockerImageLayers, len(originalDockerImageLayerTuple))

	for index, dockerImageLayer := range dockerImageLayers {
		wantSize := originalDockerImageLayerTuple[index].Desc.Size
		gotSize := uut.MapDescriptorSize(dockerImageLayer)
		assert.Equal(t, wantSize, gotSize)
	}
}

func Test_orasAddonMigration_ResolveAndMigrate_MapDescriptorSize_dockerImageFromSource_v1_0(t *testing.T) {
	// arrange
	ctx := context.Background()
	tag := "1.0"
	builder := oraswrapper.NewGraphBuilder(tag)

	manifestContent := []byte("manifestContent")
	ucImageLayerAnnotations := manifest.CreateUcManifestAnnotationsV1_0(tag, manifest.ValidManifestVersion)
	builder.AppendUcManifest(manifestContent, ucImageLayerAnnotations...)

	wantImageContent := []byte("dockerImageContent")
	dockerPlatform := &ocispec.Platform{
		Architecture: "amd64",
		OS:           "linux",
	}
	builder.AppendDockerImage(wantImageContent, dockerPlatform)

	source, err := builder.BuildAndTag(ctx)
	assert.Nil(t, err)

	_, err = source.Resolve(ctx, tag)
	assert.Nil(t, err)

	migrationStorage := memory.New()

	uut := &orasAddonMigrateRepository{
		source:           source,
		migrationStorage: migrationStorage,
	}

	// arrange
	got, err := uut.Resolve(ctx, tag)
	assert.Nil(t, err)

	// assert
	imageIndex, err := oraswrapper.FetchImageIndex(ctx, uut, got)
	assert.Nil(t, err)

	imageManifest, err := oraswrapper.FetchImageManifest(ctx, uut, imageIndex.Manifests[0])
	assert.Nil(t, err)

	platformKey := oraswrapper.FromOCIPlatform(dockerPlatform)
	originalDockerImageLayerTuple := builder.DockerImages[platformKey]
	dockerImageLayers := imageManifest.Layers[1:]

	assert.Len(t, dockerImageLayers, len(originalDockerImageLayerTuple))

	for index, dockerImageLayer := range dockerImageLayers {
		wantSize := originalDockerImageLayerTuple[index].Desc.Size
		gotSize := uut.MapDescriptorSize(dockerImageLayer)
		assert.Equal(t, wantSize, gotSize)
	}

}

// EqualOCI returns true if two OCI descriptors point to the same content.
func EqualOCI(a, b ocispec.Descriptor) bool {
	return a.Digest == b.Digest && a.Size == b.Size && a.MediaType == b.MediaType
}

func assertDescriptors(t *testing.T, want ocispec.Descriptor, got ocispec.Descriptor) {
	assert.Equal(t, want.MediaType, got.MediaType)

	assert.NotEmpty(t, got.Digest)
	if !reflect.DeepEqual(got.Annotations, want.Annotations) {
		t.Errorf("assertDescriptors annotations got = %v, want %v", got, want)
	}
}

func BuildAndTagV0(ctx context.Context, builder *oraswrapper.GraphBuilder, platform *ocispec.Platform) (oras.Target, error) {
	src := memory.New()

	err := oraswrapper.PushAll(ctx, src, builder.UcManifestTuple)
	if err != nil {
		return nil, err
	}

	platformKey := oraswrapper.FromOCIPlatform(platform)
	imageConfigTuple, imageManifestTuple, err := builder.AppendImageManifest(platformKey)
	if err != nil {
		return nil, err
	}

	err = oraswrapper.PushAll(ctx, src, builder.DockerImages[platformKey]...)
	if err != nil {
		return nil, err
	}

	err = oraswrapper.PushAll(ctx, src, imageConfigTuple)
	if err != nil {
		return nil, err
	}

	err = oraswrapper.PushAll(ctx, src, imageManifestTuple)
	if err != nil {
		return nil, err
	}

	err = src.Tag(ctx, *(imageManifestTuple.Desc), builder.Tag)

	return src, nil
}
