// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"math"
	"u-control/uc-aom/internal/pkg/config"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/errdef"
)

// Return true if mediatype is a uc image media type, otherwise false
func IsUcImageLayerMediaType(mediaType string) bool {
	return mediaType == config.UcImageLayerMediaType
}

func hasUcImageLayer(imageManifest *ocispec.Manifest) bool {
	for _, layer := range imageManifest.Layers {
		if IsUcImageLayerMediaType(layer.MediaType) {
			return true
		}
	}
	return false
}

func getUcImageLayer(imageManifest *ocispec.Manifest) (ocispec.Descriptor, error) {
	for _, layer := range imageManifest.Layers {
		if IsUcImageLayerMediaType(layer.MediaType) {
			return layer, nil
		}
	}
	return ocispec.Descriptor{}, errdef.ErrNotFound
}

func cumulativeLayerSize(repository Repository, layers []ocispec.Descriptor) uint64 {

	if repositoryWithMapDescriptorSize, hasMapFunc := repository.(DescriptorSizeMapper); hasMapFunc {
		return cumulativeLayerSizeWithMapDescriptorSize(repositoryWithMapDescriptorSize, layers)
	}

	cumulativeLayerSize := uint64(0)
	for _, desc := range layers {
		cumulativeLayerSize += uint64(desc.Size)
	}
	return cumulativeLayerSize
}

func cumulativeLayerSizeWithMapDescriptorSize(repository DescriptorSizeMapper, layers []ocispec.Descriptor) uint64 {
	cumulativeLayerSize := uint64(0)
	for _, desc := range layers {
		descSize := repository.MapDescriptorSize(desc)
		cumulativeLayerSize += uint64(descSize)
	}
	return cumulativeLayerSize
}

func estimatedInstallSizeBytes(cumulativeLayerSize uint64) uint64 {
	// Empirical data shows x2.5 increase relative to the cumulative layer size.
	estimated := 2.5 * float64(cumulativeLayerSize)
	return uint64(math.Ceil(estimated))
}
