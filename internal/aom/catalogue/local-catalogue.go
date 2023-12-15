// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package catalogue

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/aom/registry"
	model "u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
)

var (
	ErrorAddOnNotFound = errors.New("Not found.")
)

type localAddOnCatalogue struct {
	// Destination, root path, where the addon will be saved on disk
	Root string

	localfs       *manifest.LocalFSRepository
	addOnRegistry registry.AddOnRegistry
}

// NewLocalAddOnCatalogue creates an instance of LocalAddOnCatalogue
func NewLocalAddOnCatalogue(root string, addOnRegistry registry.AddOnRegistry, localfs *manifest.LocalFSRepository) LocalAddOnCatalogue {
	return &localAddOnCatalogue{Root: root, localfs: localfs, addOnRegistry: addOnRegistry}
}

// Returns the add-on's manifest from the registry using the name and version
func (c *localAddOnCatalogue) FetchManifest(name string, version string) (*model.Root, error) {
	log.Tracef("LocalCatalogue.FetchManifest('%s', '%s')", name, version)
	dir, err := os.MkdirTemp("", "*-manifest")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)

	accumulator := dockerImageAccumulator{
		destination: dir,
	}

	processor := registry.NewUcImageLayerProcessor(accumulator.action)
	_, err = c.addOnRegistry.Pull(name, version, processor)
	if err != nil {
		return nil, err
	}

	return c.localfs.ReadManifestFrom(dir)
}

func (c *localAddOnCatalogue) PullAddOn(name string, version string) (CatalogueAddOnWithImages, error) {
	log.Tracef("LocalCatalogue.PullAddOn('%s', '%s')", name, version)
	destination := c.getInstallLocation(name)

	err := os.RemoveAll(destination)
	if err != nil {
		return CatalogueAddOnWithImages{}, err
	}

	accumulator := dockerImageAccumulator{
		destination: destination,
	}

	processor := registry.NewAcceptAllManifestLayerProcessor(accumulator.action)
	estimatedInstallSize, err := c.addOnRegistry.Pull(name, version, processor)
	if err != nil {
		return CatalogueAddOnWithImages{}, err
	}
	addOn, err := c.GetAddOn(name)
	if err != nil {
		return CatalogueAddOnWithImages{}, err
	}

	return CatalogueAddOnWithImages{
			AddOn:                addOn,
			DockerImageData:      accumulator.imageReaders,
			EstimatedInstallSize: estimatedInstallSize},
		nil
}

func (c *localAddOnCatalogue) DeleteAddOn(name string) error {
	log.Tracef("LocalCatalogue.DeleteAddOn('%s')", name)
	location := c.getInstallLocation(name)
	return os.RemoveAll(location)
}

func (c *localAddOnCatalogue) GetAddOn(name string) (CatalogueAddOn, error) {
	log.Tracef("LocalCatalogue.GetAddOn('%s')", name)
	location := c.getInstallLocation(name)
	manifest, err := c.localfs.ReadManifestFrom(location)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return CatalogueAddOn{}, ErrorAddOnNotFound
		}

		log.Errorf("ReadManifestFrom('%s') error = %+v", location, err)
		return CatalogueAddOn{}, err
	}
	return CatalogueAddOn{Name: name, Version: manifest.Version, Manifest: *manifest}, nil
}

func (c *localAddOnCatalogue) GetAddOns() ([]*CatalogueAddOn, error) {
	log.Trace("LocalCatalogue.GetAddOns()")
	repositories, err := c.localfs.GetManifestsDirectories(c.Root)

	if err != nil {
		return make([]*CatalogueAddOn, 0), err
	}
	log.Tracef("Repositories: [%s]", strings.Join(repositories, ", "))
	addOns := make([]*CatalogueAddOn, 0, len(repositories))
	for _, repo := range repositories {
		addOn, err := c.GetAddOn(repo)
		if err != nil {
			continue
		}
		addOns = append(addOns, &addOn)
	}

	return addOns, nil
}

func (c *localAddOnCatalogue) getInstallLocation(name string) string {
	return filepath.Join(c.Root, name)
}

type dockerImageAccumulator struct {
	destination  string
	imageReaders []io.Reader
}

func (p *dockerImageAccumulator) action(src io.Reader, mediaType string) {
	if !registry.IsUcImageLayerMediaType(mediaType) {
		p.imageReaders = append(p.imageReaders, src)
		return
	}

	err := manifest.WriteUcManifestContentToDestination(src, p.destination)
	if err != nil {
		log.Error("WriteUcManifestContent() error =", err)
	}
}
