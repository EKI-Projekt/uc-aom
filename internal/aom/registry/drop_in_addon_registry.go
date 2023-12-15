// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	aom_manifest "u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/pkg/config"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

var (
	ErrWrongArchOrOS   = errors.New("wrong architecture or OS")
	ErrReadImageConfig = errors.New("cannot read config image")
)

type DropInAddOnRegistry struct {
	// Destination, root path, where the addon will be saved on disk
	root string

	// current running architecture
	architecture string

	// current running OS
	os string

	decompressor aom_manifest.ManifestLayerDecompressor
}

// NewDropInAddOnRegistry creates an instance of DropInAddOnRegistry
func NewDropInAddOnRegistry(root string, architecture string, os string, decompressor aom_manifest.ManifestLayerDecompressor) AddOnRegistry {
	return &DropInAddOnRegistry{root, architecture, os, decompressor}
}

// Read the artifacts from the drop in registry
// Artifacts can be filtered by mediatype via predicate
// Action is called on the artifacts that pass the filter
func (r *DropInAddOnRegistry) Pull(repository string, tag string, processor ImageManifestLayerProcessor) (uint64, error) {
	log.Tracef("DropInAddOnRegistry.Pull('%s', '%s')", repository, tag)
	basePathAddOn := path.Join(r.root, repository, tag)
	imageManifestPaths, err := findFilesWith(basePathAddOn, config.UcImageManifestDescriptorFilename)
	if err != nil {
		return 0, err
	}

	var lastError error = fs.ErrNotExist
	for _, path := range imageManifestPaths {
		imageManifest, err := r.validateImageManifestAt(path)
		if err != nil {
			lastError = err
			continue
		}

		cumulativeLayerSize := uint64(0)
		parentDirectory := filepath.Dir(path)
		for _, layer := range imageManifest.Layers {
			cumulativeLayerSize += uint64(layer.Size)
			if !processor.Filter(&layer) {
				continue
			}

			file, err := openFile(parentDirectory, &layer)
			if err != nil {
				log.Errorf("Couldn't open layer '%s': %v", layer.Digest, err)
				return 0, err
			}

			if IsUcImageLayerMediaType(layer.MediaType) {
				rc, err := r.decompressor.Decompress(file)
				if err != nil {
					log.Errorf("Decompress(): unexpected Error '%v'", err)
					return 0, err
				}
				processor.Action(rc, layer.MediaType)
			} else {
				processor.Action(bufio.NewReader(file), layer.MediaType)
			}

		}

		return estimatedInstallSizeBytes(cumulativeLayerSize), nil
	}

	return 0, lastError
}

// Delete an add-on repository from the drop-in registry
func (r *DropInAddOnRegistry) Delete(repository string, tag string) error {
	log.Tracef("DropInAddOnRegistry.Delete('%s', '%s')", repository, tag)
	if repository == "" || tag == "" {
		// fail silently
		return nil
	}

	pathToRepository := path.Join(r.root, repository)
	return os.RemoveAll(pathToRepository)
}

// Returns the repositories in the drop-in registry
func (r *DropInAddOnRegistry) Repositories() ([]string, error) {
	log.Tracef("DropInAddOnRegistry.Repositories()")
	return getDirectoriesIn(r.root)
}

// Returns the tags from the repository
func (r *DropInAddOnRegistry) Tags(repository string) ([]string, error) {
	log.Tracef("DropInAddOnRegistry.Tags('%s')", repository)
	path := path.Join(r.root, repository)
	return getDirectoriesIn(path)
}

func (r *DropInAddOnRegistry) checkIfCanBePulled(config *ocispec.Image) bool {
	return config.Architecture == r.architecture && config.OS == r.os
}

func (r *DropInAddOnRegistry) validateImageManifestAt(path string) (*ocispec.Manifest, error) {
	imageManifest, err := readImageManifest(path)
	if err != nil {
		return nil, err
	}

	config, err := readImageConfig(filepath.Dir(path), imageManifest)
	if err != nil {
		return nil, ErrReadImageConfig
	}

	canBePulled := r.checkIfCanBePulled(config)
	if !canBePulled {
		return nil, ErrWrongArchOrOS
	}

	return imageManifest, nil
}

func getDirectoriesIn(path string) ([]string, error) {
	dirList, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	repositories := make([]string, 0, len(dirList))
	for _, dir := range dirList {
		if dir.IsDir() {
			repositories = append(repositories, dir.Name())
		}
	}

	return repositories, nil
}

func openFile(rootDirectory string, descriptor *ocispec.Descriptor) (*os.File, error) {
	if fileName, ok := descriptor.Annotations[ocispec.AnnotationTitle]; ok {
		pathToFile := path.Join(rootDirectory, fileName)
		file, err := os.Open(pathToFile)
		if err != nil {
			log.Errorf("Couldn't open '%s': %v", pathToFile, err)
			return nil, err
		}
		return file, nil
	}

	return nil, fs.ErrNotExist
}

func readImageConfig(rootDirectory string, imageManifest *ocispec.Manifest) (*ocispec.Image, error) {
	if imageManifest.Config.MediaType != config.UcConfigMediaType {
		return nil, fs.ErrNotExist
	}

	configFile, err := openFile(rootDirectory, &imageManifest.Config)
	if err != nil {
		return nil, err
	}
	defer configFile.Close()

	content, err := io.ReadAll(configFile)
	if err != nil {
		return nil, err
	}

	var config ocispec.Image
	err = json.Unmarshal(content, &config)
	return &config, err
}

func readImageManifest(filepath string) (*ocispec.Manifest, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var imageManifest ocispec.Manifest
	err = json.Unmarshal(content, &imageManifest)
	return &imageManifest, err
}

func findFilesWith(root string, matchingName string) ([]string, error) {
	accumulate := make([]string, 0)
	appendPathWithMatchToAccumulate := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Errorf("Couldn't read '%s' in '%s': %v", path, root, err)
			return err
		}

		if !d.IsDir() && d.Name() == matchingName {
			accumulate = append(accumulate, path)
			return fs.SkipDir
		}

		return nil
	}

	return accumulate, filepath.WalkDir(root, appendPathWithMatchToAccumulate)
}
