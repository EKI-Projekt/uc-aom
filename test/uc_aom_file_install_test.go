// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
	aom_cmd "u-control/uc-aom/internal/aom/cmd"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	aop_cmd "u-control/uc-aom/internal/aop/cmd"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/test/bufconn"
)

type testAddOn struct {
	RepositoryName      string
	Version             string
	DropInPath          string
	expectTobeInstalled bool
}

type addonInfoType struct {
	repositoryname      string
	version             string
	expectTobeInstalled bool
}

func TestUcAomDrop(t *testing.T) {
	log.SetLevel(log.TraceLevel)
	tests := []struct {
		name       string
		addonInfos []addonInfoType
	}{{
		name: "Install one add-on via drop-in",
		addonInfos: []addonInfoType{{
			repositoryname: "test-uc-addon-status-running-addon-pkg",
			version:        "0.1.0-1",
		}},
	}, {
		name: "Install multi add-ons via drop-in",
		addonInfos: []addonInfoType{
			{
				repositoryname: "test-uc-addon-status-running-addon-pkg",
				version:        "0.1.0-1",
			},
			{
				repositoryname: "test-uc-addon-provide-ui-addon-pkg",
				version:        "0.1.0-1",
			},
		},
	},
	}

	prepareEnvironment(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// arrange
			dropInTests := make([]*testAddOn, len(tt.addonInfos))
			for index, addonInfo := range tt.addonInfos {
				dropInTest := newDropInTest(t, &addonInfo)
				dropInTests[index] = dropInTest
			}
			grpcTestListener := bufconn.Listen(bufSize)
			ctx := context.Background()
			testEnvironment, err := newTestEnvironment(ctx, grpcTestListener)
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() {
				testEnvironment.conn.Close()
			})

			// act
			startUcAom(t, grpcTestListener)

			// assert
			for _, dropInTest := range dropInTests {
				compareDropInWithInstalledAddOn(t, testEnvironment, dropInTest)
			}
		})
	}
}

func TestUcAomDropInOnlyInstallAddOnOnce(t *testing.T) {
	// Arrange
	addOnName := "test-uc-addon-status-running-addon-pkg"
	testAddOns := []addonInfoType{{
		repositoryname: addOnName,
		version:        "0.2.0-1",
	}, {
		repositoryname: addOnName,
		version:        "0.1.0-1",
	}}

	for _, testAddOn := range testAddOns {
		newDropInTest(t, &testAddOn)
	}

	grpcTestListener := bufconn.Listen(bufSize)
	ctx := context.Background()
	testEnvironment, err := newTestEnvironment(ctx, grpcTestListener)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		testEnvironment.conn.Close()
	})

	// Act
	startUcAom(t, grpcTestListener)

	// Assert
	getAddOnReq := &grpc_api.GetAddOnRequest{
		Name:    addOnName,
		Version: "",
		Filter:  grpc_api.GetAddOnRequest_INSTALLED,
		View:    grpc_api.AddOnView_FULL,
	}
	resultAddOn, err := testEnvironment.client.GetAddOn(ctx, getAddOnReq)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	expectedAddOnVersion := "0.2.0-1"
	if resultAddOn.Version != expectedAddOnVersion {
		t.Errorf("Expected addon version %s but got %s", expectedAddOnVersion, resultAddOn.Version)
	}

}

func TestUcAomWithSwuUpload(t *testing.T) {
	// Arrange
	tmpDir := t.TempDir()

	swuFileTest := &testAddOn{
		RepositoryName:      "test-uc-addon-status-running-addon-pkg",
		Version:             "0.1.0-1",
		DropInPath:          tmpDir,
		expectTobeInstalled: true,
	}

	grpcTestListener := bufconn.Listen(bufSize)
	ctx := context.Background()
	testEnvironment, err := newTestEnvironment(ctx, grpcTestListener)
	if err != nil {
		t.Fatal(err)
	}

	// Act
	startUcAom(t, grpcTestListener)

	addOnList, err := listAddOns(testEnvironment.client, testEnvironment.ctx, grpc_api.ListAddOnsRequest_INSTALLED)
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

func startUcAom(t *testing.T, grpcTestListener *bufconn.Listener) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(1)
	go func() {
		t.Helper()
		ucAom := aom_cmd.NewUcAom(grpcTestListener)
		if err := ucAom.Setup(); err != nil {
			t.Error(err)
		}
		waitGroup.Done()
		if err := ucAom.Run(); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	// check if aom was started
	waitGroup.Wait()
}

func compareDropInWithInstalledAddOn(t *testing.T, testEnvironment *TestEnvironment, dropInTest *testAddOn) {
	getAddOnReq := &grpc_api.GetAddOnRequest{
		Name:    dropInTest.RepositoryName,
		Version: dropInTest.Version,
		Filter:  grpc_api.GetAddOnRequest_INSTALLED,
		View:    grpc_api.AddOnView_FULL,
	}

	addOn, err := testEnvironment.client.GetAddOn(testEnvironment.ctx, getAddOnReq)
	if err != nil {
		t.Fatal(err)
	}

	if addOn.Version != dropInTest.Version {
		t.Errorf("Expected %s, Actual %s", dropInTest.Version, addOn.Version)
	}

	t.Cleanup(func() {
		testEnvironment.DeleteTestAddOnRoutine(t, addOn)
	})
}

func newDropInTest(t *testing.T, addInfo *addonInfoType) *testAddOn {
	t.Helper()
	dropInTest := testAddOn{
		RepositoryName:      addInfo.repositoryname,
		Version:             addInfo.version,
		expectTobeInstalled: addInfo.expectTobeInstalled,
	}
	dropInTest.DropInPath = initializeDropInPath(t, dropInTest.RepositoryName, dropInTest.Version)
	initDropInTestWithAddOnPackager(t, &dropInTest)
	return &dropInTest
}

func initDropInTestWithAddOnPackager(t *testing.T, dropInTest *testAddOn) {
	targetCredentialsFilepath := createPackagerTargetCredentials(t, dropInTest.RepositoryName)
	pullCmd := aop_cmd.NewPullCmd()
	pullCmd.SetArgs([]string{"--target-credentials", targetCredentialsFilepath, "--output", dropInTest.DropInPath, "--version", dropInTest.Version, "-x=false"})

	if got := pullCmd.Execute(); got != nil {
		t.Errorf("NewPullCmd() = %v, unexpected error", got)
	}
}

func validateAddOnInstalledWithTimeout(t *testing.T, testEnvironment *TestEnvironment, maximumRetries int, timeBetweenTries time.Duration, addOnName string) {
	retries := 0
	for {
		if retries >= maximumRetries {
			t.Fatal("Timeout: Error Add-On wasn't installed.")
		}
		err := validateAddOnInstalled(testEnvironment.client, testEnvironment.ctx, addOnName)
		if err == nil {
			break
		}
		time.Sleep(timeBetweenTries)
		retries++
	}
}

func exportAddOnAsSwu(t *testing.T, output string, addonInfo *testAddOn) string {
	t.Helper()
	targetCredentialsFilepath := createPackagerTargetCredentials(t, addonInfo.RepositoryName)
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
