// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"testing"
	"u-control/uc-aom/internal/cli"

	"github.com/stretchr/testify/assert"
)

func runCliCommand(t *testing.T, args ...string) error {
	t.Helper()
	tcmd := cli.NewAomCommand(nil)

	tcmd.SetArgs(args)
	return tcmd.Execute()
}

func TestExitStatusForEmptySubcommand(t *testing.T) {
	err := runCliCommand(t)
	assert.EqualError(t, err, "Please use a command")
}

func TestExitStatusForInvalidSubcommand(t *testing.T) {
	err := runCliCommand(t, "invalid")
	assert.EqualError(t, err, "uc-aom: 'invalid' is not a uc-aom command.\nSee 'uc-aom --help'")
}

func TestExitStatusForHelpCommand(t *testing.T) {
	err := runCliCommand(t, "--help")
	assert.NoError(t, err)
}
