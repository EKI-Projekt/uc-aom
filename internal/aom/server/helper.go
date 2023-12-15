// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const bearerTokenPrefix = "Bearer "

func getJsonWebTokenFrom(ctx context.Context) string {
	metadata, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}

	authorizationValue := metadata.Get("authorization")

	if len(authorizationValue) > 0 {
		return parseBearerTokenFrom(authorizationValue[0])
	}

	return ""
}

func parseBearerTokenFrom(tokenToParse string) string {
	isBearerToken := strings.HasPrefix(tokenToParse, bearerTokenPrefix)
	if !isBearerToken {
		return ""
	}
	token := strings.TrimPrefix(tokenToParse, bearerTokenPrefix)
	return token
}
