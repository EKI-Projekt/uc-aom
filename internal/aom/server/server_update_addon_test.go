// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server_test

import (
	"context"
	"fmt"
	"io"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/aom/status"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

func TestUpdateAddOnVersionFail(t *testing.T) {
	type testCaseData struct {
		currentVersion string
		futureVersion  string
	}

	testCases := []testCaseData{
		{"1.0.0-1", "0.9.9-1"},
	}

	for i := 0; i < len(testCases); i++ {
		current := testCases[i].currentVersion
		future := testCases[i].futureVersion

		t.Run(fmt.Sprintf("Testing Version Upgrade [%s->%s]", testCases[i].currentVersion, testCases[i].futureVersion), func(t *testing.T) {
			// Arrange
			uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
			updateStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

			currentAddOn := catalogue.CatalogueAddOn{Name: "addOn", Version: current}
			futureGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: future}

			iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
			mockObj.On("GetAddOn", "addOn").Return(currentAddOn, nil)

			updateReq := &grpc_api.UpdateAddOnRequest{
				AddOn: futureGrpcAddOn,
			}

			// Act
			err := uut.UpdateAddOn(updateReq, updateStreamMock)
			if err == nil {
				t.Fatal("Expected error but got none.")
			}

			// Assert
			mockObj.AssertExpectations(t)
		})
	}
}

func TestUpdateAddOnVersionPass(t *testing.T) {
	type testCaseData struct {
		currentVersion string
		futureVersion  string
	}

	testCases := []testCaseData{
		{"1.0.0-1", "1.0.1-1"},
		{"1.0-1", "1.1-1"},
	}

	for i := 0; i < len(testCases); i++ {
		current := testCases[i].currentVersion
		future := testCases[i].futureVersion

		t.Run(fmt.Sprintf("Testing Version Upgrade [%s->%s]", testCases[i].currentVersion, testCases[i].futureVersion), func(t *testing.T) {
			// Arrange
			uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
			updateStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

			currentAddOn := catalogue.CatalogueAddOn{
				Name:    "addOn",
				Version: current,
				Manifest: manifest.Root{
					ManifestVersion: manifest.ValidManifestVersion,
					Version:         current,
					Platform:        []string{"ucm"},
				},
			}
			futureAddOn := catalogue.CatalogueAddOnWithImages{
				AddOn: catalogue.CatalogueAddOn{
					Name:    "addOn",
					Version: future,
					Manifest: manifest.Root{
						ManifestVersion: manifest.ValidManifestVersion,
						Version:         future,
						Platform:        []string{"ucm"},
					},
				},
				DockerImageData: []io.Reader{},
			}
			futureGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: future}

			iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
			mockObj.On("GetAddOn", "addOn").Return(currentAddOn, nil).Twice()
			updateStreamMock.On("Send", mock.Anything).Return(nil)
			mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(nil)
			mockObj.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
			mockObj.On("IamPermissionWriterDelete", mock.Anything).Return(nil)
			mockObj.On("DeleteAddOn", mock.Anything).Return(nil)
			mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
			mockObj.On("GetAddOn", "addOn").Return(futureAddOn.AddOn, nil)
			mockObj.On("PullAddOn", "addOn", future).Return(futureAddOn, nil)
			mockObj.MockStackService.On("CreateStackWithDockerCompose", "addOn", mock.AnythingOfType("string"), mock.Anything).Return(nil)
			mockObj.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(nil)
			mockObj.MockStackService.On("RemoveUnusedVolumes", "addOn", mock.Anything).Return(nil)
			mockObj.On("AddOnStatusResolver", "addOn").Return([]*status.ListAddOnContainersFuncReturnType{{Status: "(healthy)"}}, nil)
			mockObj.On("FetchManifest", futureAddOn.AddOn.Name, futureAddOn.AddOn.Version).Return(&futureAddOn.AddOn.Manifest, nil)
			mockObj.On("Validate", mock.Anything).Return(nil)
			mockObj.On("AvailableSpaceInBytes").Return(uint64(1), nil)

			updateReq := &grpc_api.UpdateAddOnRequest{
				AddOn: futureGrpcAddOn,
			}

			// Act
			err := uut.UpdateAddOn(updateReq, updateStreamMock)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Assert
			mockObj.AssertExpectations(t)
		})
	}
}
