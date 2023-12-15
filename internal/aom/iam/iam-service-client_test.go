// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package iam

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func createUut(url string, name string, endpoint string, httpClient HttpClient) *IamServiceClient {
	return &IamServiceClient{Url: url, Name: name, Endpoint: endpoint, httpClient: httpClient}
}

func TestReadIamResponseWithTrueResponse(t *testing.T) {
	// arrange
	trueResponse := iamServiceResponse{
		Result: permissionResult{
			Allow: true,
		},
	}

	doFunc := createHttpStatusOKResponseDoFuncWith(trueResponse)
	httpClient := &iamHttpClientMock{DoFunc: doFunc}
	uut := createUut("", "", "", httpClient)

	// act
	result, err := uut.IsAllowed("", "")

	// assert
	if err != nil {
		t.Error(err)
		return
	}

	if result != true {
		t.Errorf("Result should be true, but is %t", result)
	}

}

func TestReadIamResponseWithFalseResponse(t *testing.T) {
	// arrange
	falseResponse := iamServiceResponse{
		Result: permissionResult{
			Allow: false,
		},
	}

	doFunc := createHttpStatusOKResponseDoFuncWith(falseResponse)
	httpClient := &iamHttpClientMock{DoFunc: doFunc}
	uut := createUut("", "", "", httpClient)

	// act
	result, err := uut.IsAllowed("", "")

	// assert
	if err != nil {
		t.Error(err)
		return
	}

	if result != false {
		t.Errorf("Result should be false, but is %t", result)
	}

}

func TestReadIamResponseWithEmptyResponse(t *testing.T) {
	// arrange
	emptyResponse := iamServiceResponse{}
	doFunc := createHttpStatusOKResponseDoFuncWith(emptyResponse)
	httpClient := &iamHttpClientMock{DoFunc: doFunc}
	uut := createUut("", "", "", httpClient)

	// act
	result, err := uut.IsAllowed("", "")

	// assert
	if err != nil {
		t.Error(err)
		return
	}

	if result != false {
		t.Errorf("Result should be false, but is %t", result)
	}

}

func TestReturnErrorFromHttpClient(t *testing.T) {
	// arrange
	doFunc := createDoFuncWithError(errors.New("test-error"))
	httpClient := &iamHttpClientMock{DoFunc: doFunc}
	uut := createUut("", "", "", httpClient)

	// act
	_, err := uut.IsAllowed("", "")

	// assert
	if err == nil {
		t.Error("Error is nil")
		return
	}
	if err.Error() != "test-error" {
		t.Errorf("Result should be 'test-error', but is %s", err.Error())
	}

}

func TestCheckHttpSettings(t *testing.T) {
	// arrange
	trueResponse := iamServiceResponse{
		Result: permissionResult{
			Allow: true,
		},
	}

	doFunc := createHttpStatusOKResponseDoFuncWith(trueResponse)

	httpClient := &iamHttpClientMock{DoFunc: doFunc}
	uut := createUut("url", "service", "/endpoint", httpClient)

	// act
	uut.IsAllowed("", "")

	// assert

	if httpClient.Req.Method != http.MethodPost {
		t.Errorf("Method should be '%s', but is %s", http.MethodPost, httpClient.Req.Method)
	}

	isExpectedEqualActual("url"+"/endpoint", httpClient.Req.URL.Path, t)

	isExpectedEqualActual("application/json", httpClient.Req.Header.Get("Content-Type"), t)
}

func TestReturnHttpStatus(t *testing.T) {
	// arrange
	emptyResponse := iamServiceResponse{}
	doFunc := createDoFuncWithStatusCodeAndResponseBody(http.StatusInternalServerError, "error-test", emptyResponse)

	httpClient := &iamHttpClientMock{DoFunc: doFunc}
	uut := createUut("url", "service", "/endpoint", httpClient)

	// act
	_, err := uut.IsAllowed("", "")

	// assert
	isExpectedEqualActual("error-test", err.Error(), t)
}

func TestReadIamResponseWithWrongTypeResponse(t *testing.T) {

	// arrange
	type WrongStruct struct {
		Result string `json:"result"`
	}
	readCloser := convertToReadCloser(WrongStruct{Result: "test"})

	// act
	_, err := readIamServiceResponseFrom(readCloser)

	// assert
	if err == nil {
		t.Errorf("Should return an error")
		return
	}
}

func isExpectedEqualActual(expected string, actual string, t *testing.T) {
	if actual != expected {
		t.Errorf("Result should be '%s', but is %s", expected, actual)
	}
}

func convertToReadCloser(testResponse interface{}) io.ReadCloser {
	reqBody, _ := json.Marshal(testResponse)
	byteReader := bytes.NewReader(reqBody)
	return io.NopCloser(byteReader)
}
