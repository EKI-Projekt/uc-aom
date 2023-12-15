// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/aom/status"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

type createUpdateResponseStreamErrorOnLastSend struct {
	*server.AddOnResponseStreamMock
}

func (s createUpdateResponseStreamErrorOnLastSend) Send(addOn *grpc_api.AddOn) error {
	s.Called(addOn)
	if len(addOn.Name) != 0 {
		return errors.New("transport is closing")
	}
	return nil
}

func TestCreateAddOnStatusContextIndependent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())

	resumeSemaphore := make(chan int)
	stopDeleteAddOn := make(chan int)

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	createStreamMock := createUpdateResponseStreamErrorOnLastSend{server.NewAddOnResponseStreamMock(ctx, iamClientMock)}

	install := "2.3.4-1"
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
		DockerImageData: []io.Reader{strings.NewReader("docker-image")},
	}
	installGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: install}

	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	createStreamMock.On("Send", mock.Anything)
	mockObj.On("PullAddOn", "addOn", install).Return(installAddOn, nil).Run(func(args mock.Arguments) {
		// Signal that the transaction is open
		resumeSemaphore <- 1

		// block.
		<-stopDeleteAddOn
	})
	mockObj.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	mockObj.MockStackService.On("ImportDockerImage", mock.Anything).Return(nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", "addOn", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(0x2b), nil)

	createReq := &grpc_api.CreateAddOnRequest{
		AddOn: installGrpcAddOn,
	}

	go func() {
		err := uut.CreateAddOn(createReq, createStreamMock)
		if err != nil {
			t.Errorf("Unexpected error: %v.", err)
		}
		resumeSemaphore <- 1
	}()

	// Wait to be told that DeleteAddOnStack has been called.
	<-resumeSemaphore

	// Act
	cancel()
	stopDeleteAddOn <- 1
	<-resumeSemaphore

	mockObj.AssertExpectations(t)
}

func TestDeleteAddOnStatusContextIndependent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())

	resumeSemaphore := make(chan int)
	stopDeleteAddOn := make(chan int)

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	deleteStreamMock := server.NewEmptyResponseStreamMock(ctx, iamClientMock)

	current := "1.0.0-1"
	currentAddOn := catalogue.CatalogueAddOn{Name: "addOn", Manifest: manifest.Root{Title: "addOn title for test", Version: current, Platform: []string{"ucm"}}, Version: current}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
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

	deleteReq := &grpc_api.DeleteAddOnRequest{
		Name: "addOn",
	}

	go func() {
		err := uut.DeleteAddOn(deleteReq, deleteStreamMock)
		if err != nil {
			t.Errorf("Unexpected error: %v.", err)
		}
		resumeSemaphore <- 1
	}()

	// Wait to be told that DeleteAddOnStack has been called.
	<-resumeSemaphore

	// Act
	cancel()
	stopDeleteAddOn <- 1
	<-resumeSemaphore

	mockObj.AssertExpectations(t)
}

func TestUpdateAddOnStatusContextIndependent(t *testing.T) {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())

	resumeSemaphore := make(chan int)
	stopDeleteAddOn := make(chan int)

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	updateStreamMock := createUpdateResponseStreamErrorOnLastSend{server.NewAddOnResponseStreamMock(ctx, iamClientMock)}

	current := "1.0.0-1"
	future := "2.0.0-1"
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
	mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(nil)
	mockObj.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterDelete", mock.Anything).Return(nil)
	mockObj.On("DeleteAddOn", mock.Anything).Return(nil)
	mockObj.On("PullAddOn", "addOn", future).Return(futureAddOn, nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", "addOn", mock.AnythingOfType("string"), mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Signal that the transaction is open
		resumeSemaphore <- 1

		// block.
		<-stopDeleteAddOn
	})
	mockObj.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
	mockObj.MockStackService.On("RemoveUnusedVolumes", "addOn", mock.Anything).Return(nil)
	mockObj.On("AddOnStatusResolver", "addOn").Return([]*status.ListAddOnContainersFuncReturnType{{Status: "(healthy)"}}, nil)
	mockObj.On("FetchManifest", futureAddOn.AddOn.Name, futureAddOn.AddOn.Version).Return(&futureAddOn.AddOn.Manifest, nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(0x3b), nil)

	updateReq := &grpc_api.UpdateAddOnRequest{
		AddOn: futureGrpcAddOn,
	}

	go func() {
		err := uut.UpdateAddOn(updateReq, updateStreamMock)
		if err != nil {
			t.Errorf("Unexpected error: %v.", err)
		}
		resumeSemaphore <- 1
	}()

	// Wait to be told that DeleteAddOnStack has been called.
	<-resumeSemaphore

	// Act
	cancel()
	stopDeleteAddOn <- 1
	<-resumeSemaphore

	mockObj.AssertExpectations(t)
}
