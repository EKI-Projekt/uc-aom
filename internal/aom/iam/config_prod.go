// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build prod
// +build prod

package iam

import (
	"u-control/uc-aom/internal/pkg/utils"
	aomUtils "u-control/uc-aom/internal/aom/utils"
)

var (
	IAM_AUTH_URL            = utils.GetEnv("IAM_AUTH_URL", "http://127.0.0.1:49155")
	IAM_AUTH_ENDPOINT       = utils.GetEnv("IAM_AUTH_ENDPOINT", "/v1/data/service/authz")
	IAM_AUTH_SERVICE_UCAOM  = utils.GetEnv("IAM_AUTH_SERVICE_UCAOM", "uc-aom")
	IAM_AUTH_SERVICE_UCAUTH = utils.GetEnv("IAM_AUTH_SERVICE_UCAUTH", "uc-auth")
	IAM_AUTH_NO_AUTH_OPT    = aomUtils.GetEnvBool("IAM_AUTH_NO_AUTH_OPT", true)
)
