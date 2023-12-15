// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"
	"u-control/uc-aom/internal/aom/system"
	"u-control/uc-aom/internal/pkg/manifest"
)

type manifestFeatureToSystemAdapter struct {
	system          system.System
	adaptedManifest *manifest.Root
}

func newManifestFeatureToSystemAdapter(system system.System) *manifestFeatureToSystemAdapter {
	return &manifestFeatureToSystemAdapter{
		system:          system,
		adaptedManifest: nil,
	}
}

// convert the provided manifest for the system to run on it.
func (c *manifestFeatureToSystemAdapter) adaptFeaturesToSystem(sourceManifest *manifest.Root) (*manifest.Root, error) {

	hasLocalPublicVolumes := manifest.HasLocalPublicVolumes(sourceManifest.Environments)

	if !hasLocalPublicVolumes {
		// nothing needs to be adapted
		return sourceManifest, nil
	}

	err := c.cloneAndAssignManifest(sourceManifest)
	if err != nil {
		return nil, err
	}

	err = c.assignAdminUserToLocalPublicVolumeServices()
	if err != nil {
		return nil, err
	}
	return c.adaptedManifest, nil

}

func (c *manifestFeatureToSystemAdapter) cloneAndAssignManifest(sourceManifest *manifest.Root) error {
	bytes, err := sourceManifest.ToBytes()
	if err != nil {
		return err
	}
	clonedManifest, err := manifest.NewFromBytes(bytes)
	if err != nil {
		return err
	}
	c.adaptedManifest = clonedManifest
	return nil
}

func (c *manifestFeatureToSystemAdapter) assignAdminUserToLocalPublicVolumeServices() error {
	adminUser, err := c.system.LookupAdminUser()
	if err != nil {
		return err
	}
	for _, s := range c.adaptedManifest.Services {
		if manifest.UsesLocalPublicVolume(s, c.adaptedManifest.Environments) {
			s.Config["user"] = fmt.Sprintf("%s:%s", adminUser.Uid, adminUser.Gid)
		}
	}
	return nil
}
