// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/cmd"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/registry"
	"u-control/uc-aom/internal/aop/credentials"

	"github.com/gorilla/websocket"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const (
	numRetries = 5
	bufSize    = 1024 * 1024
)

type addOnErrorTuple struct {
	addOn *grpc_api.AddOn
	err   error
}

type streamReturningAddOn interface {
	Recv() (*grpc_api.AddOn, error)
	grpc.ClientStream
}

func prepareEnvironment(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()

	previousDropInPath := registry.DROP_IN_PATH
	previousAssetInstallPath := catalogue.ASSETS_INSTALL_PATH
	previousAssetTmpPath := catalogue.ASSETS_TMP_PATH

	t.Cleanup(func() {
		registry.DROP_IN_PATH = previousDropInPath
		catalogue.ASSETS_INSTALL_PATH = previousAssetInstallPath
		catalogue.ASSETS_TMP_PATH = previousAssetTmpPath
	})

	registry.DROP_IN_PATH = path.Join(tmpDir, "/var/cache/uc-aom/drop-in/")

	catalogue.ASSETS_INSTALL_PATH = path.Join(tmpDir, "/var/lib/uc-aom")
	err := os.MkdirAll(catalogue.ASSETS_INSTALL_PATH, os.ModePerm)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	catalogue.ASSETS_TMP_PATH = path.Join(tmpDir, "/var/run/uc-aom")
	err = os.MkdirAll(catalogue.ASSETS_TMP_PATH, os.ModePerm)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func createAndConnectToNewUcAomInstance(t *testing.T, grpcTestListener *bufconn.Listener) error {
	prepareEnvironment(t)
	ucAom := cmd.NewUcAom(grpcTestListener)
	err := ucAom.Setup()
	if err != nil {
		return err
	}

	go func() {
		if err := ucAom.Run(); err != nil {
			log.Fatalf("Server exited with error: %v", err)
		}
	}()

	return nil
}

func contains(settings []*grpc_api.Setting, name string) bool {
	for i := range settings {
		if settings[i].Name == name {
			return true
		}
	}
	return false
}

func await(stream streamReturningAddOn) (*grpc_api.AddOn, error) {
	waitc := make(chan addOnErrorTuple)
	go func() {
		var caught *grpc_api.AddOn
		for {
			in, err := stream.Recv()
			if in != nil {
				caught = in
			}
			if err != nil {
				if errors.Is(err, io.EOF) {
					waitc <- addOnErrorTuple{caught, nil}
				} else {
					waitc <- addOnErrorTuple{caught, err}
				}
				close(waitc)
				return
			}
		}
	}()

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to close the stream: %+v", err)
	}

	// Block
	if chanResult, ok := <-waitc; ok {
		return chanResult.addOn, chanResult.err
	}
	return nil, errors.New("Failed to read channel")
}

func createAddOn(client grpc_api.AddOnServiceClient, ctx context.Context, input *grpc_api.AddOn) (*grpc_api.AddOn, error) {
	req := &grpc_api.CreateAddOnRequest{
		AddOn: input,
	}

	stream, err := client.CreateAddOn(ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

func deleteAddOn(client grpc_api.AddOnServiceClient, ctx context.Context, addOn *grpc_api.AddOn) error {
	req := &grpc_api.DeleteAddOnRequest{
		Name:  addOn.Name,
		Title: addOn.Title,
	}

	stream, err := client.DeleteAddOn(ctx, req)
	if err != nil {
		return err
	}

	waitc := make(chan struct{})
	go func() {
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Delete AddOn Failed: %+v", err)
			}
		}
	}()

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to delete AddOn: %+v", err)
	}

	// Block
	<-waitc
	return nil
}

func getInstalledAddOn(client grpc_api.AddOnServiceClient, ctx context.Context, name string, view grpc_api.AddOnView) (*grpc_api.AddOn, error) {
	req := &grpc_api.GetAddOnRequest{
		Name:   name,
		View:   view,
		Filter: grpc_api.GetAddOnRequest_INSTALLED,
	}

	return client.GetAddOn(ctx, req)
}

func getCatalogueAddOn(client grpc_api.AddOnServiceClient, ctx context.Context, name string, view grpc_api.AddOnView) (*grpc_api.AddOn, error) {
	req := &grpc_api.GetAddOnRequest{
		Name:   name,
		View:   view,
		Filter: grpc_api.GetAddOnRequest_CATALOGUE,
	}

	return client.GetAddOn(ctx, req)
}

func validateAddOnInstalled(client grpc_api.AddOnServiceClient, ctx context.Context, name string) error {
	installed, err := listAddOns(client, ctx, grpc_api.ListAddOnsRequest_INSTALLED)
	if err != nil {
		return err
	}

	for i := range installed.AddOns {
		if name != installed.AddOns[i].Name {
			continue
		}
		return nil
	}

	return fs.ErrNotExist
}

