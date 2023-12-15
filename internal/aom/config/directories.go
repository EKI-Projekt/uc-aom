// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package config

import "u-control/uc-aom/internal/pkg/utils"

// Automatic directory creation and environment variables by systemd
// https://www.freedesktop.org/software/systemd/man/systemd.exec.html#RuntimeDirectory=
var (
	UC_AOM_STATE_DIRECTORY = utils.GetEnv("STATE_DIRECTORY", "/var/lib/uc-aom")
	UC_AOM_CACHE_DIRECTORY = utils.GetEnv("CACHE_DIRECTORY", "/var/cache/uc-aom/")
)
