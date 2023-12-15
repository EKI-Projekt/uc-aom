// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package network

import (
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/docker/api/types"
	"github.com/stretchr/testify/mock"
)

type MockExternalNetworkConnector struct {
	mock.Mock
}

func (m *MockExternalNetworkConnector) Initialize() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockExternalNetworkConnector) IsConnected(manifest *model.Root) bool {
	args := m.Called(manifest)
	return args.Bool(0)
}

func (m *MockExternalNetworkConnector) Reconnect(containers []types.Container) error {
	args := m.Called(containers)
	return args.Error(0)
}
