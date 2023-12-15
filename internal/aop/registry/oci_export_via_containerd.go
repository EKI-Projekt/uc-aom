// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"bytes"
	"context"
	"errors"
	"strings"

	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/platforms"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2/registry"
)

type ExportDockerImageFunc func(registry registry.Registry, ref string, platform *specs.Platform) ([]byte, error)

// Return the docker image referenced by the given ref from the given given registry,
// as an OCI image layout tarball, or an error should it fail.
//
// See: https://github.com/opencontainers/image-spec/blob/main/image-layout.md
// See: https://github.com/containerd/containerd/blob/main/images/archive/exporter.go
func ExportUsingContainerd(reg registry.Registry, ref string, platform *specs.Platform) ([]byte, error) {
	ctx := context.Background()
	var out bytes.Buffer

	imageStore := ociRegistryBasedStore{reg: reg}

	repository, _, err := ToRepositoryAndTag(ref)
	if err != nil {
		return nil, err
	}

	ociRepository, err := reg.Repository(ctx, repository)
	if err != nil {
		log.Fatalf("Could not initialize repository: %v", err)
		return nil, err
	}

	provider := ociRepositoryBasedProvider{repository: ociRepository}
	platformComparer := platforms.Only(*platform)

	err = archive.Export(ctx, provider, &out, archive.WithPlatform(platformComparer), archive.WithImage(imageStore, ref))
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func ToRepositoryAndTag(ref string) (string, string, error) {
	if !strings.Contains(ref, ":") {
		return ref, "", errors.New("Could not find tag: " + ref)
	}
	s := strings.Split(ref, ":")
	return s[0], s[1], nil
}
