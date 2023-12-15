// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	grpc_api "u-control/uc-aom/internal/aom/grpc"

	"github.com/spf13/cobra"
)

// AddCommands adds all the commands from cmd to the root command
func AddCommands(rootCmd *cobra.Command, client grpc_api.AddOnServiceClient) {
	rootCmd.AddCommand(
		NewListCommand(client),
	)
}
