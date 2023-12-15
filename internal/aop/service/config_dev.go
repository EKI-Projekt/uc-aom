// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package service

// Registry where the add-on package will be hosted.
var REGISTRY_ADDRESS string = "registry:5000"

const docker_os string = "linux"
