// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"io"
	"u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/pkg/manifest"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Called on a given manifest layer represented by src
type ImageManifestLayerAction func(src io.Reader, mediaType string)

// Processes each layer of an image manifest.
type ImageManifestLayerProcessor interface {
	// Called on a given manifest layer represented by src, which has passed the Filter predicate.
	Action(src io.Reader, mediaType string)

	// Determins if the manifest layer should be processed by Action.
	Filter(desc *ocispec.Descriptor) bool
}

// Used to only process uc manifest layers
type ucImageLayerProcessor struct {
	action ImageManifestLayerAction
}

func NewUcImageLayerProcessor(action ImageManifestLayerAction) *ucImageLayerProcessor {
	return &ucImageLayerProcessor{action: action}
}

func (p *ucImageLayerProcessor) Action(src io.Reader, mediaType string) {
	p.action(src, mediaType)
}

func (p *ucImageLayerProcessor) Filter(desc *ocispec.Descriptor) bool {
	if !IsUcImageLayerMediaType(desc.MediaType) {
		return false
	}

	if value, ok := desc.Annotations[config.UcImageLayerAnnotationSchemaVersion]; ok {
		return value == manifest.ValidManifestVersion
	}

	return false
}

// Used to process all manifest layers expect the uc layers
type allExceptUcImageLayerProcessor struct {
	ucImageLayerProcessor
}

func NewAllExceptUcImageLayerProcessor(action ImageManifestLayerAction) *allExceptUcImageLayerProcessor {
	return &allExceptUcImageLayerProcessor{ucImageLayerProcessor{action: action}}
}

func (p *allExceptUcImageLayerProcessor) Filter(desc *ocispec.Descriptor) bool {
	return !p.ucImageLayerProcessor.Filter(desc)
}

// Used to process all manifest layers
type acceptAllManifestLayerProcessor struct {
	ucImageLayerProcessor
}

func NewAcceptAllManifestLayerProcessor(action ImageManifestLayerAction) *acceptAllManifestLayerProcessor {
	return &acceptAllManifestLayerProcessor{ucImageLayerProcessor{action: action}}
}

func (p *acceptAllManifestLayerProcessor) Filter(desc *ocispec.Descriptor) bool {
	return true
}

// Used to process none manifest layer
type acceptNoneManifestLayerProcessor struct {
	ucImageLayerProcessor
}

func NewAcceptNoneManifestLayerProcessor(action ImageManifestLayerAction) *acceptNoneManifestLayerProcessor {
	return &acceptNoneManifestLayerProcessor{ucImageLayerProcessor{action: action}}
}

func (p *acceptNoneManifestLayerProcessor) Filter(desc *ocispec.Descriptor) bool {
	return false
}
