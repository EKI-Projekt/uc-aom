// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/aom/docker"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/pkg/manifest"

	sharedConfig "u-control/uc-aom/internal/pkg/config"
	testhelpers "u-control/uc-aom/test/test-helpers"

	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/server"
	"u-control/uc-aom/internal/aom/service"

	"github.com/docker/docker/api/types"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type testCase struct {
	description     string
	name            string
	version         string
	expectedVersion string
}

var testCases = []testCase{
	{
		description:     "Empty version '' returns 0.2.0-1",
		name:            "test-uc-addon-settings-addon-pkg",
		version:         "",
		expectedVersion: "0.2.0-1",
	},
	{
		description:     "Version 0.1.0-1 returns 0.1.0-1",
		name:            "test-uc-addon-settings-addon-pkg",
		version:         "0.1.0-1",
		expectedVersion: "0.1.0-1",
	},
	{
		description:     "Version 0.2.0-1 returns 0.2.0-1",
		name:            "test-uc-addon-settings-addon-pkg",
		version:         "0.2.0-1",
		expectedVersion: "0.2.0-1",
	},
}

func catalogueFail(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange & Act
	_, err := testEnvironment.GetCatalogueAddOnWithVersion("test-uc-addon-settings-addon-pkg", "non-existant-version")

	// Assert
	if err == nil {
		t.Errorf("Expected error none received.")
	}
}

func (tc *testCase) cataloguePass(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange & Act
	addOn, err := testEnvironment.GetCatalogueAddOnWithVersion(tc.name, tc.version)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	if addOn.Version != tc.expectedVersion {
		t.Errorf("Expected %s, Actual %s", tc.expectedVersion, addOn.Version)
	}

}

func createAndDelete(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Act
	addOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-status-running-addon-pkg", "0.1.0-1")

	// Assert
	// List installed AddOns, must be present
	err := testhelpers.ValidateAddOnInstalled(testEnvironment, addOn.Name)
	if err != nil {
		t.Fatalf("Failed to validate AddOn installed: %v", err)
	}
	// Validate logo path
	actualLogo := addOn.Logo
	expectedLogoPrefix := config.URL_ASSETS_LOCAL_ROOT
	if !(strings.HasPrefix(actualLogo, expectedLogoPrefix)) {
		t.Errorf("Error: Expected URL assets root prefix '%s' didn't match actual assets path '%s'", expectedLogoPrefix, actualLogo)
	}

	// Validate that the logo can be requested through nginx
	err = testhelpers.HttpGetWithRetriesAndExpectedStatus(fmt.Sprintf("http://nginx%s", addOn.Logo), http.StatusOK)
	assert.Nil(t, err)

	// Validate addOn status
	if addOn.Status != grpc_api.AddOnStatus_STARTING && addOn.Status != grpc_api.AddOnStatus_RUNNING {
		t.Errorf("Status is %d; want %d (STARTING) or %d (RUNNING)", addOn.Status, grpc_api.AddOnStatus_STARTING, grpc_api.AddOnStatus_RUNNING)
	}

	testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	_, err = testEnvironment.GetInstalledAddOn(addOn.Name)
	if err == nil {
		t.Errorf("Failed to delete addOn. AddOn is still present.")
	}
}

func updateWithVolume(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOnInstall := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-update-with-volume-addon-pkg", "0.1.0-1")
	addOnUpdate := testEnvironment.UpdateTestAddOnRoutine(t, "test-uc-addon-update-with-volume-addon-pkg", "0.2.0-1")
	// Act
	// Test is executed by addOn. Assert of test result via addOn status.

	// Assert
	addOnStatus, err := testEnvironment.GetInstalledAddOnStatus(addOnUpdate)
	if err != nil {
		t.Errorf("Failed to get AddOn status: %v", err)
	}
	if addOnStatus != grpc_api.AddOnStatus_RUNNING {
		t.Errorf("Error: Wrong container status")
	}
	if addOnInstall.Logo == addOnUpdate.Logo {
		t.Errorf("Error: Logo is the same")
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOnUpdate)
	})
}

func vendorInformation(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	expectedVendor := grpc_api.Vendor{
		Name:    "Test Name",
		Url:     "https://www.example.com",
		Email:   "test@mail.test",
		Street:  "Test Street",
		Zip:     "12345",
		City:    "Test City",
		Country: "Test Country",
	}

	// Act
	addOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-vendor-information-addon-pkg", "0.1.0-1")

	// Assert
	if !reflect.DeepEqual(&expectedVendor, addOn.GetVendor()) {
		t.Errorf("Error: Mismatching vendor information.Expected %+v but got %+v", &expectedVendor, addOn.GetVendor())
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})
}

func addOnCommunication(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	receiverAddOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-communication-receiver-addon-pkg", "0.1.0-1")
	senderAddOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-communication-sender-addon-pkg", "0.1.0-1")

	// Act
	// Test is executed by addOn. Assert of test result via addOn status.

	// Assert
	receiverAddOnStatus, err := testEnvironment.GetInstalledAddOnStatus(senderAddOn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if receiverAddOnStatus != grpc_api.AddOnStatus_RUNNING {
		t.Errorf("Status is %d; want %d (RUNNING)", receiverAddOnStatus, grpc_api.AddOnStatus_RUNNING)
	}
	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, receiverAddOn)
		testEnvironment.DeleteTestAddOnRoutine(t, senderAddOn)
	})
}

