// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package utils_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/utils"
)

func TestHeartBeatOK(t *testing.T) {
	// Arrange
	const timeUnit = time.Millisecond
	const numOfCycles = 5
	const heartBeat = 100

	// Act
	operation := func() error {
		time.Sleep((numOfCycles * heartBeat) * timeUnit)
		return nil
	}

	var count int = 0
	callback := func() {
		count++
	}

	err := utils.ApplyOperationWithHeartBeat(operation, callback, heartBeat*timeUnit)

	// Assert
	if err != nil {
		t.Errorf("Unexpected error returned from operation: %+v", err)
	}

	if count-1 != numOfCycles {
		t.Errorf("Expected %d not equal to %d", numOfCycles, count-1)
	}
}

func TestHeartBeatOperationFail(t *testing.T) {
	// Arrange
	const timeUnit = time.Millisecond
	const numOfCycles = 6
	const heartBeat = 100
	const ErrorMsg = "TestError"

	// Act
	operation := func() error {
		time.Sleep((numOfCycles * heartBeat) * timeUnit)
		return errors.New(ErrorMsg)
	}

	var count int = 0
	callback := func() {
		count++
	}

	err := utils.ApplyOperationWithHeartBeat(operation, callback, heartBeat*timeUnit)

	// Assert
	if err == nil || err.Error() != ErrorMsg {
		t.Error("Expected error not created.")
	}

	if count-1 != numOfCycles {
		t.Errorf("Expected %d not equal to %d", numOfCycles, count-1)
	}
}

func TestReplaceSlashesWithDashes(t *testing.T) {
	// arrange
	testString := "ab/cd/ef"
	expected := "ab-cd-ef"

	// act
	result := utils.ReplaceSlashesWithDashes(testString)

	// assert
	if result != expected {
		t.Errorf("String is not right: %s", result)
	}

}

func TestGetShortHashWithCustomHashFunc(t *testing.T) {
	// arrange
	testContent := []byte("testContent")
	var gotContent []byte
	testHash := "TestString"
	testFunc := func(content []byte) (string, error) {
		gotContent = content
		return testHash, nil
	}
	// act
	hash, err := utils.GetShortHashFromWithCustomHashFunc(testContent, testFunc)

	// assert
	if string(gotContent) != string(testContent) {
		t.Errorf("Unexpected content. Expected %s but got %s.", testContent, gotContent)
	}
	if err != nil {
		t.Errorf("Unexpected error. Expected none but got %s.", err)
	}
	if hash != testHash[0:5] {
		t.Errorf("Unexpected hash. Expected %s but got %s.", testHash, hash)
	}
}

func TestGetShortHashWithCustomHashFuncWithError(t *testing.T) {
	// arrange
	testContent := []byte("testContent")
	testHash := "TestString"
	testFunc := func(content []byte) (string, error) {

		return testHash, errors.New("testFunc Error")
	}
	// act
	hash, err := utils.GetShortHashFromWithCustomHashFunc(testContent, testFunc)

	// assert
	if err == nil {
		t.Errorf("Expected error but got none.")
	}
	if len(hash) != 0 {
		t.Errorf("Unexpected hash. Expected empty hash but got %s.", hash)
	}
}

func TestGetShortSHA1HashFrom(t *testing.T) {
	// arrange
	expectedHash := "17513"
	testContent := []byte("testContent")

	// act
	resultHash, err := utils.GetShortSHA1HashFrom(testContent)

	// assert
	if err != nil {
		t.Errorf("Expected no error but got %s.", err)
	}
	if resultHash != expectedHash {
		t.Errorf("Expected hash to be %s but got %s", expectedHash, resultHash)
	}
}

func TestRetry_CallbackError(t *testing.T) {
	// Arrange
	expectedRetries := 3
	retryCounter := 0
	callbackFunc := func() (interface{}, error) {
		retryCounter++
		return nil, errors.New("UNEXPECTED_ERROR")
	}

	// Act
	_, err := utils.Retry(expectedRetries, 0, callbackFunc)

	// Assert
	if err == nil {
		t.Error("Expected an error but got none")
	}

	if retryCounter != expectedRetries {
		t.Errorf("Expected %d retries but got %d", expectedRetries, retryCounter)
	}
}

func TestRetry_CallbackResult(t *testing.T) {
	// Arrange
	expectedName := "ABC"
	totalRetries := 5
	expectedRetries := 3
	type result struct {
		Name string
	}
	var retryCounter = 0
	callbackFunc := func() (interface{}, error) {
		retryCounter++
		if retryCounter == expectedRetries {
			return result{Name: expectedName}, nil
		}
		return nil, errors.New("Unexpected error")
	}

	// Act
	res, err := utils.Retry(totalRetries, 0, callbackFunc)

	// Assert
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}

	if retryCounter != expectedRetries {
		t.Errorf("Expected %d retries but got %d", expectedRetries, retryCounter)
	}

	resName := res.(result).Name
	if resName != expectedName {
		t.Errorf("Expected name in result to be %s but got %s", expectedName, resName)
	}
}

func TestGetEnvBool(t *testing.T) {
	type args struct {
		key           string
		value         string
		fallBack      bool
		expectedValue bool
	}

	testCases := []args{
		{key: "TEST_VALUE", value: "false", fallBack: true, expectedValue: false},
		{key: "TEST_VALUE", value: "", fallBack: true, expectedValue: true},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s should have the value of %v", tc.key, tc.expectedValue), func(t *testing.T) {
			// Arrange
			if tc.value != "" {
				os.Setenv(tc.key, tc.value)
				t.Cleanup(func() {
					os.Unsetenv(tc.key)
				})
			}

			// Act
			res := utils.GetEnvBool(tc.key, tc.fallBack)

			// Assert
			if res != tc.expectedValue {
				t.Errorf("Expected result to be %v but got %v", tc.fallBack, res)
			}
		})
	}
}
