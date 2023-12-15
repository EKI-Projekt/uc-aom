// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	aom_manifest "u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/pkg/manifest"
	"u-control/uc-aom/internal/pkg/manifest/v0_1"
	oraswrapper "u-control/uc-aom/internal/pkg/oras-wrapper"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/errdef"
)

// Interface of an add-on repository
type Repository interface {
	content.Storage
	content.Resolver
}

// Interface which is used to map size of an descriptor if the property size can be used direcly
type DescriptorSizeMapper interface {
	MapDescriptorSize(ocispec.Descriptor) int64
}

type orasAddonMigrateRepository struct {
	source                         oras.Target
	migrationStorage               oras.Target
	dockerImageSourceDescriptorMap map[string]ocispec.Descriptor
	decompressor                   aom_manifest.ManifestLayerDecompressor
}

func NewOrasAddonMigrateRepository(source oras.Target, decompressor aom_manifest.ManifestLayerDecompressor) *orasAddonMigrateRepository {
	return &orasAddonMigrateRepository{
		source:                         source,
		migrationStorage:               memory.New(),
		dockerImageSourceDescriptorMap: make(map[string]ocispec.Descriptor),
		decompressor:                   decompressor,
	}
}

func (m *orasAddonMigrateRepository) Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error) {

	exists, err := m.Exists(ctx, target)
	if err != nil {
		return nil, err
	}

	if exists {
		if orginalDockerImageDesc, isDummyDescriptor := m.isDummyDockerImageDescriptor(target); isDummyDescriptor {
			return m.source.Fetch(ctx, orginalDockerImageDesc)
		}
		return m.migrationStorage.Fetch(ctx, target)
	}

	if !IsUcImageLayerMediaType(target.MediaType) {
		return m.source.Fetch(ctx, target)
	}
	return m.fetchAndDecompressUcManifestTarget(ctx, m.source, target)

}

func (m *orasAddonMigrateRepository) Exists(ctx context.Context, target ocispec.Descriptor) (bool, error) {
	return m.migrationStorage.Exists(ctx, target)
}

func (m *orasAddonMigrateRepository) Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error {
	return errdef.ErrUnsupported
}

func (m *orasAddonMigrateRepository) Resolve(ctx context.Context, tag string) (ocispec.Descriptor, error) {
	if m.hasTagBeenMigrated(ctx, tag) {
		log.Debug("migrationStorage.Resolve()")
		return m.migrationStorage.Resolve(ctx, tag)
	}

	log.Debug("source.Resolve()")
	desc, err := m.source.Resolve(ctx, tag)
	if err != nil {
		log.Error("source.Resolve() error =", err)
		return ocispec.Descriptor{}, err

	}

	return m.migrate(ctx, desc, tag)
}

func (m *orasAddonMigrateRepository) MapDescriptorSize(desc ocispec.Descriptor) int64 {
	// Because of the dummy docker images we need to map if there is a dummy docker image descriptor
	if orginalDockerImageDesc, isDummyDescriptor := m.isDummyDockerImageDescriptor(desc); isDummyDescriptor {
		return orginalDockerImageDesc.Size
	}

	return desc.Size
}

func (m *orasAddonMigrateRepository) isDummyDockerImageDescriptor(desc ocispec.Descriptor) (originalDescriptor ocispec.Descriptor, isDummyDescriptor bool) {
	if originalDescriptor, ok := m.dockerImageSourceDescriptorMap[string(desc.Digest)]; ok {
		return originalDescriptor, ok
	}
	return ocispec.Descriptor{}, false
}

func (m *orasAddonMigrateRepository) hasTagBeenMigrated(ctx context.Context, tag string) bool {
	_, err := m.migrationStorage.Resolve(ctx, tag)
	return !errors.Is(err, errdef.ErrNotFound)
}

func (m *orasAddonMigrateRepository) migrate(ctx context.Context, desc ocispec.Descriptor, tag string) (ocispec.Descriptor, error) {
	switch desc.MediaType {
	case ocispec.MediaTypeImageManifest:
		return m.migrateImageManifestDescV0_to_V1_0(ctx, m.source, desc, tag)

	case ocispec.MediaTypeImageIndex:
		return m.migrateImageIndexDesc(ctx, m.source, desc, tag)

	}

	return ocispec.Descriptor{}, errors.New(fmt.Sprintf("Unexpected mediatype: '%s'", desc.MediaType))
}