func withWebUI(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Act
	installedAddOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-provide-ui-addon-pkg", "0.1.0-1")

	// Assert
	if !strings.HasPrefix(installedAddOn.Location, "/test-uc-addon-provide-ui-addon-pkg") {
		t.Errorf("Add-on name fails to prefix the publish location. '%s'", installedAddOn.Location)
	}

	expectedAddOnWebUIUrl := fmt.Sprintf("http://nginx%s", installedAddOn.Location)
	expectedStatusCode := 200
	err := testhelpers.HttpGetWithRetriesAndExpectedStatus(expectedAddOnWebUIUrl, expectedStatusCode)
	if err != nil {
		t.Fatalf("Failed to get add-on web ui: %v", err)
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, installedAddOn)
	})
}

func invalidManifest(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {

	// Act & Assert
	createAddOnResult, err := testEnvironment.CreateAddOn("test-uc-addon-invalid-manifest-version-addon-pkg", "0.1.0-1")
	if err == nil {
		t.Errorf("Error is not nil, result of createAddon: %v", createAddOnResult)
	}
}

func withWebSocket(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	installedaddOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-websocket-addon-pkg", "0.1.0-1")
	websocketPath := fmt.Sprintf("%s/echo", installedaddOn.Location)
	websocketUrl := url.URL{Scheme: "ws", Host: "nginx", Path: websocketPath}
	websocketString := websocketUrl.String()

	protocolPath := fmt.Sprintf("%s/protocol", installedaddOn.Location)
	protocolUrl := url.URL{Scheme: "http", Host: "nginx", Path: protocolPath}

	// Act
	echoConnection, err := testhelpers.WsDialWithRetries(websocketString)
	if err != nil {
		t.Errorf("Unable to create websocket connection to %s: %v", websocketString, err)
	}
	defer echoConnection.Close()

	protocolAtApp, err := testhelpers.HttpGetBodyWithRetries(protocolUrl.String())
	if err != nil {
		t.Errorf("Unable to read received protocol from add-on: %v", err)
	}

	// Assert
	testhelpers.WsAssertEchoWorking(t, echoConnection)

	if protocolAtApp != "HTTP/1.1" {
		t.Errorf("Add-On is not receiving HTTP/1.1 but %v", protocolAtApp)
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, installedaddOn)
	})
}

func identicalPublishKeys(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOnNames := []string{"test-uc-addon-provide-ui-addon-pkg", "test-uc-addon-provide-ui-b-addon-pkg"}
	addOns := make([]*grpc_api.AddOn, len(addOnNames))

	// Act
	for i, addOnName := range addOnNames {
		addOn := testEnvironment.CreateTestAddOnRoutine(t, addOnName, "0.1.0-1")
		addOns[i] = addOn
	}

	// Assert
	expectedFilepaths := make([]string, 0)
	for _, addOnName := range addOnNames {
		expectedFilepaths = append(expectedFilepaths,
			fmt.Sprintf("%s/%s-provideui-proxy.map", routes.ROUTES_MAP_AVAILABLE_PATH, addOnName),
			fmt.Sprintf("%s/%s-provideui-proxy.map", routes.ROUTES_MAP_ENABLED_PATH, addOnName),
			fmt.Sprintf("%s/%s-provideui.http.conf", routes.SITES_AVAILABLE_PATH, addOnName),
			fmt.Sprintf("%s/%s-provideui.http.conf", routes.SITES_ENABLED_PATH, addOnName),
		)
	}

	for _, path := range expectedFilepaths {
		_, err := os.Open(path)
		if err != nil {
			t.Errorf("Failed to find file '%s' %s", path, err.Error())
		}
	}

	t.Cleanup(func() {
		for _, addOn := range addOns {
			testEnvironment.DeleteTestAddOnRoutine(t, addOn)
		}
	})
}

func prefixPublishKey(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOnName := "test-uc-addon-provide-ui-addon-pkg"
	addOn := testEnvironment.CreateTestAddOnRoutine(t, addOnName, "0.1.0-1")

	// Act
	testEnvironment.DeleteTestAddOnRoutine(t, addOn)

	// Assert
	paths := []string{
		fmt.Sprintf("%s/%s-provideui-proxy.map", routes.ROUTES_MAP_AVAILABLE_PATH, addOnName),
		fmt.Sprintf("%s/%s-provideui-proxy.map", routes.ROUTES_MAP_ENABLED_PATH, addOnName),
		fmt.Sprintf("%s/%s-provideui.http.conf", routes.SITES_AVAILABLE_PATH, addOnName),
		fmt.Sprintf("%s/%s-provideui.http.conf", routes.SITES_ENABLED_PATH, addOnName),
	}

	for _, path := range paths {
		_, err := os.Open(path)
		if err == nil {
			t.Errorf("Expected '%s' to be deleted but still exist", path)
		}
	}
}

