// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package iam

import "net/http"

// Interface to ask if the token has a specific permission
type IamClient interface {
	IsAllowed(token string, permission string) (bool, error)
}

// Interface to abstract the http client
type HttpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// function to type for the http client do function
type HttpClientDoRequestFunc func(req *http.Request) (*http.Response, error)
