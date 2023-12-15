// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build prod
// +build prod

package iam

import "net/http"

// NewIamServiceClient creates an instance of IamServiceClient
func NewIamServiceClient(url string, name string, endpoint string) *IamServiceClient {
	httpClient := &http.Client{}
	return &IamServiceClient{Url: url, Name: name, Endpoint: endpoint, httpClient: httpClient}
}
