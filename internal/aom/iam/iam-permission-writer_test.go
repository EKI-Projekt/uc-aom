// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package iam_test

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aom/iam"

	log "github.com/sirupsen/logrus"
)

type TestCase struct {
	Description   string
	ReloadMessage string
	ExpectError   bool
}

func init() {
	log.SetLevel(log.TraceLevel)
}

func setUp() (*iam.IamPermissionWriter, *bytes.Buffer) {
	var buf bytes.Buffer
	writeToBuffer := func(name string, writeContent func(io.Writer) error) error {
		log.Tracef("Created buffer for: '%s'", name)
		return writeContent(&buf)
	}

	deleteFile := func(name string) error {
		log.Tracef("Deleting file: '%s'", name)
		return nil
	}

	uut := iam.NewIamPermissionWriter("testPermissionPath", writeToBuffer, deleteFile)
	return uut, &buf
}

func TestIamPermissionWriterCreate(t *testing.T) {
	// Arrange
	uut, buf := setUp()
	expected := `
{
  "compatibility-version": "1.0",
  "service": "uc-auth",
  "group-display-name": "Web server - Protected applications",
  "permissions": [
    {
      "id": "wm-telemetry.access",
      "display-name": "Access to Weidmueller Telemetry",
      "no-auth-option": true
    }
  ]
}

`
	// Act and assert
	permission := &iam.IamPermission{"Weidmueller Telemetry", "wm-telemetry", true}
	err := uut.Create(permission)
	if err != nil {
		t.Errorf("Received unexpected error: '%+v'", err)
	}

	actual := fmt.Sprint(buf)
	if normalizeTmpl(actual) != normalizeTmpl(expected) {
		t.Errorf("Not Equal. Expected '%+v' Actual '%+v'", expected, actual)
	}

}

func TestIamPermissionWriterCreate_DisableNoAuthOpt(t *testing.T) {
	// Arrange
	uut, buf := setUp()
	expected := `
{
  "compatibility-version": "1.0",
  "service": "uc-auth",
  "group-display-name": "Web server - Protected applications",
  "permissions": [
    {
      "id": "wm-telemetry.access",
      "display-name": "Access to Weidmueller Telemetry",
      "no-auth-option": false
    }
  ]
}

`
	// Act and assert
	permission := &iam.IamPermission{"Weidmueller Telemetry", "wm-telemetry", false}
	err := uut.Create(permission)
	if err != nil {
		t.Errorf("Received unexpected error: '%+v'", err)
	}

	actual := fmt.Sprint(buf)
	if normalizeTmpl(actual) != normalizeTmpl(expected) {
		t.Errorf("Not Equal. Expected '%+v' Actual '%+v'", expected, actual)
	}

}

func normalizeTmpl(tmpl string) string {
	tmpl = strings.ReplaceAll(tmpl, " ", "")
	tmpl = strings.ReplaceAll(tmpl, "\t", "")
	tmpl = strings.ReplaceAll(tmpl, "\n", "")
	return tmpl
}