func updateWithSettings(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	// Configuration for install
	addonName := "test-uc-addon-settings-addon-pkg"
	addonInstallVersion := "0.1.0-1"
	param1InstallValue := "p1 install"
	param2InstallValue := "abc"
	//param3 is only in 0.2.0-1 version available
	param4InstallValue := "p4 install"

	installConfig := &grpc_api.AddOn{
		Name:    addonName,
		Version: addonInstallVersion,
		Settings: []*grpc_api.Setting{
			{
				Name: "param1", SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: param1InstallValue,
					},
				},
			},
			{
				Name: "param2", SettingOneof: &grpc_api.Setting_DropDownList{
					DropDownList: &grpc_api.DropDownList{
						Elements: []*grpc_api.DropDownItem{
							{
								Value: param2InstallValue, Selected: true,
							},
						},
					},
				},
			},
			{
				Name: "param4", SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: param4InstallValue,
					},
				},
			},
		},
	}

	addOnInstall, err := testEnvironment.CreateAddOn(installConfig.Name, installConfig.Version)
	if err != nil {
		t.Fatalf("Failed to install the add-on: %v", err)
	}

	// Configuration for update
	addonUpdateVersion := "0.2.0-1"
	param1UpdateValue := "p1 updated"
	param2UpdateValue := "qwe"
	param3UpdateValue := "p3 updated"
	//param4 is only in 0.1.0-1 version available

	updateConfig := &grpc_api.AddOn{
		Name:    addonName,
		Version: addonUpdateVersion,
		Settings: []*grpc_api.Setting{
			{
				Name: "param1", SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: param1UpdateValue,
					},
				},
			},
			{
				Name: "param2", SettingOneof: &grpc_api.Setting_DropDownList{
					DropDownList: &grpc_api.DropDownList{
						Elements: []*grpc_api.DropDownItem{
							{
								Value: param2UpdateValue, Selected: true,
							},
						},
					},
				},
			},
			{
				Name: "param3", SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: param3UpdateValue,
					},
				},
			},
		},
	}

	// Act
	addOnUpdate, err := testEnvironment.UpdateAddOn(updateConfig)
	if err != nil {
		t.Fatalf("Failed to update the add-on: %v", err)
	}

	// Assert
	if addOnInstall.Logo == addOnUpdate.Logo {
		t.Errorf("Error: Logo is the same")
	}

	for _, setting := range addOnUpdate.Settings {
		if setting.Name == "param1" {
			if setting.GetTextBox().Value != param1UpdateValue {
				t.Errorf("Unexpected setting for param1: %#v", setting.GetTextBox())
			}
		} else if setting.Name == "param2" {
			dropDown := setting.GetDropDownList()
			for i := range dropDown.Elements {
				if !dropDown.Elements[i].Selected {
					continue
				}
				if dropDown.Elements[i].Value != param2UpdateValue {
					t.Errorf("Unexpected setting param2: %#v", dropDown.Elements[i])
				}
			}
		} else if setting.Name == "param3" {
			if setting.GetTextBox().Value != param3UpdateValue {
				t.Errorf("Unexpected setting for param3: %#v", setting.GetTextBox())
			}
		} else {
			t.Errorf("Unexpected setting: %#v", setting)
		}
	}

	// Cleanup
	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOnUpdate)
	})
}

func withSettings(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOnName := "test-uc-addon-settings-addon-pkg"
	addOnVersion := "0.2.0-1"
	settings := []*grpc_api.Setting{
		{
			Name: "param1", SettingOneof: &grpc_api.Setting_TextBox{
				TextBox: &grpc_api.TextBox{
					Value: "test value = something",
				},
			},
		},
		{
			Name: "param2", SettingOneof: &grpc_api.Setting_DropDownList{
				DropDownList: &grpc_api.DropDownList{
					Elements: []*grpc_api.DropDownItem{
						{
							Value: "abc", Selected: true,
						},
					},
				},
			},
		},
	}

	addOn, err := testEnvironment.CreateAddOn(addOnName, addOnVersion, settings...)
	if err != nil {
		t.Fatalf("Failed to get the AddOn: %v", err)
	}

	// Act
	addOn, err = testEnvironment.GetInstalledAddOn(addOn.Name)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error thrown: %v", err)
	}

	if len(addOn.Settings) != 3 {
		t.Errorf("Unexpected value for Settings: %#v", addOn.Settings)
	}

	for _, setting := range addOn.Settings {
		if setting.Name == "param1" {
			if setting.GetTextBox().Value != "test value = something" {
				t.Errorf("Unexpected setting for param1: %#v", setting.GetTextBox())
			}
		} else if setting.Name == "param2" {
			dropDown := setting.GetDropDownList()
			for i := range dropDown.Elements {
				if !dropDown.Elements[i].Selected {
					continue
				}
				if dropDown.Elements[i].Value != "abc" {
					t.Errorf("Unexpected setting param2: %#v", dropDown.Elements[i])
				}
			}
		} else if setting.Name == "param3" {
			if setting.GetTextBox().Value != "default" {
				t.Errorf("Unexpected setting for param3: %#v", setting.GetTextBox())
			}
		} else {
			t.Errorf("Unexpected setting: %#v", setting)
		}
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})
}

func withSettingsReconfigure(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	// Defaults
	//   - param1: abc
	//   - param2: xyz
	param1UpdateValue := "reconfigured"
	param2UpdateValue := "abc"

	addOnName := "test-uc-addon-settings-addon-pkg"
	addOnVersion := "0.1.0-1"

	addOn, err := testEnvironment.CreateAddOn(addOnName, addOnVersion)
	if err != nil {
		t.Fatalf("Failed to get the AddOn: %v", err)
	}

	reconfigure := &grpc_api.AddOn{
		Name:    addOnName,
		Version: addOnVersion,
		Settings: []*grpc_api.Setting{
			{
				Name: "param1", SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: param1UpdateValue,
					},
				},
			},
			{
				Name: "param2", SettingOneof: &grpc_api.Setting_DropDownList{
					DropDownList: &grpc_api.DropDownList{
						Elements: []*grpc_api.DropDownItem{
							{
								Value: param2UpdateValue, Selected: true,
							},
						},
					},
				},
			},
			{
				Name: "param4", SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: "p4 default",
					},
				},
			},
		},
	}

	addOn, err = testEnvironment.UpdateAddOn(reconfigure)
	if err != nil {
		t.Fatalf("Failed to get the AddOn: %v", err)
	}

	addOn, err = testEnvironment.GetInstalledAddOn(addOn.Name)
	if err != nil {
		t.Fatalf("Unexpected error thrown: %v", err)
	}

	if len(addOn.Settings) != 3 {
		t.Errorf("Unexpected value for Settings: %#v", addOn.Settings)
	}

	for _, setting := range addOn.Settings {
		if setting.Name == "param1" {
			if setting.GetTextBox().Value != param1UpdateValue {
				t.Errorf("Unexpected setting for param1: %#v", setting.GetTextBox())
			}
		} else if setting.Name == "param2" {
			dropDown := setting.GetDropDownList()
			for i := range dropDown.Elements {
				if !dropDown.Elements[i].Selected {
					continue
				}
				if dropDown.Elements[i].Value != param2UpdateValue {
					t.Errorf("Unexpected setting param2: %#v", dropDown.Elements[i])
				}
			}
		} else if setting.Name == "param4" {
			if setting.GetTextBox().Value != "p4 default" {
				t.Errorf("Unexpected setting for param4: %#v", setting.GetTextBox())
			}
		} else {
			t.Errorf("Unexpected setting: %#v", setting)
		}
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})
}

