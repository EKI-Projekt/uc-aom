// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package credentials_test

import (
	"io/fs"
	"testing"
	"u-control/uc-aom/internal/aop/credentials"
)

var testCases = []testCase{
	{
		description:    "Valid: username, password, repositoryname",
		payload:        []byte(`{ "username": "Robert", "password": "Griesemer", "repositoryname": "go-creators/first-addon" }`),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isNilError,
		assertUsername:       func(x string) bool { return x == "Robert" },
		assertPassword:       func(x string) bool { return x == "Griesemer" },
		assertRepositoryName: func(x string) bool { return x == "go-creators/first-addon" },
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == false },
	},
	{
		description:    "Invalid: credential file does not exist",
		err:            fs.ErrNotExist,
		validationOpts: []credentials.ValidationOpt{},

		assertError:          isSomeError,
		assertUsername:       isEmpty,
		assertPassword:       isEmpty,
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == true },
	},
	{
		description:    "Invalid: corrput JSON file",
		payload:        []byte(`{ "identitytoken: "Ken Thompson", "repositoryname": "go-creators/first-addon"}`),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isSomeError,
		assertUsername:       isEmpty,
		assertPassword:       isEmpty,
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == true },
	},
	{
		description:    "Invalid: empty JSON file but repositoryname validation option active",
		payload:        []byte("{}"),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isSomeError,
		assertUsername:       isEmpty,
		assertPassword:       isEmpty,
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == true },
	},
	{
		description:    "Invalid: empty JSON file but serveraddress validation option active",
		payload:        []byte("{}"),
		validationOpts: []credentials.ValidationOpt{credentials.ServerAddressSet()},

		assertError:          isSomeError,
		assertUsername:       isEmpty,
		assertPassword:       isEmpty,
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == true },
	},
	{
		description:    "Invalid: not a credentials JSON file",
		payload:        []byte(`{ "map": {"key": 42}, "repositoryname": "" }`),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isSomeError,
		assertUsername:       isEmpty,
		assertPassword:       isEmpty,
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == true },
	},
	{
		description:    "Invalid: has username and password but no repositoryname",
		payload:        []byte(`{ "username": "Rob", "password": "Pike" }`),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isSomeError,
		assertUsername:       func(x string) bool { return x == "Rob" },
		assertPassword:       func(x string) bool { return x == "Pike" },
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == false },
	},
	{
		description:    "Invalid: has username and password but repositoryname is empty string",
		payload:        []byte(`{ "username": "Ken", "password": "Thompson", "repositoryname": "" }`),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isSomeError,
		assertUsername:       func(x string) bool { return x == "Ken" },
		assertPassword:       func(x string) bool { return x == "Thompson" },
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == false },
	},
	{
		description:    "Invalid: has username and repositoryname but no password",
		payload:        []byte(`{ "username": "Ken", "repositoryname": "go-creators/first-addon" }`),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isSomeError,
		assertUsername:       func(x string) bool { return x == "Ken" },
		assertPassword:       isEmpty,
		assertRepositoryName: func(x string) bool { return x == "go-creators/first-addon" },
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == false },
	},
	{
		description:    "Invalid: has password and repositoryname but no username",
		payload:        []byte(`{ "password": "Thompson", "repositoryname": "go-creators/first-addon" }`),
		validationOpts: []credentials.ValidationOpt{credentials.RepositoryNameSet()},

		assertError:          isSomeError,
		assertUsername:       isEmpty,
		assertPassword:       func(x string) bool { return x == "Thompson" },
		assertRepositoryName: func(x string) bool { return x == "go-creators/first-addon" },
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == false },
	},
	{
		description:    "Valid: has username, password and serveraddress",
		payload:        []byte(`{ "username": "Robert", "password": "Griesemer", "serveraddress": "test.registry.io"  }`),
		validationOpts: []credentials.ValidationOpt{credentials.ServerAddressSet()},

		assertError:          isNilError,
		assertUsername:       func(x string) bool { return x == "Robert" },
		assertPassword:       func(x string) bool { return x == "Griesemer" },
		assertRepositoryName: isEmpty,
		assertServerAddress:  func(x string) bool { return x == "test.registry.io" },
		assertInsecureServer: func(x bool) bool { return x == false },
	},
	{
		description:    "Valid: has serveraddress",
		payload:        []byte(`{ "serveraddress": "localhost:4321"  }`),
		validationOpts: []credentials.ValidationOpt{credentials.ServerAddressSet()},

		assertError:          isNilError,
		assertUsername:       isEmpty,
		assertPassword:       isEmpty,
		assertRepositoryName: isEmpty,
		assertServerAddress:  func(x string) bool { return x == "localhost:4321" },
		assertInsecureServer: func(x bool) bool { return x == true },
	},
	{
		description:    "Invalid: has username and password but empty serveraddress",
		payload:        []byte(`{ "username": "Robert", "password": "Griesemer", "serveraddress": "" }`),
		validationOpts: []credentials.ValidationOpt{credentials.ServerAddressSet()},

		assertError:          isSomeError,
		assertUsername:       func(x string) bool { return x == "Robert" },
		assertPassword:       func(x string) bool { return x == "Griesemer" },
		assertRepositoryName: isEmpty,
		assertServerAddress:  isEmpty,
		assertInsecureServer: func(x bool) bool { return x == false },
	},
}

func TestCredentialsParseAndValidation(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.description, tc.testCredentialsParseAndValidationP)
	}
}

func (testCase *testCase) testCredentialsParseAndValidationP(t *testing.T) {
	// Act
	credentials, err := credentials.ParseAndValidate(testCase.readFileFuncMock, "", testCase.validationOpts...)

	// Assert
	if !testCase.assertError(err) {
		t.Errorf("Unexpected error: %v", err)
	}

	if !testCase.assertRepositoryName(credentials.RepositoryName) {
		t.Errorf("Unexpected RepositoryName: '%s'", credentials.RepositoryName)
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
}

type testCase struct {
	description string

	payload        []byte
	err            error
	validationOpts []credentials.ValidationOpt

	assertError          func(error) bool
	assertUsername       func(string) bool
	assertPassword       func(string) bool
	assertRepositoryName func(string) bool
	assertServerAddress  func(string) bool
	assertInsecureServer func(bool) bool
}

func (r *testCase) readFileFuncMock(string) ([]byte, error) {
	return r.payload, r.err
}

func isEmpty(input string) bool {
	return len(input) == 0
}

func isNilError(err error) bool {
	return err == nil
}

func isSomeError(err error) bool {
	return !isNilError(err)
}

func TestSetRegistryAddress(t *testing.T) {
	type args struct {
		description           string
		serverAddress         string
		expectedServerAddress string
	}
	testCases := []args{
		{

			description:           "Should get the default registry address",
			serverAddress:         "",
			expectedServerAddress: "registry:5000",
		},
		{
			description:           "Should overwrite the default server address",
			serverAddress:         "registry:7777",
			expectedServerAddress: "registry:7777",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			// Arrange
			targetCredentials := &credentials.Credentials{ServerAddress: tc.serverAddress}

			// Act
			credentials.SetRegistryServerAddress(targetCredentials)

			// Assert
			if targetCredentials.ServerAddress != tc.expectedServerAddress {
				t.Errorf("Expected server address to be %s but got %s", tc.expectedServerAddress, targetCredentials.ServerAddress)
			}
		})
	}
}
