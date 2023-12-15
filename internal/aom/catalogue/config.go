// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package catalogue

import "u-control/uc-aom/internal/pkg/utils"

var (
	ASSETS_TMP_PATH     = utils.GetEnv("ASSETS_TMP_PATH", "/var/run/uc-aom")
	ASSETS_INSTALL_PATH = utils.GetEnv("ASSETS_INSTALL_PATH", "/var/lib/uc-aom")
)