func withPreRelease(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOnName := "test-uc-addon-status-running-addon-pkg"
	addOnInstall := testEnvironment.CreateTestAddOnRoutine(t, addOnName, "0.2.0-1")

	// Act
	prereleaseVersions := []string{"0.2.0-1-rc.1", "0.2.0-rc.1-1"}

	for _, version := range prereleaseVersions {
		addOnUpdate, err := testEnvironment.UpdateTestAddOnRoutineWithError(addOnName, version)

		// assert
		if err == nil {
			t.Fatalf("Expect Error but got nil")
		}
		if addOnUpdate != nil {
			t.Fatalf("Expect addOnUpdate to be nil but got %v", addOnUpdate)
		}
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOnInstall)
	})
}

func withLargeSize(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-large-size-addon-pkg", "0.1.0-1")

	// Assert
	// List installed AddOns, must be present
	err := testhelpers.ValidateAddOnInstalled(testEnvironment, addOn.Name)
	if err != nil {
		t.Fatalf("Failed to validate AddOn installed: %v", err)
	}

	// Validate logo path
	actualLogo := addOn.Logo
	expectedLogoPrefix := config.URL_ASSETS_LOCAL_ROOT
	if !(strings.HasPrefix(actualLogo, expectedLogoPrefix)) {
		t.Errorf("Error: Expected URL assets root prefix '%s' didn't match actual assets path '%s'", expectedLogoPrefix, actualLogo)
	}
	// Validate addOn status
	if addOn.Status != grpc_api.AddOnStatus_STARTING && addOn.Status != grpc_api.AddOnStatus_RUNNING {
		t.Errorf("Status is %d; want %d (STARTING) or %d (RUNNING)", addOn.Status, grpc_api.AddOnStatus_STARTING, grpc_api.AddOnStatus_RUNNING)
	}

	testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	_, err = testEnvironment.GetInstalledAddOn(addOn.Name)
	if err == nil {
		t.Errorf("Failed to delete addOn. AddOn is still present.")
	}
}

func withInstallAddOnOnlyOnce(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOnName := "test-uc-addon-status-running-addon-pkg"
	firstAddOnVersion := "0.1.0-1"
	secondAddOnVersion := "0.2.0-1"

	firstAddOn := testEnvironment.CreateTestAddOnRoutine(t, addOnName, firstAddOnVersion)
	defer testEnvironment.DeleteTestAddOnRoutine(t, firstAddOn)

	err := testhelpers.ValidateAddOnInstalled(testEnvironment, firstAddOn.Name)
	if err != nil {
		t.Fatalf("Failed to validate AddOn installed: %v", err)
	}

	// Act
	resultAddOn, err := testEnvironment.CreateAddOn(addOnName, secondAddOnVersion)

	// Assert
	expectedError := status.Error(codes.AlreadyExists, service.ErrorAddOnAlreadyInstalled.Error())
	if !errors.Is(err, expectedError) {
		t.Fatalf("Expected error %v but got %v.", expectedError, err)
	}
	if reflect.DeepEqual(resultAddOn, grpc_api.AddOn{}) {
		t.Fatalf("Expected empty AddOn but got: %v", resultAddOn)
	}

	installedAddOn, err := testEnvironment.GetInstalledAddOn(addOnName)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if installedAddOn.Version != firstAddOnVersion {
		t.Fatalf("Expected AddOn version %s but got %s", firstAddOnVersion, installedAddOn.Version)
	}
}

func withNewTestEnvironment(listener *bufconn.Listener, fun func(*testhelpers.TestEnvironment, *testing.T)) func(*testing.T) {
	return func(t *testing.T) {
		testEnvironment, err := testhelpers.NewTestEnvironment(context.Background(), listener)
		if err != nil {
			t.Fatal(err)
		}

		t.Cleanup(func() {
			testEnvironment.CloseConnection()
		})

		fun(testEnvironment, t)
	}
}

