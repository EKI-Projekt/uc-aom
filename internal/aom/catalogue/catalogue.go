// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package catalogue

import (
	"io"
	"u-control/uc-aom/internal/pkg/manifest"
)

type CatalogueAddOn struct {
	// Uniquely identifies the AddOn
	Name string
	// The version of the AddOn
	Version string
	// The manifest of the AddOn
	Manifest manifest.Root
}

// Decorates a CatalogueAddOn with docker images
type CatalogueAddOnWithImages struct {
	// Estimated install size in bytes
	EstimatedInstallSize uint64

	// An add-on instance
	AddOn CatalogueAddOn

	// An array of gzipped tarballs in binary form, each of which
	// represents a docker image as a repository containing all images and metadata.
	// The repository maps to those referenced in the add-on's manifest JSON file.
	DockerImageData []io.Reader
}

type RemoteAddOnCatalogue interface {
	// Returns the names of all AddOns associated with the remote catalogue.
	// An AddOn name is a unique identifier.
	GetAddOnNames() ([]string, error)

	// Returns the versions for the AddOn identified by name, in ascending order.
	GetAddOnVersions(name string) ([]string, error)

	// Returns the AddOn identified by name with the given version from the remote catalogue.
	GetAddOn(name string, version string) (CatalogueAddOn, error)

	// Returns the latest version of all AddOns from the remote catalogue.
	GetLatestAddOns() ([]*CatalogueAddOn, error)
}

type LocalAddOnCatalogue interface {
	// Pulls the manifest and associated artefacts,
	// for the AddOn identified by name in the given version,
	// to the local catalogue and returns the result.
	PullAddOn(name string, version string) (CatalogueAddOnWithImages, error)

	// Deletes the manifest and associated artefacts,
	// for the AddOn identified by name,
	// from the local catalogue.
	DeleteAddOn(name string) error

	// Returns the AddOn identified by name from the local catalogue.
	GetAddOn(name string) (CatalogueAddOn, error)

	// Returns all AddOns from the local catalogue.
	GetAddOns() ([]*CatalogueAddOn, error)

	// Fetches and returns the manifest for the add-on identified by name and version.
	FetchManifest(name string, version string) (*manifest.Root, error)
}
