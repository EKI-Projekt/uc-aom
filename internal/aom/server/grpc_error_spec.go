// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server

import (
	"errors"
	"u-control/uc-aom/internal/aom/service"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	domain                             = "weidmueller.ucontrol.aom"
	unsupported_platform               = "UNSUPPORTED_PLATFORM"
	invalid_manifest_version           = "INVALID_MANIFEST_VERSION"
	remote_registry_connection_problem = "REMOTE_REGISTRY_CONNECTION_PROBLEM"
	update_downgrade_error             = "UPDATE_DOWNGRADE_ERROR"
	feature_root_access_not_enabled    = "FEATURE_ROOT_ACCESS_NOT_ENABLED"
	not_enough_disk_space              = "NOT_ENOUGH_DISK_SPACE"
)

func convertToGrpcError(err error) error {
	if errors.Is(err, service.ErrorAddOnAlreadyInstalled) {
		return status.Error(codes.AlreadyExists, err.Error())
	}
	if invalidVersion, ok := err.(*service.InvalidManifestError); ok {
		return ConvertToGrpcInvalidManifestVersionError(invalidVersion)
	}
	if unsupportedPlatform, ok := err.(*service.UnsupportedPlatformError); ok {
		return ConvertToGrpcUnsupportedPlatformError(unsupportedPlatform)
	}
	if errors.Is(err, service.SshRootAccessNotEnabledError) {
		return ConvertToGrpcRootAccessNotEnabledError(err)
	}
	if notEnoughDiskSpace, ok := err.(*service.NotEnoughDiskSpaceError); ok {
		return ConvertToGrpcNotEnoughDiskSpaceError(notEnoughDiskSpace)
	}

	return status.Error(codes.FailedPrecondition, err.Error())
}

// Generates an invalid platform error.
func ConvertToGrpcUnsupportedPlatformError(err error) error {
	return createInvalidArgumentGrpcStatusErrorWithReasonAndError(unsupported_platform, err)
}

// Generates an invalid manifest version error
func ConvertToGrpcInvalidManifestVersionError(err error) error {
	return createInvalidArgumentGrpcStatusErrorWithReasonAndError(invalid_manifest_version, err)
}

// Generates a remote registry connection error
func ConvertToGrpcRemoteRegistryConnectionError(err error) error {
	return createInvalidArgumentGrpcStatusErrorWithReasonAndError(remote_registry_connection_problem, err)
}

// Generates a update downgrade error
func ConvertToGrpcUpdateDowngradeError(err error) *status.Status {
	statusWithDetails, err := createInvalidArgumentGrpcStatusWithReasonAndError(update_downgrade_error, err)

	if err != nil {
		errorStatus, _ := status.FromError(err)
		return errorStatus
	}

	return statusWithDetails
}

// Generates a remote registry connection error
func ConvertToGrpcRootAccessNotEnabledError(err error) error {
	return createInvalidArgumentGrpcStatusErrorWithReasonAndError(feature_root_access_not_enabled, err)
}

func ConvertToGrpcNotEnoughDiskSpaceError(err error) error {
	return createResourceExhaustedGrpcStatusWithReasonAndError(not_enough_disk_space, err)
}

func createInvalidArgumentGrpcStatusErrorWithReasonAndError(reason string, err error) error {
	status, err := createInvalidArgumentGrpcStatusWithReasonAndError(reason, err)
	if err != nil {
		return err
	}
	return status.Err()
}

func createInvalidArgumentGrpcStatusWithReasonAndError(reason string, err error) (*status.Status, error) {
	statusWithDetails, err := status.New(codes.InvalidArgument, err.Error()).WithDetails(newErrorInfo(reason))
	if err != nil {
		return nil, err
	}
	return statusWithDetails, nil
}

func createResourceExhaustedGrpcStatusWithReasonAndError(reason string, err error) error {
	statusWithDetails, err := status.New(codes.ResourceExhausted, err.Error()).WithDetails(newErrorInfo(reason))
	if err != nil {
		return err
	}

	return statusWithDetails.Err()
}

func newErrorInfo(reason string) *errdetails.ErrorInfo {
	return &errdetails.ErrorInfo{
		Reason: reason,
		Domain: domain,
	}
}
