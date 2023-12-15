// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/pkg/config"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

type packageExporter struct {
	tarballFunc           fileio.TarballFunc
	createCpioArchiveFunc fileio.CreateCpioArchiveFunc
}

// Create a new instance of the packager exporter
// createCpioArchiveFunc will be used to create a cpio archive
// tarballFunc will be used to generate a tar archive.
func NewPackageExporter(
	createCpioArchiveFunc fileio.CreateCpioArchiveFunc,
	tarballFunc fileio.TarballFunc) *packageExporter {
	return &packageExporter{
		createCpioArchiveFunc: createCpioArchiveFunc,
		tarballFunc:           tarballFunc}
}

// Export the given addOnTarget to the provided outputFilepath as a swu file which includes the add-on as a archive.
// Uses the packager reader to pull the add-on.
func (r *packageExporter) Export(ctx context.Context, addOnTarget registry.AddOnRepositoryTarget, parentDirectory string, filename string) error {
	pullAddOnTempDir, err := createPullAddonTempDir()
	if err != nil {
		return err
	}
	defer removeTempDirectory(pullAddOnTempDir)
	log.Infof("Download directory: '%s'", pullAddOnTempDir)

	packagerReader := NewPackageReader(fileio.UnGzipTarball)
	platforms, err := packagerReader.Pull(ctx, addOnTarget, &PullOptions{DestDir: pullAddOnTempDir, Extract: false})
	if err != nil {
		return fmt.Errorf("Could not store the add-on to the output path %s: %v", pullAddOnTempDir, err)
	}

	for _, platform := range platforms {
		pulledAddOnDirPath := GetPullDirectory(addOnTarget, platform, pullAddOnTempDir)
		outputDirectory := filepath.Join(parentDirectory, GetSubdirectoryNameFor(platform))
		headerName := headerNameFunc(pullAddOnTempDir, platform)

		err = r.exportAddonDirToOutputPathAsSwufile(pulledAddOnDirPath, outputDirectory, filename, headerName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *packageExporter) exportAddonDirToOutputPathAsSwufile(
	pulledAddOnDirPath string,
	outputDirectory string,
	filename string,
	headerNameFunc fileio.HeaderNameFunc) error {

	addOnArchive, err := r.createTarForAddon(pulledAddOnDirPath, headerNameFunc)
	if err != nil {
		return fmt.Errorf("Could not create tar-archive: %v", err)
	}

	err = os.MkdirAll(outputDirectory, 0755)
	if err != nil {
		return err
	}
	fullOutputPath := filepath.Join(outputDirectory, filename)

	err = r.exportAddonArchiveToOutputPathAsSwufile(addOnArchive, fullOutputPath)
	if err != nil {
		return fmt.Errorf("Could not export add-on as swu-file: %v", err)
	}

	return nil
}

func headerNameFunc(pullDirectory string, platform *v1.Platform) func(string, string) (string, error) {
	removeSubDirectoryRegex := regexp.MustCompile(GetSubdirectoryNameFor(platform) + string(filepath.Separator) + `?`)
	return func(basename string, filename string) (string, error) {
		relative, err := filepath.Rel(pullDirectory, filename)
		if err != nil {
			return "", err
		}
		return removeSubDirectoryRegex.ReplaceAllString(relative, ""), nil
	}
}

func (r *packageExporter) createTarForAddon(directory string, headerNameFunc fileio.HeaderNameFunc) (fileio.ArchiveFileEntry, error) {
	predicateToIncludeAllFiles := func(_ string) bool {
		return true
	}

	tarFileContent, err := r.tarballFunc(directory, predicateToIncludeAllFiles, headerNameFunc)
	if err != nil {
		return fileio.ArchiveFileEntry{}, fmt.Errorf("Could not create tarball from directory: %v", err)
	}

	return fileio.ArchiveFileEntry{Name: config.UcSwuFilePayloadName, Body: tarFileContent}, nil
}

func (r *packageExporter) exportAddonArchiveToOutputPathAsSwufile(addOnArchive fileio.ArchiveFileEntry, outputFilepath string) error {
	swdescription := fileio.ArchiveFileEntry{Name: "sw-description", Body: createSwdescriptionContent()}
	err := r.createCpioArchiveFunc(outputFilepath, []fileio.ArchiveFileEntry{swdescription, addOnArchive})
	if err != nil {
		return fmt.Errorf("Can not create swu-file: %v", err)
	}
	return nil
}

func createPullAddonTempDir() (string, error) {
	useDefaultTempDir := ""

	// wildcard (*) includes a random number in the directory, so that we get always a new directory
	tempSubfolderPattern := "uc-aom-export-*"
	tempPullAddOnDir, err := os.MkdirTemp(useDefaultTempDir, tempSubfolderPattern)
	if err != nil {
		return "", fmt.Errorf("Could not create temp-directory for export: %v", err)
	}
	return tempPullAddOnDir, nil
}

func removeTempDirectory(tempdirectory string) {
	err := os.RemoveAll(tempdirectory)
	if err != nil {
		log.Errorf("Could not cleanup temp-directory: %v", err)
		// No error handling here because we are using a temp directory which is removed by the os
		// Moveover, we create a new subfolder for each export routine in the default temp dir.
	}
}

// Both bootloader marker are set to 'false' to prevent swupdate from updating bootloader environment variables.
// A update of bootloader environment variables is only valid for image/firmware related updates.
// Also see: https://sbabic.github.io/swupdate/sw-description.html?%20marker#update-transaction-and-status-marker
func createSwdescriptionContent() []byte {
	return []byte(fmt.Sprintf(
		`software =
{
	version = "0.1.0";
	hardware-compatibility: [ "#RE:.*" ];
	bootloader_state_marker = false;
	bootloader_transaction_marker = false;

	files: (
		{
			filename = "%v";
			path = "%s";
			type = "archive";
		}
	);
}`,
		config.UcSwuFilePayloadName, config.CACHE_DROP_IN_PATH))
}
