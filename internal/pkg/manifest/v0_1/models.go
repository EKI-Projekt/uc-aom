// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package v0_1

const (
	ValidManifestVersion = "0.1"

	// The internal add-on network is a special network which is created and recreated on system bootup.
	// It can be used by an add-on to communicate between other add-ons.
	// Futhermore, on devices with a firmware version 1.1x.xx it can be used to access the global variable list (GVL) by using the zeroMQ port (5555).
	InternalAddOnNetworkName = "internal-bridge"
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
	Platform        Platform                `json:"platform"`               // optional platforms that this add-on requires.
}

type Service struct {
	Type   string                 `json:"type"`   // the type of the service, currently only docker-compose is supported
	Config map[string]interface{} `json:"config"` // configuration of the service
}

type Environment struct {
	Type   string            `json:"type"`   // the type of the environment, currently only docker-compose is supported
	Config EnvironmentConfig `json:"config"` // configuration of the environment
}

type EnvironmentConfig struct {
	Volumes  map[string]map[string]interface{} `json:"volumes,omitempty"`  // volume settings of docker compose
	Networks map[string]map[string]interface{} `json:"networks,omitempty"` // network settings of docker compose
}

type Setting struct {
	Name     string  `json:"name,omitempty"`     // the name of the setting
	Label    string  `json:"label,omitempty"`    // the label of the setting
	Value    string  `json:"default,omitempty"`  // Variant - the default value of the setting (TextBox)
	Required bool    `json:"required,omitempty"` // whether the settings is required to have a value
	Pattern  string  `json:"pattern,omitempty"`  // A regex to validate user input
	Select   []*Item `json:"select,omitempty"`   // Variant - a DropDownList
}

type Item struct {
	Label    string `json:"label"`   // the label of the drop-down item
	Value    string `json:"value"`   // the value of the drop-down item
	Selected bool   `json:"default"` // Whether this drop-down item should be selected
}

type ProxyRoute struct {
	From string `json:"from"` // An absolute URL consisting of protocol, origin host and port e.g. http://localhost:8456
	To   string `json:"to"`   // A relative URL path e.g. /my/addon
}

// Vendor contains the add-ons vendor contact information
type Vendor struct {
	Name    string `json:"name"`    // Name of the vendor
	Url     string `json:"url"`     // website url consisting of protocol, origin and host e.g. https://www.abc.de
	Email   string `json:"email"`   // email address of the vendor
	Street  string `json:"street"`  // Street including and house number
	Zip     string `json:"zip"`     // Zip code e.g. 63456
	City    string `json:"city"`    // City of the vendor
	Country string `json:"country"` // Country of the vendor
}

type Platform []string
