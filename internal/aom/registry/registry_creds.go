// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"u-control/uc-aom/internal/pkg/utils"
)

var (
	DEV_CREDENTIALS_ROOT = utils.GetEnv("DEV_CREDENTIALS_ROOT", "/var/lib/uc-aom")
	REL_CREDENTIALS_ROOT = utils.GetEnv("REL_CREDENTIALS_ROOT", "/usr/share/uc-aom")
)
