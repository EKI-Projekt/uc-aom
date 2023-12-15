// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server_test

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/catalogue"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

func TestUpdateAddOnFailTC1(t *testing.T) {
	// Arrange
	current := "1.0.0-1"
	future := "2.0.0-1"

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	updateStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	currentAddOn := catalogue.CatalogueAddOn{Name: "addOn", Version: current, Manifest: manifest.Root{Version: current, ManifestVersion: manifest.ValidManifestVersion, Platform: []string{"ucm"}}}
	futureGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: future}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("GetAddOn", "addOn").Return(currentAddOn, nil)
	mockObj.On("FetchManifest", futureGrpcAddOn.Name, futureGrpcAddOn.Version).Return(&currentAddOn.Manifest, nil)
	updateStreamMock.On("Send", mock.Anything).Return(nil)
	mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(fmt.Errorf("Delete Stack Failed")).Run(func(args mock.Arguments) {
		time.Sleep(time.Millisecond * 10)
	})

	updateReq := &grpc_api.UpdateAddOnRequest{
		AddOn: futureGrpcAddOn,
	}

	// Act
	uut.UpdateAddOn(updateReq, updateStreamMock)

	// Assert
	mockObj.AssertExpectations(t)
}

func TestCreateAddOnFailTC1(t *testing.T) {
	// Arrange
	install := "5.6.7-1"

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	createStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)
	installGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: install}
	installPullError := fmt.Errorf("Pull Failed")

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("DeleteAddOn", "addOn").Return(nil)
	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	mockObj.On("PullAddOn", "addOn", install).Return(catalogue.CatalogueAddOnWithImages{}, installPullError)
	createStreamMock.On("Send", mock.Anything).Return(nil)

	createReq := &grpc_api.CreateAddOnRequest{
		AddOn: installGrpcAddOn,
	}

	// Act
	err := uut.CreateAddOn(createReq, createStreamMock)

	// Assert
	if !strings.HasSuffix(err.Error(), installPullError.Error()) {
		t.Errorf("Expected %v, Actual %v", installPullError, err)
	}
	mockObj.AssertExpectations(t)

}

func TestCreateAddOnFailTC2(t *testing.T) {
	// Arrange
	install := "6.7.8-1"

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	createStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)
	addOn := newAddOn("addOn", install, "docker-image", "test-volume")
	installAddOn := catalogue.CatalogueAddOnWithImages{
		AddOn:           *addOn,
		DockerImageData: []io.Reader{},
	}
	installGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: install}

	createStackError := fmt.Errorf("Create Stack Failed")

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("DeleteAddOn", "addOn").Return(nil)
	mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(nil)
	mockObj.MockStackService.On("RemoveUnusedVolumes", "addOn", mock.Anything).Return(nil)
	mockObj.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	mockObj.On("PullAddOn", "addOn", install).Return(installAddOn, nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", "addOn", mock.AnythingOfType("string"), mock.Anything).Return(createStackError)
	createStreamMock.On("Send", mock.Anything).Return(nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(0xdeadbeef), nil)

	createReq := &grpc_api.CreateAddOnRequest{
		AddOn: installGrpcAddOn,
	}

	// Act
	err := uut.CreateAddOn(createReq, createStreamMock)

	// Assert
	if !strings.HasSuffix(err.Error(), createStackError.Error()) {
		t.Errorf("Expected %v, Actual %v", createStackError, err)
	}
	mockObj.AssertExpectations(t)
}

func TestCreateAddOnFailTC3(t *testing.T) {
	// Arrange
	install := "7.8.9-1"

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	createStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	addOn := newAddOn("addOn", install, "docker-image", "test-volume")
	installAddOn := catalogue.CatalogueAddOnWithImages{
		AddOn:           *addOn,
		DockerImageData: []io.Reader{},
	}
	installGrpcAddOn := &grpc_api.AddOn{Name: "addOn", Version: install}

	iamPermissionError := fmt.Errorf("IAM permission error")

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("DeleteAddOn", "addOn").Return(nil)
	mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(nil)
	mockObj.MockStackService.On("RemoveUnusedVolumes", "addOn", mock.Anything).Return(nil)
	mockObj.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil)
	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	mockObj.On("PullAddOn", "addOn", install).Return(installAddOn, nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", "addOn", mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterWrite", mock.Anything, mock.Anything).Return(iamPermissionError)
	mockObj.On("IamPermissionWriterDelete", mock.Anything).Return(nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(0xc0de), nil)
	createStreamMock.On("Send", mock.Anything).Return(nil)

	createReq := &grpc_api.CreateAddOnRequest{
		AddOn: installGrpcAddOn,
	}

	// Act
	err := uut.CreateAddOn(createReq, createStreamMock)

	// Assert
	if !strings.HasSuffix(err.Error(), iamPermissionError.Error()) {
		t.Errorf("Expected %v, Actual %v", iamPermissionError, err)
	}
	mockObj.AssertExpectations(t)
}

