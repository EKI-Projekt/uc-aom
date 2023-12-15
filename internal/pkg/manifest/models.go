// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"encoding/json"
	"errors"
	manifestV0_1 "u-control/uc-aom/internal/pkg/manifest/v0_1"
)

const (
	ValidManifestVersion = "0.2"

	LocalPublicVolumeDriverName       = "local-public"
	LocalPublicVolumeAccessDriverName = "local-public-access"

	// The internal add-on network is a special network which is created and recreated on system bootup.
	// It can be used by an add-on to communicate between other add-ons.
	// Futhermore, on devices with a firmware version 1.1x.xx it can be used to access the global variable list (GVL) by using the zeroMQ port (5555).
	InternalAddOnNetworkName = manifestV0_1.InternalAddOnNetworkName
)

type Root struct {
	ManifestVersion string                  `json:"manifestVersion"`        // version of the add-on manifest
	Version         string                  `json:"version"`                // version of the add-on
	Title           string                  `json:"title"`                  // title of the add-on
	Description     string                  `json:"description"`            // description of the add-on
	Logo            string                  `json:"logo"`                   // logo of the add-on, is presented in the ui
	Services        map[string]*Service     `json:"services"`               // settings of the add-on services
	Environments    map[string]*Environment `json:"environments,omitempty"` // settings of the add-on environment
	Settings        map[string][]*Setting   `json:"settings,omitempty"`     // settings
	Publish         map[string]*ProxyRoute  `json:"publish,omitempty"`      // publish defines an optional proxy route where a UI is made available.
	Vendor          *Vendor                 `json:"vendor,omitempty"`       // vendor information, see Vendor for details.
	Features        []Feature               `json:"features,omitempty"`     // features that the app depends on, see Feature for details.
	Platform        []string                `json:"platform"`               // optional platforms that this add-on requires.
}

// UnmarshalManifestVersionFrom return the ManifestVersion from the byte content or error if not possible
func UnmarshalManifestVersionFrom(manifestRawByteContent []byte) (string, error) {

	type onlyWithManifestVersion struct {
		ManifestVersion string `json:"manifestVersion"`
	}

	model := &onlyWithManifestVersion{}
	if err := json.Unmarshal(manifestRawByteContent, model); err != nil {
		return "", err
	}

	if model.ManifestVersion == "" {
		return "", errors.New("No ManifestVersion in byte content")
	}

	return model.ManifestVersion, nil
}

// Creates a new instance of Root based on the provided byte content
func NewFromBytes(content []byte) (*Root, error) {
	root := &Root{}
	if err := json.Unmarshal(content, root); err != nil {
		return nil, err
	}
	return root, nil
}

// ToBytes returns the JSON encoding byte slice
func (r *Root) ToBytes() ([]byte, error) {
	return json.Marshal(r)
}

type Service manifestV0_1.Service

type Environment struct {
	manifestV0_1.Environment
	Config EnvironmentConfig `json:"config"` // configuration of the environment
}

func NewEnvironment(envType string) *Environment {
	return &Environment{
		Environment: manifestV0_1.Environment{Type: envType},
	}
}

func (e *Environment) WithVolumes(volumes map[string]map[string]interface{}) *Environment {
	e.Config.Volumes = volumes
	return e
}

func (e *Environment) WithNetworks(networks map[string]map[string]interface{}) *Environment {
	e.Config.Networks = networks
	return e
}

type EnvironmentConfig manifestV0_1.EnvironmentConfig

type Setting struct {
	manifestV0_1.Setting
	Select []*Item `json:"select,omitempty"` // Variant - a DropDownList
}

func NewSettings(name string, label string, required bool) *Setting {
	s := manifestV0_1.Setting{
		Name:     name,
		Label:    label,
		Required: required,
	}
	return &Setting{Setting: s}
}

func (s *Setting) WithTextBoxValue(value string) *Setting {
	s.Value = value
	return s
}

func (s *Setting) WithSelectItems(items ...*Item) *Setting {
	s.Select = items
	return s
}

// Select the item with the same value, deselect the others.
func (s *Setting) SelectValue(value string) {
	for _, item := range s.Select {
		item.Selected = item.Value == value
	}
}

// Return true if setting is a text box, otherwise false.
func (s *Setting) IsTextBox() bool {
	return s.Value != "" && len(s.Select) == 0
}

type Item manifestV0_1.Item

type ProxyRoute manifestV0_1.ProxyRoute

type Vendor manifestV0_1.Vendor

type Platform manifestV0_1.Platform

// Declares a single hardware or software feature that is used by the application.
// The purpose of a `features` declaration is to inform about the set of hardware and software features on which your application depends
type Feature struct {
	Name     string `json:"name"`               // Specifies a single hardware or software feature used by the application, as a descriptor string.
	Required *bool  `json:"required,omitempty"` // Boolean value that indicates whether the application requires the feature specified in `name`. The default value if not declared is `true`.
}