func (m *orasAddonMigrateRepository) migrateImageManifestDescV0_to_V1_0(ctx context.Context, source oras.Target, desc ocispec.Descriptor, tag string) (ocispec.Descriptor, error) {
	log.Debug("migrateImageManifestDesc()")
	builder := oraswrapper.NewGraphBuilder(tag)

	imageManifestBlob, err := oraswrapper.FetchImageManifest(ctx, source, desc)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("oraswrapper.FetchImageManifest(): Unexpected err: %v, desc: %v", err, desc))
	}

	ucImageDescLayer, err := getUcImageLayer(imageManifestBlob)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("getUcImageLayer(): Unexpected err: %v", err))
	}

	rc, err := m.fetchAndDecompressUcManifestTarget(ctx, source, ucImageDescLayer)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("content.FetchAll(): Unexpected err: %v, desc: %v", err, ucImageDescLayer))
	}

	content, err := aom_manifest.UnmarshalDecompressedContent(rc)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("UnmarshalDecompress(): Unexpected err: %v", err))
	}

	content.Manifest, err = manifest.MigrateUcManifest(v0_1.ValidManifestVersion, content.Manifest)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("manifest.MigrateUcManifest(): Unexpected err: %v, desc: %v", err, ucImageDescLayer))
	}

	ucManifestBlob, err := json.Marshal(content)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("json.Marshal(): Unexpected err: %v, content: %v", err, content))
	}

	ucManifestLayerAnnotations := manifest.CreateUcManifestAnnotationsV1_0(tag, manifest.ValidManifestVersion)
	builder.AppendUcManifest(ucManifestBlob, ucManifestLayerAnnotations...)

	platform, err := m.resolvePlatformFromImageConfig(ctx, source, imageManifestBlob.Config)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("builder.BuildAndTag(): Unexpected err: %v", err))
	}

	// The migration does only need to work on the uc manifest layer.
	// To create a complete oci image index and image manifest descriptor, we also need to include the docker images.
	// However, the size of the docker image layers is much larger than the size of the uc manifest layer.
	// As a result, we can't fetch the docker layers and store them into the migration memory storage.
	// We create dummy docker images which act as a placeholder for the original docker images.
	dockerImageLayers := imageManifestBlob.Layers[1:]
	m.createAndAppendDummyDockerImage(builder, dockerImageLayers, platform)

	addOnTarget, err := builder.BuildAndTag(ctx)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("builder.BuildAndTag(): Unexpected err: %v", err))
	}

	migratedImageIndexDesc, err := oras.Copy(ctx, addOnTarget, tag, m.migrationStorage, "", oras.DefaultCopyOptions)
	return migratedImageIndexDesc, err
}

func (*orasAddonMigrateRepository) resolvePlatformFromImageConfig(ctx context.Context, source oras.Target, configDescriptor ocispec.Descriptor) (*ocispec.Platform, error) {
	imageConfigBlob, err := oraswrapper.FetchImageConfig(ctx, source, configDescriptor)
	if err != nil {
		return &ocispec.Platform{}, errors.New(fmt.Sprintf("oraswrapper.FetchImageConfig(): Unexpected err: %v, desc: %v", err, configDescriptor))
	}

	platform := &ocispec.Platform{
		Architecture: imageConfigBlob.Architecture,
		OS:           imageConfigBlob.OS,
	}
	return platform, nil
}

func (m *orasAddonMigrateRepository) createAndAppendDummyDockerImage(builder *oraswrapper.GraphBuilder, dockerImageLayers []ocispec.Descriptor, platform *ocispec.Platform) {
	for index, dockerImageLayerDesc := range dockerImageLayers {
		dummyDockerBlob := append([]byte(dockerImageLayerDesc.Digest), byte(dockerImageLayerDesc.Size), byte(index))
		builder.AppendDockerImage(dummyDockerBlob, platform)

		platformKey := oraswrapper.FromOCIPlatform(platform)
		dummyDockerTuple := builder.DockerImages[platformKey][index]
		m.dockerImageSourceDescriptorMap[string(dummyDockerTuple.Desc.Digest)] = dockerImageLayerDesc
	}
}

func (m *orasAddonMigrateRepository) migrateImageIndexDesc(ctx context.Context, source oras.Target, desc ocispec.Descriptor, tag string) (ocispec.Descriptor, error) {

	imageIndex, err := oraswrapper.FetchImageIndex(ctx, source, desc)
	if err != nil {
		return ocispec.Descriptor{}, errors.New(fmt.Sprintf("[FetchImageIndex]: %v", err))
	}

	if version, ok := imageIndex.Annotations[ocispec.AnnotationVersion]; ok {

		switch version {
		case config.PackageVersionV1_0:
			return desc, nil

		default:
			return ocispec.Descriptor{}, errors.New(fmt.Sprintf("Unexpected package version: '%s'", version))
		}
	}

	return ocispec.Descriptor{}, errors.New(fmt.Sprintf("Missing annotation version in descriptor: %v", desc))

}

func (m *orasAddonMigrateRepository) fetchAndDecompressUcManifestTarget(ctx context.Context, fetcher content.Fetcher, ucManifestTarget ocispec.Descriptor) (io.ReadCloser, error) {
	layerReader, err := fetcher.Fetch(ctx, ucManifestTarget)
	if err != nil {
		return nil, err
	}

	return m.decompressor.Decompress(layerReader)

}
