// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

// Define the registry interface that stores the information about add-ons
type AddOnRegistry interface {
	// Return all names of add-on repositories
	Repositories() ([]string, error)

	// Return all tags/versions of the provided add-on repository
	Tags(repository string) ([]string, error)

	// Fetch all data associated with the app uniquely identified by the repository and tag.
	// Returns the estimated install size in bytes and an error which describes the success status.
	Pull(repository string, tag string, processor ImageManifestLayerProcessor) (uint64, error)

	// Delete an add-on from the registry
	Delete(repository string, tag string) error
}
