// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package iam

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
)

// Mock for http client. is used in test and den environment
type iamHttpClientMock struct {
	Req    *http.Request
	DoFunc HttpClientDoRequestFunc
}

func (m *iamHttpClientMock) Do(req *http.Request) (*http.Response, error) {
	m.Req = req
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return &http.Response{}, nil
}

func createHttpStatusOKResponseDoFuncWith(responseBody iamServiceResponse) HttpClientDoRequestFunc {
	return createDoFuncWithStatusCodeAndResponseBody(http.StatusOK, "", responseBody)
}

func createDoFuncWithStatusCodeAndResponseBody(statusCode int, status string, responseBody iamServiceResponse) HttpClientDoRequestFunc {
	response := &http.Response{}
	response.StatusCode = statusCode
	response.Status = status
	respBody, err := json.Marshal(responseBody)
	if err != nil {
		return createDoFuncWithError(err)
	}
	return createDoFuncWithHttpResponseAndBody(response, respBody)
}

func createDoFuncWithHttpResponseAndBody(response *http.Response, respBody []byte) HttpClientDoRequestFunc {
	// We cannot make a closure on a reader instance as the contents
	// will be consumed on the first execution of the returned function.
	// In all subsequent calls, the http.Response body will be empty!
	doFunc := func(req *http.Request) (*http.Response, error) {
		byteReader := bytes.NewReader(respBody)
		response.Body = io.NopCloser(byteReader)
		return response, nil
	}
	return doFunc
}

func createDoFuncWithError(err error) HttpClientDoRequestFunc {
	doFunc := func(req *http.Request) (*http.Response, error) {
		return &http.Response{}, err
	}
	return doFunc
}
