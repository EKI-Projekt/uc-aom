// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/aom/utils"
	"u-control/uc-aom/internal/pkg/config"
	model "u-control/uc-aom/internal/pkg/manifest"
)

func TestDecompress_ShallAddHashValueToLogoFile(t *testing.T) {
	// arrange
	logoTarFile := createLogo("logo.png")
	manifestTarFile := createManifestWith(logoTarFile)

	reader := createManifestLogoTarReader(t, &manifestTarFile, &logoTarFile)
	tempDir := t.TempDir()
	decompressor := manifest.ManifestTarGzipDecompressor{}

	// act
	rc, err := decompressor.Decompress(reader)
	if err != nil {
		t.Fatalf("Decompress(), Unexpected error %v", err)
	}

	if err := manifest.WriteUcManifestContentToDestination(rc, tempDir); err != nil {
		t.Fatalf("Decompress() failed! %v", err)
	}

	// assert
	fileDirEntries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("os.ReadDir(tempDir) %v", err)
	}

	hash, err := utils.GetShortSHA1HashFrom(logoTarFile.Content)
	if err != nil {
		t.Fatalf("manifest.GetShortHashFrom(): %v", err)
	}

	expectedLogoFile := "logo-" + hash + ".png"
	foundFile := false
	for _, entry := range fileDirEntries {
		if entry.Name() == expectedLogoFile {
			foundFile = true
		}
	}

	if !foundFile {
		t.Fatalf("File not found, want %s", expectedLogoFile)
	}

	expectedManifestFilePath := filepath.Join(tempDir, config.UcImageManifestFilename)
	manifestRaw, err := os.ReadFile(expectedManifestFilePath)
	if err != nil {
		t.Fatalf("os.ReadFile(expectedManifestFilePath): %v", err)
	}

	gotManifest, err := model.NewFromBytes(manifestRaw)
	if err != nil {
		t.Fatalf("Couldn't read decompressed manifest.")
	}
	if gotManifest.Logo != expectedLogoFile {
		t.Fatalf("Expect file name for logo %s but got %s", expectedLogoFile, gotManifest.Logo)
	}
}

func TestDecompress_ShallReturnErrorIfManifestNotExist(t *testing.T) {
	// arrange
	logoTarFile := createLogo("logo.png")

	reader := createManifestLogoTarReader(t, &logoTarFile)
	tempDir := t.TempDir()

	// act
	err := manifest.WriteUcManifestContentToDestination(reader, tempDir)

	// assert
	if err == nil {
		t.Fatalf("Expected test to fail!")
	}

	if errors.Is(err, errors.New("Manifest wasn't found!")) {
		t.Fatalf("Unexpected error! %s", err)
	}

	fileDirEntries, err := os.ReadDir(tempDir)
	if len(fileDirEntries) != 0 {
		t.Fatalf("Expected no files to be written, but %d exist.", len(fileDirEntries))
	}
}

func TestDecompress_ShallReturnErrorIfLogoNotExist(t *testing.T) {
	// arrange
	logoTarFile := createLogo("logo.png")

	reader := createManifestLogoTarReader(t, &logoTarFile)
	tempDir := t.TempDir()

	// act
	err := manifest.WriteUcManifestContentToDestination(reader, tempDir)

	// assert
	if err == nil {
		t.Fatalf("Expected test to fail!")
	}

	if errors.Is(err, errors.New("Logo wasn't found!")) {
		t.Fatalf("Unexpected error! %s", err)
	}

	fileDirEntries, err := os.ReadDir(tempDir)
	if len(fileDirEntries) != 0 {
		t.Fatalf("Expected no files to be written, but %d exist.", len(fileDirEntries))
	}
}

func createLogo(name string) tarArchiveFile {
	logoTarFile := tarArchiveFile{
		Name:    name,
		Content: []byte("logo-content"),
	}
	return logoTarFile
}

func createManifestWith(logoTarFile tarArchiveFile) tarArchiveFile {
	addOnManifest := model.Root{
		Logo: logoTarFile.Name,
	}
	manifestContent, _ := json.Marshal(addOnManifest)
	manifestTarFile := tarArchiveFile{
		Name:    config.UcImageManifestFilename,
		Content: manifestContent,
	}
	return manifestTarFile
}

func createManifestLogoTarReader(t *testing.T, tarArchiveFiles ...*tarArchiveFile) io.ReadCloser {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, tarArchiveFile := range tarArchiveFiles {
		header := tar.Header{
			Name: tarArchiveFile.Name,
			Size: int64(len(tarArchiveFile.Content)),
		}
		tarWriter.WriteHeader(&header)

		if _, err := tarWriter.Write(tarArchiveFile.Content); err != nil {
			t.Fatalf("tarWriter.Write(logoContent) %v", err)
		}
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Can not close tarWriter %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Can not close gzipWriter  %v", err)
	}
	reader := bytes.NewReader(buf.Bytes())
	return io.NopCloser(reader)
}
