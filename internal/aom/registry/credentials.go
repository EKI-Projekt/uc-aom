// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Reads the contents of the file at path and returns the result or an error.
type ReadCredentialsFunc func(path string) ([]byte, error)

// Credentials holds the information username and password for the related serveraddress.
// For an unsecured server just the serveraddress is needed.
type Credentials struct {
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	ServerAddress string `json:"serveraddress,omitempty"`
}

// Functions checks if credentials for a custom/dev registry were provided.
// If no file was found the production credentials are used as fallback.
func GetRegistryCredentials() (*Credentials, error) {
	var credentialsPath string

	if hasOverwriteCredentialsAt(DEV_CREDENTIALS_ROOT) {
		credentialsPath = filepath.Join(DEV_CREDENTIALS_ROOT, "registrycredentials.json")
	} else {
		credentialsPath = filepath.Join(REL_CREDENTIALS_ROOT, "registrycredentials.json")
	}
	credentials, err := ReadAndValidateCredentials(os.ReadFile, credentialsPath)
	return credentials, err
}

// Reads and (rudimentary) validatest the credentials from the provided path.
func ReadAndValidateCredentials(readCredentialsFunc ReadCredentialsFunc, filepath string) (*Credentials, error) {
	content, err := readCredentialsFunc(filepath)
	if err != nil {
		return nil, err
	}
	credentials, err := convertCredentials(content)
	if err != nil {
		return nil, err
	}

	err = validateCredentials(credentials)
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

func validateCredentials(credentials *Credentials) error {
	if len(credentials.ServerAddress) == 0 {
		return errors.New("No serveradress provided!")
	}

	if len(credentials.Username) == 0 && len(credentials.Password) != 0 {
		return errors.New("Password was provided but no username!")
	}

	if len(credentials.Username) != 0 && len(credentials.Password) == 0 {
		return errors.New("Username was provided but no password!")
	}
	return nil
}

func convertCredentials(content []byte) (*Credentials, error) {
	credentials := Credentials{}

	err := json.Unmarshal(content, &credentials)
	if err != nil {
		return nil, err
	}
	return &credentials, nil
}

// Checks if username and password are set.
func (c *Credentials) IsInsecureServer() bool {
	return len(c.Username) == 0 && len(c.Password) == 0
}

func hasOverwriteCredentialsAt(path string) bool {
	pathCredentials := filepath.Join(path, "registrycredentials.json")
	if _, err := os.Stat(pathCredentials); err == nil {
		return true
	} else {
		return false
	}
}