func getInstalledAddOnStatus(TestEnvironment *TestEnvironment, addOn *grpc_api.AddOn) (grpc_api.AddOnStatus, error) {
	channel := make(chan grpc_api.AddOnStatus)
	go func() {
		retries := 3
		for ; retries > 0; retries-- {
			returnAddOn, err := getInstalledAddOn(TestEnvironment.client, TestEnvironment.ctx, addOn.Name, grpc_api.AddOnView_FULL)
			if err != nil {
				log.Fatalf("Failed to get installed AddOn: %+v", err)
			}
			if returnAddOn.Status != grpc_api.AddOnStatus_STARTING {
				channel <- returnAddOn.Status
				break
			}
			time.Sleep(time.Second * 1)
		}
	}()
	return <-channel, nil
}

func listAddOns(client grpc_api.AddOnServiceClient, ctx context.Context, filter grpc_api.ListAddOnsRequest_Filter) (*grpc_api.ListAddOnsResponse, error) {
	req := &grpc_api.ListAddOnsRequest{
		Name:   "",
		View:   grpc_api.AddOnView_BASIC,
		Filter: filter,
	}

	stream, err := client.ListAddOns(ctx, req)
	if err != nil {
		return nil, err
	}

	waitc := make(chan *grpc_api.ListAddOnsResponse)
	go func() {
		var caught *grpc_api.ListAddOnsResponse
		for {
			in, err := stream.Recv()
			if in != nil {
				caught = in
			}
			if err == io.EOF {
				waitc <- caught
				close(waitc)
				return
			}
			if err != nil {
				log.Fatalf("Failed to receive the AddOn: %+v", err)
			}
		}
	}()

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to create the AddOn: %+v", err)
	}

	// Block
	if last, ok := <-waitc; ok {
		return last, nil
	}
	return nil, errors.New("Failed to read channel")
}

func updateAddOn(client grpc_api.AddOnServiceClient, ctx context.Context, addOn *grpc_api.AddOn) (*grpc_api.AddOn, error) {
	req := &grpc_api.UpdateAddOnRequest{
		AddOn: addOn,
	}

	stream, err := client.UpdateAddOn(ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

func httpGetBodyWithRetries(url string) (string, error) {
	var resp *http.Response
	var err error
	for i := 0; i < numRetries; i++ {
		resp, err = http.Get(url)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			err = errors.New(fmt.Sprintf("Error in http response status code: %d", resp.StatusCode))
			continue
		}

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		return string(content), nil
	}

	return "", err
}

func httpGetWithRetriesAndExpectedStatus(url string, expectedStatus int) error {
	var resp *http.Response
	var err error
	for i := 0; i < numRetries; i++ {
		resp, err = http.Get(url)
		if resp != nil && resp.StatusCode == expectedStatus {
			return nil
		}
	}
	if err != nil {
		return err
	}
	return errors.New(fmt.Sprintf("Error in http response status code: %d", resp.StatusCode))
}

func wsDialWithRetries(url string) (*websocket.Conn, error) {
	var con *websocket.Conn
	var err error
	for i := 0; i < numRetries; i++ {
		con, _, err = websocket.DefaultDialer.Dial(url, nil)
		if con != nil {
			return con, nil
		}
	}
	return nil, err
}

func wsAssertEchoWorking(t *testing.T, con *websocket.Conn) {
	testMessage := []byte("test-message")

	err := con.WriteMessage(websocket.TextMessage, testMessage)
	if err != nil {
		t.Errorf("Unable to write to websocket: %v", err)
	}

	_, response, err := con.ReadMessage()
	if err != nil {
		t.Errorf("Unable to read from websocket: %v", err)
	}

	if bytes.Compare(response, testMessage) != 0 {
		t.Errorf("Websocket echo not working, sent %v, but got: %v", testMessage, response)
	}
}

// Create a file at a temp directory with insecure credentials and return the filepath
func createPackagerTargetCredentials(t *testing.T, repositoryname string) string {
	t.Helper()
	c := credentials.Credentials{}
	c.RepositoryName = repositoryname
	data, _ := json.Marshal(&c)
	targetCredentialsFilepath := filepath.Join(t.TempDir(), "target-credentials.json")
	os.WriteFile(targetCredentialsFilepath, data, 0644)
	return targetCredentialsFilepath
}

// create a drop in path for the add-on, delete after test and
func initializeDropInPath(t *testing.T, repositoryName string, addOnVersion string) string {
	dropInPath := filepath.Join(registry.DROP_IN_PATH)

	err := os.MkdirAll(dropInPath, os.ModePerm)
	if err != nil {
		t.Errorf("os.Mkdir() = %v, unexpected error", err)
	}

	cleanUpCallback := func() {
		err = os.RemoveAll(registry.DROP_IN_PATH)
		if err != nil {
			t.Logf("os.RemoveAll() = %v, unexpected error", err)
		}
	}

	t.Cleanup(cleanUpCallback)
	return dropInPath
}
