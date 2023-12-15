// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"u-control/uc-aom/internal/aom/system"
	"u-control/uc-aom/internal/pkg/manifest"
)

// Represents an unsupported platform.
type UnsupportedPlatformError struct {
	message string
}

func (r *UnsupportedPlatformError) Error() string {
	return r.message
}

var (
	SshRootAccessNotEnabledError = errors.New("System ssh root access is not enabled")
)

type Capabilities struct {
	Platforms []string

	system   system.System
	features []manifest.Feature
}

func NewCapabilities(system system.System, platform ...string) *Capabilities {
	return &Capabilities{system: system, Platforms: platform, features: make([]manifest.Feature, 0)}
}

// Adds the provided features to thecapabilities
func (r *Capabilities) WithFeatures(features []manifest.Feature) *Capabilities {
	r.features = features
	return r
}

// Validates platform and provided features. Failing a validation will result in an error
func (r *Capabilities) Validate() error {
	err := r.validatePlatform()
	if err != nil {
		return err
	}

	if len(r.features) > 0 {
		return r.validateFeatures()
	}

	return nil
}

func (r *Capabilities) validatePlatform() error {
	hostname, err := getHostPlatform()
	if err != nil {
		return &UnsupportedPlatformError{err.Error()}
	}

	for _, platform := range r.Platforms {
		if platform == hostname {
			return nil
		}
	}

	return &UnsupportedPlatformError{"The platform is not supported"}
}

func (r *Capabilities) validateFeatures() error {
	for _, feature := range r.features {
		if feature.Required == nil || *feature.Required == true {
			switch feature.Name {
			case "ucontrol.software.root_access":
				returnValue, err := r.system.IsSshRootAccessEnabled()
				if err != nil {
					return err
				}
				if returnValue {
					continue
				}
				return SshRootAccessNotEnabledError
			}
		}
	}

	return nil
}
