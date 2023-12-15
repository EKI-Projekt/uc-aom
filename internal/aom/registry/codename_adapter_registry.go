// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"errors"
	sharedRegistry "u-control/uc-aom/internal/pkg/registry"
)

var repositoryNotFoundError = errors.New("Not Found")

type codeNameAdapterRegistry struct {
	registry    AddOnRegistry
	codeNameMap map[string]string
}

func NewCodeNameAdapterRegistry(registy AddOnRegistry) AddOnRegistry {
	return &codeNameAdapterRegistry{registry: registy, codeNameMap: make(map[string]string)}
}

// Returns all normalized repositories known to this ORAS registry.
func (r *codeNameAdapterRegistry) Repositories() ([]string, error) {
	respositoriesWithCodeName, err := r.registry.Repositories()
	if err != nil {
		return nil, err
	}
	normalizedRepositories := make([]string, 0, len(respositoriesWithCodeName))
	for _, respositoryWithCodeName := range respositoriesWithCodeName {
		repository := sharedRegistry.NormalizeCodeName(respositoryWithCodeName)
		normalizedRepositories = append(normalizedRepositories, repository)
	}
	return normalizedRepositories, nil
}

// fetch the repository with code name before calling the registry tags
func (r *codeNameAdapterRegistry) Tags(repository string) ([]string, error) {
	repositoryWithCodeName, err := r.getRepositoryWithCodeName(repository)
	if err != nil {
		return make([]string, 0), err
	}
	return r.registry.Tags(repositoryWithCodeName)
}

// fetch the repositoty with code name before calling the registry pull
func (r *codeNameAdapterRegistry) Pull(repository string, tag string, processor ImageManifestLayerProcessor) (uint64, error) {
	repositoryWithCodeName, err := r.getRepositoryWithCodeName(repository)
	if err != nil {
		return uint64(0), err
	}
	return r.registry.Pull(repositoryWithCodeName, tag, processor)
}

// fetch the repository with code name before calling the registry delete
func (r *codeNameAdapterRegistry) Delete(repository string, tag string) error {
	repositoryWithCodeName, err := r.getRepositoryWithCodeName(repository)
	if err != nil {
		return err
	}
	return r.registry.Delete(repositoryWithCodeName, tag)
}

func (r *codeNameAdapterRegistry) getRepositoryWithCodeName(name string) (string, error) {
	respositoriesWithCodeName, err := r.registry.Repositories()
	if err != nil {
		return "", err
	}
	for _, repositoryWithCodeName := range respositoriesWithCodeName {
		normalizedName := sharedRegistry.NormalizeCodeName(repositoryWithCodeName)
		if normalizedName == name {
			return repositoryWithCodeName, nil
		}
	}
	return "", repositoryNotFoundError
}
