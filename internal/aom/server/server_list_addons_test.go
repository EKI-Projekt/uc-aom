// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/dbus"
	"u-control/uc-aom/internal/aom/env"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/iam"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/service"
	addonstatus "u-control/uc-aom/internal/aom/status"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

var manifestJson = `
{
  "manifestVersion": "0.1",
  "version": "%s",
  "title": "%s",
  "description": "Description",
  "logo": "logo.png",
  "services": {
    "cloudadapter": {
      "type": "docker-compose",
      "config": {
        "image": "docker/image",
        "restart": "no",
        "containerName": "abc",
        "ports": ["127.0.0.1:8888:8888"]
      }
    }
  },
  "platform": ["ucg", "ucm"],
  "vendor": {
    "name": "Weidmüller GmbH & Co KG",
    "url": "https://www.weidmueller.de",
    "email": "datenschutz@weidmueller.de",
    "street": "Klingenbergstraße 26",
    "zip": "32758",
    "city": "Detmold",
    "country": "Germany"
  }
}
`

type remoteCatalogueMock struct {
	mock.Mock
}

func (r *remoteCatalogueMock) GetAddOnNames() ([]string, error) {
	return nil, nil
}

func (r *remoteCatalogueMock) GetAddOnVersions(name string) ([]string, error) {
	return nil, nil
}

func (r *remoteCatalogueMock) GetAddOn(name string, version string) (catalogue.CatalogueAddOn, error) {
	return catalogue.CatalogueAddOn{}, nil
}

func (r *remoteCatalogueMock) GetLatestAddOns() ([]*catalogue.CatalogueAddOn, error) {
	args := r.Called()
	return args.Get(0).([]*catalogue.CatalogueAddOn), args.Error(1)
}

func mockCatalogueAddons(name string, version string) []*catalogue.CatalogueAddOn {
	manifestJson := fmt.Sprintf(manifestJson, version, name)
	var root manifest.Root
	json.Unmarshal([]byte(manifestJson), &root)
	addOn := &catalogue.CatalogueAddOn{
		Name:     name,
		Version:  version,
		Manifest: root,
	}
	return []*catalogue.CatalogueAddOn{addOn}
}

func mockService(t *testing.T) *service.Service {
	manifestValidator, err := manifest.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	mockObj := &service.ServiceMultiComponentMock{}
	reverseProxy := routes.NewReverseProxy(dbus.Initialize(), "", "", "", "",
		mockObj.ReverseProxyWrite,
		mockObj.ReverseProxyDelete,
		mockObj.ReverseProxyCreateSymbolicLink,
		mockObj.ReverseProxyRemoveSymbolicLink)
	iamPermissionWriter := iam.NewIamPermissionWriter("", mockObj.IamPermissionWriterWrite, mockObj.IamPermissionWriterDelete)
	service := service.NewService(&mockObj.MockStackService, reverseProxy, iamPermissionWriter, mockObj, manifestValidator, mockObj, nil)
	return service
}

func TestAddOnServer_ListAddOns(t *testing.T) {
	type fields struct {
		service                  *service.Service
		stackCreateTimeout       time.Duration
		addonsAssetsLocalPath    string
		addonsAssetsRemotePath   string
		localCatalogue           catalogue.LocalAddOnCatalogue
		remoteCatalogue          *remoteCatalogueMock
		iamServiceUcAomClient    iam.IamClient
		iamServiceUcAuthClient   iam.IamClient
		addOnStatusResolver      *addonstatus.AddOnStatusResolver
		addOnEnvironmentResolver *env.AddOnEnvironmentResolver
		transactionScheduler     *service.TransactionScheduler
	}
	type args struct {
		request *grpc_api.ListAddOnsRequest
		stream  *ListAddOnResponseStreamMock
	}

	// Use a buffered channel so that we don't block when sending
	captureListAddOnResult := make(chan *grpc_api.ListAddOnsResponse, 2)

	listAddOnArgs := args{
		request: &grpc_api.ListAddOnsRequest{
			Filter: grpc_api.ListAddOnsRequest_CATALOGUE,
		},
		stream: &ListAddOnResponseStreamMock{
			capture: captureListAddOnResult,
		},
	}

	uutFields := fields{
		service:                  mockService(t),
		stackCreateTimeout:       10 * time.Second,
		addonsAssetsLocalPath:    "",
		addonsAssetsRemotePath:   "",
		localCatalogue:           nil,
		remoteCatalogue:          nil,
		iamServiceUcAomClient:    nil,
		iamServiceUcAuthClient:   nil,
		addOnStatusResolver:      nil,
		addOnEnvironmentResolver: nil,
		transactionScheduler:     nil,
	}

	tests := []struct {
		name         string
		fields       fields
		args         args
		wantErr      bool
		addOnName    string
		addOnVersion string
		isValid      bool
	}{
		{
			name:         "Valid addons",
			fields:       uutFields,
			args:         listAddOnArgs,
			wantErr:      false,
			addOnName:    "abc",
			addOnVersion: "0.1.0-1",
			isValid:      true,
		},
		{
			name:         "Invalid addons",
			fields:       uutFields,
			args:         listAddOnArgs,
			wantErr:      false,
			addOnName:    "xyz",
			addOnVersion: "0.1",
			isValid:      false,
		},
	}
	for _, tt := range tests {
		tt.fields.remoteCatalogue = &remoteCatalogueMock{}
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			s := &AddOnServer{
				service:                  tt.fields.service,
				stackCreateTimeout:       tt.fields.stackCreateTimeout,
				addonsAssetsLocalPath:    tt.fields.addonsAssetsLocalPath,
				addonsAssetsRemotePath:   tt.fields.addonsAssetsRemotePath,
				localCatalogue:           tt.fields.localCatalogue,
				remoteCatalogue:          tt.fields.remoteCatalogue,
				iamServiceUcAomClient:    tt.fields.iamServiceUcAomClient,
				iamServiceUcAuthClient:   tt.fields.iamServiceUcAuthClient,
				addOnStatusResolver:      tt.fields.addOnStatusResolver,
				addOnEnvironmentResolver: tt.fields.addOnEnvironmentResolver,
				transactionScheduler:     tt.fields.transactionScheduler,
			}

			mockedAddons := mockCatalogueAddons(tt.addOnName, tt.addOnVersion)
			tt.fields.remoteCatalogue.On("GetLatestAddOns").Return(mockedAddons, nil)
			tt.args.stream.On("Send", mock.Anything).Return(nil)

			// Act
			if err := s.ListAddOns(tt.args.request, tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("AddOnServer.ListAddOns() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Read the result
			var addOns []*grpc_api.AddOn
			for addOns == nil {
				result := <-captureListAddOnResult
				addOns = result.GetAddOns()
			}

			// Assert
			if tt.isValid {
				if len(addOns) == 0 {
					t.Error("Expected to have a valid addon but got none")
				}
				if addOns[0].Name != tt.addOnName {
					t.Errorf("Expected addon name to be  %s but got %s", tt.addOnName, addOns[0].Name)
				}
				if addOns[0].Version != tt.addOnVersion {
					t.Errorf("Expected addon version to be  %s but got %s", tt.addOnVersion, addOns[0].Version)
				}
			}

			if !tt.isValid {
				if len(addOns) > 0 {
					t.Errorf("Expected addon list to be empty but got %s", addOns[0].Name)
				}
			}
		})
	}
}
