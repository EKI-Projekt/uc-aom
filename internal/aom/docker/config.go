// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"path"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/pkg/utils"
)

var (
	// Root Path where the public volumes will be created.
	PUBLIC_VOLUMES_PATH = utils.GetEnv("PUBLIC_VOLUMES_PATH", path.Join(config.UC_AOM_STATE_DIRECTORY, "volumes-public"))
)

const (
	// Defines the current version of the stack artefact
	// 0.1.0 can not be determined by docker labels, these stacks was created by portainer.
	// 0.2.0 was included in 0.5.0 but had a migration bug because of prefilled volumes.
	// 0.2.1 fixed the volumes migration issue.

	StackVersion = "0.2.1"
)
