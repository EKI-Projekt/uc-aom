// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager

import (
	"bytes"
	"context"
	"encoding/json"
	"time"
	"u-control/uc-aom/internal/aop/company"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/pkg/config"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/content/memory"
)

type indexOrasGraphBuilder struct {
	manifestLogo *descriptorBlobTuple
	dockerImages map[*ocispec.Platform][]*descriptorBlobTuple
	tag          string
}

type descriptorBlobTuple struct {
	desc *ocispec.Descriptor
	blob []byte
}

// Creates a new instance of OrasGraphBuilder
func NewIndexOrasGraphBuilder(tag string) *indexOrasGraphBuilder {
	builder := &indexOrasGraphBuilder{
		dockerImages: make(map[*ocispec.Platform][]*descriptorBlobTuple, 0),
		tag:          tag,
	}

	return builder
}

func (r *indexOrasGraphBuilder) AppendManifestAndLogo(mediaType string, blob []byte, annotations ...string) {
	descriptor := createDescriptorFromBlob(mediaType, blob, annotations...)
	r.manifestLogo = &descriptorBlobTuple{desc: descriptor, blob: blob}
}

func (r *indexOrasGraphBuilder) AppendDockerImage(mediaType string, blob []byte, platform *ocispec.Platform, annotations ...string) {
	descriptor := createDescriptorFromBlob(mediaType, blob, annotations...)
	dockerImageTuple := &descriptorBlobTuple{desc: descriptor, blob: blob}

	currentLayerTuples, ok := r.dockerImages[platform]
	if !ok {
		initial := make([]*descriptorBlobTuple, 0)
		currentLayerTuples = initial
	}
	currentLayerTuples = append(currentLayerTuples, dockerImageTuple)
	r.dockerImages[platform] = currentLayerTuples
}

// Create an AddOnTarget based on the elements which have been given by `Append`
func (r *indexOrasGraphBuilder) BuildAndTagOrasTarget(ctx context.Context) (registry.AddOnTarget, error) {
	src := memory.New()

	err := push(ctx, src, r.manifestLogo)
	if err != nil {
		return nil, err
	}

	imageManifestTuples := make([]*descriptorBlobTuple, 0, len(r.dockerImages))
	for platform, blobs := range r.dockerImages {
		imageConfigTuple, imageManifestTuple, err := createImageConfigTupleAndImageManifestTuple(platform, r.manifestLogo, blobs)
		if err != nil {
			return nil, err
		}
		imageManifestTuple.desc.Platform = platform

		imageManifestTuples = append(imageManifestTuples, imageManifestTuple)

		err = push(ctx, src, blobs...)
		if err != nil {
			return nil, err
		}

		push(ctx, src, imageConfigTuple)
		if err != nil {
			return nil, err
		}

		push(ctx, src, imageManifestTuple)
		if err != nil {
			return nil, err
		}
	}

	imageManifestIndexTuple, err := createManifestIndexTuple(imageManifestTuples)
	if err != nil {
		return nil, err
	}

	push(ctx, src, imageManifestIndexTuple)
	if err != nil {
		return nil, err
	}

	log.Debugf("src.Tag root: %v", *imageManifestIndexTuple.desc)
	err = src.Tag(ctx, *imageManifestIndexTuple.desc, r.tag)
	if err != nil {
		return nil, err
	}

	return registry.NewOciTargetDecorator(src, r.tag), nil
}

func createImageConfigTupleAndImageManifestTuple(platform *ocispec.Platform, manifestLogo *descriptorBlobTuple, dockerImages []*descriptorBlobTuple) (*descriptorBlobTuple, *descriptorBlobTuple, error) {
	imageConfigBlob, err := createImageConfigTuple(platform)
	if err != nil {
		return nil, nil, err
	}

	desc := make([]ocispec.Descriptor, 0, len(dockerImages)+1)
	desc = append(desc, *manifestLogo.desc)
	for _, image := range dockerImages {
		desc = append(desc, *image.desc)
	}

	imageManifestBlob, err := createImageManifestTuple(*imageConfigBlob.desc, desc)
	if err != nil {
		return imageConfigBlob, nil, err
	}

	return imageConfigBlob, imageManifestBlob, nil
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

func createManifestIndexTuple(imageManifests []*descriptorBlobTuple) (*descriptorBlobTuple, error) {
	// https://github.com/opencontainers/image-spec/blob/main/media-types.md

	// Image Index -> Image Manifest -> [Image Config, Layer_1, Layer_2, ..., Layer_n]
	// Layer_1               - manifest and logo.
	// Layer_2, ..., Layer_n - docker images

	desc := make([]ocispec.Descriptor, 0, len(imageManifests))
	for _, im := range imageManifests {
		desc = append(desc, *im.desc)
	}

	index := ocispec.Index{
		// Historical value, does not pertain to OCI or docker version
		Versioned: specs.Versioned{SchemaVersion: 2},
		Manifests: desc,
	}

	indexJSON, err := json.Marshal(index)
	if err != nil {
		return nil, err
	}

	descriptor := createDescriptorFromBlob(ocispec.MediaTypeImageIndex, indexJSON)
	return &descriptorBlobTuple{desc: descriptor, blob: indexJSON}, nil
}

func createImageManifestTuple(config ocispec.Descriptor, layers []ocispec.Descriptor) (*descriptorBlobTuple, error) {
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

	descriptor := createDescriptorFromBlob(ocispec.MediaTypeImageManifest, manifestJSON)
	return &descriptorBlobTuple{desc: descriptor, blob: manifestJSON}, nil
}

func createImageConfigTuple(platform *ocispec.Platform) (*descriptorBlobTuple, error) {
	now := time.Now().UTC()
	imageConfig := ocispec.Image{
		Architecture: platform.Architecture,
		Author:       company.ShortAuthorInfo(),
		Created:      &now,
		OS:           platform.OS,
	}
	imageConfigJSON, err := json.Marshal(imageConfig)
	if err != nil {
		return nil, err
	}

	descriptor := createDescriptorFromBlob(config.UcConfigMediaType, imageConfigJSON, ocispec.AnnotationTitle, config.UcConfigAnnotationTitle)
	return &descriptorBlobTuple{desc: descriptor, blob: imageConfigJSON}, nil
}

func push(ctx context.Context, store *memory.Store, tuples ...*descriptorBlobTuple) error {
	log.Tracef("push")
	for _, tuple := range tuples {
		log.Debugf("pushing to in-memory store: %v", tuple.desc)
		err := store.Push(ctx, *tuple.desc, bytes.NewReader(tuple.blob))
		if err != nil {
			return err
		}
	}
	return nil
}
