// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package dropin

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	aop_cmd "u-control/uc-aom/internal/aop/cmd"
	"u-control/uc-aom/internal/pkg/config"
	testhelpers "u-control/uc-aom/test/test-helpers"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/test/bufconn"
)

type dropInTestType struct {
	RepositoryName      string
	Version             string
	DropInPath          string
	expectTobeInstalled bool
}

func TestUcAomDrop(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	tests := []struct {
		name        string
		dropInTests []dropInTestType
	}{
		{
			name: "Install one add-on via drop-in",
			dropInTests: []dropInTestType{{
				RepositoryName: "test-uc-addon-status-running-addon-pkg",
				Version:        "0.1.0-1",
				DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
			}},
		},
		{
			name: "Install one add-on via persistence drop-in",
			dropInTests: []dropInTestType{{
				RepositoryName: "test-uc-addon-status-running-addon-pkg",
				Version:        "0.1.0-1",
				DropInPath:     filepath.Join(config.PERSISTENCE_DROP_IN_PATH),
			}},
		},
		{
			name: "Install multi add-ons via drop-in",
			dropInTests: []dropInTestType{

				{
					RepositoryName: "test-uc-addon-status-running-addon-pkg",
					Version:        "0.1.0-1",
					DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
				},
				{
					RepositoryName: "test-uc-addon-provide-ui-addon-pkg",
					Version:        "0.1.0-1",
					DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			testhelpers.PrepareEnvironment(t)
			for _, addon := range tt.dropInTests {
				initDropInTest(t, &addon)
			}
			grpcTestListener := bufconn.Listen(testhelpers.BufSize)
			ctx := context.Background()
			testEnvironment, err := testhelpers.NewTestEnvironment(ctx, grpcTestListener)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				testEnvironment.CloseConnection()
			})

			appendCleanupFuncForDropInTests(t, tt.dropInTests...)

			// act
			testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)

			// assert
			for _, dropInTest := range tt.dropInTests {
				compareDropInWithInstalledAddOn(t, testEnvironment, &dropInTest)
			}
		})
	}
}

func TestUcAomDropUpdate(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	tests := []struct {
		name        string
		dropInTests []dropInTestType
	}{{
		name: "Install one add-on via drop-in",
		dropInTests: []dropInTestType{{
			RepositoryName: "test-uc-addon-status-running-addon-pkg",
			Version:        "0.1.0-1",
			DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
		}},
	}, {
		name: "Install update add-ons via drop-in",
		dropInTests: []dropInTestType{
			{
				RepositoryName: "test-uc-addon-status-running-addon-pkg",
				Version:        "0.2.0-1",
				DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
			},
		},
	},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			testhelpers.PrepareEnvironment(t)
			for _, dropInTest := range tt.dropInTests {
				initDropInTest(t, &dropInTest)
			}
			grpcTestListener := bufconn.Listen(testhelpers.BufSize)
			ctx := context.Background()
			testEnvironment, err := testhelpers.NewTestEnvironment(ctx, grpcTestListener)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				testEnvironment.CloseConnection()
			})
			appendCleanupFuncForDropInTests(t, tt.dropInTests...)

			// act
			testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)

			// assert
			for _, dropInTest := range tt.dropInTests {
				if tt.name == "Install update add-ons via drop-in" {
					compareDropInWithInstalledAddOn(t, testEnvironment, &dropInTest)
				} else {
					compareDropInWithInstalledAddOnWithoutCleanup(t, testEnvironment, &dropInTest)
				}
			}
		})
	}
}