func InstallWithFeatureRootAccess(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	type configFile struct {
		RootAccessEnabled int `json:"rootAccessEnabled"`
	}
	configFilePath := os.Getenv("ROOT_ACCESS_CONFIG_FILE")
	assert.NotEmpty(t, configFilePath)

	preTestContent, err := os.ReadFile(configFilePath)
	assert.Nil(t, err)

	writeConfigFileContent := func(contentToWrite []byte) error {
		return os.WriteFile(configFilePath, contentToWrite, os.ModePerm)
	}

	defer writeConfigFileContent(preTestContent)

	type fields struct {
		rootAccessEnabled int
	}

	type args struct {
		name    string
		version string
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "Should not be installed if root access is disabled",
			args: args{
				name:    "test-uc-addon-ssh-root-access-addon-pkg",
				version: "0.1.0-1",
			},
			fields: fields{
				rootAccessEnabled: 0,
			},
			wantErr: true,
		},
		{
			name: "Should be installed if root access is enabled",
			args: args{
				name:    "test-uc-addon-ssh-root-access-addon-pkg",
				version: "0.1.0-1",
			},
			fields: fields{
				rootAccessEnabled: 1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			testhelpers.SetRootAccess(t, tt.fields.rootAccessEnabled)

			// act
			addOn, err := testEnvironment.CreateAddOn(tt.args.name, tt.args.version)
			if err == nil {
				defer testEnvironment.DeleteTestAddOnRoutine(t, addOn)
			}

			// assert
			if (err != nil) != tt.wantErr {
				t.Errorf("createAddOn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			expectedError := server.ConvertToGrpcRootAccessNotEnabledError(service.SshRootAccessNotEnabledError)
			if tt.wantErr && !errors.Is(err, expectedError) {
				t.Fatalf("Expected error %v but got %v.", service.SshRootAccessNotEnabledError, err)
			}
		})
	}

}

func updateWithFeatureRootAccess(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	type configFile struct {
		RootAccessEnabled int `json:"rootAccessEnabled"`
	}
	configFilePath := os.Getenv("ROOT_ACCESS_CONFIG_FILE")
	assert.NotEmpty(t, configFilePath)

	preTestContent, err := os.ReadFile(configFilePath)
	assert.Nil(t, err)

	writeConfigFileContent := func(contentToWrite []byte) error {
		return os.WriteFile(configFilePath, contentToWrite, os.ModePerm)
	}

	defer writeConfigFileContent(preTestContent)
	type args struct {
		initialVersion    string
		replaceVersion    string
		rootAccessEnabled int
	}

	tests := []struct {
		name    string
		wantErr bool
		args    args
	}{
		{
			name:    "Should be updated if root access is enabled",
			wantErr: false,
			args: args{
				initialVersion:    "0.1.0-1",
				replaceVersion:    "0.2.0-1",
				rootAccessEnabled: 1,
			},
		},
		{
			name:    "Should not be updated if root acces is disabled",
			wantErr: true,
			args: args{
				initialVersion:    "0.1.0-1",
				replaceVersion:    "0.2.0-1",
				rootAccessEnabled: 0,
			},
		},
		{
			name:    "Should be configured if root acces is disabled",
			wantErr: false,
			args: args{
				initialVersion:    "0.2.0-1",
				replaceVersion:    "0.2.0-1",
				rootAccessEnabled: 0,
			},
		},
	}

	testAddOnName := "test-uc-addon-ssh-root-access-addon-pkg"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			testhelpers.SetRootAccess(t, 1)
			initialAddOn, err := testEnvironment.CreateAddOn(testAddOnName, tt.args.initialVersion)
			if err == nil {
				defer testEnvironment.DeleteTestAddOnRoutine(t, initialAddOn)
			}

			testhelpers.SetRootAccess(t, tt.args.rootAccessEnabled)

			updateConfig := &grpc_api.AddOn{
				Name:    testAddOnName,
				Version: tt.args.replaceVersion,
				Settings: []*grpc_api.Setting{
					{
						Name: "TestParameter", SettingOneof: &grpc_api.Setting_TextBox{
							TextBox: &grpc_api.TextBox{
								Value: "UpdateValue",
							},
						},
					},
				},
			}

			_, err = testEnvironment.UpdateAddOn(updateConfig)

			// Assert
			if (err != nil) != tt.wantErr {
				t.Errorf("updateAddOn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			expectedError := server.ConvertToGrpcRootAccessNotEnabledError(service.SshRootAccessNotEnabledError)
			if tt.wantErr && !errors.Is(err, expectedError) {
				t.Fatalf("Expected error %v but got %v.", service.SshRootAccessNotEnabledError, err)
			}
		})
	}
}

func withPublicVolume(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange

	installedAddOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-public-volume-addon-pkg", "0.1.0-1")
	expectedVolumePublicPath := path.Join(docker.PUBLIC_VOLUMES_PATH, fmt.Sprintf("%s_%s", installedAddOn.Name, "uc-addon-public-volume"))

	// Assert
	wantUserId := uint32(1000)
	wantGroupId := uint32(1000)
	gotPublicVolumeFileInfo, err := os.Stat(expectedVolumePublicPath)
	assert.Nil(t, err)
	assert.True(t, gotPublicVolumeFileInfo.IsDir())

	gotPublicVolumeSys := gotPublicVolumeFileInfo.Sys().(*syscall.Stat_t)
	assert.Equal(t, wantUserId, gotPublicVolumeSys.Uid)
	assert.Equal(t, wantGroupId, gotPublicVolumeSys.Gid)

	// The "testDir" directory is created by the app during the start-up
	wantDirectory := path.Join(expectedVolumePublicPath, "testDir")
	gotDirectoryFileInfo, err := os.Stat(wantDirectory)
	assert.Nil(t, err)

	gotDirectorySys := gotDirectoryFileInfo.Sys().(*syscall.Stat_t)
	assert.Equal(t, wantUserId, gotDirectorySys.Uid)
	assert.Equal(t, wantGroupId, gotDirectorySys.Gid)

	testEnvironment.DeleteTestAddOnRoutine(t, installedAddOn)
	_, err = os.Stat(expectedVolumePublicPath)
	assert.Error(t, err)
}

