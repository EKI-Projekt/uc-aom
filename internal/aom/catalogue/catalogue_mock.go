// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package catalogue

import (
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

type CatalogueMock struct {
	mock.Mock
}

func (m CatalogueMock) PullAddOn(name string, version string) (CatalogueAddOnWithImages, error) {
	args := m.Called(name, version)
	return args.Get(0).(CatalogueAddOnWithImages), args.Error(1)
}

func (m CatalogueMock) DeleteAddOn(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m CatalogueMock) GetAddOn(name string) (CatalogueAddOn, error) {
	args := m.Called(name)
	return args.Get(0).(CatalogueAddOn), args.Error(1)
}

func (m CatalogueMock) GetAddOns() ([]*CatalogueAddOn, error) {
	args := m.Called()
	return args.Get(0).([]*CatalogueAddOn), args.Error(1)
}

func (m CatalogueMock) FetchManifest(name string, version string) (*manifest.Root, error) {
	args := m.Called(name, version)
	return args.Get(0).(*manifest.Root), args.Error(1)
}
