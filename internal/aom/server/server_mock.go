// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package server

import (
	"u-control/uc-aom/internal/aom/env"
	"u-control/uc-aom/internal/aom/service"
	"u-control/uc-aom/internal/aom/status"

	"github.com/stretchr/testify/mock"
)

type IamClientMock struct {
	mock.Mock
}

func (r *IamClientMock) IsAllowed(token string, permission string) (bool, error) {
	args := r.Called(token, permission)
	return args.Get(0).(bool), args.Error(1)
}

func NewServerUsingServiceMultiComponentMock() (*AddOnServer, *service.ServiceMultiComponentMock, *IamClientMock) {
	mockObj := &service.ServiceMultiComponentMock{}
	serviceStub := mockObj.NewServiceUsingServiceMultiComponentMock()
	iamClientMock := &IamClientMock{}
	statusResolver := status.NewAddOnStatusResolver(mockObj.AddOnStatusResolver)
	envResolver := env.NewAddOnEnvironmentResolver(mockObj)
	transactionResolver := service.NewTransactionScheduler()

	uut := NewServer(serviceStub, "", "", mockObj, nil, iamClientMock, iamClientMock, statusResolver, envResolver, transactionResolver)
	return uut, mockObj, iamClientMock
}
