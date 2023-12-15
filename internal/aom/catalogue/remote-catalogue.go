// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package catalogue

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/aom/registry"
	model "u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
)

type ORASRemoteAddOnCatalogue struct {
	// Location where temporary data will be stored.
	Root string

	manifestReader model.ManifestFileReader
	registry       registry.AddOnRegistry
}

// NewORASRemoteAddOnCatalogue creates an instance of ORASRemoteAddOnCatalogue
func NewORASRemoteAddOnCatalogue(root string, registry registry.AddOnRegistry, manifestReader model.ManifestFileReader) *ORASRemoteAddOnCatalogue {
	return &ORASRemoteAddOnCatalogue{Root: root, manifestReader: manifestReader, registry: registry}
}

func (catalogue *ORASRemoteAddOnCatalogue) GetAddOnNames() ([]string, error) {
	log.Trace("RemoteCatalogue.GetAddOnNames()")
	return catalogue.registry.Repositories()
}

func (catalogue *ORASRemoteAddOnCatalogue) GetAddOnVersions(name string) ([]string, error) {
	log.Tracef("RemoteCatalogue.GetAddOnVersions(%s)", name)
	versions, err := catalogue.registry.Tags(name)
	if err != nil {
		return nil, err
	}

	sort.Sort(model.ByAddOnVersion(versions))
	return versions, nil
}

func (catalogue *ORASRemoteAddOnCatalogue) GetAddOn(name string, version string) (CatalogueAddOn, error) {
	log.Tracef("RemoteCatalogue.GetAddOn('%s', '%s')", name, version)
	destination := filepath.Join(catalogue.Root, name)

	err := os.RemoveAll(destination)
	if err != nil {
		return CatalogueAddOn{}, err
	}
	processor := registry.NewUcImageLayerProcessor(catalogue.action(destination))
	_, err = catalogue.registry.Pull(name, version, processor)
	if err != nil {
		return CatalogueAddOn{}, err
	}

	manifest, err := catalogue.manifestReader.ReadManifestFrom(destination)
	if err != nil {
		return CatalogueAddOn{}, err
	}
	return CatalogueAddOn{Name: name, Version: version, Manifest: *manifest}, nil
}

func (catalogue *ORASRemoteAddOnCatalogue) GetLatestAddOns() ([]*CatalogueAddOn, error) {
	log.Trace("RemoteCatalogue.GetLatestAddOns()")
	repositories, err := catalogue.GetAddOnNames()
	if err != nil {
		return nil, NewRemoteRegistryConnectionError(err)
	}

	addOns := make([]*CatalogueAddOn, 0, len(repositories))
	for _, repo := range repositories {
		versions, err := catalogue.GetAddOnVersions(repo)
		if err != nil {
			log.Tracef("Skipping repository '%s': %v", repo, err)
			continue
		}

		log.Tracef("Repository: %s, Tags: [%s]", repo, strings.Join(versions, ", "))
		latest := versions[len(versions)-1]
		catalogueAddOn, err := catalogue.GetAddOn(repo, latest)
		if err != nil {
			log.Warnf("Failed to fetch AddOn from repository '%s': %v", repo, err)
			continue
		}
		addOns = append(addOns, &catalogueAddOn)
	}
	return addOns, nil
}

func (catalogue *ORASRemoteAddOnCatalogue) action(destination string) func(src io.Reader, mediaType string) {
	return func(src io.Reader, mediaType string) {
		err := manifest.WriteUcManifestContentToDestination(src, destination)
		if err != nil {
			log.Error("Error decompressing layer =", err)
		}
	}
}
