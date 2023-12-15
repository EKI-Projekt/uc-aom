// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/dbus"
	"u-control/uc-aom/internal/aom/iam"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/service"
	"u-control/uc-aom/internal/pkg/manifest"

	"google.golang.org/grpc/codes"
	grpcStatus "google.golang.org/grpc/status"

	"github.com/stretchr/testify/mock"
)

func newAddOn(name string, title string, version string, dockerImageName string, volumeName string) *catalogue.CatalogueAddOn {
	addOn := catalogue.CatalogueAddOn{Name: name, Version: version}
	serviceConfig := map[string]interface{}{"image": fmt.Sprintf("%s:%s", dockerImageName, version)}
	addOn.Manifest.Title = title
	addOn.Manifest.ManifestVersion = manifest.ValidManifestVersion
	addOn.Manifest.Services = map[string]*manifest.Service{"test-service": {Type: "docker-compose", Config: serviceConfig}}
	addOn.Manifest.Publish = map[string]*manifest.ProxyRoute{"publish": {From: "/from", To: "/To"}}

	testEnvironment := manifest.NewEnvironment("")
	volumes := map[string]map[string]interface{}{volumeName: {"name": volumeName}}
	testEnvironment.WithVolumes(volumes)
	addOn.Manifest.Environments = map[string]*manifest.Environment{"": testEnvironment}
	addOn.Manifest.Platform = []string{"ucg", "ucm"}
	return &addOn
}

func dockerImages(content ...string) []io.Reader {
	retValues := make([]io.Reader, 0, len(content))
	for _, payload := range content {
		retValues = append(retValues, strings.NewReader(payload))
	}
	return retValues
}

func createUut(tc *service.ServiceMultiComponentMock) *service.Service {
	reverseProxy := routes.NewReverseProxy(dbus.Initialize(), "", "", "", "",
		tc.ReverseProxyWrite,
		tc.ReverseProxyDelete,
		tc.ReverseProxyCreateSymbolicLink,
		tc.ReverseProxyRemoveSymbolicLink)
	iamPermissionWriter := iam.NewIamPermissionWriter("", tc.IamPermissionWriterWrite, tc.IamPermissionWriterDelete)
	return service.NewService(tc, reverseProxy, iamPermissionWriter, tc, tc, tc, tc)
}

func TestCreateAddOnRoutineSuccess(t *testing.T) {
	// Arrange
	mockObj := &service.ServiceMultiComponentMock{}
	addOn := newAddOn("addontest", "add-on-test", "4.3.2", "docker-image", "test-volume")
	dockerImages := dockerImages("docker-image")
	uut := createUut(mockObj)

	addOnWithDockerImages := catalogue.CatalogueAddOnWithImages{AddOn: *addOn, DockerImageData: dockerImages}

	mockObj.On("GetAddOn", addOn.Name).Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	mockObj.On("PullAddOn", addOn.Name, addOn.Version).Return(addOnWithDockerImages, nil)
	mockObj.MockStackService.On("ImportDockerImage", dockerImages[0]).Return(nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", addOn.Name, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterWrite", "/addontest-proxy.json", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyWrite", "/addontest-publish.http.conf", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		httpConfGeneratorFunc := args.Get(1).(func(writer io.Writer) error)
		var buf bytes.Buffer
		httpConfGeneratorFunc(&buf)
		mockObj.TestData()["HTTP_CONF"] = buf.String()
	})
	mockObj.On("ReverseProxyWrite", "/addontest-publish-proxy.map", mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		httpConfGeneratorFunc := args.Get(1).(func(writer io.Writer) error)
		var buf bytes.Buffer
		httpConfGeneratorFunc(&buf)
		mockObj.TestData()["PROXY_MAP"] = buf.String()
	})
	mockObj.On("ReverseProxyCreateSymbolicLink", "/addontest-publish.http.conf", "/addontest-publish.http.conf", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyCreateSymbolicLink", "/addontest-publish-proxy.map", "/addontest-publish-proxy.map", mock.Anything).Return(nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(1), nil)

	expectedHttpConf := `
# {"com.weidmueller.uc.aom.reverse-proxy.version":"0.2.0","com.weidmueller.uc.aom.version":"0.5.3"}
location /addontest/To {

    return 301 $scheme://$host/addontest/To/$request_uri;

    location /addontest/To/ {
        proxy_pass /from/;

        proxy_http_version 1.1;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        client_max_body_size 0;
        client_body_timeout 30m;
    }
}
`

	expectedProxyMap := `
# {"com.weidmueller.uc.aom.reverse-proxy.version":"0.2.0","com.weidmueller.uc.aom.version":"0.5.3"}
# add-on-test Add-on UI
~^/+addontest/+To.*$    addontest.access;

`

	// Act
	transactionScheduler := service.NewTransactionScheduler()
	tx, err := transactionScheduler.CreateTransaction(context.Background(), uut)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer tx.Rollback()

	err = tx.CreateAddOnRoutine(addOn.Name, addOn.Version)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	actualHttpConf := mockObj.TestData()["HTTP_CONF"]
	if actualHttpConf != expectedHttpConf {
		t.Errorf("Not Equal. Expected '%+v' Actual '%+v'", expectedHttpConf, actualHttpConf)
	}

	actualProxyMap := mockObj.TestData()["PROXY_MAP"]
	if actualProxyMap != expectedProxyMap {
		t.Errorf("Not Equal. Expected '%+v' Actual '%+v'", expectedProxyMap, actualProxyMap)
	}

	// Assert
	mockObj.AssertExpectations(t)
}

