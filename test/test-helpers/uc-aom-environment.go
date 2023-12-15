// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package testhelpers

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
	grpc_api "u-control/uc-aom/internal/aom/grpc"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type TestEnvironment struct {
	Ctx    context.Context
	Client grpc_api.AddOnServiceClient
	conn   *grpc.ClientConn
}

func NewTestEnvironment(ctx context.Context, grpcTestListener *bufconn.Listener) (*TestEnvironment, error) {
	testEnvironment := &TestEnvironment{Ctx: ctx}
	bufDialer := grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return grpcTestListener.Dial()
	})
	conn, err := grpc.DialContext(testEnvironment.Ctx,
		grpcTestListener.Addr().String(),
		bufDialer, grpc.WithInsecure(),
	)
	if err != nil {
		log.Errorf("Failed to dial bufnet: %v", err)
		return nil, err
	}
	testEnvironment.Client = grpc_api.NewAddOnServiceClient(conn)
	testEnvironment.conn = conn
	return testEnvironment, nil
}

func NewTestEnvironmentWithGrpcAddr(ctx context.Context, grpcAddr string) (*TestEnvironment, error) {
	conn, err := grpc.DialContext(ctx, grpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	// defer conn.Close()

	testEnvironment := &TestEnvironment{
		Ctx:    ctx,
		Client: grpc_api.NewAddOnServiceClient(conn),
		conn:   conn,
	}
	return testEnvironment, nil
}

func (tenv *TestEnvironment) CloseConnection() error {
	return tenv.conn.Close()
}

func (tenv *TestEnvironment) GetCatalogueAddOn(name string) (*grpc_api.AddOn, error) {
	req := &grpc_api.GetAddOnRequest{
		Name:   name,
		View:   grpc_api.AddOnView_FULL,
		Filter: grpc_api.GetAddOnRequest_CATALOGUE,
	}

	stream, err := tenv.Client.GetAddOn(tenv.Ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

func (tenv *TestEnvironment) GetCatalogueAddOnWithVersion(name string, version string) (*grpc_api.AddOn, error) {
	req := &grpc_api.GetAddOnRequest{
		Name:    name,
		Version: version,
		View:    grpc_api.AddOnView_FULL,
		Filter:  grpc_api.GetAddOnRequest_CATALOGUE,
	}

	stream, err := tenv.Client.GetAddOn(tenv.Ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

func (tenv *TestEnvironment) GetCatalogue() (*grpc_api.ListAddOnsResponse, error) {
	return listAddOns(tenv, grpc_api.ListAddOnsRequest_CATALOGUE)
}

func (tenv *TestEnvironment) GetInstalledAddOns() (*grpc_api.ListAddOnsResponse, error) {
	return listAddOns(tenv, grpc_api.ListAddOnsRequest_INSTALLED)
}

func (tenv *TestEnvironment) GetInstalledAddOn(name string) (*grpc_api.AddOn, error) {
	req := &grpc_api.GetAddOnRequest{
		Name:   name,
		View:   grpc_api.AddOnView_FULL,
		Filter: grpc_api.GetAddOnRequest_INSTALLED,
	}

	stream, err := tenv.Client.GetAddOn(tenv.Ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

func (tenv *TestEnvironment) GetInstalledAddOnStatus(addOn *grpc_api.AddOn) (grpc_api.AddOnStatus, error) {
	channel := make(chan struct {
		status grpc_api.AddOnStatus
		err    error
	})
	go func() {
		retries := 5
		var lastError error
		for ; retries > 0; retries-- {
			returnAddOn, err := tenv.GetInstalledAddOn(addOn.Name)
			if err != nil {
				lastError = err
			}

			if returnAddOn != nil && returnAddOn.Status != grpc_api.AddOnStatus_STARTING {
				channel <- struct {
					status grpc_api.AddOnStatus
					err    error
				}{returnAddOn.Status, nil}
				return
			}
			time.Sleep(time.Second * 1)
		}

		if lastError == nil {
			lastError = fmt.Errorf("Failed to get installed AddOn after %d retries", retries)
		}

		channel <- struct {
			status grpc_api.AddOnStatus
			err    error
		}{grpc_api.AddOnStatus_ERROR, lastError}

	}()
	result := <-channel
	return result.status, result.err
}

func (tenv *TestEnvironment) CreateAddOn(addOnName string, addOnVersion string, settings ...*grpc_api.Setting) (*grpc_api.AddOn, error) {
	req := &grpc_api.CreateAddOnRequest{
		AddOn: &grpc_api.AddOn{Name: addOnName, Version: addOnVersion, Settings: settings},
	}

	stream, err := tenv.Client.CreateAddOn(tenv.Ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

func (tenv *TestEnvironment) UpdateAddOn(addOn *grpc_api.AddOn) (*grpc_api.AddOn, error) {
	req := &grpc_api.UpdateAddOnRequest{
		AddOn: addOn,
	}

	stream, err := tenv.Client.UpdateAddOn(tenv.Ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

// Routine to create a addOn for testing
func (tenv *TestEnvironment) CreateTestAddOnRoutine(t *testing.T, addOnName string, addOnVersion string, settings ...*grpc_api.Setting) *grpc_api.AddOn {
	addOn, err := tenv.CreateAddOn(addOnName, addOnVersion, settings...)
	if err != nil {
		t.Fatalf("Failed to create AddOn: %v", err)
	}

	// Return the installed AddOn
	addOn, err = tenv.GetInstalledAddOn(addOn.Name)
	if err != nil {
		t.Fatalf("Failed to get the AddOn: %v", err)
	}
	return addOn
}

// Routine to delete a addOn from the test environment
func (tenv *TestEnvironment) DeleteTestAddOnRoutine(t *testing.T, addOn *grpc_api.AddOn) {
	// Delete the AddOn
	err := DeleteAddOn(tenv.Client, tenv.Ctx, addOn)
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
	addOn, err := tenv.UpdateAddOn(&grpc_api.AddOn{Name: addOnName, Version: addOnVersion})
	if err != nil {
		return nil, err
	}
	// Return the installed AddOn
	addOn, err = tenv.GetInstalledAddOn(addOn.Name)
	if err != nil {
		return nil, err
	}
	return addOn, nil
}
