// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry_test

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"path"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aop/credentials"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/pkg/config"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type testCase4ContainerdExport struct {
	description string

	registryAddress string
	username        string
	password        string

	repository string
	tag        string

	os           string
	architecture string

	assertIndexJson    func(contents []byte) bool
	assertManifestJson func(contents []byte) bool
}

var testCases2 = []testCase4ContainerdExport{
	{
		description:        "uc-aom-unhealthy",
		registryAddress:    "registry:5000",
		repository:         "test/uc-aom-unhealthy",
		tag:                "0.1",
		os:                 "linux",
		architecture:       "amd64",
		assertIndexJson:    func(contents []byte) bool { return true },
		assertManifestJson: func(contents []byte) bool { return true },
	},
	{
		description:        "uc-aom-starting",
		registryAddress:    "registry:5000",
		repository:         "test/uc-aom-starting",
		tag:                "0.1",
		os:                 "linux",
		architecture:       "arm",
		assertIndexJson:    func(contents []byte) bool { return true },
		assertManifestJson: func(contents []byte) bool { return true },
	},
}

var testCases3 = []testCase4ContainerdExport{
	{
		description:        "linux/riscv64",
		registryAddress:    "registry:5000",
		repository:         "test/uc-aom-unhealthy",
		tag:                "0.1",
		os:                 "linux",
		architecture:       "riscv64",
		assertIndexJson:    func(contents []byte) bool { return false },
		assertManifestJson: func(contents []byte) bool { return false },
	},
	{
		description:        "ios/amd64",
		registryAddress:    "registry:5000",
		repository:         "test/uc-aom-starting",
		tag:                "0.1",
		os:                 "ios",
		architecture:       "amd64",
		assertIndexJson:    func(contents []byte) bool { return false },
		assertManifestJson: func(contents []byte) bool { return false },
	},
	{
		description:        "openbsd/mips64",
		registryAddress:    "registry:5000",
		repository:         "test/uc-aom-starting",
		tag:                "0.1",
		os:                 "openbsd",
		architecture:       "mips64",
		assertIndexJson:    func(contents []byte) bool { return false },
		assertManifestJson: func(contents []byte) bool { return false },
	},
}

func TestContainerdExport(t *testing.T) {
	for _, tc := range testCases2 {
		t.Run(tc.description, tc.testContainerdExportP)
	}
}

func TestExportUsingContainerdUnknownPlatform(t *testing.T) {
	for _, tc := range testCases3 {
		t.Run(tc.description, tc.testExportUsingContainerdUnknownPlatformP)
	}
}

func (testCase *testCase4ContainerdExport) testContainerdExportP(t *testing.T) {
	// Arrange
	ctx := context.Background()
	credentials := &credentials.Credentials{Username: testCase.username, Password: testCase.password, ServerAddress: testCase.registryAddress}

	orasRegistry, err := registry.InitializeRegistry(ctx, credentials)
	if err != nil {
		t.Fatal(err)
	}

	platform := &ocispec.Platform{
		OS:           testCase.os,
		Architecture: testCase.architecture,
	}

	// Act
	tarball, err := registry.ExportUsingContainerd(orasRegistry, fmt.Sprintf("%s:%s", testCase.repository, testCase.tag), platform)
	if err != nil {
		t.Fatal(err)
	}

	// Assert
	blobsCountActual := 0
	hasOciLayoutActual := false
	hasIndexJsonActual := false
	hashManifestJsonActual := false

	tr := tar.NewReader(bytes.NewReader(tarball))
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			t.Fatal(err)
		}

		var contents bytes.Buffer
		if _, err := io.Copy(&contents, tr); err != nil {
			t.Fatal(err)
		}

		if strings.Contains(hdr.Name, "blobs") {
			if hdr.Typeflag != tar.TypeReg {
				continue
			}
			blobsCountActual++
			expected := path.Base(hdr.Name)
			actual := fmt.Sprintf("%x", sha256.Sum256(contents.Bytes()))
			if expected != actual {
				t.Errorf("Mismatching SHA256. Expected '%s', Actual '%s'", expected, actual)
			}
		} else {
			switch hdr.Name {
			case "oci-layout":
				hasOciLayoutActual = true
				if !validateOciLayoutContent(contents.Bytes()) {
					t.Errorf("oci-layout content validation Failed.")
				}
				break
			case config.OciImageIndexFilename:
				hasIndexJsonActual = true
				if !testCase.assertIndexJson(contents.Bytes()) {
					t.Errorf("Assert %s failed.", config.OciImageIndexFilename)
				}
				break
			case config.UcImageManifestFilename:
				hashManifestJsonActual = true
				if !testCase.assertManifestJson(contents.Bytes()) {
					t.Errorf("Assert %s failed.", config.UcImageManifestFilename)
				}
				break
			default:
				t.Errorf("Unexpected Tar entry: %+v", hdr)
			}
		}
	}

	if blobsCountActual == 0 {
		t.Fatalf("There MUST be at least one blob.")
	}

	if !hasOciLayoutActual {
		t.Fatalf("There MUST be a file oci-layout.")
	}

	if !hasIndexJsonActual {
		t.Fatalf("There MUST be a file %s", config.OciImageIndexFilename)
	}

	if !hashManifestJsonActual {
		t.Fatalf("There MUST be a file %s.", config.UcImageManifestFilename)
	}
}

func validateOciLayoutContent(actual []byte) bool {
	expected := []byte(`{"imageLayoutVersion":"1.0.0"}`)
	if len(actual) != len(expected) {
		return false
	}

	for i := range expected {
		if expected[i] != actual[i] {
			return false
		}
	}

	return true
}

func TestExportUsingContainerdRequiresTag(t *testing.T) {
	// Arrange
	platform := &ocispec.Platform{
		OS:           "linux",
		Architecture: "amd64",
	}

	// Act
	_, err := registry.ExportUsingContainerd(nil, "registryNameWithNoTag", platform)

	// Assert
	assertIsTagNotFoundError(err, t)
}

func (testCase *testCase4ContainerdExport) testExportUsingContainerdUnknownPlatformP(t *testing.T) {
	// Arrange
	ctx := context.Background()
	credentials := &credentials.Credentials{Username: testCase.username, Password: testCase.password, ServerAddress: testCase.registryAddress}

	orasRegistry, err := registry.InitializeRegistry(ctx, credentials)
	if err != nil {
		t.Fatal(err)
	}

	platform := &ocispec.Platform{
		OS:           testCase.os,
		Architecture: testCase.architecture,
	}

	// Act
	_, err = registry.ExportUsingContainerd(orasRegistry, fmt.Sprintf("%s:%s", testCase.repository, testCase.tag), platform)

	// Assert
	assertIsNoManifestForPlatformError(err, t)
}

func assertIsTagNotFoundError(err error, t *testing.T) {
	assertErrorMessage(t, err, "Could not find tag: registryNameWithNoTag")
}

func assertIsNoManifestForPlatformError(err error, t *testing.T) {
	assertErrorMessage(t, err, "no manifest found for platform")
}

func assertErrorMessage(t *testing.T, err error, message string) {
	if err == nil || !strings.Contains(err.Error(), message) {
		t.Fatalf("Expected error '%s' Got'%v'", message, err)
	}
}
