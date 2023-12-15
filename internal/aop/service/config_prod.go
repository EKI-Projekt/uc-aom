// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build prod
// +build prod

package service

import "u-control/uc-aom/internal/pkg/utils"

// Registry where the add-on package will be hosted.
var REGISTRY_ADDRESS string = utils.GetEnv("DEFAULT_REGISTRY_SERVER_ADDRESS", "wmucdev.azurecr.io")

const docker_os string = "linux"
