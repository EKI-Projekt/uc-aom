// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/mock"
)

type mockManifestReader struct {
	mock.Mock
}

func (r *mockManifestReader) ReadManifestFrom(directoryOfManifest string) (*manifest.Root, error) {
	args := r.Called(directoryOfManifest)
	return args.Get(0).(*manifest.Root), args.Error(1)
}

type testCase struct {
	description      string
	manifestFilePath string
}

var validManifestTestCaseData = []testCase{
	{
		"Testcase SimpleManifest",
		"testdata/simple-manifest.json",
	},
	{
		"Testcase SimpleManifest with enviornments",
		"testdata/simple-manifest-with-environments.json",
	},
	{
		"Testcase anyviz example",
		"../../../api/examples/anyviz.manifest.min.json",
	},
	{
		"Testcase Testcase anyviz with environments example",
		"../../../api/examples/anyviz.manifest.vpn.json",
	},
	{
		"Testcase SimpleManifest with settings",
		"testdata/simple-manifest-with-settings.json",
	},
	{
		"Testcase SimpleManifest with publish",
		"testdata/simple-manifest-with-publish.json",
	},
	{
		"Testcase node red example",
		"../../../api/examples/node-red/manifest.json",
	},
	{
		"Testcase vendor example",
		"testdata/simple-manifest-with-vendor.json",
	},
}

var invalidManifestTestCaseData = []testCase{
	{
		"Testcase invalid service name",
		"testdata/invalid-service-name-manifest.json",
	},
	{
		"Testcase empty services object",
		"testdata/empty-services-manifest.json",
	},
	{
		"Testcase service type is not docker",
		"testdata/wrong-service-type-manifest.json",
	},
	{
		"Testcase invalid environments name",
		"testdata/invalid-simple-manifest-with-environments.json",
	},
	{
		"Testcase invalid manifest version 1.1.1",
		"testdata/invalid-manifest-version-0.1.1.json",
	},
	{
		"Testcase invalid manifest version a.b",
		"testdata/invalid-manifest-version-a.b.json",
	},
	{
		"Testcase invalid manifest version .1",
		"testdata/invalid-manifest-version-.1.json",
	},
	{
		"Testcase invalid manifest version 1.",
		"testdata/invalid-manifest-version-1.json",
	},
	{
		"Testcase invalid manifest version 0",
		"testdata/invalid-manifest-version-0.json",
	},
	{
		"Testcase invalid manifest settings with missing required props",
		"testdata/invalid-settings-required-props.json",
	},
	{
		"Testcase invalid manifest settings with missing required select options properties",
		"testdata/invalid-settings-required-select-options.json",
	},
	{
		"Testcase invalid manifest settings with undefined properties",
		"testdata/invalid-settings-undefined-property.json",
	},
	{
		"Testcase invalid manifest settings with undefined select properties",
		"testdata/invalid-settings-undefined-select-property.json",
	},
	{
		"Testcase invalid manifest with undefined vendor properties",
		"testdata/invalid-vendor-property-manifest.json",
	},
	{
		"Testcase invalid manifest with invalid format of email property",
		"testdata/invalid-vendor-email-format-manifest.json",
	},
	{
		"Testcase invalid manifest with invalid format of url property",
		"testdata/invalid-vendor-uri-format-manifest.json",
	},
	{
		"Testcase invalid manifest with missing platform property",
		"testdata/invalid-manifest-missing-platform-property.json",
	},
	{
		"Testcase invalid manifest wrong platform property type - string",
		"testdata/invalid-manifest-platform-property-type-string.json",
	},
	{
		"Testcase invalid manifest wrong platform property type - number",
		"testdata/invalid-manifest-platform-property-type-number.json",
	},
	{
		"Testcase invalid manifest wrong platform property item type",
		"testdata/invalid-manifest-platform-property-item-type-number.json",
	},
	{
		"Testcase invalid manifest wrong logo name",
		"testdata/invalid-manifest-logo.json",
	},
}

func (tc *testCase) importManifest() (interface{}, error) {
	bytes, err := os.ReadFile(tc.manifestFilePath)
	if err != nil {
		return nil, err
	}

	var v interface{}
	return v, json.Unmarshal(bytes, &v)
}

func intializeManifestSchema() *jsonschema.Schema {
	compiler := jsonschema.NewCompiler()
	compiler.AssertFormat = true
	compiler.AssertContent = true
	return compiler.MustCompile("../../../api/uc-manifest.schema.json")
}

