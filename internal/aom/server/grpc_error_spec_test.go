// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server

import (
	"errors"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConvertToGrpcUpdateDowngradeError(t *testing.T) {
	// Arrange
	testError := errors.New("Test error")

	// Act
	err := ConvertToGrpcUpdateDowngradeError(testError)

	// Assert
	if err == nil {
		t.Fatal("Failed to generate error!")
	}

	checkStatusDetails(err.Details(), t)

	gotError := err.Err()
	checkErrorMessage(t, gotError, testError, codes.InvalidArgument)
}

func TestConvertToGrpc(t *testing.T) {
	type args struct {
		err        error
		statusCode codes.Code
	}
	tests := []struct {
		name    string
		uut     func(error) error
		args    args
		wantErr bool
	}{
		{
			name: "UnsupportedPlatform",
			uut:  ConvertToGrpcUnsupportedPlatformError,
			args: args{
				err:        errors.New("unsupported platform"),
				statusCode: codes.InvalidArgument,
			},
			wantErr: true,
		},
		{
			name: "InvalidManifestVersion",
			uut:  ConvertToGrpcInvalidManifestVersionError,
			args: args{
				err:        errors.New("invalid manifest version"),
				statusCode: codes.InvalidArgument,
			},
			wantErr: true,
		},
		{
			name: "RootAccessNotEnabled",
			uut:  ConvertToGrpcRootAccessNotEnabledError,
			args: args{
				err:        errors.New("System ssh root access is not enabled"),
				statusCode: codes.InvalidArgument,
			},
			wantErr: true,
		},
		{
			name: "NotEnoughDiskSpace",
			uut:  ConvertToGrpcNotEnoughDiskSpaceError,
			args: args{
				err:        errors.New("Not enough disk space. Available 11 (bytes), Approx Required 17 (bytes)."),
				statusCode: codes.ResourceExhausted,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.uut(tt.args.err)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s() Failed to generate error!", getFunctionName(tt.uut))
			}
			checkErrorMessage(t, err, tt.args.err, tt.args.statusCode)
		})
	}
}

func getFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func checkErrorMessage(t *testing.T, gotError error, testError error, testStatusCode codes.Code) {
	gotErrorAsStatus := status.Convert(gotError)
	if gotErrorAsStatus == nil {
		t.Errorf("Failed to cast %v to status.Error pointer!", gotError)
	}

	gotStatusCode := gotErrorAsStatus.Code()
	if gotStatusCode != testStatusCode {
		t.Errorf("Mismatching gRPC error codes. Want: %d Got: %d", testStatusCode, gotStatusCode)
	}

	gotMessage := gotError.Error()
	if !strings.Contains(gotMessage, testError.Error()) {
		t.Errorf("'%s' not found in '%s'", testError.Error(), gotMessage)
	}
}

func checkStatusDetails(details []interface{}, t *testing.T) {

	if len(details) < 1 {
		t.Fatalf("checkStatusDetails: expect at least one element in details")
	}

	info := details[0].(*errdetails.ErrorInfo)

	if info.Domain != "weidmueller.ucontrol.aom" {
		t.Errorf("info.Domain: want %s, got %s", "weidmueller.ucontrol.aom", info.Domain)
	}

	if info.Reason != "UPDATE_DOWNGRADE_ERROR" {
		t.Errorf("info.Reason: want %s, got %s", "UPDATE_DOWNGRADE_ERROR", info.Domain)
	}
}
