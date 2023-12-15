// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package config

const (
	// Package version V1.0 was introduced because of the multi architecture registry format.
	// The registry format has been moved from an ImageManifest to an ImageIndex which can contain various ImageManifests for each platform.
	// Version V0.0 hasn't been defined explixit. Therefore, if that version cannot be found and we found an ImageManifest, it must be version V0.0.
	// Package version V0.0 was used in version 0.1, 0.2 and 0.3 which has been released in 1.16.0.
	// Package version V1.0 was introduced in version 0.4 which has been released in 2.0.0.
	PackageVersionV1_0 = "1.0"
)

const (
	// AnnotationVersion used to identify the package format version in the oras registry
	UcPackageVersion = PackageVersionV1_0

	// Reference: https://github.com/opencontainers/image-spec/blob/v1.1.0-rc2/image-layout.md#indexjson-file
	OciImageIndexFilename = "index.json"

	// MediaType used to identity our config
	UcConfigMediaType = "application/vnd.weidmueller.uc.config.v1+json"

	// AnnotationTitle used to identity our config and used as a file name
	UcConfigAnnotationTitle = "config.json"

	// Used as the filename when serializing the image manifest descriptor.
	UcImageManifestDescriptorFilename = "image-manifest.json"

	// MediaType used to identity our layer
	UcImageLayerMediaType = "application/vnd.weidmueller.uc.image.layer.v1.tar+gzip"

	// AnnotationTitle used to identity our layer and used as a file name
	UcImageLayerAnnotationTitle = "weidmueller.uc.aom"

	// Annotation schema version to identify our schema version
	UcImageLayerAnnotationSchemaVersion = "com.weidmueller.uc.image.schema.version"

	// Used as the filename when the image layer is created, compressed or decompressed.
	UcImageManifestFilename = "manifest.json"

	// Name of the tar archive in the swu-file containing the files needed for an drop-in installation
	UcSwuFilePayloadName = "payload.tar"
)