func withPublicVolumeAndUpdate(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	const expectedStatusCode = 200
	testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-public-volume-addon-pkg", "0.1.0-1")
	updatedPublicAddOn := testEnvironment.UpdateTestAddOnRoutine(t, "test-uc-addon-public-volume-addon-pkg", "0.2.0-1")
	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, updatedPublicAddOn)
	})

	err := testhelpers.HttpGetWithRetriesAndExpectedStatus(fmt.Sprintf("http://nginx%s", updatedPublicAddOn.Location), expectedStatusCode)
	assert.Nil(t, err)

	publicBaseUrl := fmt.Sprintf("http://nginx%s", updatedPublicAddOn.Location)

	// This directory and file is created by the public add-on in the entrypoint
	expectedRoutes := []string{"testDir", "testFile.txt"}

	// Act & Assert
	testhelpers.WalkThroughExpectedRoutesWithBaseUrl(t, publicBaseUrl, expectedRoutes)

}

func withPublicVolumeAndAccess(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	const expectedStatusCode = 200
	installedPublicAddOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-public-volume-addon-pkg", "0.1.0-1")
	installedPublicAccessAddOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-public-volume-access-addon-pkg", "0.1.0-1")
	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, installedPublicAddOn)
		testEnvironment.DeleteTestAddOnRoutine(t, installedPublicAccessAddOn)
	})

	err := testhelpers.HttpGetWithRetriesAndExpectedStatus(fmt.Sprintf("http://nginx%s", installedPublicAddOn.Location), expectedStatusCode)
	assert.Nil(t, err)

	err = testhelpers.HttpGetWithRetriesAndExpectedStatus(fmt.Sprintf("http://nginx%s", installedPublicAccessAddOn.Location), expectedStatusCode)
	assert.Nil(t, err)

	publicBaseUrl := fmt.Sprintf("http://nginx%s", installedPublicAddOn.Location)

	expectedPublicVolume := fmt.Sprintf("%s_%s", installedPublicAddOn.Name, "uc-addon-public-volume")
	publicAccessBaseUrl := fmt.Sprintf("http://nginx%s/%s", installedPublicAccessAddOn.Location, expectedPublicVolume)

	// This directory and file is created by the public add-on in the entrypoint
	expectedRoutes := []string{"testDir", "testFile.txt"}

	// Act & Assert
	testhelpers.WalkThroughExpectedRoutesWithBaseUrl(t, publicBaseUrl, expectedRoutes)
	testhelpers.WalkThroughExpectedRoutesWithBaseUrl(t, publicAccessBaseUrl, expectedRoutes)
}

func withCodeName(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Act
	addOn, err := testEnvironment.GetCatalogueAddOn("test-uc-addon-posuma-addon-pkg")
	if err != nil {
		t.Fatal(err)
	}

	createdAddOn := testEnvironment.CreateTestAddOnRoutine(t, addOn.Name, addOn.Version)

	// Assert
	expectedLocation := "/test-uc-addon-posuma-addon-pkg/test-ui"
	if !assert.Equal(t, expectedLocation, createdAddOn.Location) {
		assert.FailNow(t, "wrong location")
	}

	err = testhelpers.HttpGetWithRetriesAndExpectedStatus(fmt.Sprintf("http://nginx%s", createdAddOn.Location), http.StatusOK)
	assert.Nil(t, err)

	err = testhelpers.ValidateAddOnInstalled(testEnvironment, addOn.Name)
	if err != nil {
		t.Fatal(err)
	}

	testEnvironment.DeleteTestAddOnRoutine(t, addOn)
}

func withPortCheck(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	addOnName := "test-uc-addon-port-check-addon-pkg"
	addOnVersion := "0.1.0-1"

	// port of the local registry should be available in the dev environment
	wantSettingPortValue := "5000"
	settings := &grpc_api.Setting{
		Name: "PORT", SettingOneof: &grpc_api.Setting_TextBox{
			TextBox: &grpc_api.TextBox{
				Value: wantSettingPortValue,
			},
		}}

	addOn, err := testEnvironment.CreateAddOn(addOnName, addOnVersion, settings)
	if err != nil {
		t.Fatalf("Failed to get the AddOn: %v", err)
	}

	// Act
	addOnStatus, err := testEnvironment.GetInstalledAddOnStatus(addOn)

	// Assert
	assert.NoError(t, err)

	assert.Len(t, addOn.Settings, 1)
	assert.Equal(t, wantSettingPortValue, addOn.Settings[0].GetTextBox().Value)
	assert.Equal(t, grpc_api.AddOnStatus_RUNNING, addOnStatus)

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})
}

func withMultiService(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Act
	addOn, err := testEnvironment.GetCatalogueAddOn("test-uc-addon-multi-service-addon-pkg")
	if err != nil {
		t.Fatal(err)
	}

	createdAddOn := testEnvironment.CreateTestAddOnRoutine(t, addOn.Name, addOn.Version)

	// Assert
	err = testhelpers.ValidateAddOnInstalled(testEnvironment, addOn.Name)
	assert.NoError(t, err)

	status, err := testEnvironment.GetInstalledAddOnStatus(createdAddOn)
	assert.NoError(t, err)
	assert.Equal(t, status, grpc_api.AddOnStatus_RUNNING)

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})
}

