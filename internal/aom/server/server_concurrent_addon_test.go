// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server_test

import (
	"context"
	"io"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/catalogue"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/aom/status"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCreateAddOnStatusConcurrentGetAddOn(t *testing.T) {
	// Arrange
	install := "2.3.4-1"

	stopCreateAddOn := make(chan int)
	resumeSemaphore := make(chan int)

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	createStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	installAddOn := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "addOn",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "addOn title for test",
				Version:         install,
				Platform:        []string{"ucm"},
			},
			Version: install,
		},
		DockerImageData: []io.Reader{},
	}
	installGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: install}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	createStreamMock.On("Send", mock.Anything).Return(nil)
	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	mockObj.On("PullAddOn", "addOn", install).Return(installAddOn, nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", "addOn", mock.AnythingOfType("string"), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Signal that the transaction is open
		resumeSemaphore <- 1

		// block.
		<-stopCreateAddOn
	})
	mockObj.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(0xdeadbeef), nil)

	createReq := &grpc_api.CreateAddOnRequest{
		AddOn: installGrpcAddOn,
	}

	go func() {
		err := uut.CreateAddOn(createReq, createStreamMock)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		resumeSemaphore <- 1
	}()

	// Wait to be told that CreateStackWithDockerCompose has been called.
	<-resumeSemaphore

	// Act
	addOn, err := callGetAddOn(uut, iamClientMock, installAddOn.AddOn.Name)

	// Assert
	assert.NoError(t, err)

	if addOn.Status != grpc_api.AddOnStatus_INSTALLING {
		t.Errorf("Expected status %v, Actual %v", grpc_api.AddOnStatus_INSTALLING, addOn.Status)
	}
	if addOn.Name != installAddOn.AddOn.Name {
		t.Errorf("Expected %s, Actual %s", installAddOn.AddOn.Name, addOn.Name)
	}
	if addOn.Title != installAddOn.AddOn.Manifest.Title {
		t.Errorf("Expected %s, Actual %s", installAddOn.AddOn.Manifest.Title, addOn.Title)
	}

	stopCreateAddOn <- 1
	<-resumeSemaphore

	mockObj.AssertExpectations(t)
}

func TestDeleteAddOnStatusConcurrentGetAddOn(t *testing.T) {
	// Arrange
	current := "1.0.0-1"

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second*2))

	resumeSemaphore := make(chan int)
	stopDeleteAddOn := make(chan int)

	// Use a buffered channel so that we don't block when sending
	captureListAddOnResult := make(chan *grpc_api.ListAddOnsResponse, 100)

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	deleteStreamMock := server.NewEmptyResponseStreamMock(ctx, iamClientMock)
	listStreamMock := server.NewListAddOnResponseStreamMock(ctx, iamClientMock, captureListAddOnResult)

	currentAddOn := catalogue.CatalogueAddOn{Name: "addOn", Manifest: manifest.Root{Title: "addOn title for test", Version: current, Platform: []string{"ucm"}}, Version: current}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	listStreamMock.On("Send", mock.Anything).Return(nil)
	mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(nil).Run(func(args mock.Arguments) {
		// Signal that the transaction is open
		resumeSemaphore <- 1

		// block.
		<-stopDeleteAddOn
	})
	mockObj.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterDelete", mock.Anything).Return(nil)
	mockObj.On("DeleteAddOn", mock.Anything).Return(nil)
	mockObj.MockStackService.On("RemoveUnusedVolumes", "addOn", mock.Anything).Return(nil)
	mockObj.On("GetAddOn", "addOn").Return(currentAddOn, nil)
	deleteStreamMock.On("Send", mock.Anything).Return(nil)
	mockObj.On("GetAddOns").Return([]*catalogue.CatalogueAddOn{&currentAddOn}, nil)

	deleteReq := &grpc_api.DeleteAddOnRequest{
		Name: "addOn",
	}

	listReq := &grpc_api.ListAddOnsRequest{
		Filter: grpc_api.ListAddOnsRequest_INSTALLED,
	}

	go func() {
		err := uut.DeleteAddOn(deleteReq, deleteStreamMock)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		resumeSemaphore <- 1
	}()

	// Wait to be told that CreateStackWithDockerCompose has been called.
	<-resumeSemaphore

	// Act
	err := uut.ListAddOns(listReq, listStreamMock)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Read the result
	var addOns []*grpc_api.AddOn
	for addOns == nil {
		result := <-captureListAddOnResult
		addOns = result.GetAddOns()
	}

	if len(addOns) != 1 {
		t.Errorf("Expected exactly one Add-on, Actual: %#v", addOns)
	}
	addOn := addOns[0]

	if addOn.Status != grpc_api.AddOnStatus_DELETING {
		t.Errorf("Expected status %v, Actual %v", grpc_api.AddOnStatus_DELETING, addOn.Status)
	}
	if addOn.Name != currentAddOn.Name {
		t.Errorf("Expected %s, Actual %s", currentAddOn.Name, addOn.Name)
	}
	if addOn.Title != currentAddOn.Manifest.Title {
		t.Errorf("Expected %s, Actual %s", currentAddOn.Manifest.Title, addOn.Title)
	}

	stopDeleteAddOn <- 1
	<-resumeSemaphore

	mockObj.AssertExpectations(t)
	cancel()
}

