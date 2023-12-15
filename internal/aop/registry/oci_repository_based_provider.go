// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"bytes"
	"context"
	"io"

	"github.com/containerd/containerd/content"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/registry"
)

type ociRepositoryBasedProvider struct {
	repository registry.Repository
}

// ReaderAt only requires desc.Digest to be set.
// Other fields in the descriptor may be used internally for resolving
// the location of the actual data.
func (r ociRepositoryBasedProvider) ReaderAt(ctx context.Context, desc ocispec.Descriptor) (content.ReaderAt, error) {
	rc, err := r.repository.Fetch(ctx, desc)
	if err != nil {
		return nil, err
	}

	buff := bytes.NewBuffer([]byte{})
	_, err = io.Copy(buff, rc)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(buff.Bytes())
	return &byteReaderAt{reader: reader}, nil
}

type byteReaderAt struct {
	io.ReaderAt
	io.Closer

	reader *bytes.Reader
}

func (r *byteReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	return r.reader.ReadAt(p, off)
}

func (r *byteReaderAt) Size() int64 {
	return r.reader.Size()
}

func (r *byteReaderAt) Close() error {
	return nil
}
