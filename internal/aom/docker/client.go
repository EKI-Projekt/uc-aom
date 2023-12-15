// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package docker

import (
	"github.com/docker/cli/cli/command"
	cliflags "github.com/docker/cli/cli/flags"
	log "github.com/sirupsen/logrus"
)

// NewDockerCli creates a new DockerCli
func NewDockerCli() (command.Cli, error) {
	dockerCli, err := command.NewDockerCli(command.WithStandardStreams())

	if err != nil {
		return nil, err
	}

	clientOptions := cliflags.NewClientOptions()
	clientOptions.Common.LogLevel = log.GetLevel().String()
	err = dockerCli.Initialize(clientOptions)
	if err != nil {
		return nil, err
	}
	return dockerCli, nil
}
