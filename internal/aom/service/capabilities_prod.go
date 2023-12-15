// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build prod
// +build prod

package service

import "os"

func getHostPlatform() (string, error) {
	return os.Hostname()
}
