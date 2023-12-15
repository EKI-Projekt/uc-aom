// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"os"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/cli"
	"u-control/uc-aom/internal/cli/config"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

func main() {
	grpcAddress := config.GrpcAddress()
	conn, err := grpc.Dial(grpcAddress, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Couldn't connect to daemon: %v", err)
	}
	defer conn.Close()
	client := grpc_api.NewAddOnServiceClient(conn)

	if err := runUcAom(client); err != nil {
		log.Errorln(err)
		os.Exit(1)
	}
}

func runUcAom(client grpc_api.AddOnServiceClient) error {
	cmd := cli.NewAomCommand(client)
	setDefaultCmdLogOutput(cmd)
	return cmd.Execute()
}

func setDefaultCmdLogOutput(cmd *cobra.Command) {
	commandOutput := cmd.OutOrStdout()
	log.SetOutput(commandOutput)
}
