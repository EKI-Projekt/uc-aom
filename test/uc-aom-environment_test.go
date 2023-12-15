// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package test

import (
	"context"
	"net"
	"testing"
	grpc_api "u-control/uc-aom/internal/aom/grpc"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type TestEnvironment struct {
	ctx    context.Context
	client grpc_api.AddOnServiceClient
	conn   *grpc.ClientConn
}

func newTestEnvironment(ctx context.Context, grpcTestListener *bufconn.Listener) (*TestEnvironment, error) {
	testEnvironment := &TestEnvironment{ctx: ctx}
	bufDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return grpcTestListener.Dial()
	})
	conn, err := grpc.DialContext(testEnvironment.ctx, "bufnet", bufDialer, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Errorf("Failed to dial bufnet: %v", err)
		return nil, err
	}
	testEnvironment.client = grpc_api.NewAddOnServiceClient(conn)
	testEnvironment.conn = conn
	return testEnvironment, nil
}

// Routine to create a addOn for testing
func (tenv *TestEnvironment) CreateTestAddOnRoutine(t *testing.T, addOnName string, addOnVersion string) *grpc_api.AddOn {
	addOn, err := createAddOn(tenv.client, tenv.ctx, &grpc_api.AddOn{Name: addOnName, Version: addOnVersion})
	if err != nil {
		t.Fatalf("Failed to create AddOn: %v", err)
	}

	// Return the installed AddOn
	addOn, err = getInstalledAddOn(tenv.client, tenv.ctx, addOn.Name, grpc_api.AddOnView_FULL)
	if err != nil {
		t.Fatalf("Failed to get the AddOn: %v", err)
	}
	return addOn
}

// Routine to delete a addOn from the test environment
func (tenv *TestEnvironment) DeleteTestAddOnRoutine(t *testing.T, addOn *grpc_api.AddOn) {
	// Delete the AddOn
	err := deleteAddOn(tenv.client, tenv.ctx, addOn)
	if err != nil {
		t.Fatalf("Failed to delete AddOn: %v", err)
	}
}

// Update a test addOn to the provided version and return it with the full information
func (tenv *TestEnvironment) UpdateTestAddOnRoutine(t *testing.T, addOnName string, addOnVersion string) *grpc_api.AddOn {
	addOn, err := tenv.UpdateTestAddOnRoutineWithError(addOnName, addOnVersion)
	if err != nil {
		t.Fatalf("Failed to update add-on: %v", err)
	}
	return addOn
}

// Update a test addOn to the provided version and return it with the full information otherwise the error
func (tenv *TestEnvironment) UpdateTestAddOnRoutineWithError(addOnName string, addOnVersion string) (*grpc_api.AddOn, error) {
	addOn, err := updateAddOn(tenv.client, tenv.ctx, &grpc_api.AddOn{Name: addOnName, Version: addOnVersion})
	if err != nil {
		return nil, err
	}
	// Return the installed AddOn
	addOn, err = getInstalledAddOn(tenv.client, tenv.ctx, addOn.Name, grpc_api.AddOnView_FULL)
	if err != nil {
		return nil, err
	}
	return addOn, nil
}