func checkStackForVersionLabel(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	dockerCli, err := docker.NewDockerCli()
	assert.NoError(t, err)
	dockerClient := dockerCli.Client()

	// Act
	addOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-update-with-volume-addon-pkg", "0.1.0-1")

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})

	// Assert
	containerInfo, err := dockerClient.ContainerInspect(testEnvironment.Ctx, "uc-addon-update-with-volume")
	assert.NoError(t, err)

	containsVersionLabels := func(labels map[string]string) {
		if assert.Contains(t, labels, config.UcAomVersionLabel) {
			assert.Equal(t, labels[config.UcAomVersionLabel], config.UcAomVersion)
		}
		if assert.Contains(t, labels, docker.UcAomStackVersionLabel) {
			assert.Equal(t, labels[docker.UcAomStackVersionLabel], docker.StackVersion)
		}
	}

	containsVersionLabels(containerInfo.Config.Labels)

	volumeName := fmt.Sprintf("%s_%s", addOn.Name, "uc-addon-test-volume-stay")
	volumeInfo, err := dockerClient.VolumeInspect(testEnvironment.Ctx, volumeName)
	assert.NoError(t, err)

	containsVersionLabels(volumeInfo.Labels)

	networkName := fmt.Sprintf("%s_%s", addOn.Name, "default")
	networkInfo, err := dockerClient.NetworkInspect(testEnvironment.Ctx, networkName, types.NetworkInspectOptions{})
	assert.NoError(t, err)
	containsVersionLabels(networkInfo.Labels)
}

func withLargeFileUpload(testEnvironment *testhelpers.TestEnvironment, t *testing.T) {
	// Arrange
	f, err := os.CreateTemp(t.TempDir(), "upload-file")
	assert.NoError(t, err)

	size10MB := int64(10 * 1024 * 1024)
	err = f.Truncate(size10MB)
	assert.NoError(t, err)

	addOn := testEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-provide-ui-addon-pkg", "0.1.0-1")
	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})

	// Act
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("myFile", f.Name())
	assert.NoError(t, err)
	_, err = io.Copy(fw, f)
	assert.NoError(t, err)

	w.Close()

	resp, err := http.Post(fmt.Sprintf("http://nginx%s/upload", addOn.Logo), w.FormDataContentType(), &b)

	assert.Equal(t, 200, resp.StatusCode)
}
func TestUcAom(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	grpcTestListener := bufconn.Listen(testhelpers.BufSize)

	err := testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)
	if err != nil {
		t.Fatalf("Unable to connect to UcAom: %v", err)
	}

	t.Run("TestGetAddOnFromCatalogueFail", withNewTestEnvironment(grpcTestListener, catalogueFail))
	for _, testCase := range testCases {
		t.Run(testCase.description, withNewTestEnvironment(grpcTestListener, testCase.cataloguePass))
	}
	t.Run("TestAddOnCreateAndDelete", withNewTestEnvironment(grpcTestListener, createAndDelete))
	t.Run("TestAddOnUpdateWithVolume", withNewTestEnvironment(grpcTestListener, updateWithVolume))
	t.Run("TestAddOnVendorInformation", withNewTestEnvironment(grpcTestListener, vendorInformation))
	t.Run("TestAddOnCommunication", withNewTestEnvironment(grpcTestListener, addOnCommunication))
	t.Run("TestAddOnWithWebUI", withNewTestEnvironment(grpcTestListener, withWebUI))
	t.Run("TestNotInstallAddonWithInvalidManifest", withNewTestEnvironment(grpcTestListener, invalidManifest))
	t.Run("TestAddOnWithWebsocket", withNewTestEnvironment(grpcTestListener, withWebSocket))
	t.Run("TestInstallAddonsWithIdenticalPublishKeys", withNewTestEnvironment(grpcTestListener, identicalPublishKeys))
	t.Run("TestDeleteAddOnWithPrefixedPublishKey", withNewTestEnvironment(grpcTestListener, prefixPublishKey))
	t.Run("TestUpdateAddOnWithSettings", withNewTestEnvironment(grpcTestListener, updateWithSettings))
	t.Run("TestAddOnWithSettings", withNewTestEnvironment(grpcTestListener, withSettings))
	t.Run("TestAddOnWithSettingsReconfigure", withNewTestEnvironment(grpcTestListener, withSettingsReconfigure))
	t.Run("TestAddOnWithPrerelease", withNewTestEnvironment(grpcTestListener, withPreRelease))
	t.Run("TestAddOnWithLargeSize", withNewTestEnvironment(grpcTestListener, withLargeSize))
	t.Run("TestInstallAddOnOnlyOnce", withNewTestEnvironment(grpcTestListener, withInstallAddOnOnlyOnce))
	t.Run("TestAddOnInstallWithFeatureRootAccess", withNewTestEnvironment(grpcTestListener, InstallWithFeatureRootAccess))
	t.Run("TestAddOnUpdateWithFeatureRootAccess", withNewTestEnvironment(grpcTestListener, updateWithFeatureRootAccess))
	t.Run("TestInstallAddOnWithPublicVolume", withNewTestEnvironment(grpcTestListener, withPublicVolume))
	t.Run("TestInstallAddOnWithPublicVolumeAndUpdate", withNewTestEnvironment(grpcTestListener, withPublicVolumeAndUpdate))
	t.Run("TestInstallAddOnWithPublicVolumeAndAccess", withNewTestEnvironment(grpcTestListener, withPublicVolumeAndAccess))
	t.Run("TestInstallAddOnWithCodeName", withNewTestEnvironment(grpcTestListener, withCodeName))
	t.Run("Test_InstallAddOn_PortCheck", withNewTestEnvironment(grpcTestListener, withPortCheck))
	t.Run("TestInstallAddOnWithMultiService", withNewTestEnvironment(grpcTestListener, withMultiService))
	t.Run("TestCheckStackForVersionLabel", withNewTestEnvironment(grpcTestListener, checkStackForVersionLabel))
	t.Run("TestLargeFileUpload", withNewTestEnvironment(grpcTestListener, withLargeFileUpload))
}

