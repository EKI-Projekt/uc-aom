// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"context"
	"u-control/uc-aom/internal/aop/credentials"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

// Initialize a connection to the OCI registry using the given credentials.
func InitializeRegistry(ctx context.Context, credentials *credentials.Credentials) (*remote.Registry, error) {
	reg, err := remote.NewRegistry(credentials.ServerAddress)
	if err != nil {
		return nil, err
	}

	creds := auth.Credential{
		Username: credentials.Username,
		Password: credentials.Password,
	}

	reg.RepositoryOptions.Client = &auth.Client{
		Credential: func(c context.Context, s string) (auth.Credential, error) {
			return creds, nil
		},
	}

	reg.PlainHTTP = credentials.IsInsecureServer()

	return reg, reg.Ping(ctx)
}

// Given the source, which is a rooted directed acyclic graph (DAG)
// with the root node tagged with manifest.Version, copies it to the destination.
func Copy(ctx context.Context, source AddOnTarget, destination oras.Target, opts oras.CopyOptions) (ocispec.Descriptor, error) {
	log.Debugf("oras.Copy source: %v\tdestination: %v", source, destination)
	return oras.Copy(ctx, source, source.AddOnVersion(), destination, "", opts)
}
