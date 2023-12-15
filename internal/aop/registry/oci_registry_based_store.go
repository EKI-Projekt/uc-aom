// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"context"

	"github.com/containerd/containerd/images"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/registry"
)

type ociRegistryBasedStore struct {
	reg registry.Registry
}

func (r ociRegistryBasedStore) Get(ctx context.Context, name string) (images.Image, error) {
	repository, tag, err := ToRepositoryAndTag(name)
	if err != nil {
		return images.Image{}, err
	}

	ociRepository, err := r.reg.Repository(ctx, repository)
	if err != nil {
		log.Fatalf("Could not initialize repository: %v", err)
		return images.Image{}, err
	}

	manifest, err := ociRepository.Resolve(ctx, tag)
	if err != nil {
		return images.Image{}, err
	}

	image := images.Image{
		Name:   name,
		Target: manifest,
	}

	return image, nil
}

func (r ociRegistryBasedStore) List(ctx context.Context, filters ...string) ([]images.Image, error) {
	panic("[List] not implemented")
}

func (r ociRegistryBasedStore) Create(ctx context.Context, image images.Image) (images.Image, error) {
	panic("[Create] not implemented")
}

// Update will replace the data in the store with the provided image. If
// one or more fieldpaths are provided, only those fields will be updated.
func (r ociRegistryBasedStore) Update(ctx context.Context, image images.Image, fieldpaths ...string) (images.Image, error) {
	panic("[Update] not implemented")
}

func (r ociRegistryBasedStore) Delete(ctx context.Context, name string, opts ...images.DeleteOpt) error {
	panic("[Delete] not implemented")
}