func TestAssetsDirPermission(t *testing.T) {
	// Arrange
	initialGrpcTestListener := bufconn.Listen(testhelpers.BufSize)
	testEnvironment, err := testhelpers.NewTestEnvironment(context.Background(), initialGrpcTestListener)
	assert.NoError(t, err)

	testhelpers.CreateAndConnectToNewUcAomInstance(t, initialGrpcTestListener)

	// Act
	listAddOnsResponse, err := testEnvironment.GetCatalogue()
	assert.NoError(t, err)

	// Assert
	for _, addOn := range listAddOnsResponse.AddOns {
		addOnPath := path.Join(catalogue.ASSETS_TMP_PATH, addOn.Name)
		manifest := path.Join(addOnPath, sharedConfig.UcImageManifestFilename)
		logoName := path.Base(addOn.Logo)
		logo := path.Join(addOnPath, logoName)

		addOnPathInfo, err := os.Stat(addOnPath)
		assert.Nil(t, err)
		assert.Equal(t, addOnPathInfo.Mode(), os.ModeDir|os.FileMode(0755))

		manifestInfo, err := os.Stat(manifest)
		assert.Nil(t, err)
		assert.Equal(t, manifestInfo.Mode(), os.FileMode(0644))

		logoInfo, err := os.Stat(logo)
		assert.Nil(t, err)
		assert.Equal(t, logoInfo.Mode(), os.FileMode(0644))

		err = testhelpers.HttpGetWithRetriesAndExpectedStatus(fmt.Sprintf("http://nginx%s", addOn.Logo), http.StatusOK)
		assert.Nil(t, err)
	}
}

func TestStartAddOnOnBoot(t *testing.T) {
	// Arrange
	initialGrpcTestListener := bufconn.Listen(testhelpers.BufSize)
	initialTestEnvironment, err := testhelpers.NewTestEnvironment(context.Background(), initialGrpcTestListener)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	testhelpers.CreateAndConnectToNewUcAomInstance(t, initialGrpcTestListener)
	addOn := initialTestEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-stop-after-creation-addon-pkg", "0.1.0-1")
	addOnStatus, err := initialTestEnvironment.GetInstalledAddOnStatus(addOn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if addOnStatus != grpc_api.AddOnStatus_ERROR {
		t.Errorf("Error: Wrong container status")
	}
	initialTestEnvironment.CloseConnection()

	grpcTestListener := bufconn.Listen(testhelpers.BufSize)
	testEnvironment, err := testhelpers.NewTestEnvironment(context.Background(), grpcTestListener)
	defer testEnvironment.CloseConnection()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Act
	testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)
	defer testEnvironment.DeleteTestAddOnRoutine(t, addOn)

	// Assert
	addOnStatus, err = testEnvironment.GetInstalledAddOnStatus(addOn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if addOnStatus != grpc_api.AddOnStatus_RUNNING {
		t.Errorf("Error: Wrong container status")
	}
}

func TestRecreatedInternalBridgeNetwork(t *testing.T) {
	// arrange
	initialGrpcTestListener := bufconn.Listen(testhelpers.BufSize)
	ctx := context.Background()
	initialTestEnvironment, err := testhelpers.NewTestEnvironment(ctx, initialGrpcTestListener)
	assert.NoError(t, err)
	testhelpers.CreateAndConnectToNewUcAomInstance(t, initialGrpcTestListener)
	addOnWithInternalBridge := initialTestEnvironment.CreateTestAddOnRoutine(t, "test-uc-addon-communication-receiver-addon-pkg", "0.1.0-1")
	addOnStatus, err := initialTestEnvironment.GetInstalledAddOnStatus(addOnWithInternalBridge)
	assert.NoError(t, err)
	assert.Equal(t, addOnStatus, grpc_api.AddOnStatus_RUNNING)
	initialTestEnvironment.CloseConnection()

	dockerCli, err := docker.NewDockerCli()
	assert.NoError(t, err)
	dockerClient := dockerCli.Client()
	inspectContainerJSON, err := dockerClient.ContainerInspect(initialTestEnvironment.Ctx, addOnWithInternalBridge.Title)
	assert.NoError(t, err)
	aliasesAfterCreate := inspectContainerJSON.NetworkSettings.Networks[manifest.InternalAddOnNetworkName].Aliases

	// act
	grpcTestListener := bufconn.Listen(testhelpers.BufSize)
	testEnvironment, err := testhelpers.NewTestEnvironment(context.Background(), grpcTestListener)
	assert.NoError(t, err)
	testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)

	// assert
	addOnStatusAfterReboot, err := testEnvironment.GetInstalledAddOnStatus(addOnWithInternalBridge)
	assert.NoError(t, err)
	assert.Equal(t, addOnStatusAfterReboot, grpc_api.AddOnStatus_RUNNING)

	inspectContainerJSONAfterReconnect, err := dockerClient.ContainerInspect(initialTestEnvironment.Ctx, addOnWithInternalBridge.Title)
	assert.NoError(t, err)
	aliasesAfterReconnect := inspectContainerJSONAfterReconnect.NetworkSettings.Networks[manifest.InternalAddOnNetworkName].Aliases

	assert.Equal(t, aliasesAfterCreate, aliasesAfterReconnect)

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOnWithInternalBridge)
		testEnvironment.CloseConnection()
	})

}
