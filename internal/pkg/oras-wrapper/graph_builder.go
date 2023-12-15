// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package oraswrapper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"u-control/uc-aom/internal/pkg/config"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
)

// Platform represents a string concatenated by the platform os, arch and variant
type PlatformKey string

// FromOCIPlatform creates a new platform key by splitting the os, arch and var fields from
// the ocispec.Platform
func FromOCIPlatform(platform *ocispec.Platform) PlatformKey {
	concatedString := fmt.Sprintf("%s-%s", platform.OS, platform.Architecture)
	if platform.Variant != "" {
		concatedString += fmt.Sprintf("-%s", platform.Variant)
	}
	return PlatformKey(concatedString)
}

// ToOCIPlatform convert the platform key back to the ocispec platform
func (p *PlatformKey) ToOCIPlatform() (*ocispec.Platform, error) {
	pAsString := string(*p)
	fields := strings.Split(pAsString, "-")
	if len(fields) < 2 {
		return nil, errors.New("Invalid arguments")
	}
	os, arch := fields[0], fields[1]
	var variant string = ""
	if len(fields) == 3 {
		variant = fields[2]
	}
	return &ocispec.Platform{
		OS:           os,
		Architecture: arch,
		Variant:      variant,
	}, nil
}

type GraphBuilder struct {
	UcManifestTuple     *DescriptorBlobTuple
	DockerImages        map[PlatformKey][]*DescriptorBlobTuple
	ImageManifestTuples []*DescriptorBlobTuple
	Tag                 string
	author              string
}

// Creates a new instance of GraphBuilder
func NewGraphBuilder(tag string) *GraphBuilder {
	builder := &GraphBuilder{
		DockerImages:        make(map[PlatformKey][]*DescriptorBlobTuple, 0),
		ImageManifestTuples: make([]*DescriptorBlobTuple, 0),
		Tag:                 tag,
		author:              "",
	}
	return builder
}

// Add author information to config image
func (r *GraphBuilder) WithAuthor(author string) {
	r.author = author
}

// Append the uc manifest the internal datastructure
func (r *GraphBuilder) AppendUcManifest(blob []byte, annotations ...string) {
	descriptor := createDescriptorFromBlob(config.UcImageLayerMediaType, blob, annotations...)
	r.UcManifestTuple = &DescriptorBlobTuple{Desc: descriptor, Blob: blob}
}

// Append a platform specific docker imageto the internal datastructure
func (r *GraphBuilder) AppendDockerImage(blob []byte, platform *ocispec.Platform, annotations ...string) {
	descriptor := createDescriptorFromBlob(ocispec.MediaTypeImageLayer, blob, annotations...)
	dockerImageTuple := &DescriptorBlobTuple{Desc: descriptor, Blob: blob}

	platformKey := FromOCIPlatform(platform)
	currentLayerTuples, ok := r.DockerImages[platformKey]
	if !ok {
		initial := make([]*DescriptorBlobTuple, 0)
		currentLayerTuples = initial
	}
	currentLayerTuples = append(currentLayerTuples, dockerImageTuple)
	r.DockerImages[platformKey] = currentLayerTuples
}

// Append an platform specific image manifest to the internal datastructure
// Before calling this function it is necessary to append an uc manifest and corresponding docker images with the same platform
func (r *GraphBuilder) AppendImageManifest(platformKey PlatformKey) (*DescriptorBlobTuple, *DescriptorBlobTuple, error) {
	dockerImageTuples := r.DockerImages[platformKey]
	imageConfigTuple, imageManifestTuple, err := CreateImageConfigTupleAndImageManifestTuple(platformKey, r.author, r.UcManifestTuple, dockerImageTuples)
	if err != nil {
		return nil, nil, err
	}
	r.ImageManifestTuples = append(r.ImageManifestTuples, imageManifestTuple)

	return imageConfigTuple, imageManifestTuple, nil
}

// Create an AddOnTarget based on the elements which have been given by `Append`
func (r *GraphBuilder) BuildAndTag(ctx context.Context) (oras.Target, error) {
	src := memory.New()

	err := PushAll(ctx, src, r.UcManifestTuple)
	if err != nil {
		return nil, err
	}

	for platformKey, dockerImageTuples := range r.DockerImages {
		imageConfigTuple, imageManifestTuple, err := r.AppendImageManifest(platformKey)
		if err != nil {
			return nil, err
		}

		err = PushAll(ctx, src, dockerImageTuples...)
		if err != nil {
			return nil, err
		}

		err = PushAll(ctx, src, imageConfigTuple)
		if err != nil {
			return nil, err
		}

		err = PushAll(ctx, src, imageManifestTuple)
		if err != nil {
			return nil, err
		}
	}

	imageIndexTuple, err := CreateImageIndexTuple(r.ImageManifestTuples)
	if err != nil {
		return nil, err
	}

	PushAll(ctx, src, imageIndexTuple)
	if err != nil {
		return nil, err
	}

	log.Debugf("src.Tag root: %v", *imageIndexTuple.Desc)
	err = src.Tag(ctx, *imageIndexTuple.Desc, r.Tag)
	if err != nil {
		return nil, err
	}

	return src, nil
}

