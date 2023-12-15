// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"strconv"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/aom/network"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddOnStarter_StartInstalledAddOns(t *testing.T) {
	type fields struct {
		localCatalogue           catalogue.LocalAddOnCatalogue
		stackService             docker.StackServiceAPI
		externalNetworkConnector network.ExternalNetworkConnector
	}

	installedAddOns := []*catalogue.CatalogueAddOn{
		{
			Name: "firstAddOn", Manifest: manifest.Root{
				Version:  "1.0.0-1",
				Title:    "addOn title",
				Platform: []string{"ucm"},
			},
			Version: "1.0.0-1",
		},
		{
			Name: "secondAddOn", Manifest: manifest.Root{
				Version:  "1.0.0-1",
				Title:    "addOn title",
				Platform: []string{"ucm"},
			},
			Version: "1.0.0-1",
		},
		{
			Name: "thirdAddOn", Manifest: manifest.Root{
				Version:  "1.0.0-1",
				Title:    "addOn title",
				Platform: []string{"ucm"},
			},
			Version: "1.0.0-1",
		},
	}

	localCatalogueMock := &catalogue.CatalogueMock{}
	localCatalogueMock.On("GetAddOns").Return(installedAddOns, nil)
	mockService := &docker.MockStackService{}
	mockExternalNetworkConnector := &network.MockExternalNetworkConnector{}
	mockExternalNetworkConnector.On("Initialize").Return(nil)
	for _, addOn := range installedAddOns {
		mockExternalNetworkConnector.On("IsConnected", &addOn.Manifest).Return(false)
		mockService.On("StartupStackNonBlocking", addOn.Name).Return(nil).Once()
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Shall call startup for all installed add-ons",
			fields: fields{
				localCatalogue:           localCatalogueMock,
				stackService:             mockService,
				externalNetworkConnector: mockExternalNetworkConnector,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AddOnStarter{
				localCatalogue:           tt.fields.localCatalogue,
				stackService:             tt.fields.stackService,
				externalNetworkConnector: tt.fields.externalNetworkConnector,
			}
			if err := s.StartInstalledAddOns(); (err != nil) != tt.wantErr {
				t.Errorf("AddOnStarter.StartInstalledAddOns() error = %v, wantErr %v", err, tt.wantErr)
			}

			mock.AssertExpectationsForObjects(t, tt.fields.localCatalogue)
			mock.AssertExpectationsForObjects(t, tt.fields.stackService)
			mock.AssertExpectationsForObjects(t, tt.fields.externalNetworkConnector)
		})
	}
}

func TestAddOnStarter_StartInstalledAddOnsWithExternalNetwork(t *testing.T) {
	type fields struct {
		localCatalogue           catalogue.LocalAddOnCatalogue
		stackService             docker.StackServiceAPI
		externalNetworkConnector network.ExternalNetworkConnector
	}

	installedAddOns := []*catalogue.CatalogueAddOn{
		{
			Name: "addOnWithExternalNetwork", Manifest: manifest.Root{
				Version:  "1.0.0-1",
				Title:    "addOn title",
				Platform: []string{"ucm"},
			},
			Version: "1.0.0-1",
		},
	}

	localCatalogueMock := &catalogue.CatalogueMock{}
	localCatalogueMock.On("GetAddOns").Return(installedAddOns, nil)
	mockService := &docker.MockStackService{}
	mockExternalNetworkConnector := &network.MockExternalNetworkConnector{}
	mockExternalNetworkConnector.On("Initialize").Return(nil)
	for i, addOn := range installedAddOns {
		addOnContainers := []types.Container{{ID: strconv.Itoa(i)}}
		mockExternalNetworkConnector.On("IsConnected", &addOn.Manifest).Return(true).Once()
		mockExternalNetworkConnector.On("Reconnect", addOnContainers).Return(nil).Once()
		mockService.On("ListAllStackContainers", addOn.Name).Return(addOnContainers, nil).Once()
		mockService.On("StartupStackNonBlocking", addOn.Name).Return(nil).Once()
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Shall call startup for all installed add-ons",
			fields: fields{
				localCatalogue:           localCatalogueMock,
				stackService:             mockService,
				externalNetworkConnector: mockExternalNetworkConnector,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AddOnStarter{
				localCatalogue:           tt.fields.localCatalogue,
				stackService:             tt.fields.stackService,
				externalNetworkConnector: tt.fields.externalNetworkConnector,
			}
			if err := s.StartInstalledAddOns(); (err != nil) != tt.wantErr {
				t.Errorf("AddOnStarter.StartInstalledAddOns() error = %v, wantErr %v", err, tt.wantErr)
			}

			mock.AssertExpectationsForObjects(t, tt.fields.localCatalogue)
			mock.AssertExpectationsForObjects(t, tt.fields.stackService)
			mock.AssertExpectationsForObjects(t, tt.fields.externalNetworkConnector)
		})
	}
}

func TestShallReturnErrorIfNetworkInitializedReturnsError(t *testing.T) {
	// Arrange
	localCatalogueMock := &catalogue.CatalogueMock{}
	localCatalogueMock.On("GetAddOns").Return([]*catalogue.CatalogueAddOn{}, nil)
	mockService := &docker.MockStackService{}
	mockExternalNetworkConnector := &network.MockExternalNetworkConnector{}

	wantErr := errors.New("InitError")
	mockExternalNetworkConnector.On("Initialize").Return(wantErr)

	s := &AddOnStarter{
		localCatalogue:           localCatalogueMock,
		stackService:             mockService,
		externalNetworkConnector: mockExternalNetworkConnector,
	}

	// Act
	gotErr := s.StartInstalledAddOns()

	// Assert
	assert.ErrorIs(t, gotErr, wantErr)

}
