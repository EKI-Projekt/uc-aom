// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package routes

import "github.com/stretchr/testify/mock"

type MockReverseProxyCreater struct {
	mock.Mock
}

func (m *MockReverseProxyCreater) Create(filenameId string, reverseProxyMap *ReverseProxyMap, reverseProxyHttpConf *ReverseProxyHttpConf) error {
	args := m.Called(filenameId, reverseProxyMap, reverseProxyHttpConf)
	return args.Error(0)
}

func (m *MockReverseProxyCreater) Delete(filenameId string) error {
	args := m.Called(filenameId)
	return args.Error(0)
}
