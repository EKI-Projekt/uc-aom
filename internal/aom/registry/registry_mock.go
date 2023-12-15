// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package registry

import "github.com/stretchr/testify/mock"

type MockRegistry struct {
	mock.Mock
}

func (r *MockRegistry) Repositories() ([]string, error) {
	args := r.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (r *MockRegistry) Tags(repository string) ([]string, error) {
	args := r.Called(repository)
	return args.Get(0).([]string), args.Error(1)
}

func (r *MockRegistry) Pull(repository string, tag string, processor ImageManifestLayerProcessor) (uint64, error) {
	args := r.Called(repository, tag, processor)
	return args.Get(0).(uint64), args.Error(1)
}

func (r *MockRegistry) Delete(repository string, tag string) error {
	args := r.Called(repository, tag)
	return args.Error(0)
}