func TestUpdateAddOnStatusConcurrentGetAddOn(t *testing.T) {
	// Arrange
	current := "1.0.0-1"
	future := "2.0.0-1"

	stopDeleteAddOn := make(chan int)
	resumeSemaphore := make(chan int)

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	updateStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	currentAddOn := catalogue.CatalogueAddOn{
		Name: "addOn",
		Manifest: manifest.Root{
			ManifestVersion: manifest.ValidManifestVersion,
			Title:           "old addOn title for test",
			Version:         current,
			Platform:        []string{"ucm"},
		},
		Version: current,
	}
	futureAddOn := catalogue.CatalogueAddOnWithImages{
		AddOn: catalogue.CatalogueAddOn{
			Name: "addOn",
			Manifest: manifest.Root{
				ManifestVersion: manifest.ValidManifestVersion,
				Title:           "new addOn title for test",
				Version:         future,
				Platform:        []string{"ucm"},
			},
			Version: future,
		},
		DockerImageData: []io.Reader{},
	}
	futureGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: future}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("GetAddOn", "addOn").Return(currentAddOn, nil).Twice()
	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	mockObj.On("GetAddOn", "addOn").Return(futureAddOn.AddOn, nil)
	updateStreamMock.On("Send", mock.Anything).Return(nil)
	mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(nil).Run(func(args mock.Arguments) {
		// Signal that the transaction is open
		resumeSemaphore <- 1

		// block.
		<-stopDeleteAddOn
	})
	mockObj.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterDelete", mock.Anything).Return(nil)
	mockObj.On("DeleteAddOn", mock.Anything).Return(nil)
	mockObj.On("PullAddOn", "addOn", future).Return(futureAddOn, nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", "addOn", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	mockObj.MockStackService.On("RemoveUnusedVolumes", "addOn", mock.Anything).Return(nil)
	mockObj.On("AddOnStatusResolver", "addOn").Return([]*status.ListAddOnContainersFuncReturnType{{Status: "(healthy)"}}, nil)
	mockObj.On("FetchManifest", futureAddOn.AddOn.Name, futureAddOn.AddOn.Version).Return(&futureAddOn.AddOn.Manifest, nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(0xdeadbeef), nil)

	updateReq := &grpc_api.UpdateAddOnRequest{
		AddOn: futureGrpcAddOn,
	}

	go func() {
		err := uut.UpdateAddOn(updateReq, updateStreamMock)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		resumeSemaphore <- 1
	}()

	// Wait to be told that CreateStackWithDockerCompose has been called.
	<-resumeSemaphore

	// Act
	addOn, err := callGetAddOn(uut, iamClientMock, currentAddOn.Name)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if addOn.Status != grpc_api.AddOnStatus_UPDATING {
		t.Errorf("Expected status %v, Actual %v", grpc_api.AddOnStatus_UPDATING, addOn.Status)
	}
	if addOn.Name != currentAddOn.Name {
		t.Errorf("Expected %s, Actual %s", currentAddOn.Name, addOn.Name)
	}
	if addOn.Title != currentAddOn.Manifest.Title {
		t.Errorf("Expected %s, Actual %s", currentAddOn.Manifest.Title, addOn.Title)
	}

	stopDeleteAddOn <- 1
	<-resumeSemaphore

	mockObj.AssertExpectations(t)
}

func callGetAddOn(uut *server.AddOnServer, iamClientMock *server.IamClientMock, addOnName string) (*grpc_api.AddOn, error) {
	getReq := &grpc_api.GetAddOnRequest{
		Name:   addOnName,
		Filter: grpc_api.GetAddOnRequest_INSTALLED,
	}

	getAddOnStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	var addOn *grpc_api.AddOn
	getAddOnStreamMock.On("Send", mock.Anything).Run(func(args mock.Arguments) {
		addOn = args.Get(0).(*grpc_api.AddOn)
	}).Return(nil)

	err := uut.GetAddOn(getReq, getAddOnStreamMock)
	return addOn, err
}
