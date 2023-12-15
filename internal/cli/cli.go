// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cli

import (
	"fmt"
	"u-control/uc-aom/internal/cli/cmd"

	grpc_api "u-control/uc-aom/internal/aom/grpc"

	"github.com/spf13/cobra"
)

func NewAomCommand(client grpc_api.AddOnServiceClient) *cobra.Command {
	topLevelCommand := &cobra.Command{
		Use:              "uc-aom COMMAND [ARG...]",
		Short:            "App manager for installing custom user apps on the u-OS",
		SilenceUsage:     true,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("Please use a command")
			}
			return fmt.Errorf("uc-aom: '%s' is not a uc-aom command.\nSee 'uc-aom --help'", args[0])

		},
		DisableFlagsInUseLine: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   false,
			HiddenDefaultCmd:    true,
			DisableDescriptions: true,
		},
	}
	cmd.AddCommands(topLevelCommand, client)
	return topLevelCommand
}
