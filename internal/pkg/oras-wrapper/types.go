// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package oraswrapper

import ocispec "github.com/opencontainers/image-spec/specs-go/v1"

type DescriptorBlobTuple struct {
	Desc *ocispec.Descriptor
	Blob []byte
}
