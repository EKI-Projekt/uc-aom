// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager

import (
	"context"
	"errors"
	"io"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/pkg/config"
	oraswrapper "u-control/uc-aom/internal/pkg/oras-wrapper"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type addOnOCIPackageIndex struct {
	OciImageIndexDescriptor ocispec.Descriptor
	OciImageIndex           *ocispec.Index
	addOnOCIPackage         []*addOnOCIPackage
}

type addOnOCIPackage struct {
	OciImageManifestDescriptor ocispec.Descriptor
	OciImageManifest           *ocispec.Manifest
	OciConfiguration           *ocispec.Image
	OciImageManifestLayersMap  map[string][]ocispec.Descriptor
}

func (p *addOnOCIPackage) GetAllDockerImageDescriptors() []ocispec.Descriptor {
	return p.OciImageManifestLayersMap[ocispec.MediaTypeImageLayer]
}

func (p *addOnOCIPackage) GetAllAddOnManifestDescriptors() []ocispec.Descriptor {
	return p.OciImageManifestLayersMap[config.UcImageLayerMediaType]
}

func (p *addOnOCIPackage) GetAddOnManifestDescriptor() (ocispec.Descriptor, error) {

	addOnManifestDescriptors := p.GetAllAddOnManifestDescriptors()

	if len(addOnManifestDescriptors) > 0 {
		return addOnManifestDescriptors[0], nil
	}
	return ocispec.Descriptor{}, errors.New("Add-on manifest layer not Found")
}

// Responsible for validate an add-on package.
type packageResolver struct {
	ctx         context.Context
	addOnTarget registry.AddOnTarget
}

// Create a new instance of the packager validator
func newPackageResolver(ctx context.Context, addOnTarget registry.AddOnTarget) *packageResolver {
	return &packageResolver{ctx, addOnTarget}
}

func (r *packageResolver) resolve() (*addOnOCIPackageIndex, error) {
	ociPackageIndex := addOnOCIPackageIndex{}
	ociPackageIndex.addOnOCIPackage = make([]*addOnOCIPackage, 0, 2)

	addOnOciImageIndex, err := r.resolveAddOnOciImageIndex()
	if err != nil {
		return &addOnOCIPackageIndex{}, err
	}
	ociPackageIndex.OciImageIndexDescriptor = addOnOciImageIndex

	ociImageIndex, err := r.fetchAddOnOciImageIndex(ociPackageIndex.OciImageIndexDescriptor)
	if err != nil {
		return &addOnOCIPackageIndex{}, err
	}
	ociPackageIndex.OciImageIndex = ociImageIndex

	for _, manifest := range ociPackageIndex.OciImageIndex.Manifests {
		ociPackage := addOnOCIPackage{}
		ociPackage.OciImageManifestDescriptor = manifest
		ociImageManifest, err := r.fetchAddOnOciImageManifest(manifest)
		if err != nil {
			continue
		}

		ociPackage.OciImageManifest = ociImageManifest
		ociConfiguration, err := r.fetchAddOnOciConfiguration(ociPackage.OciImageManifest.Config)
		if err != nil {
			continue
		}

		ociPackage.OciConfiguration = ociConfiguration
		ociPackage.OciImageManifestLayersMap = r.createLayersMediaTypeMap(ociPackage.OciImageManifest)

		ociPackageIndex.addOnOCIPackage = append(ociPackageIndex.addOnOCIPackage, &ociPackage)
	}

	return &ociPackageIndex, nil
}

func (r *packageResolver) fetchLayerContent(layerDescriptor ocispec.Descriptor) (io.ReadCloser, error) {
	return r.addOnTarget.Fetch(r.ctx, layerDescriptor)
}

func (r *packageResolver) resolveAddOnOciImageIndex() (ocispec.Descriptor, error) {
	addOnOciImageIndex, err := r.addOnTarget.Resolve(r.ctx, r.addOnTarget.AddOnVersion())
	if err != nil {
		return ocispec.Descriptor{}, err
	}
	return addOnOciImageIndex, nil
}

func (r *packageResolver) fetchAddOnOciImageIndex(addOnOciImageIndexDescriptor ocispec.Descriptor) (*ocispec.Index, error) {
	indexJson, err := oraswrapper.FetchImageIndex(r.ctx, r.addOnTarget, addOnOciImageIndexDescriptor)
	return indexJson, err
}

func (r *packageResolver) fetchAddOnOciImageManifest(addOnOciImageManifestDescriptor ocispec.Descriptor) (*ocispec.Manifest, error) {
	manifestJson, err := oraswrapper.FetchImageManifest(r.ctx, r.addOnTarget, addOnOciImageManifestDescriptor)
	return manifestJson, err
}

func (r *packageResolver) fetchAddOnOciConfiguration(addOnOciConfigurationDescriptor ocispec.Descriptor) (*ocispec.Image, error) {
	configContent, err := oraswrapper.FetchImageConfig(r.ctx, r.addOnTarget, addOnOciConfigurationDescriptor)
	return configContent, err
}

func (r *packageResolver) createLayersMediaTypeMap(ociImageManifest *ocispec.Manifest) map[string][]ocispec.Descriptor {

	layersMap := make(map[string][]ocispec.Descriptor, 0)

	for _, layer := range ociImageManifest.Layers {
		descs := layersMap[layer.MediaType]
		if descs == nil {
			descs = make([]ocispec.Descriptor, 0)
		}

		layersMap[layer.MediaType] = append(descs, layer)
	}

	return layersMap
}