func TestDeleteAddOnRoutineSuccess(t *testing.T) {
	// Arrange
	mockObj := &service.ServiceMultiComponentMock{}
	addOn := newAddOn("addontest", "add-on-test", "4.3.2", "docker-image", "test-volume")
	uut := createUut(mockObj)

	mockObj.On("GetAddOn", addOn.Name).Return(*addOn, nil)
	mockObj.MockStackService.On("DeleteAddOnStack", addOn.Name).Return(nil)
	mockObj.MockStackService.On("DeleteDockerImages", []string{"docker-image:4.3.2"}).Return(nil)
	mockObj.MockStackService.On("RemoveUnusedVolumes", addOn.Name, []string{"test-volume"}).Return(nil)
	mockObj.On("ReverseProxyDelete", "/addontest-publish.http.conf").Return(nil)
	mockObj.On("ReverseProxyDelete", "/addontest-publish-proxy.map").Return(nil)
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addontest-publish.http.conf").Return(nil)
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addontest-publish-proxy.map").Return(nil)
	mockObj.On("IamPermissionWriterDelete", "/addontest-proxy.json").Return(nil)
	mockObj.On("DeleteAddOn", addOn.Name).Return(nil)

	// Act
	transactionScheduler := service.NewTransactionScheduler()
	tx, err := transactionScheduler.CreateTransaction(context.Background(), uut)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer tx.Rollback()

	err = tx.DeleteAddOnRoutine(addOn.Name)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert
	mockObj.AssertExpectations(t)
}

