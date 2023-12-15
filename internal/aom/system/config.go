// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package system

import "u-control/uc-aom/internal/pkg/utils"

var (
	root_access_script_path = utils.GetEnv("ROOT_ACCESS_SCRIPT_PATH", "/usr/sbin/configure-root-access.sh")
	admin_uid               = utils.GetEnv("ADMIN_UID", "1000")
	admin_gid               = utils.GetEnv("ADMIN_GID", "1000")
)
