// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server_test

import (
	"context"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

func TestCreateAddOnPlatformFail(t *testing.T) {
	// Arrange
	current := "1.0.0-1"

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	updateStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	addOn := catalogue.CatalogueAddOn{Name: "addOn", Version: current, Manifest: manifest.Root{Version: current, Platform: []string{"ucg"}}}
	addOnWithImages := catalogue.CatalogueAddOnWithImages{AddOn: addOn}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	mockObj.On("PullAddOn", addOn.Name, addOn.Version).Return(addOnWithImages, nil)
	mockObj.On("DeleteAddOn", "addOn").Return(nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(0xdeadbeef), nil)
	updateStreamMock.On("Send", mock.Anything).Return(nil)

	createReq := &grpc_api.CreateAddOnRequest{
		AddOn: &grpc_api.AddOn{Name: addOn.Name, Version: addOn.Version},
	}

	// Act
	err := uut.CreateAddOn(createReq, updateStreamMock)

	// Assert
	if err == nil {
		t.Error("Expected error but none received")
	}

	mockObj.AssertExpectations(t)
}

func TestUpdateAddOnPlatformFail(t *testing.T) {
	// Arrange
	current := "1.0.0-1"
	future := "2.0.0-1"

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	updateStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	currentAddOn := catalogue.CatalogueAddOn{Name: "addOn", Version: current, Manifest: manifest.Root{Version: current, Platform: []string{"ucg"}}}
	futureGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: future}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("GetAddOn", futureGrpcAddOn.Name).Return(currentAddOn, nil)
	mockObj.On("FetchManifest", futureGrpcAddOn.Name, futureGrpcAddOn.Version).Return(&currentAddOn.Manifest, nil)
	updateStreamMock.On("Send", mock.Anything).Return(nil)

	updateReq := &grpc_api.UpdateAddOnRequest{
		AddOn: futureGrpcAddOn,
	}

	// Act
	err := uut.UpdateAddOn(updateReq, updateStreamMock)

	// Assert
	if err == nil {
		t.Error("Expected error but none received")
	}

	mockObj.AssertExpectations(t)
}
