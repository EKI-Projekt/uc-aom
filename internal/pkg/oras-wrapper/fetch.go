// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package oraswrapper

import (
	"context"
	"encoding/json"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content"
)

// Fetch the ocispec Index parsed from the JSON-encoded data
func FetchImageIndex(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) (*ocispec.Index, error) {
	data, err := content.FetchAll(ctx, fetcher, desc)
	if err != nil {
		return nil, err
	}
	var imageIndex ocispec.Index
	err = json.Unmarshal(data, &imageIndex)
	return &imageIndex, err
}

// Returns the ocispec Manifest parsed from the JSON-encoded
func FetchImageManifest(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) (*ocispec.Manifest, error) {
	data, err := content.FetchAll(ctx, fetcher, desc)
	if err != nil {
		return nil, err
	}
	var imageManifest ocispec.Manifest
	err = json.Unmarshal(data, &imageManifest)
	return &imageManifest, err
}

// Returns the oci ImageConfig parsed from the JSON-encoded
func FetchImageConfig(ctx context.Context, fetcher content.Fetcher, desc ocispec.Descriptor) (*ocispec.Image, error) {
	data, err := content.FetchAll(ctx, fetcher, desc)
	if err != nil {
		return nil, err
	}
	var imageConfig ocispec.Image
	err = json.Unmarshal(data, &imageConfig)
	return &imageConfig, err
}
