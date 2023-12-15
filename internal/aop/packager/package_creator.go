// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"u-control/uc-aom/internal/aop/company"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/aop/manifest"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/pkg/config"
	sharedManifest "u-control/uc-aom/internal/pkg/manifest"
	oraswrapper "u-control/uc-aom/internal/pkg/oras-wrapper"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"

	orasRegistry "oras.land/oras-go/v2/registry"
)

var (
	SingleArchImageError = errors.New("A multi-platform image build is required")
)

// Responsible for creating an add-on package that can be hosted by an OCI capable registry.
type PackageCreator struct {
	exportDockerImageFunc registry.ExportDockerImageFunc
	gzipTarballFunc       fileio.GzipTarballFunc
	logoBaseName          string
}

// Create a new instance of the packager
// gzipTarballFunc will be used to tar and gzip the directory containing manifest.json
// exportDockerImageFunc will be used to export any referenced docker images in manifest.json
func NewPackageCreator(gzipTarballFunc fileio.GzipTarballFunc, exportDockerImageFunc registry.ExportDockerImageFunc) *PackageCreator {
	return &PackageCreator{exportDockerImageFunc: exportDockerImageFunc, gzipTarballFunc: gzipTarballFunc}
}

// Create a rooted directed acyclic graph (DAG) with the root node tagged with tag,
// which can be subsequently pushed to an OCI registry.
func (r *PackageCreator) CreateOciArtifact(ctx context.Context, manifest *manifest.AddOnManifest, reg orasRegistry.Registry) (registry.AddOnTarget, error) {
	builder := oraswrapper.NewGraphBuilder(manifest.Version)
	builder.WithAuthor(company.ShortAuthorInfo())

	r.logoBaseName = manifest.Logo
	ucImageLayerAnnotations := sharedManifest.CreateUcManifestAnnotationsV1_0(manifest.Version, manifest.ManifestVersion)
	err := r.appendManifestAndLogo(builder, manifest.ManifestBaseDirectory(), ucImageLayerAnnotations)
	if err != nil {
		return nil, err
	}

	refs := sharedManifest.GetDockerImageReferences(manifest.Services)
	err = r.appendDockerImages(builder, ctx, reg, refs)
	if err != nil {
		return nil, err
	}

	if orasTarget, err := builder.BuildAndTag(ctx); err == nil {
		return registry.NewOciTargetDecorator(orasTarget, manifest.Version), nil
	}

	return nil, err
}

func (r *PackageCreator) predicate(path string) bool {
	base := filepath.Base(path)
	switch base {
	case config.UcImageManifestFilename, r.logoBaseName:
		return true
	default:
		return false
	}
}

func (r *PackageCreator) appendManifestAndLogo(builder *oraswrapper.GraphBuilder, contentDirPath string, annotations []string) error {
	tarGzipBytes, err := r.gzipTarballFunc(contentDirPath, r.predicate)
	if err != nil {
		return err
	}
	builder.AppendUcManifest(tarGzipBytes, annotations...)
	return nil
}

func (r *PackageCreator) appendDockerImages(builder *oraswrapper.GraphBuilder, ctx context.Context, reg orasRegistry.Registry, dockerImageRefs []string) error {
	for _, ref := range dockerImageRefs {
		log.Debugf("export docker image at '%s'", ref)
		platforms, err := getSupportedPlatforms(ctx, reg, ref)
		if err != nil {
			return err
		}

		if len(platforms) == 0 {
			return SingleArchImageError
		}

		for _, platform := range platforms {
			ociTarballBytes, err := r.exportDockerImageFunc(reg, ref, platform)
			if err != nil || ociTarballBytes == nil {
				return err
			}

			builder.AppendDockerImage(ociTarballBytes, platform, ocispec.AnnotationTitle, ref)
		}
	}

	return nil
}

func getSupportedPlatforms(ctx context.Context, reg orasRegistry.Registry, ref string) ([]*ocispec.Platform, error) {
	repository, tag, err := registry.ToRepositoryAndTag(ref)
	if err != nil {
		return nil, err
	}

	ociRepository, err := reg.Repository(ctx, repository)
	if err != nil {
		return nil, fmt.Errorf("[Repository] Unexpected error = %w", err)
	}

	desc, err := ociRepository.Resolve(ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("[Resolve] Unexpected error = %w", err)
	}

	rc, err := ociRepository.Fetch(ctx, desc)
	if err != nil {
		return nil, fmt.Errorf("[Fetch] Unexpected error = %w", err)
	}

	content, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("[ReadAll] Unexpected error = %w", err)
	}

	var index ocispec.Index
	err = json.Unmarshal(content, &index)
	if err != nil {
		return nil, fmt.Errorf("[Unmarshal] Unexpected error = %w", err)
	}

	platforms := make([]*ocispec.Platform, 0, len(index.Manifests))
	for _, manifest := range index.Manifests {
		// Docker Buildkit supports creating and attaching attestations to build artifacts.
		// At the moment, the packager doesn't package these attestations to the addon artifact.
		if isAttestationManifest(manifest) {
			continue
		}
		platforms = append(platforms, manifest.Platform)
	}

	return platforms, nil
}

func isAttestationManifest(manifest ocispec.Descriptor) bool {
	// The attestation manifest can be determined by the platform property
	// See for details: https://github.com/moby/buildkit/blob/v0.11.0/docs/attestations/attestation-storage.md
	attestationPlatform := "unknown"
	return strings.Contains(manifest.Platform.Architecture, attestationPlatform) && strings.Contains(manifest.Platform.OS, attestationPlatform)
}
