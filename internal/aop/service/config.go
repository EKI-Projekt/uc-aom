// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

// Returns the supported OCI image OS.
func OS() string {
	return docker_os
}