// Helper function to create an image manifest with a config image
func CreateImageConfigTupleAndImageManifestTuple(platformKey PlatformKey, author string, ucManifestTuple *DescriptorBlobTuple, dockerImageTuples []*DescriptorBlobTuple) (*DescriptorBlobTuple, *DescriptorBlobTuple, error) {
	imageConfigTuple, err := CreateImageConfigTuple(platformKey, author)
	if err != nil {
		return nil, nil, err
	}

	lengthOfLayers := len(dockerImageTuples) + 1 // add lengh of one because of the uc manifest layer
	layerDesc := make([]ocispec.Descriptor, 0, lengthOfLayers)

	layerDesc = append(layerDesc, *ucManifestTuple.Desc)
	for _, image := range dockerImageTuples {
		layerDesc = append(layerDesc, *image.Desc)
	}

	imageManifestTuple, err := CreateImageManifestTuple(platformKey, *imageConfigTuple.Desc, layerDesc)
	if err != nil {
		return nil, nil, err
	}
	return imageConfigTuple, imageManifestTuple, nil
}

// Helper function to create an image index
func CreateImageIndexTuple(imageManifests []*DescriptorBlobTuple) (*DescriptorBlobTuple, error) {
	// https://github.com/opencontainers/image-spec/blob/main/media-types.md

	// Image Index -> Image Manifest -> [Image Config, Layer_1, Layer_2, ..., Layer_n]
	// Layer_1               - manifest and logo.
	// Layer_2, ..., Layer_n - docker images

	manifestDesc := make([]ocispec.Descriptor, 0, len(imageManifests))
	for _, im := range imageManifests {
		manifestDesc = append(manifestDesc, *im.Desc)
	}

	annotations := map[string]string{ocispec.AnnotationVersion: config.UcPackageVersion}
	index := ocispec.Index{
		// Historical value, does not pertain to OCI or docker version
		Versioned:   specs.Versioned{SchemaVersion: 2},
		Manifests:   manifestDesc,
		Annotations: annotations,
	}

	indexJSON, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}

	descriptor := createDescriptorFromBlob(ocispec.MediaTypeImageIndex, indexJSON)
	return &DescriptorBlobTuple{Desc: descriptor, Blob: indexJSON}, nil
}

// Helper functino to create an image manifest
func CreateImageManifestTuple(platformKey PlatformKey, config ocispec.Descriptor, layers []ocispec.Descriptor) (*DescriptorBlobTuple, error) {
	manifest := ocispec.Manifest{
		// Historical value, does not pertain to OCI or docker version
		Versioned: specs.Versioned{SchemaVersion: 2},
		Config:    config,
		Layers:    layers,
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	platform, err := platformKey.ToOCIPlatform()
	if err != nil {
		return nil, err
	}
	descriptor := createDescriptorFromBlob(ocispec.MediaTypeImageManifest, manifestJSON)
	descriptor.Platform = platform
	return &DescriptorBlobTuple{Desc: descriptor, Blob: manifestJSON}, nil
}

// Helper function to create an image config
func CreateImageConfigTuple(platformKey PlatformKey, author string) (*DescriptorBlobTuple, error) {
	now := time.Now().UTC()
	platform, err := platformKey.ToOCIPlatform()
	if err != nil {
		return nil, err
	}

	imageConfig := ocispec.Image{
		Architecture: platform.Architecture,
		Author:       author,
		Created:      &now,
		OS:           platform.OS,
	}
	imageConfigJSON, err := json.Marshal(imageConfig)
	if err != nil {
		return nil, err
	}

	descriptor := createDescriptorFromBlob(config.UcConfigMediaType, imageConfigJSON, ocispec.AnnotationTitle, config.UcConfigAnnotationTitle)
	return &DescriptorBlobTuple{Desc: descriptor, Blob: imageConfigJSON}, nil
}

func createDescriptorFromBlob(mediaType string, blob []byte, annotations ...string) *ocispec.Descriptor {
	annotationMap := make(map[string]string, len(annotations)/2)
	for i := 0; i < len(annotations); i += 2 {
		annotationMap[annotations[i]] = annotations[i+1]
	}

	desc := ocispec.Descriptor{
		MediaType:   mediaType,
		Digest:      digest.FromBytes(blob),
		Size:        int64(len(blob)),
		Annotations: annotationMap,
	}

	return &desc
}
