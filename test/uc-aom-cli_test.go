// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package test

import (
	"bytes"
	"context"
	"io/ioutil"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/test/bufconn"

	"u-control/uc-aom/internal/cli"
	testhelpers "u-control/uc-aom/test/test-helpers"
)

func TestUcAomCli(t *testing.T) {
	log.SetLevel(log.InfoLevel)
	grpcTestListener := bufconn.Listen(testhelpers.BufSize)

	err := testhelpers.CreateAndConnectToNewUcAomInstance(t, grpcTestListener)
	if err != nil {
		t.Fatalf("Unable to connect to UcAom: %v", err)
	}

	testenv, err := testhelpers.NewTestEnvironment(context.Background(), grpcTestListener)
	if err != nil {
		t.Fatal(err)
	}

	// act
	uut := cli.NewAomCommand(testenv.Client)
	b := bytes.NewBufferString("")
	uut.SetOut(b)

	uut.SetArgs([]string{"ls"})

	// assert
	got := uut.Execute()

	assert.NoError(t, got)
	_, err = ioutil.ReadAll(b)
	assert.NoError(t, err)
}
