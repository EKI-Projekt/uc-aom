// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

// Package grpc contains the grpc api interfaces.
package grpc_api

//go:generate go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
//go:generate go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
//go:generate protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --proto_path=../../../api addon_service.proto
