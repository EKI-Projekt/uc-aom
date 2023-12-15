// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build prod
// +build prod

package manifest

import "u-control/uc-aom/internal/pkg/utils"

var (
	BASE_PATH = utils.GetEnv("MANIFESTS_BASE_PATH", "/usr/share/uc-aom/manifests")
)
