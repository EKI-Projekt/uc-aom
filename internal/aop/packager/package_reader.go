// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/pkg/config"
	pkgRegistry "u-control/uc-aom/internal/pkg/registry"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/file"
)

// Responsible for consuming an add-on package.
type PackageReader struct {
	unGzipTarballFunc fileio.UnGzipTarballFunc
}

type PullOptions struct {
	DestDir string
	Extract bool
}

// Create a new instance of the packager reader
// unGzipTarballFunc will be used to ungzip and untar the archive that contains manifest.json add asset files.
func NewPackageReader(unGzipTarballFunc fileio.UnGzipTarballFunc) *PackageReader {
	return &PackageReader{unGzipTarballFunc}
}

// Pull the given addOnTarget to the destDir, where it will be unpacked.
// Returns the platforms for which the addOnTarget is available.
func (r *PackageReader) Pull(ctx context.Context, addOnTarget registry.AddOnRepositoryTarget, options *PullOptions) ([]*ocispec.Platform, error) {

	_, err := os.Stat(options.DestDir)
	if err != nil {
		return nil, err
	}

	resolver := newPackageResolver(ctx, addOnTarget)
	addOnPackageIndex, err := resolver.resolve()
	if err != nil {
		return nil, err
	}

	platforms := make([]*ocispec.Platform, 0, len(addOnPackageIndex.addOnOCIPackage))
	for _, pkg := range addOnPackageIndex.addOnOCIPackage {
		platform := pkg.OciImageManifestDescriptor.Platform

		pullDirectory, err := createPullDirectory(options.DestDir, addOnTarget, platform)
		if err != nil {
			return nil, err
		}

		destination := file.New(pullDirectory)
		copyOptions := registry.NewOrasCopyOptions(&pkg.OciImageManifestDescriptor, config.UcImageManifestDescriptorFilename, addOnTarget, pullDirectory)
		_, err = registry.Copy(ctx, addOnTarget, destination, copyOptions)
		if err != nil {
			return nil, err
		}

		if options.Extract {
			err = r.extractAddOnArchive(addOnPackageIndex.addOnOCIPackage[0], options)
			if err != nil {
				return nil, err
			}
		}

		platforms = append(platforms, pkg.OciImageManifestDescriptor.Platform)
	}

	return platforms, nil
}

// Returns a subdirectory name for the given platform.
func GetSubdirectoryNameFor(platform *ocispec.Platform) string {
	return platform.OS + "-" + platform.Architecture
}

// Returns the path where the given addOn for the given platform has been pulled to.
func GetPullDirectory(addOnTarget registry.AddOnRepositoryTarget, platform *v1.Platform, parentDirectory string) string {
	platformDirectory := GetSubdirectoryNameFor(platform)
	normalizedRepository := pkgRegistry.NormalizeCodeName(addOnTarget.AddOnRepository())
	return filepath.Join(parentDirectory, normalizedRepository, addOnTarget.AddOnVersion(), platformDirectory)
}

func createPullDirectory(
	parentDirectory string,
	addOnTarget registry.AddOnRepositoryTarget,
	platform *ocispec.Platform) (string, error) {

	pullDirectory := GetPullDirectory(addOnTarget, platform, parentDirectory)
	err := os.MkdirAll(pullDirectory, 0755)
	if err != nil {
		return "", fmt.Errorf("Could not create add-on pull directory '%s': %v", pullDirectory, err)
	}
	return pullDirectory, nil
}

func (r *PackageReader) extractAddOnArchive(addOnPackage *addOnOCIPackage, options *PullOptions) error {
	descriptor, err := addOnPackage.GetAddOnManifestDescriptor()
	if err != nil {
		return err
	}
	manifestFileName := descriptor.Annotations[ocispec.AnnotationTitle]
	return r.extractManifestAndLogoTarball(options.DestDir, manifestFileName)
}

func (r *PackageReader) extractManifestAndLogoTarball(destDir string, manifestFileName string) error {

	return filepath.WalkDir(destDir, func(path string, d fs.DirEntry, err error) error {

		if d.IsDir() || d.Name() != manifestFileName {
			return nil
		}

		fileReader, err := os.Open(path)
		if err != nil {
			return err
		}

		defer fileReader.Close()
		err = r.unGzipTarballFunc(destDir, fileReader)
		if err != nil {
			return err
		}

		return os.Remove(path)
	})
}
