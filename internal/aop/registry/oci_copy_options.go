// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"io"
	"os"
	"path/filepath"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content"
)

type descriptorClosure struct {
	imageManifest         *ocispec.Descriptor
	imageManifestFilename string

	orasRepository oras.Target
	destination    string
}

func (r *descriptorClosure) capturedImageManifest(ctx context.Context, src content.Storage, root ocispec.Descriptor) (ocispec.Descriptor, error) {
	return *r.imageManifest, nil
}

func (r *descriptorClosure) writeDescriptorContent(ctx context.Context, desc ocispec.Descriptor) error {
	if r.imageManifest.Digest != desc.Digest {
		return nil
	}

	rc, err := r.orasRepository.Fetch(ctx, desc)
	if err != nil {
		return err
	}
	defer rc.Close()
	return writeToDestination(r.destination, r.imageManifestFilename, rc)
}

func writeToDestination(destination string, filename string, rc io.ReadCloser) error {
	target := filepath.Join(destination, filename)
	fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if _, err := io.Copy(fileToWrite, rc); err != nil {
		return err
	}
	return fileToWrite.Close()
}

func NewOrasCopyOptions(imageManifest *ocispec.Descriptor, imageManifestFilename string, orasRepository oras.Target, destination string) oras.CopyOptions {
	closure := descriptorClosure{imageManifest: imageManifest, imageManifestFilename: imageManifestFilename, orasRepository: orasRepository, destination: destination}
	copyGraphOptions := oras.CopyGraphOptions{PostCopy: closure.writeDescriptorContent}
	return oras.CopyOptions{CopyGraphOptions: copyGraphOptions, MapRoot: closure.capturedImageManifest}
}
