// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager

import (
	"context"
	"fmt"
	"io"
	"u-control/uc-aom/internal/aop/registry"

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
