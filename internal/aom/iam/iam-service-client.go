// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package iam

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type IamServiceClient struct {
	// Url location of an OPA i.e. http://localhost:49155
	Url string

	// Service name e.g. uc-aom
	Name string

	// Service endpoint
	Endpoint string

	httpClient HttpClient
}

type iamServiceRequest struct {
	Input permissionInput `json:"input"`
}

type permissionInput struct {
	Token      string `json:"token"`
	Permission string `json:"permission"`
	Service    string `json:"service"`
}

type iamServiceResponse struct {
	Result permissionResult `json:"result"`
}

type permissionResult struct {
	Allow bool `json:"allow"`
}

func (iamServiceClient *IamServiceClient) IsAllowed(jwt string, permission string) (bool, error) {
	request := iamServiceRequest{
		Input: permissionInput{
			Token:      jwt,
			Permission: permission,
			Service:    iamServiceClient.Name,
		},
	}

	hasPermission, err := iamServiceClient.askIamServiceForPermission(request)
	return hasPermission, err
}

func (iamServiceClient *IamServiceClient) askIamServiceForPermission(request iamServiceRequest) (bool, error) {
	reqBody, err := json.Marshal(request)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest(http.MethodPost, iamServiceClient.Url+iamServiceClient.Endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := iamServiceClient.httpClient.Do(req)
	if err != nil {
		return false, err
	}

	// If the returned error is nil, the Response will contain a non-nil Body which the user is expected to close.
	// See: https://pkg.go.dev/net/http@go1.17.7#Client.Do

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return readIamServiceResponseFrom(resp.Body)
	}

	return false, errors.New(resp.Status)
}

func readIamServiceResponseFrom(httpBody io.ReadCloser) (bool, error) {
	respBody, err := io.ReadAll(httpBody)
	if err != nil {
		return false, err
	}

	response := iamServiceResponse{}
	err = json.Unmarshal([]byte(respBody), &response)
	if err != nil {
		return false, err
	}

	return response.Result.Allow, nil
}
