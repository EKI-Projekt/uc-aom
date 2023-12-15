// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry_test

import (
	"io/fs"
	"testing"
	"u-control/uc-aom/internal/aom/registry"
)

func TestValidCredentialsParser(t *testing.T) {
	validTestCases := []testCase{
		{
			description: "Valid: serveraddress without username/password (insecure)",
			payload:     []byte(`{ "serveraddress": "registry:5000" }`),

			assertUsername:       isEmpty,
			assertPassword:       isEmpty,
			assertServerAddress:  func(x string) bool { return x == "registry:5000" },
			assertInsecureServer: func(x bool) bool { return x == true },
		},
		{
			description: "Valid: username, password, serveraddress",
			payload:     []byte(`{ "username": "Robert", "password": "Griesemer", "serveraddress": "registry:5000" }`),

			assertUsername:       func(x string) bool { return x == "Robert" },
			assertPassword:       func(x string) bool { return x == "Griesemer" },
			assertServerAddress:  func(x string) bool { return x == "registry:5000" },
			assertInsecureServer: func(x bool) bool { return x == false },
		},
	}

	for _, testCase := range validTestCases {
		t.Run(testCase.description, func(t *testing.T) {
			credentials, err := registry.ReadAndValidateCredentials(testCase.ReadFile, "")

			// Assert
			if err != nil {
				t.Fatalf("Unexpected error: %s", err.Error())
			}

			if !testCase.assertServerAddress(credentials.ServerAddress) {
				t.Errorf("Unexpected ServerAddress: '%s'", credentials.ServerAddress)
			}

			if !testCase.assertUsername(credentials.Username) {
				t.Errorf("Unexpected Usersername: '%s'", credentials.Username)
			}

			if !testCase.assertPassword(credentials.Password) {
				t.Errorf("Unexpected Password: '%s'", credentials.Password)
			}

			if !testCase.assertInsecureServer(credentials.IsInsecureServer()) {
				t.Errorf("Assert IsInsecureServer failed. credentials.IsInsecureServer() = %v", credentials.IsInsecureServer())
			}
		})
	}
}
func TestInvalidCredentialsParser(t *testing.T) {
	invalidTestCases := []testCase{
		{
			description:          "credential file does not exist",
			err:                  fs.ErrNotExist,
			assertUsername:       isEmpty,
			assertPassword:       isEmpty,
			assertServerAddress:  isEmpty,
			assertInsecureServer: func(x bool) bool { return x == true },
		},
		{
			description: "corrput JSON file",
			payload:     []byte(`{ "identitytoken: "Ken Thompson", "serveraddress": "registry:5002" }`),

			assertUsername:       isEmpty,
			assertPassword:       isEmpty,
			assertServerAddress:  isEmpty,
			assertInsecureServer: func(x bool) bool { return x == true },
		},
		{
			description: "empty JSON file",
			payload:     []byte("{}"),

			assertUsername:       isEmpty,
			assertPassword:       isEmpty,
			assertServerAddress:  isEmpty,
			assertInsecureServer: func(x bool) bool { return x == true },
		},
		{
			description: "not a credentials JSON file",
			payload:     []byte(`{ "map": {"key": 42}, "serveraddress": "" }`),

			assertUsername:       isEmpty,
			assertPassword:       isEmpty,
			assertServerAddress:  isEmpty,
			assertInsecureServer: func(x bool) bool { return x == true },
		},
		{
			description: "has username and password but no serveraddress",
			payload:     []byte(`{ "username": "Ken", "password": "Thompson" }`),

			assertUsername:       func(x string) bool { return x == "Ken" },
			assertPassword:       func(x string) bool { return x == "Thompson" },
			assertServerAddress:  isEmpty,
			assertInsecureServer: func(x bool) bool { return x == false },
		},
		{
			description: "has serveraddress and username but no password",
			payload:     []byte(`{ "username": "Ken", "serveraddress": "registry:5003" }`),

			assertUsername:       func(x string) bool { return x == "Ken" },
			assertPassword:       isEmpty,
			assertServerAddress:  func(x string) bool { return x == "registry:5003" },
			assertInsecureServer: func(x bool) bool { return x == false },
		},
		{
			description:          "has serveraddress and password but no username",
			payload:              []byte(`{ "password": "Thompson", "serveraddress": "registry:5004" }`),
			assertUsername:       isEmpty,
			assertPassword:       func(x string) bool { return x == "Thompson" },
			assertServerAddress:  func(x string) bool { return x == "registry:5004" },
			assertInsecureServer: func(x bool) bool { return x == false },
		},
	}

	for _, tc := range invalidTestCases {
		t.Run(tc.description, func(t *testing.T) {
			// Act
			callback := func(path string) ([]byte, error) {
				payload, err := tc.ReadFile(path)
				return payload, err
			}
			_, err := registry.ReadAndValidateCredentials(callback, "")

			// Assert
			if err == nil {
				t.Errorf("error is nil")
			}
		})
	}
}

type testCase struct {
	description string

	payload []byte
	err     error

	assertUsername       func(string) bool
	assertPassword       func(string) bool
	assertServerAddress  func(string) bool
	assertInsecureServer func(bool) bool
}

func (t *testCase) ReadFile(string) ([]byte, error) {
	return t.payload, t.err
}

func isEmpty(input string) bool {
	return len(input) == 0
}