func (tc *testCase) withSchema(schema *jsonschema.Schema, assert func(*testing.T, error)) func(*testing.T) {
	return func(t *testing.T) {
		// Arrange
		root, err := tc.importManifest()
		if err != nil {
			t.Fatalf("Unable to initialize TestCase: %v", err)
		}

		// Act
		err = schema.Validate(root)

		assert(t, err)
	}
}

func assertValid(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected error to be nil, Actual: %v", err)
	}
}

func assertNotValid(t *testing.T, err error) {
	if err == nil {
		t.Errorf("Expected error, Actual: nil")
	}
}

func TestValidManifest(t *testing.T) {
	schema := intializeManifestSchema()
	for _, testCase := range validManifestTestCaseData {
		t.Run(testCase.description, testCase.withSchema(schema, assertValid))
	}
}

func TestInvalidManifest(t *testing.T) {
	schema := intializeManifestSchema()
	for _, testCase := range invalidManifestTestCaseData {
		t.Run(testCase.description, testCase.withSchema(schema, assertNotValid))
	}
}

var simpleManifest = `
{
  "manifestVersion": "0.1",
  "version": "%s",
  "title": "AnyViz Cloud Adapter",
  "description": "The AnyViz cloud solution allows you to remotely monitor, control and analyse industrial PLCs, sensors and meters.",
  "logo": "logoanyviz.png",
  "services": {
    "cloudadapter": {
      "type": "docker-compose",
      "config": {
        "image": "anyviz/cloudadapter",
        "restart": "no",
        "containerName": "anyviz",
        "ports": ["127.0.0.1:8888:8888"]
      }
    }
  },
  "platform": ["ucg", "ucm"],
  "vendor": {
    "name": "Weidmüller GmbH & Co KG",
    "url": "https://www.weidmueller.de",
    "email": "datenschutz@weidmueller.de",
    "street": "Klingenbergstraße 26",
    "zip": "32758",
    "city": "Detmold",
    "country": "Germany"
  }
}
`

func TestValidManifestAddOnVersion(t *testing.T) {
	addOnVersions := []string{
		"0.1-1",
		"0.1.0-1",
		"0.9.3.3-1",
		"0.9.3.3-12",
		"0.9.3.3-alpha.1-1",
		"0.9.3.3-alpha.2-1",
		"0.9.3.3-alpha.10-1",
		"0.9.3.3-beta.1-1",
		"0.9.3.3-beta.2-1",
		"0.9.3.3-beta.10-1",
		"0.9.3.3-rc.1-12",
		"0.9.3.3-rc.2-12",
		"0.9.3.3-rc.10-12",
		"0.9.3.3-1-alpha.1",
		"0.9.3.3-1-alpha.10",
		"0.9.3.3-1-beta.1",
		"0.9.3.3-1-beta.10",
		"0.9.3.3-1-rc.1",
		"0.9.3.3-1-rc.10",
	}

	schema, err := manifest.NewValidator()
	if err != nil {
		t.Fatal(err)
	}

	root, err := manifest.NewFromBytes([]byte(simpleManifest))
	if err != nil {
		t.Fatal(err)
	}

	for _, addOnVersion := range addOnVersions {
		testcase := fmt.Sprintf("Testcase valid add-on version %s", addOnVersion)
		t.Run(testcase, func(t *testing.T) {
			// arrange
			root.Version = addOnVersion

			// act
			validateResult := schema.Validate(root)

			// assert
			if validateResult != nil {
				t.Errorf("Add-on addOnManifest version is invalid %s", validateResult.Error())
			}
		})
	}
}

func TestInValidManifestAddOnVersion(t *testing.T) {
	addOnVersions := []string{
		"1",
		"0.",
		"0.1",
		"0.1.1",
		"0.1.1.0",
		"0.1.-beta",
		"0.1-abcd",
		"0.1-1-beta",
		"0.1-1-beta1",
		"0.1-1-beta.1-beta.1",
	}
	schema, err := manifest.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	root, err := manifest.NewFromBytes([]byte(simpleManifest))
	if err != nil {
		t.Fatal(err)
	}
	for _, addOnVersion := range addOnVersions {
		testcase := fmt.Sprintf("Testcase valid add-on version %s", addOnVersion)
		t.Run(testcase, func(t *testing.T) {
			// arrange
			root.Version = addOnVersion

			// act
			validateResult := schema.Validate(root)

			// assert
			if validateResult == nil {
				t.Errorf("Add-on manifest version shall be invalid")
			}
		})
	}
}
