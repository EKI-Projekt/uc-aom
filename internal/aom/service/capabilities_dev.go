// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package service

const hostname = "ucm"

func getHostPlatform() (string, error) {
	return hostname, nil
}
