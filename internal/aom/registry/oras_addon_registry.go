// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"errors"
	"fmt"
	"time"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/aom/utils"
	oraswrapper "u-control/uc-aom/internal/pkg/oras-wrapper"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/content"
	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Callback function that returns the repository by the given name
type GetRepositoryFn func(ctx context.Context, registry registry.Registry, name string) (Repository, error)

type ORASAddOnRegistry struct {
	registry              registry.Registry
	getRepositoryCallback GetRepositoryFn
	architecture          string
	os                    string
}

// Function that wraps the GetRepository function so that migration can be executed if required
func GetRepositoryWithMigration() GetRepositoryFn {

	registryFascade := make(map[string]Repository, 0)

	return func(ctx context.Context, registry registry.Registry, name string) (Repository, error) {

		if target, ok := registryFascade[name]; ok {
			return target, nil
		}

		source, err := registry.Repository(ctx, name)
		if err != nil {
			return nil, err
		}

		decompressor := &manifest.ManifestTarGzipDecompressor{}
		migrateRepository := NewOrasAddonMigrateRepository(source, decompressor)
		registryFascade[name] = migrateRepository
		return migrateRepository, nil
	}
}

func NewORASAddOnRegistry(registry registry.Registry, localfs *manifest.LocalFSRepository, architecture string, os string) *ORASAddOnRegistry {
	getRepositoryWithMigrationFn := GetRepositoryWithMigration()
	return &ORASAddOnRegistry{registry: registry, getRepositoryCallback: getRepositoryWithMigrationFn, architecture: architecture, os: os}
}

// Returns all repositories known to this ORAS registry.
func (r *ORASAddOnRegistry) Repositories() ([]string, error) {
	delay := time.Duration(10) * time.Second
	res, err := utils.Retry(5, delay, func() (interface{}, error) {
		ctx := context.Background()
		names, err := registry.Repositories(ctx, r.registry)
		return names, err
	})
	if err != nil {
		log.Error("Unable to list repositories: ", err)
		return make([]string, 0), err
	}
	repositories, ok := res.([]string)
	if !ok {
		return make([]string, 0), fmt.Errorf("Could not cast %T into string", res)
	}
	return repositories, nil
}

// Returns all tags of the given repository or an error should it fail or
// the tuple repository and tag not represent an AddOn.
func (r *ORASAddOnRegistry) Tags(repository string) ([]string, error) {
	ctx := context.Background()
	repo, err := r.getRepositoryCallback(ctx, r.registry, repository)
	if err != nil {
		return nil, err
	}

	tags, err := r.getTagsFromRemoteRepository(ctx, repository)
	if err != nil {
		return nil, err
	}

	return r.onlyAddOnTags(context.Background(), repo, tags)
}

// Downloads the artifact from the Registry identified by repository and tag,
// calls the action on any image manifest layers that pass the predicate.
func (r *ORASAddOnRegistry) Pull(repository string, tag string, processor ImageManifestLayerProcessor) (uint64, error) {
	ctx := context.Background()
	repo, err := r.getRepositoryCallback(ctx, r.registry, repository)
	if err != nil {
		log.Error("Unable to read repository: ", err)
		return 0, err
	}

	imageManifest, err := r.deserializeImageManifest(ctx, repo, tag)
	if err != nil {
		return 0, err
	}

	for _, item := range imageManifest.Layers {
		if !processor.Filter(&item) {
			continue
		}

		reader, err := repo.Fetch(ctx, item)
		if err != nil {
			log.Error("Store.Fetch() error =", err)
			continue
		}
		processor.Action(reader, item.MediaType)
	}

	cumulativeLayerSize := cumulativeLayerSize(repo, imageManifest.Layers)

	return estimatedInstallSizeBytes(cumulativeLayerSize), err
}

func (r *ORASAddOnRegistry) Delete(repository string, tag string) error {
	panic("Not implemented")
}

// Initialize an ORAS registry instance given the credentials.
func InitializeRegistry(credentials *Credentials) (registry.Registry, error) {
	reg, err := remote.NewRegistry(credentials.ServerAddress)
	if err != nil {
		return nil, err
	}

	var creds = auth.Credential{Username: credentials.Username, Password: credentials.Password}
	reg.RepositoryOptions.Client = &auth.Client{
		Credential: func(c context.Context, s string) (auth.Credential, error) {
			return creds, nil
		},
	}
	reg.PlainHTTP = credentials.IsInsecureServer()
	return reg, nil
}

func (r *ORASAddOnRegistry) deserializeImageManifest(ctx context.Context, repository Repository, tag string) (*ocispec.Manifest, error) {
	log.Debug("repository.Resolve()")
	desc, err := repository.Resolve(ctx, tag)
	if err != nil {
		log.Error("repository.Resolve() error =", err)
		return nil, err
	}

	switch desc.MediaType {
	case ocispec.MediaTypeImageIndex:
		imageIndex, err := oraswrapper.FetchImageIndex(ctx, repository, desc)
		if err != nil {
			return nil, err
		}

		return r.findSupportedManifest(repository, ctx, imageIndex.Manifests...)

	default:
		return nil, errors.New(fmt.Sprintf("Unexpected mediatype: '%s'", desc.MediaType))
	}

}

func (r *ORASAddOnRegistry) findSupportedManifest(fetcher content.Fetcher, ctx context.Context, manifestDescriptors ...ocispec.Descriptor) (*ocispec.Manifest, error) {
	for _, manifestDescriptor := range manifestDescriptors {
		platform := manifestDescriptor.Platform
		if platform.OS == r.os && platform.Architecture == r.architecture {
			return oraswrapper.FetchImageManifest(ctx, fetcher, manifestDescriptor)
		}
	}

	return nil, errors.New(fmt.Sprintf("Support architecture %s and OS %s is not included!", r.architecture, r.os))
}

func (r *ORASAddOnRegistry) onlyAddOnTags(ctx context.Context, target Repository, tags []string) ([]string, error) {

	filtered := make([]string, 0, len(tags))

	for i := range tags {
		ok, err := r.validateIsAddOn(ctx, target, tags[i])
		if err != nil {
			log.Error("Unable to read repository: ", err)
			continue
		}
		if ok {
			filtered = append(filtered, tags[i])
		}
	}

	if len(filtered) == 0 {
		return nil, errors.New("Invalid AddOn")
	}

	return filtered, nil
}

func (r *ORASAddOnRegistry) validateIsAddOn(ctx context.Context, repository Repository, tag string) (bool, error) {
	imageManifest, err := r.deserializeImageManifest(ctx, repository, tag)
	if err != nil {
		return false, err
	}

	hasUcImageLayer := hasUcImageLayer(imageManifest)
	return hasUcImageLayer, nil
}

func (r *ORASAddOnRegistry) getTagsFromRemoteRepository(ctx context.Context, name string) ([]string, error) {
	repository, err := r.registry.Repository(ctx, name)
	if err != nil {
		return []string{}, err
	}
	return registry.Tags(ctx, repository)
}
