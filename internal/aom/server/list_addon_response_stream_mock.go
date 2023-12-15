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

type ListAddOnResponseStreamMock struct {
	mock.Mock
	capture chan *grpc_api.ListAddOnsResponse
}

func (r *ListAddOnResponseStreamMock) Context() context.Context {
	args := r.Called()
	return args.Get(0).(context.Context)
}

func (r *ListAddOnResponseStreamMock) RecvMsg(m interface{}) error {
	args := r.Called(m)
	return args.Error(0)
}

func (r *ListAddOnResponseStreamMock) Send(listAddOn *grpc_api.ListAddOnsResponse) error {
	if r.capture != nil {
		r.capture <- listAddOn
	}
	args := r.Called(listAddOn)
	return args.Error(0)
}

func (r *ListAddOnResponseStreamMock) SendHeader(md metadata.MD) error {
	args := r.Called(md)
	return args.Error(0)
}

func (r *ListAddOnResponseStreamMock) SendMsg(m interface{}) error {
	args := r.Called(m)
	return args.Error(0)
}

func (r *ListAddOnResponseStreamMock) SetHeader(md metadata.MD) error {
	args := r.Called(md)
	return args.Error(0)
}

func (r *ListAddOnResponseStreamMock) SetTrailer(md metadata.MD) {
	r.Called(md)
}

func NewListAddOnResponseStreamMock(ctx context.Context, iamClientMock *IamClientMock, capture chan *grpc_api.ListAddOnsResponse) *ListAddOnResponseStreamMock {
	mockObj := &ListAddOnResponseStreamMock{capture: capture}
	md := metadata.New(map[string]string{"authorization": "Bearer 12345"})
	ctx = metadata.NewIncomingContext(ctx, md)
	mockObj.On("Context").Return(ctx)
	iamClientMock.On("IsAllowed", "12345", mock.AnythingOfType("string")).Return(true, nil)
	return mockObj
}