func TestUcAomDropWithSettingsUpdate(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	tests := []struct {
		name       string
		dropInTest dropInTestType
		want       []*grpc_api.Setting
		update     []*grpc_api.Setting
	}{{
		name: "Install one add-on via drop-in",
		dropInTest: dropInTestType{
			RepositoryName: "test-uc-addon-settings-addon-pkg",
			Version:        "0.1.0-1",
			DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
		},
		want: []*grpc_api.Setting{
			{
				Name:     "param1",
				Label:    "Param 1",
				Required: true,
				SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: "aaa",
					},
				},
			},
			{
				Name:  "param2",
				Label: "Param 2",
				SettingOneof: &grpc_api.Setting_DropDownList{
					DropDownList: &grpc_api.DropDownList{
						Elements: []*grpc_api.DropDownItem{
							{
								Label: "Abc",
								Value: "abc",
							},
							{
								Label: "Qwe",
								Value: "qwe",
							},
							{
								Label:    "Xyz",
								Value:    "xyz",
								Selected: true,
							},
						},
					},
				},
			},
			{
				Name:     "param4",
				Label:    "Param 4",
				Required: true,
				SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: "p4 default",
					},
				},
			},
		},
		update: []*grpc_api.Setting{
			{
				Name:     "param1",
				Label:    "Param 1",
				Required: true,
				SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: "123456",
					},
				},
			},
			{
				Name:  "param2",
				Label: "Param 2",
				SettingOneof: &grpc_api.Setting_DropDownList{
					DropDownList: &grpc_api.DropDownList{
						Elements: []*grpc_api.DropDownItem{
							{
								Label: "Abc",
								Value: "abc",
							},
							{
								Label:    "Qwe",
								Value:    "qwe",
								Selected: true,
							},
							{
								Label: "Xyz",
								Value: "xyz",
							},
						},
					},
				},
			},
			{
				Name:     "param4",
				Label:    "Param 4",
				Required: true,
				SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{
						Value: "p4 default",
					},
				},
			},
		},
	}, {
		name: "Install update add-ons via drop-in",
		dropInTest: dropInTestType{
			RepositoryName: "test-uc-addon-settings-addon-pkg",
			Version:        "0.2.0-1",
			DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
		},
		want: []*grpc_api.Setting{
			{
				Name:     "param1",
				Label:    "Param 1",
				Required: true,
				SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{Value: "123456"},
				},
			},
			{
				Name:  "param2",
				Label: "Param 2",
				SettingOneof: &grpc_api.Setting_DropDownList{
					DropDownList: &grpc_api.DropDownList{
						Elements: []*grpc_api.DropDownItem{
							{
								Label: "Abc",
								Value: "abc",
							},
							{
								Label:    "Qwe",
								Value:    "qwe",
								Selected: true,
							},
							{
								Label: "Xyz",
								Value: "xyz",
							},
						},
					},
				},
			},
			{
				Name:     "param3",
				Label:    "Param 3",
				Required: true,
				SettingOneof: &grpc_api.Setting_TextBox{
					TextBox: &grpc_api.TextBox{Value: "default"},
				},
			},
		},
	}}
	testhelpers.PrepareEnvironment(t)
	for index, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			initDropInTest(t, &tt.dropInTest)
			grpcTestListener := bufconn.Listen(testhelpers.BufSize)
			ctx := context.Background()
			testEnvironment, err := testhelpers.NewTestEnvironment(ctx, grpcTestListener)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				testEnvironment.CloseConnection()
			})
			appendCleanupFuncForDropInTests(t, tt.dropInTest)

			// act
			testhelpers.CreateAndConnectToNewUcAomInstanceWithoutPrepare(t, grpcTestListener)

			// assert
			var installedAddOnViaDropIn *grpc_api.AddOn

			lastIndex := len(tests) - 1
			if index == lastIndex {
				installedAddOnViaDropIn = compareDropInWithInstalledAddOn(t, testEnvironment, &tt.dropInTest)
			} else {
				installedAddOnViaDropIn = compareDropInWithInstalledAddOnWithoutCleanup(t, testEnvironment, &tt.dropInTest)
			}

			assert.Len(t, installedAddOnViaDropIn.Settings, len(tt.want))

			if !reflect.DeepEqual(installedAddOnViaDropIn.Settings, tt.want) {
				t.Errorf("Addon setting = %v, want %v", installedAddOnViaDropIn.Settings, tt.want)
			}

			if len(tt.update) > 0 {
				installedAddOnViaDropIn.Settings = tt.update
				_, err = testEnvironment.UpdateAddOn(installedAddOnViaDropIn)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
			}

		})
	}
}

func TestUcAomDropInOnlyInstallAddOnOnce(t *testing.T) {
	// Arrange
	testhelpers.PrepareEnvironment(t)
	addOnName := "test-uc-addon-status-running-addon-pkg"
	dropInTests := []dropInTestType{{
		RepositoryName: addOnName,
		Version:        "0.2.0-1",
		DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
	}, {
		RepositoryName: addOnName,
		Version:        "0.1.0-1",
		DropInPath:     filepath.Join(config.CACHE_DROP_IN_PATH),
	}}

	for _, dropInTest := range dropInTests {
		initDropInTest(t, &dropInTest)
	}

	grpcTestListener := bufconn.Listen(testhelpers.BufSize)
	ctx := context.Background()
	testEnvironment, err := testhelpers.NewTestEnvironment(ctx, grpcTestListener)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testEnvironment.CloseConnection()
	})
	appendCleanupFuncForDropInTests(t, dropInTests...)

	// Act
	testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)

	// Assert
	resultAddOn, err := testEnvironment.GetInstalledAddOn(addOnName)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	expectedAddOnVersion := "0.2.0-1"
	if resultAddOn.Version != expectedAddOnVersion {
		t.Errorf("Expected addon version %s but got %s", expectedAddOnVersion, resultAddOn.Version)
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, resultAddOn)
	})
}

