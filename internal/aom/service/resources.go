// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"fmt"
	"u-control/uc-aom/internal/aom/system"
)

// Represents insufficient disk space.
type NotEnoughDiskSpaceError struct {
	message string
}

func (r *NotEnoughDiskSpaceError) Error() string {
	return r.message
}

// Returns an error if the requiredSizeBytes is
// equal or larger than the space available
// as defined by uOSSystem.
func CheckDiskSpace(uOSSystem system.System, requiredSizeBytes uint64) error {
	availableBytes, err := uOSSystem.AvailableSpaceInBytes()
	if err != nil {
		return err
	}

	if availableBytes < requiredSizeBytes {
		message := fmt.Sprintf(
			"Insufficient disk space. Available %d (bytes), Approx Required %d (bytes)",
			availableBytes,
			requiredSizeBytes,
		)
		return &NotEnoughDiskSpaceError{message: message}
	}

	return nil
}
