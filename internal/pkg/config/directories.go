// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package config

import (
	"path"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/pkg/utils"
)

// Automatic directory creation and environment variables by systemd
// https://www.freedesktop.org/software/systemd/man/systemd.exec.html#RuntimeDirectory=

// The CACHE_DROP_IN_PATH environment variable is used by both the aom and aop package.
// In order to avoid duplicate code and ensure that both packages have access to this variable,
// it has been placed inside the shared package.
var (
	DROP_IN_FOLDER_NAME      = "drop-in"
	CACHE_DROP_IN_PATH       = utils.GetEnv("CACHE_DROP_IN_PATH", path.Join(config.UC_AOM_CACHE_DIRECTORY, DROP_IN_FOLDER_NAME))
	PERSISTENCE_DROP_IN_PATH = utils.GetEnv("PERSISTENCE_DROP_IN_PATH", path.Join(config.UC_AOM_STATE_DIRECTORY, DROP_IN_FOLDER_NAME))
)
