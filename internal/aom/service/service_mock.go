// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package service

import (
	"io"
	"os/user"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/dbus"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/aom/iam"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/aom/status"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

type ServiceMultiComponentMock struct {
	mock.Mock
	docker.MockStackService
}

func (r *ServiceMultiComponentMock) NewServiceUsingServiceMultiComponentMock() *Service {
	reverseProxy := routes.NewReverseProxy(dbus.Initialize(), "", "", "", "",
		r.ReverseProxyWrite,
		r.ReverseProxyDelete,
		r.ReverseProxyCreateSymbolicLink,
		r.ReverseProxyRemoveSymbolicLink)
	iamPermissionWriter := iam.NewIamPermissionWriter("", r.IamPermissionWriterWrite, r.IamPermissionWriterDelete)
	return NewService(&r.MockStackService, reverseProxy, iamPermissionWriter, r, r, r, r)
}

func (r *ServiceMultiComponentMock) AddOnStatusResolver(name string) ([]*status.ListAddOnContainersFuncReturnType, error) {
	args := r.Called(name)
	return args.Get(0).([]*status.ListAddOnContainersFuncReturnType), args.Error(1)
}

func (r *ServiceMultiComponentMock) InspectContainer(containerId string) (*docker.ContainerInfo, error) {
	args := r.Called(containerId)
	return args.Get(0).(*docker.ContainerInfo), args.Error(1)
}

func (r *ServiceMultiComponentMock) ReverseProxyWrite(name string, writeContent func(io.Writer) error) error {
	args := r.Called(name, writeContent)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) ReverseProxyDelete(name string) error {
	args := r.Called(name)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) ReverseProxyCreateSymbolicLink(target string, linkname string) error {
	args := r.Called(target, linkname)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) ReverseProxyRemoveSymbolicLink(linkname string) error {
	args := r.Called(linkname)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) IamPermissionWriterWrite(name string, writeContent func(io.Writer) error) error {
	args := r.Called(name, writeContent)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) IamPermissionWriterDelete(name string) error {
	args := r.Called(name)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) PullAddOn(name string, version string) (catalogue.CatalogueAddOnWithImages, error) {
	args := r.Called(name, version)
	return args.Get(0).(catalogue.CatalogueAddOnWithImages), args.Error(1)
}

func (r *ServiceMultiComponentMock) DeleteAddOn(name string) error {
	args := r.Called(name)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) GetAddOn(name string) (catalogue.CatalogueAddOn, error) {
	args := r.Called(name)
	return args.Get(0).(catalogue.CatalogueAddOn), args.Error(1)
}

func (r *ServiceMultiComponentMock) GetAddOns() ([]*catalogue.CatalogueAddOn, error) {
	args := r.Called()
	return args.Get(0).([]*catalogue.CatalogueAddOn), args.Error(1)
}

func (r *ServiceMultiComponentMock) FetchManifest(name string, version string) (*manifest.Root, error) {
	args := r.Called(name, version)
	return args.Get(0).(*manifest.Root), args.Error(1)
}

func (r *ServiceMultiComponentMock) Validate(manifest *manifest.Root) error {
	args := r.Called(manifest)
	return args.Error(0)
}

func (r *ServiceMultiComponentMock) GetAddOnEnvironment(name string) (map[string]string, error) {
	args := r.Called(name)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (r *ServiceMultiComponentMock) AvailableSpaceInBytes() (uint64, error) {
	args := r.Called()
	return args.Get(0).(uint64), args.Error(1)
}

func (r *ServiceMultiComponentMock) IsSshRootAccessEnabled() (bool, error) {
	args := r.Called()
	return args.Get(0).(bool), args.Error(1)
}

func (r *ServiceMultiComponentMock) LookupAdminUser() (*user.User, error) {
	args := r.Called()
	return args.Get(0).(*user.User), args.Error(1)
}