func TestReplaceAddOnRoutineSuccess(t *testing.T) {
	// Arrange
	mockObj := &service.ServiceMultiComponentMock{}
	oldAddOn := newAddOn("addontest", "add-on-test", "4.3.2", "docker-image", "test-volume-old")
	newAddOn := newAddOn("addontest", "add-on-test", "5.0.0", "docker-image", "test-volume-new")
	dockerImages := dockerImages("docker-image")
	uut := createUut(mockObj)

	mockObj.On("GetAddOn", oldAddOn.Name).Return(*oldAddOn, nil).Once()
	mockObj.MockStackService.On("DeleteAddOnStack", oldAddOn.Name).Return(nil)
	mockObj.MockStackService.On("DeleteDockerImages", []string{"docker-image:4.3.2"}).Return(nil)
	mockObj.On("ReverseProxyDelete", "/addontest-publish.http.conf").Return(nil)
	mockObj.On("ReverseProxyDelete", "/addontest-publish-proxy.map").Return(nil)
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addontest-publish.http.conf").Return(nil)
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addontest-publish-proxy.map").Return(nil)
	mockObj.On("IamPermissionWriterDelete", "/addontest-proxy.json").Return(nil)
	mockObj.On("DeleteAddOn", oldAddOn.Name).Return(nil)

	newAddOnWithDockerImages := catalogue.CatalogueAddOnWithImages{AddOn: *newAddOn, DockerImageData: dockerImages}
	mockObj.On("GetAddOn", newAddOn.Name).Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	mockObj.On("PullAddOn", newAddOn.Name, newAddOn.Version).Return(newAddOnWithDockerImages, nil)
	mockObj.MockStackService.On("ImportDockerImage", dockerImages[0]).Return(nil)
	mockObj.MockStackService.On("CreateStackWithDockerCompose", newAddOn.Name, mock.AnythingOfType("string"), mock.Anything).Return(nil)
	mockObj.On("IamPermissionWriterWrite", "/addontest-proxy.json", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyWrite", "/addontest-publish.http.conf", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyWrite", "/addontest-publish-proxy.map", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyCreateSymbolicLink", "/addontest-publish.http.conf", "/addontest-publish.http.conf", mock.Anything).Return(nil)
	mockObj.On("ReverseProxyCreateSymbolicLink", "/addontest-publish-proxy.map", "/addontest-publish-proxy.map", mock.Anything).Return(nil)
	mockObj.On("FetchManifest", newAddOn.Name, newAddOn.Version).Return(&newAddOn.Manifest, nil)

	mockObj.MockStackService.On("RemoveUnusedVolumes", newAddOn.Name, []string{"test-volume-old"}).Return(nil)
	mockObj.On("Validate", mock.Anything).Return(nil)
	mockObj.On("AvailableSpaceInBytes").Return(uint64(2), nil)

	// Act
	transactionScheduler := service.NewTransactionScheduler()
	tx, err := transactionScheduler.CreateTransaction(context.Background(), uut)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer tx.Rollback()

	err = tx.ReplaceAddOnRoutine(newAddOn.Name, newAddOn.Version)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert
	mockObj.AssertExpectations(t)
}

func TestCreateAddOnRoutineFailureNotEnoughSpace(t *testing.T) {
	// Arrange
	addOnInstallSize := uint64(0xdeadbeef)
	mockObj := &service.ServiceMultiComponentMock{}
	// TEST CASE: We have run out of space for install.
	mockObj.On("AvailableSpaceInBytes").Return(uint64(addOnInstallSize-1), nil)

	addOn := newAddOn("addontest", "add-on-test", "4.3.2", "docker-image", "test-volume")
	dockerImages := dockerImages("docker-image")
	uut := createUut(mockObj)
	addOnWithDockerImages := catalogue.CatalogueAddOnWithImages{
		AddOn:                *addOn,
		DockerImageData:      dockerImages,
		EstimatedInstallSize: addOnInstallSize,
	}

	mockObj.On("GetAddOn", addOn.Name).Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	mockObj.On("PullAddOn", addOn.Name, addOn.Version).Return(addOnWithDockerImages, nil)

	mockObj.AssertNotCalled(t, "ImportDockerImage", mock.Anything)
	mockObj.AssertNotCalled(t, "CreateStackWithDockerCompose", addOn.Name, mock.AnythingOfType("string"), mock.Anything)
	mockObj.AssertNotCalled(t, "IamPermissionWriterWrite", "/addontest-proxy.json", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyWrite", "/addontest-publish.http.conf", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyWrite", "/addontest-publish-proxy.map", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyCreateSymbolicLink", "/addontest-publish.http.conf", "/addontest-publish.http.conf", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyCreateSymbolicLink", "/addontest-publish-proxy.map", "/addontest-publish-proxy.map", mock.Anything)
	mockObj.AssertNotCalled(t, "Validate", mock.Anything)
	mockObj.On("DeleteAddOn", addOn.Name).Return(nil)

	// Act
	transactionScheduler := service.NewTransactionScheduler()
	tx, err := transactionScheduler.CreateTransaction(context.Background(), uut)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = tx.CreateAddOnRoutine(addOn.Name, addOn.Version)
	if err == nil {
		t.Fatal("Expected error, none received.")
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert
	mockObj.AssertExpectations(t)
}

func TestReplaceAddOnRoutineFailureNotEnoughSpace(t *testing.T) {
	// Arrange
	addOnInstallSize := uint64(0xdeadbeef)
	mockObj := &service.ServiceMultiComponentMock{}
	// TEST CASE: We simulate a disk space look-up error.
	mockObj.On("AvailableSpaceInBytes").Return(addOnInstallSize-1, bytes.ErrTooLarge)

	oldAddOn := newAddOn("addontest", "add-on-test", "4.3.2", "docker-image", "test-volume-old")
	newAddOn := newAddOn("addontest", "add-on-test", "5.0.0", "docker-image", "test-volume-new")
	dockerImages := dockerImages("docker-image")
	uut := createUut(mockObj)

	mockObj.On("GetAddOn", oldAddOn.Name).Return(*oldAddOn, nil).Once()
	mockObj.MockStackService.On("DeleteAddOnStack", oldAddOn.Name).Return(nil)
	mockObj.MockStackService.On("DeleteDockerImages", []string{"docker-image:4.3.2"}).Return(nil)
	mockObj.On("ReverseProxyDelete", "/addontest-publish.http.conf").Return(nil)
	mockObj.On("ReverseProxyDelete", "/addontest-publish-proxy.map").Return(nil)
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addontest-publish.http.conf").Return(nil)
	mockObj.On("ReverseProxyRemoveSymbolicLink", "/addontest-publish-proxy.map").Return(nil)
	mockObj.On("IamPermissionWriterDelete", "/addontest-proxy.json").Return(nil)
	mockObj.On("DeleteAddOn", oldAddOn.Name).Return(nil).Twice()

	newAddOnWithDockerImages := catalogue.CatalogueAddOnWithImages{
		AddOn:                *newAddOn,
		DockerImageData:      dockerImages,
		EstimatedInstallSize: addOnInstallSize,
	}

	mockObj.On("GetAddOn", newAddOn.Name).Return(catalogue.CatalogueAddOn{}, catalogue.ErrorAddOnNotFound)
	mockObj.On("PullAddOn", newAddOn.Name, newAddOn.Version).Return(newAddOnWithDockerImages, nil)
	mockObj.AssertNotCalled(t, "ImportDockerImage", mock.Anything)
	mockObj.AssertNotCalled(t, "CreateStackWithDockerCompose", newAddOn.Name, mock.AnythingOfType("string"), mock.Anything)
	mockObj.AssertNotCalled(t, "IamPermissionWriterWrite", "/addontest-proxy.json", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyWrite", "/addontest-publish.http.conf", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyWrite", "/addontest-publish-proxy.map", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyCreateSymbolicLink", "/addontest-publish.http.conf", "/addontest-publish.http.conf", mock.Anything)
	mockObj.AssertNotCalled(t, "ReverseProxyCreateSymbolicLink", "/addontest-publish-proxy.map", "/addontest-publish-proxy.map", mock.Anything)
	mockObj.AssertNotCalled(t, "Validate", mock.Anything)

	mockObj.On("FetchManifest", newAddOn.Name, newAddOn.Version).Return(&newAddOn.Manifest, nil)

	// Act
	transactionScheduler := service.NewTransactionScheduler()
	tx, err := transactionScheduler.CreateTransaction(context.Background(), uut)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = tx.ReplaceAddOnRoutine(newAddOn.Name, newAddOn.Version)
	if err == nil {
		t.Fatal("Expected error, none received.")
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Assert
	mockObj.AssertExpectations(t)
}

func TestDeleteAddOnRoutineCodesys(t *testing.T) {
	// Arrange
	mockObj := &service.ServiceMultiComponentMock{}
	addOn := newAddOn("test-uc-addon-codesys-pkg", "test-uc-addon-codesys", "0.1.0-1", "docker-image", "test-uc-addon-codesys")
	uut := createUut(mockObj)

	mockObj.On("GetAddOn", addOn.Name).Return(*addOn, nil)

	// Act
	transactionScheduler := service.NewTransactionScheduler()
	tx, err := transactionScheduler.CreateTransaction(context.Background(), uut)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer tx.Rollback()

	err = tx.DeleteAddOnRoutine(addOn.Name)

	// Assert
	if err == nil {
		t.Fatal("Expected an error but got none")
	}

	if grpcErr, ok := grpcStatus.FromError(err); ok {
		if grpcErr.Code() != codes.Unimplemented {
			t.Errorf("Expected an unimplemented error but got %s", grpcErr.Code())
		}
	}
}

func TestReplaceAddOnRoutineCodesys(t *testing.T) {
	// Arrange
	mockObj := &service.ServiceMultiComponentMock{}
	newAddOn := newAddOn("test-uc-addon-codesys-pkg", "test-uc-addon-codesys", "0.2.0-1", "docker-image", "test-uc-addon-codesys")

	uut := createUut(mockObj)

	// Act
	transactionScheduler := service.NewTransactionScheduler()
	tx, err := transactionScheduler.CreateTransaction(context.Background(), uut)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	defer tx.Rollback()

	err = tx.ReplaceAddOnRoutine(newAddOn.Name, newAddOn.Version)

	// Assert
	if err == nil {
		t.Fatalf("Expected an error but got none")
	}

	if grpcErr, ok := grpcStatus.FromError(err); ok {
		if grpcErr.Code() != codes.Unimplemented {
			t.Errorf("Expected an unimplemented error but got %s", grpcErr.Code())
		}
	}
}
