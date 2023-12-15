// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build prod
// +build prod

package dbus

import (
	"context"

	"github.com/coreos/go-systemd/v22/dbus"
)

const systemd_mode = "replace"
const systemd_service = "nginx.service"

type DBusConnectionWrapper struct {
	sdc *dbus.Conn
}

func (r DBusConnectionWrapper) Close() {
	r.sdc.Close()
}

func Initialize() Factory {
	return func(ctx context.Context) (DBusConnection, error) {
		sdc, err := dbus.NewSystemdConnectionContext(ctx)
		return &DBusConnectionWrapper{sdc: sdc}, err
	}
}

func (r DBusConnectionWrapper) ReloadUnitContext(ctx context.Context, ch chan<- string) error {
	_, err := r.sdc.ReloadUnitContext(ctx, systemd_service, systemd_mode, ch)
	return err
}
