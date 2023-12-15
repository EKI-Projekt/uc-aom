// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package testhelpers

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
	"path/filepath"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/cmd"
	grpc_api "u-control/uc-aom/internal/aom/grpc"
	"u-control/uc-aom/internal/aom/utils"
	"u-control/uc-aom/internal/aop/credentials"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"

	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const (
	numRetries = 5
	BufSize    = 1024 * 1024
)

type addOnErrorTuple struct {
	addOn *grpc_api.AddOn
	err   error
}

type streamReturningAddOn interface {
	Recv() (*grpc_api.AddOn, error)
	grpc.ClientStream
}

func PrepareEnvironment(t *testing.T) {
	t.Helper()

	err := os.MkdirAll(catalogue.ASSETS_INSTALL_PATH, os.ModePerm)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	err = os.MkdirAll(catalogue.ASSETS_TMP_PATH, os.ModePerm)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(catalogue.ASSETS_INSTALL_PATH)
		os.RemoveAll(catalogue.ASSETS_TMP_PATH)
	})

}

func CreateAndConnectToNewUcAomInstance(t *testing.T, grpcTestListener *bufconn.Listener) error {
	PrepareEnvironment(t)
	return CreateAndConnectToNewUcAomInstanceWithoutPrepare(t, grpcTestListener)
}
func CreateAndConnectToNewUcAomInstanceWithoutPrepare(t *testing.T, grpcTestListener *bufconn.Listener) error {
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

func CreateAddOn(client grpc_api.AddOnServiceClient, ctx context.Context, input *grpc_api.AddOn) (*grpc_api.AddOn, error) {
	req := &grpc_api.CreateAddOnRequest{
		AddOn: input,
	}

	stream, err := client.CreateAddOn(ctx, req)
	if err != nil {
		return nil, err
	}

	return await(stream)
}

func DeleteAddOn(client grpc_api.AddOnServiceClient, ctx context.Context, addOn *grpc_api.AddOn) error {
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

func ValidateAddOnInstalled(testEnv *TestEnvironment, name string) error {
	installed, err := listAddOns(testEnv, grpc_api.ListAddOnsRequest_INSTALLED)
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

func listAddOns(testEnv *TestEnvironment, filter grpc_api.ListAddOnsRequest_Filter) (*grpc_api.ListAddOnsResponse, error) {
	req := &grpc_api.ListAddOnsRequest{
		Name:   "",
		View:   grpc_api.AddOnView_BASIC,
		Filter: filter,
	}

	stream, err := testEnv.Client.ListAddOns(testEnv.Ctx, req)
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

func HttpGetBodyWithRetries(url string) (string, error) {
	res, _ := utils.Retry(numRetries, 0, func() (interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}

		if resp.StatusCode != http.StatusOK {
			err = errors.New(fmt.Sprintf("Error in http response status code: %d", resp.StatusCode))
			return "", err
		}

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return string(content), nil
	})
	return res.(string), nil

}

func HttpGetWithRetriesAndExpectedStatus(url string, expectedStatus int) error {
	_, err := utils.Retry(numRetries, 0, func() (interface{}, error) {
		resp, err := http.Get(url)
		if err != nil {
			return "", err
		}
		if resp.StatusCode == expectedStatus {
			return "", nil
		}
		return "", errors.New(fmt.Sprintf("Error in http response status code: %d", resp.StatusCode))
	})

	return err
}

func WsDialWithRetries(url string) (*websocket.Conn, error) {
	res, err := utils.Retry(numRetries, 0, func() (interface{}, error) {
		con, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			return nil, err
		}
		return con, nil
	})

	if err != nil {
		return nil, err
	}

	return res.(*websocket.Conn), nil

}

func WsAssertEchoWorking(t *testing.T, con *websocket.Conn) {
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
func CreatePackagerTargetCredentials(t *testing.T, repositoryname string) string {
	t.Helper()
	c := credentials.Credentials{}
	c.RepositoryName = repositoryname
	data, _ := json.Marshal(&c)
	targetCredentialsFilepath := filepath.Join(t.TempDir(), "target-credentials.json")
	os.WriteFile(targetCredentialsFilepath, data, 0644)
	return targetCredentialsFilepath
}

// create a drop in path for the add-on, delete after test and
func InitializeDropInPath(t *testing.T, dropInPath string) {
	err := os.MkdirAll(dropInPath, os.ModePerm)
	if err != nil {
		t.Errorf("os.Mkdir() = %v, unexpected error", err)
	}

	cleanUpCallback := func() {
		err = os.RemoveAll(dropInPath)
		if err != nil {
			t.Logf("os.RemoveAll() = %v, unexpected error", err)
		}
	}

	t.Cleanup(cleanUpCallback)
}

// Set root access enabled to true(1) or false(0)
func SetRootAccess(t *testing.T, setValue int) {
	type configFileJson struct {
		RootAccessEnabled int `json:"rootAccessEnabled"`
	}

	configFile := configFileJson{RootAccessEnabled: setValue}
	contentToWrite, err := json.Marshal(&configFile)
	assert.Nil(t, err)

	configFilePath := os.Getenv("ROOT_ACCESS_CONFIG_FILE")
	assert.NotEmpty(t, configFilePath)

	err = os.WriteFile(configFilePath, contentToWrite, os.ModePerm)
	assert.Nil(t, err)
}

// check if the provided subroutes are accessable
func WalkThroughExpectedRoutesWithBaseUrl(t *testing.T, baseUrl string, expectedSubRoutes []string) {
	const expectedStatusCode = 200
	for index := range expectedSubRoutes {
		urlIndex := index + 1
		urlparts := append([]string{baseUrl}, expectedSubRoutes[:urlIndex]...)
		url := strings.Join(urlparts, "/")

		if urlIndex != len(expectedSubRoutes) {
			// append "/" to access a directory
			url += "/"
		}

		err := HttpGetWithRetriesAndExpectedStatus(url, expectedStatusCode)
		if !assert.Nil(t, err, url) {
			assert.FailNow(t, err.Error())
		}
	}
}
