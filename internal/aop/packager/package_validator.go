// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"u-control/uc-aom/internal/aop/company"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/aop/service"
	"u-control/uc-aom/internal/pkg/config"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

// Responsible for validate an add-on package.
type PackageValidator struct {
	resolver          *packageResolver
	addOnPackageIndex *addOnOCIPackageIndex
}

type AnnotationPair struct {
	Key   string
	Value string
}

// Create a new instance of the packager validator
func NewPackageValidator(ctx context.Context, addOnTarget registry.AddOnTarget) (*PackageValidator, error) {
	resolver := newPackageResolver(ctx, addOnTarget)
	addOnPackageIndex, err := resolver.resolve()
	if err != nil {
		return nil, err
	}

	return newPackageValidator(resolver, addOnPackageIndex), nil
}

func newPackageValidator(resolver *packageResolver, addOnPackageIndex *addOnOCIPackageIndex) *PackageValidator {
	return &PackageValidator{resolver, addOnPackageIndex}
}

// validate the meta information (MediaTypes and Annotations) of the add-on target
// addOnTarget will be validated
func (v *PackageValidator) ValidateMetaInfosOnlyOf() error {
	err := v.validateOciImageManifest()
	if err != nil {
		return err
	}

	err = v.validateConfig()
	if err != nil {
		return err
	}

	err = v.validateMetaInfoOfLayers()
	if err != nil {
		return err
	}

	return nil
}

func (v *PackageValidator) ValidateLayerDescriptorAnnotations(expectedAnnotations ...[]*AnnotationPair) error {
	for _, index := range v.addOnPackageIndex.addOnOCIPackage {
		for layerIndex, layerDescriptor := range index.OciImageManifest.Layers {
			err := validateAnnotations(layerDescriptor.Annotations,
				expectedAnnotations[layerIndex]...,
			)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (v *PackageValidator) ValidateLayerContent(compareFn func(content io.ReadCloser, descriptor ocispec.Descriptor, index int) bool) error {
	for _, index := range v.addOnPackageIndex.addOnOCIPackage {
		for layerIndex, layerDescriptor := range index.OciImageManifest.Layers {
			content, err := v.resolver.fetchLayerContent(layerDescriptor)
			if err != nil {
				return fmt.Errorf("Could not fetch content of layer %d", layerIndex)
			}

			equal := compareFn(content, layerDescriptor, layerIndex)
			if !equal {
				return fmt.Errorf("Content of layer %d is not equal", layerIndex)
			}
		}
	}

	return nil

}

func (v *PackageValidator) validateOciImageManifest() error {
	for _, index := range v.addOnPackageIndex.addOnOCIPackage {
		if ocispec.MediaTypeImageManifest != index.OciImageManifestDescriptor.MediaType {
			return errors.New("MediaType mismatch")
		}

		if index.OciImageManifest.SchemaVersion != 2 {
			return fmt.Errorf("Wrong manifestJson SchemaVersion actual %d, 2", index.OciImageManifest.SchemaVersion)
		}
	}

	return nil
}
func (v *PackageValidator) validateConfig() error {
	for _, index := range v.addOnPackageIndex.addOnOCIPackage {
		if index.OciImageManifest.Config.MediaType != config.UcConfigMediaType {
			return fmt.Errorf("[MediaTypeMismatch] Unexpected config media type = %s", index.OciImageManifest.Config.MediaType)
		}

		err := validateAnnotations(index.OciImageManifest.Config.Annotations, &AnnotationPair{ocispec.AnnotationTitle, config.UcConfigAnnotationTitle})
		if err != nil {
			return err
		}

		if !strings.Contains(index.OciConfiguration.Author, company.AuthorName) {
			return fmt.Errorf("Author mismatch. Expected '%s', Actual '%s'", company.AuthorName, index.OciConfiguration.Author)
		}
		if !strings.Contains(index.OciConfiguration.OS, service.OS()) {
			return fmt.Errorf("OS mismatch. Expected '%s', Actual '%s'", service.OS(), index.OciConfiguration.OS)
		}
	}

	return nil
}

func validateAnnotations(actualAnnotations map[string]string, expectedAnnotationPairs ...*AnnotationPair) error {
	for _, pair := range expectedAnnotationPairs {
		if actualValue, ok := actualAnnotations[pair.Key]; ok {
			if actualValue != pair.Value {
				return fmt.Errorf("Unexpected annotation value: want '%s', but got '%s' for key '%s'", pair.Value, actualValue, pair.Key)
			}
		} else {
			return fmt.Errorf("Annotation key '%s' not Found", pair.Key)
		}
	}

	return nil
}

func (v *PackageValidator) validateMetaInfoOfLayers() error {
	for _, index := range v.addOnPackageIndex.addOnOCIPackage {
		if len(index.OciImageManifest.Layers) != 2 {
			return fmt.Errorf("Layers Count mismatch. Expected 2, Actual %d", len(index.OciImageManifest.Layers))
		}

		addOnManifestLayer, err := index.GetAddOnManifestDescriptor()
		if err != nil {
			return err
		}

		if _, ok := addOnManifestLayer.Annotations[ocispec.AnnotationTitle]; !ok {
			return fmt.Errorf("Manifest descriptor does not include an annotations for %s", ocispec.AnnotationTitle)
		}
	}
	return v.validateDockerLayers()
}

func (v *PackageValidator) validateDockerLayers() error {
	for _, index := range v.addOnPackageIndex.addOnOCIPackage {
		dockerLayers := index.GetAllDockerImageDescriptors()
		if len(dockerLayers) == 0 {
			return fmt.Errorf("No docker images in add-on")
		}
	}

	return nil
}
