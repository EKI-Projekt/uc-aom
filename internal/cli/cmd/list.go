// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
	"io"
	grpc_api "u-control/uc-aom/internal/aom/grpc"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewListCommand(client grpc_api.AddOnServiceClient) *cobra.Command {
	listCmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all apps",
		Run: func(cmd *cobra.Command, args []string) {
			runList(client)
		},
	}

	return listCmd
}

func runList(client grpc_api.AddOnServiceClient) {
	ctx := context.Background()
	r, err := client.ListAddOns(ctx, &grpc_api.ListAddOnsRequest{})
	if err != nil {
		log.Fatalf("Requesting AddOn list failed:: %v", err)
	}

	done := make(chan bool)

	go func() {
		for {
			listAddOnResp, err := r.Recv()
			if err == io.EOF {
				done <- true //close(done)
				return
			}
			if err != nil {
				log.Fatalf("Receiving  AddOn list failed: %v", err)
			}
			if len(listAddOnResp.AddOns) > 0 {
				log.Info("%s", listAddOnResp.AddOns)
			}
		}
	}()

	<-done
}
