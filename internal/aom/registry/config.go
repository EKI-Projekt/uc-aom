// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"path"
	"u-control/uc-aom/internal/aom/utils"
)

var (
	cache_directory = utils.GetEnv("CACHE_DIRECTORY", "/var/cache/uc-aom/")
	DROP_IN_PATH    = utils.GetEnv("DROP_IN_PATH", path.Join(cache_directory, "drop-in"))
)
