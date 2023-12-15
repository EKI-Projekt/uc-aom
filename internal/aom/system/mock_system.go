// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package system

import (
	"os/user"

	"github.com/stretchr/testify/mock"
)

type MockSystem struct {
	mock.Mock
}

func (r *MockSystem) IsSshRootAccessEnabled() (bool, error) {
	args := r.Called()
	return args.Get(0).(bool), args.Error(1)

}

func (r *MockSystem) LookupAdminUser() (*user.User, error) {
	args := r.Called()
	return args.Get(0).(*user.User), args.Error(1)

}

func (r *MockSystem) AvailableSpaceInBytes() (uint64, error) {
	args := r.Called()
	return args.Get(0).(uint64), args.Error(1)
}
