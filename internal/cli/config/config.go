// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package config

import "u-control/uc-aom/internal/pkg/utils"

// Replaced at build-time.
var grpcAddress string

// Returns the GRPC address.
func GrpcAddress() string {
	if len(grpcAddress) == 0 {
		grpcAddress = utils.GetEnv("GRPC_SERVER_URI", "uc-aom:3800")
	}
	return grpcAddress
}
