// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package catalogue

import "fmt"

// Represents a problem connecting to the remote registry.
type RemoteRegistryConnectionError struct {
	message string
}

func (r *RemoteRegistryConnectionError) Error() string {
	return r.message
}

func NewRemoteRegistryConnectionError(err error) error {
	message := fmt.Sprintf("Unable to connect to remote registry, err =%v", err.Error())
	return &RemoteRegistryConnectionError{message: message}
}
