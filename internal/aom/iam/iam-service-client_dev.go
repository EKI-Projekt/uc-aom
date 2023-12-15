// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package iam

// NewIamServiceClient creates an instance of IamServiceClient
func NewIamServiceClient(url string, name string, endpoint string) *IamServiceClient {
	alwaysAllowInDevModeResponse := iamServiceResponse{
		Result: permissionResult{
			Allow: true,
		},
	}
	doFunc := createHttpStatusOKResponseDoFuncWith(alwaysAllowInDevModeResponse)
	httpClient := &iamHttpClientMock{DoFunc: doFunc}
	return &IamServiceClient{Url: url, Name: name, Endpoint: endpoint, httpClient: httpClient}
}