func TestCreateAddOnFailTC4(t *testing.T) {
	// Arrange
	install := "7.8.9.1"

	uut, mockObj, iamClientMock := server.NewServerUsingServiceMultiComponentMock()
	createStreamMock := server.NewAddOnResponseStreamMock(context.Background(), iamClientMock)

	addOn := newAddOn("addOn", install, "docker-image", "test-volume")
	dockerImages := dockerImages("docker-image")
	installAddOn := catalogue.CatalogueAddOnWithImages{AddOn: *addOn, DockerImageData: dockerImages}
	installGrpcAddOn := &grpc_api.AddOn{Name: addOn.Name, Version: addOn.Version}

	nginxRouteError := fmt.Errorf("NGINX route error")

	rollbackOrderActual := make([]string, 0)
	rollbackOrderExpected := []string{
		"ReverseProxyRemoveSymbolicLink",
		"ReverseProxyDelete",
		"ReverseProxyRemoveSymbolicLink",
		"ReverseProxyDelete",
		"IamPermissionWriterDelete",
		"DeleteAddOnStack",
		"RemoveUnusedVolumes",
		"DeleteDockerImages",
		"DeleteAddOn",
	}

	iamClientMock.On("IsAllowed", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(true, nil)
	mockObj.On("GetAddOn", "addOn").Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound).Once()
	mockObj.On("PullAddOn", addOn.Name, addOn.Version).Return(installAddOn, nil)
	mockObj.MockStackService.On("ImportDockerImage", dockerImages[0]).Return(nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", addOn.Name, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockObj.On("DeleteAddOn", "addOn").Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "DeleteAddOn")
	})
	mockObj.MockStackService.On("DeleteAddOnStack", "addOn").Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "DeleteAddOnStack")
	})
	mockObj.MockStackService.On("RemoveUnusedVolumes", "addOn", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "RemoveUnusedVolumes")
	})
	mockObj.MockStackService.On("DeleteDockerImages", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "DeleteDockerImages")
	})

	mockObj.On("IamPermissionWriterWrite", "/addOn-proxy.json", mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterDelete", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "IamPermissionWriterDelete")
	})

	mockObj.On("ReverseProxyWrite", "/addOn-publish.http.conf", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyWrite", "/addOn-publish-proxy.map", mock.Anything).Return(nginxRouteError)
	mockObj.On("ReverseProxyCreateSymbolicLink", "/addOn-publish.http.conf", "/addOn-publish.http.conf", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addOn-publish.http.conf").Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "ReverseProxyRemoveSymbolicLink")
	})
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addOn-publish-proxy.map").Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "ReverseProxyRemoveSymbolicLink")
	})
	mockObj.On("ReverseProxyDelete", "/addOn-publish.http.conf", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "ReverseProxyDelete")
	})
	mockObj.On("ReverseProxyDelete", "/addOn-publish-proxy.map", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		// Rollback
		rollbackOrderActual = append(rollbackOrderActual, "ReverseProxyDelete")
	})
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(1), nil)

	createStreamMock.On("Send", mock.Anything).Return(nil)

	createReq := &grpc_api.CreateAddOnRequest{
		AddOn: installGrpcAddOn,
	}

	// Act
	err := uut.CreateAddOn(createReq, createStreamMock)

	// Assert
	if !strings.HasSuffix(err.Error(), nginxRouteError.Error()) {
		t.Errorf("Expected %v, Actual %v", nginxRouteError, err)
	}

	if !reflect.DeepEqual(rollbackOrderActual, rollbackOrderExpected) {
		t.Errorf("Expected %#v, Actual %#v", rollbackOrderExpected, rollbackOrderActual)
	}

	mockObj.AssertExpectations(t)
}

func newAddOn(name string, version string, dockerImageName string, volumeName string) *catalogue.CatalogueAddOn {
	addOn := catalogue.CatalogueAddOn{Name: name, Version: version}
	serviceConfig := map[string]interface{}{"image": fmt.Sprintf("%s:%s", dockerImageName, version)}

	addOn.Manifest.ManifestVersion = manifest.ValidManifestVersion
	addOn.Manifest.Services = map[string]*manifest.Service{"test-service": {Type: "docker-compose", Config: serviceConfig}}
	addOn.Manifest.Publish = map[string]*manifest.ProxyRoute{"publish": {From: "from", To: "To"}}

	testEnvironment := manifest.NewEnvironment("")
	volumes := map[string]map[string]interface{}{volumeName: {"name": volumeName}}
	testEnvironment.WithVolumes(volumes)
	addOn.Manifest.Environments = map[string]*manifest.Environment{"": testEnvironment}
	addOn.Manifest.Platform = []string{"ucm"}
	return &addOn
}

func dockerImages(content ...string) []io.Reader {
	retValues := make([]io.Reader, 0, len(content))
	for _, payload := range content {
		retValues = append(retValues, strings.NewReader(payload))
	}
	return retValues
}
