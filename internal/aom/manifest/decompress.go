// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"u-control/uc-aom/internal/aom/utils"
	"u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/pkg/manifest"
)

// Access struct of the decompressed uc manifest layer
type ManifestLayerContent struct {
	// Raw content of the manifest
	Manifest []byte

	// Raw content of the logo
	Logo []byte

	// Name of the logo
	LogoName string
}

// Interface to decompress the manifest layer blob
type ManifestLayerDecompressor interface {
	Decompress(layerReader io.ReadCloser) (io.ReadCloser, error)
}

type ManifestTarGzipDecompressor struct {
}

// DecompressLayer decompress the manifest layer and return its content.
func (d *ManifestTarGzipDecompressor) Decompress(layerReader io.ReadCloser) (io.ReadCloser, error) {
	tarReader, err := readDataFromTgz(layerReader)
	if err != nil {
		return nil, err
	}

	content, err := readLogoAndManifestFromTar(tarReader)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(content)
	return io.NopCloser(buf), err
}

func readDataFromTgz(tgz io.Reader) (*tar.Reader, error) {
	gzipReader, err := gzip.NewReader(tgz)
	if err != nil {
		return nil, err
	}
	tarReader := tar.NewReader(gzipReader)
	return tarReader, nil
}

func readLogoAndManifestFromTar(tarReader *tar.Reader) (*ManifestLayerContent, error) {

	manifestDecompressLayer := &ManifestLayerContent{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
		if header.Name == config.UcImageManifestFilename {
			manifestDecompressLayer.Manifest, err = ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
		} else {
			// the logo is the only other accepted file
			manifestDecompressLayer.LogoName = header.Name
			manifestDecompressLayer.Logo, err = ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
		}
	}
	return manifestDecompressLayer, nil
}

// WriteUcManifestContentToDestination the reader, add hash to logo and extract to destination.
func WriteUcManifestContentToDestination(reader io.Reader, destination string) error {
	manifestLayer, err := UnmarshalDecompressedContent(reader)
	if err != nil {
		return err
	}

	hash, err := utils.GetShortSHA1HashFrom(manifestLayer.Logo)
	if err != nil {
		return err
	}

	newLogoName := addHashToFilename(manifestLayer.LogoName, hash)

	maifestRawChanged, err := changeLogoInManifest(manifestLayer.Manifest, newLogoName)
	if err != nil {
		return err
	}

	if err := utils.MkDirAll(destination, 0755); err != nil {
		return err
	}

	if err = utils.WriteFileToDestination(config.UcImageManifestFilename, maifestRawChanged, destination); err != nil {
		if err := os.RemoveAll(destination); err != nil {
			return err
		}
		return err
	}
	if err = utils.WriteFileToDestination(newLogoName, manifestLayer.Logo, destination); err != nil {
		if err := os.RemoveAll(destination); err != nil {
			return err
		}
		return err
	}
	return nil
}

func UnmarshalDecompressedContent(reader io.Reader) (*ManifestLayerContent, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	manifestLayer := &ManifestLayerContent{}
	err = json.Unmarshal(content, manifestLayer)
	if err != nil {
		return nil, err
	}
	return manifestLayer, nil
}

func addHashToFilename(filename string, hash string) string {
	fileExt := filepath.Ext(filename)
	pureFilename := strings.TrimSuffix(filename, fileExt)
	newFilename := pureFilename + "-" + hash + fileExt
	return newFilename
}

func changeLogoInManifest(manifestRaw []byte, newLogoName string) ([]byte, error) {
	manifest, err := manifest.NewFromBytes(manifestRaw)
	if err != nil {
		return nil, err
	}
	manifest.Logo = newLogoName
	return manifest.ToBytes()
}