func TestUcAomWithSwuUpload(t *testing.T) {
	// Arrange
	testhelpers.PrepareEnvironment(t)
	tmpDir := t.TempDir()

	swuFileTest := &dropInTestType{
		RepositoryName:      "test-uc-addon-status-running-addon-pkg",
		Version:             "0.1.0-1",
		DropInPath:          tmpDir,
		expectTobeInstalled: true,
	}

	grpcTestListener := bufconn.Listen(testhelpers.BufSize)
	ctx := context.Background()
	testEnvironment, err := testhelpers.NewTestEnvironment(ctx, grpcTestListener)
	if err != nil {
		t.Fatal(err)
	}

	// Act
	testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)

	addOnList, err := testEnvironment.GetInstalledAddOns()
	if err != nil {
		t.Fatal(err)
	}
	if len(addOnList.AddOns) != 0 {
		testEnvironment.DeleteTestAddOnRoutine(t, addOnList.AddOns[0])
		t.Fatalf("Expected no Add-Ons to be installed but got '%d'(%v)", len(addOnList.AddOns), addOnList.AddOns)
	}

	// create and upload swu file
	swuFileName := exportAddOnAsSwu(t, tmpDir, swuFileTest)
	pathToSwu := path.Join(tmpDir, swuFileName)
	uploadSwuFile(t, pathToSwu)

	//check if addOn installation is finished
	validateAddOnInstalledWithTimeout(t, testEnvironment, 5, time.Second*3, swuFileTest.RepositoryName)

	// Assert
	compareDropInWithInstalledAddOn(t, testEnvironment, swuFileTest)
}

func compareDropInWithInstalledAddOn(t *testing.T, testEnvironment *testhelpers.TestEnvironment, dropInTest *dropInTestType) *grpc_api.AddOn {
	addOn := compareDropInWithInstalledAddOnWithoutCleanup(t, testEnvironment, dropInTest)
	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})
	return addOn
}

func compareDropInWithInstalledAddOnWithoutCleanup(t *testing.T, testEnvironment *testhelpers.TestEnvironment, dropInTest *dropInTestType) *grpc_api.AddOn {
	addOn, err := testEnvironment.GetInstalledAddOn(dropInTest.RepositoryName)
	if err != nil {
		t.Fatal(err)
		return nil
	}

	if addOn.Version != dropInTest.Version {
		t.Errorf("Expected %s, Actual %s", dropInTest.Version, addOn.Version)
	}
	return addOn
}

func initDropInTest(t *testing.T, dropInTest *dropInTestType) {
	t.Helper()
	testhelpers.InitializeDropInPath(t, dropInTest.DropInPath)
	initDropInTestWithAddOnPackager(t, dropInTest)
}

func initDropInTestWithAddOnPackager(t *testing.T, dropInTest *dropInTestType) {
	targetCredentialsFilepath := testhelpers.CreatePackagerTargetCredentials(t, dropInTest.RepositoryName)
	pullCmd := aop_cmd.NewPullCmd()
	pullCmd.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", dropInTest.DropInPath, "--version", dropInTest.Version, "-x=false"})

	if got := pullCmd.Execute(); got != nil {
		t.Errorf("NewPullCmd() = %v, unexpected error", got)
	}
}

func appendCleanupFuncForDropInTests(t *testing.T, dropInTests ...dropInTestType) {
	for _, dropInTest := range dropInTests {
		t.Cleanup(func() {
			if _, err := os.Stat(dropInTest.DropInPath); err != nil {
				os.RemoveAll(dropInTest.DropInPath)
			}
		})
	}
}

func validateAddOnInstalledWithTimeout(t *testing.T, testEnvironment *testhelpers.TestEnvironment, maximumRetries int, timeBetweenTries time.Duration, addOnName string) {
	retries := 0
	for {
		if retries >= maximumRetries {
			t.Fatal("Timeout: Error Add-On wasn't installed.")
		}
		err := testhelpers.ValidateAddOnInstalled(testEnvironment, addOnName)
		if err == nil {
			break
		}
		time.Sleep(timeBetweenTries)
		retries++
	}
}

func exportAddOnAsSwu(t *testing.T, output string, addonInfo *dropInTestType) string {
	t.Helper()
	targetCredentialsFilepath := testhelpers.CreatePackagerTargetCredentials(t, addonInfo.RepositoryName)
	exportCmd := aop_cmd.NewExportCmd()
	exportCmd.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", output, "--version", addonInfo.Version})
	if got := exportCmd.Execute(); got != nil {
		t.Errorf("NewExportCmd() = %v, unexpected error", got)
	}

	swuFileName := addonInfo.RepositoryName + "_" + addonInfo.Version + ".swu"
	subfoldername := strings.Join([]string{runtime.GOOS, runtime.GOARCH}, "-")
	return path.Join(subfoldername, swuFileName)
}

func uploadSwuFile(t *testing.T, pathToSwu string) {
	t.Helper()
	swuFile, err := os.Open(pathToSwu)
	defer swuFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	httpRequestBody := &bytes.Buffer{}
	writer := multipart.NewWriter(httpRequestBody)
	part, err := writer.CreateFormFile("file", filepath.Base(swuFile.Name()))
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(part, swuFile)
	writer.Close()
	swuUrl := "http://swupdate:8090/upload"

	httpClient := &http.Client{}
	response, err := httpClient.Post(swuUrl, writer.FormDataContentType(), httpRequestBody)
	t.Logf("response.Body: %v", response.Body)
	if err != nil {
		t.Fatalf("Unexpected error uploading swu-file: %v", err)
	}
	expectedStatusCode := 200
	if response.StatusCode != expectedStatusCode {
		t.Fatalf("Error uploading swu-file. Expected status '%d' but got '%d'", expectedStatusCode, response.StatusCode)
	}
}
