// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package server

import (
	"context"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/metadata"
)

type EmptyResponseStreamMock struct {
	mock.Mock
}

func (r *EmptyResponseStreamMock) Context() context.Context {
	args := r.Called()
	return args.Get(0).(context.Context)
}

func (r *EmptyResponseStreamMock) RecvMsg(m interface{}) error {
	args := r.Called(m)
	return args.Error(0)
}

func (r *EmptyResponseStreamMock) Send(e *empty.Empty) error {
	args := r.Called(e)
	return args.Error(0)
}

func (r *EmptyResponseStreamMock) SendHeader(md metadata.MD) error {
	args := r.Called(md)
	return args.Error(0)
}

func (r *EmptyResponseStreamMock) SendMsg(m interface{}) error {
	args := r.Called(m)
	return args.Error(0)
}

func (r *EmptyResponseStreamMock) SetHeader(md metadata.MD) error {
	args := r.Called(md)
	return args.Error(0)
}

func (r *EmptyResponseStreamMock) SetTrailer(md metadata.MD) {
	r.Called(md)
}

func NewEmptyResponseStreamMock(ctx context.Context, iamClientMock *IamClientMock) *EmptyResponseStreamMock {
	mockObj := &EmptyResponseStreamMock{}
	md := metadata.New(map[string]string{"authorization": "Bearer 12345"})
	ctx = metadata.NewIncomingContext(ctx, md)
	mockObj.On("Context").Return(ctx)
	iamClientMock.On("IsAllowed", "12345", mock.AnythingOfType("string")).Return(true, nil)
	return mockObj
}
