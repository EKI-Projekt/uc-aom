// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package routes

import "u-control/uc-aom/internal/pkg/utils"

var (
	SITES_AVAILABLE_PATH      = utils.GetEnv("SITES_AVAILABLE_PATH", "/var/lib/nginx/user-sites-available")
	SITES_ENABLED_PATH        = utils.GetEnv("SITES_ENABLED_PATH", "/var/lib/nginx/user-sites-enabled")
	ROUTES_MAP_AVAILABLE_PATH = utils.GetEnv("ROUTES_MAP_AVAILABLE_PATH", "/var/lib/nginx/user-routes-available")
	ROUTES_MAP_ENABLED_PATH   = utils.GetEnv("ROUTES_MAP_ENABLED_PATH", "/var/lib/nginx/user-routes-enabled")
)

const (
	TemplateVersionV0_1_0 = "0.1.0"
	TemplateVersion       = "0.2.0"
	TemplateVersionLabel  = "com.weidmueller.uc.aom.reverse-proxy.version"
)
