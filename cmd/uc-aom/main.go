// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package main

import (
	"net"
	"u-control/uc-aom/internal/aom/cmd"
	"u-control/uc-aom/internal/aom/utils"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func handleArguments() {
	var verbosity *int = flag.CountP("verbose", "v", "explain what is being done.")
	flag.Parse()

	switch *verbosity {
	case 0:
		log.SetLevel(log.WarnLevel)
	case 1:
		log.SetLevel(log.InfoLevel)
	case 2:
		log.SetLevel(log.DebugLevel)
	default:
		log.SetLevel(log.TraceLevel)
	}
}

func main() {
	handleArguments()

	grpcAddress := utils.GetEnv("GRPC_SERVER_URI", "uc-aom:3800")
	grpcListener, portErr := net.Listen("tcp", grpcAddress)
	if portErr != nil {
		log.Fatalf("Grpc port error: %v", portErr)
	}

	ucAom := cmd.NewUcAom(grpcListener)
	if err := ucAom.Setup(); err != nil {
		log.Fatalf("Aom.Setup() failed with error: %v", err)
	}

	if err := ucAom.Run(); err != nil {
		log.Fatalf("Server exited with error: %v", err)
	}
}
