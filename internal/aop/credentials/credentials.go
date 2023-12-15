// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package credentials

import (
	"encoding/json"
	"u-control/uc-aom/internal/aop/fileio"
	"u-control/uc-aom/internal/aop/service"
)

type ValidationOpt func(c *Credentials) error

// Returns a Validation option that checks that the RepositoryName is set.
func RepositoryNameSet() ValidationOpt {
	return func(c *Credentials) error {
		if len(c.RepositoryName) == 0 {
			return &CredentialsValidationError{message: "repositoryname not set"}
		}
		return nil
	}
}

// Returns a Validation option that checks that the ServerAddress is set.
func ServerAddressSet() ValidationOpt {
	return func(c *Credentials) error {
		if len(c.ServerAddress) == 0 {
			return &CredentialsValidationError{message: "serveraddress not set"}
		}
		return nil
	}
}

type Credentials struct {
	// Optional Username. If provided entails Password.
	Username string `json:"username,omitempty"`

	// Optional Password. If provided entails Username.
	Password string `json:"password,omitempty"`

	// Reference where the add-on will be stored in the registry.
	RepositoryName string `json:"repositoryname,omitempty"`

	// URL of the an OCI capable registry.
	ServerAddress string `json:"serveraddress,omitempty"`

	readFileFunc fileio.ReadFileFunc
}

// Represents credential validation errors.
type CredentialsValidationError struct {
	message string
}

func (r *CredentialsValidationError) Error() string {
	return r.message
}

// Parse, validate and return the credentials at the given filepath.
func ParseAndValidate(readFileFunc fileio.ReadFileFunc, filepath string, opts ...ValidationOpt) (*Credentials, error) {
	credentials := &Credentials{readFileFunc: readFileFunc}
	err := credentials.parse(filepath)
	if err != nil {
		return credentials, err
	}
	return credentials, credentials.validate(opts...)
}

// Returns true if the server does not require authentication (username with password).
func (r *Credentials) IsInsecureServer() bool {
	return len(r.Username) == 0 && len(r.Password) == 0
}

func (r *Credentials) parse(filepath string) error {
	content, err := r.readFileFunc(filepath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(content, r)
	if err != nil {
		return err
	}

	return nil
}

// Performs rudimentary credential validation returning an error should it fail.
func (r *Credentials) validate(opts ...ValidationOpt) error {
	for _, opt := range opts {
		if err := opt(r); err != nil {
			return err
		}
	}

	if len(r.Username) == 0 {
		if len(r.Password) == 0 {
			// OK: Username and Password are all empty.
			return nil
		}
		return &CredentialsValidationError{message: "username not set although password provided"}
	}

	if len(r.Password) == 0 {
		return &CredentialsValidationError{message: "password not set although username provided"}
	}

	return nil
}

func SetRegistryServerAddress(credentials *Credentials) {
	if credentials.ServerAddress == "" {
		credentials.ServerAddress = service.REGISTRY_ADDRESS
	}
}
