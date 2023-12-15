// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package system

import (
	"os/exec"
	"os/user"

	"golang.org/x/sys/unix"
)

type System interface {
	// Check if the SSH root access is enabled
	IsSshRootAccessEnabled() (bool, error)

	// LookupAdminUser looks up the admin user. If the user cannot be found, the
	// returned error is of type UnknownUserError.
	LookupAdminUser() (*user.User, error)

	// Return the available disk space in bytes or an error should it fail.
	AvailableSpaceInBytes() (uint64, error)
}

type uOSSystem struct {
	dataPartition string
}

func NewuOSSystem(dataPartition string) *uOSSystem {
	return &uOSSystem{dataPartition: dataPartition}
}

// Check if the SSH root access is enabled.
func (s *uOSSystem) IsSshRootAccessEnabled() (bool, error) {
	cmd := exec.Command(root_access_script_path, "is_blocked")
	err := cmd.Run()

	// Script will return 0(=nil) if SSH root access is enabled
	if err == nil {
		return true, nil
	}

	if exitCode, ok := err.(*exec.ExitError); ok {
		if exitCode.ExitCode() == 1 {
			return false, nil
		}
	}
	return false, err
}

// LookupAdminUser looks up the admin user on the u-OS system.
func (s *uOSSystem) LookupAdminUser() (*user.User, error) {
	// we create the user object manually here,
	// because the build-in GO function user.Lookup() doesn't find the admin user on our devices.
	return &user.User{
		Uid:      admin_uid,
		Gid:      admin_gid,
		Username: "admin",
		Name:     "admin",
	}, nil
}

// Return the available disk space in bytes.
func (s *uOSSystem) AvailableSpaceInBytes() (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(s.dataPartition, &stat); err != nil {
		return 0, err
	}

	return stat.Bavail * uint64(stat.Bsize), nil
}
