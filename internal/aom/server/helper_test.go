// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

// test with metadata
func TestMetaDataContextValue(t *testing.T) {
	// arrange
	testMetadata := metadata.New(map[string]string{"authorization": "Bearer abcd"})
	context := metadata.NewIncomingContext(context.Background(), testMetadata)

	result, ok := metadata.FromIncomingContext(context)

	if ok == false {
		t.Errorf("ok is false")
		return
	}

	if result.Get("authorization")[0] != "Bearer abcd" {
		t.Errorf("authorization value is wrong")
		t.Error(result.Get("authorization"))
	}

}

func TestGetJsonWebTokenFrom(t *testing.T) {
	// arrange
	testMetadata := metadata.New(map[string]string{"authorization": "Bearer abcd"})
	context := metadata.NewIncomingContext(context.Background(), testMetadata)

	// act
	result := getJsonWebTokenFrom(context)

	// assert
	if result != "abcd" {
		t.Errorf("Token is not right %s", result)
	}

}

func TestShallReturnEmptyIfNotBearer(t *testing.T) {
	// arrange
	testMetadata := metadata.New(map[string]string{"authorization": "NotBearer abcd"})
	context := metadata.NewIncomingContext(context.Background(), testMetadata)

	// act
	result := getJsonWebTokenFrom(context)

	// assert
	if result != "" {
		t.Errorf("Token is not empty %s", result)
	}

}

// shall return an empty token if not token was set
func TestShallReturnEmptyToken(t *testing.T) {
	// arrange
	testMetadata := metadata.New(map[string]string{})
	context := metadata.NewIncomingContext(context.Background(), testMetadata)

	// act
	result := getJsonWebTokenFrom(context)

	// assert
	if result != "" {
		t.Errorf("Token is not right %s", result)
	}

}

// shall return an empty token if context is not grpc metadata
func TestShallReturnEmptyTokenIfContextIsWrong(t *testing.T) {
	// arrange
	testMetadata := (map[string]string{"authorization": "Bearer abcd"})
	context := context.WithValue(context.Background(), "authorization", testMetadata)

	// act
	result := getJsonWebTokenFrom(context)

	// assert
	if result != "" {
		t.Errorf("Token is not right %s", result)
	}

}
