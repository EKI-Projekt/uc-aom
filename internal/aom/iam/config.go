// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package iam

import "u-control/uc-aom/internal/pkg/utils"

var (
	IAM_PERMISSION_PATH = utils.GetEnv("IAM_PERMISSION_PATH", "/var/lib/uc-iam/permissions")
)
