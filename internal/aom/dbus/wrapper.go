// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package dbus

import (
	"context"
)

// Abstraction of the underlying system D-Bus.
type DBusConnection interface {
	// Clients should close the connection once they have finished
	// using the D-Bus connection.
	// SeeAlso: Factory
	Close()

	// Reload the unit, using the given context and writing
	// the returned message from D-Bus reload operation to ch.
	ReloadUnitContext(ctx context.Context, ch chan<- string) error
}

// Creates a new instance given the context.
// Clients should call Close() when finished with the connection.
type Factory func(context.Context) (DBusConnection, error)
