// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build prod
// +build prod

package config

import "u-control/uc-aom/internal/pkg/utils"

var (
	URL_ASSETS_LOCAL_ROOT  = utils.GetEnv("URL_ASSETS_ROOT", "/add-ons/assets")
	URL_ASSETS_REMOTE_ROOT = utils.GetEnv("URL_ASSETS_REMOTE_ROOT", "/add-ons/assets-remote")
)
