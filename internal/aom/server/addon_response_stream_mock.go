// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package server

import (
	"context"
	grpc_api "u-control/uc-aom/internal/aom/grpc"

	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

type AddOnResponseStreamMock struct {
	mock.Mock
}

func (r *AddOnResponseStreamMock) Context() context.Context {
	args := r.Called()
	return args.Get(0).(context.Context)
}

func (r *AddOnResponseStreamMock) RecvMsg(m interface{}) error {
	args := r.Called(m)
	return args.Error(0)
}

func (r *AddOnResponseStreamMock) Send(addOn *grpc_api.AddOn) error {
	args := r.Called(addOn)
	return args.Error(0)
}

func (r *AddOnResponseStreamMock) SendHeader(md metadata.MD) error {
	args := r.Called(md)
	return args.Error(0)
}

func (r *AddOnResponseStreamMock) SendMsg(m interface{}) error {
	args := r.Called(m)
	return args.Error(0)
}

func (r *AddOnResponseStreamMock) SetHeader(md metadata.MD) error {
	args := r.Called(md)
	return args.Error(0)
}

func (r *AddOnResponseStreamMock) SetTrailer(md metadata.MD) {
	r.Called(md)
}

func NewAddOnResponseStreamMock(ctx context.Context, iamClientMock *IamClientMock) *AddOnResponseStreamMock {
	mockObj := &AddOnResponseStreamMock{}
	md := metadata.New(map[string]string{"authorization": "Bearer 12345"})
	ctx = metadata.NewIncomingContext(ctx, md)
	mockObj.On("Context").Return(ctx)
	iamClientMock.On("IsAllowed", "12345", mock.AnythingOfType("string")).Return(true, nil)
	return mockObj
}
