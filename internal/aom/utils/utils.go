// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package utils

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"log"
	"os"
	"strings"
	"time"
)

// Call an operation and trigger a callback on a regular interval,
// finally return the error status of the given operation.
// operation - A potentially long running operation whose final error is to be returned.
// heartBeatCallback - A function to be called at the regular interval given by heartBeat.
// heartBeat - Interval used to trigger the heartBeatCallback.
func ApplyOperationWithHeartBeat(operation func() error, heartBeatCallback func(), heartBeat time.Duration) error {
	heartBeatTicker := time.NewTicker(heartBeat)
	defer heartBeatTicker.Stop()

	// send the first pulse to avoid the case where the operation finishes before
	// the first heartbeat, which fails to trigger the "end" event on the client side.
	heartBeatCallback()

	done := make(chan error)
	go func() {
		done <- operation()
	}()

	for {
		select {
		case err, ok := <-done:
			if ok {
				return err
			} else {
				log.Println("Channel closed")
			}
		case <-heartBeatTicker.C:
			heartBeatCallback()
		}
	}
}

// Does exactly what it says on the tin.
func ReplaceSlashesWithDashes(input string) string {
	return strings.ReplaceAll(input, "/", "-")
}

// check for the environment variable with a given key
// and return the bool value
func GetEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return value == "true"
	}
	return fallback
}

// Function returns a 5 digit SHA1 hash string for the provided content.
func GetShortSHA1HashFrom(content []byte) (string, error) {
	sha1HashFunc := func(content []byte) (string, error) {
		sha1 := sha1.New()
		if _, err := sha1.Write(content); err != nil {
			return "", err
		}
		return hex.EncodeToString(sha1.Sum(nil)), nil
	}

	return GetShortHashFromWithCustomHashFunc(content, sha1HashFunc)
}

// Function returns a 5 digit hash string for the provided content.
// The provided hash function defines the used hash algorithm.
func GetShortHashFromWithCustomHashFunc(content []byte, customHashFunc func([]byte) (string, error)) (string, error) {
	hash, err := customHashFunc(content)
	if err != nil {
		return "", err
	}
	return hash[0:5], nil
}

// RetryFunc represents the callback function that will be invoked n times,
// where n is the number of retries, before returning an error if no result is available
type RetryFunc func() (interface{}, error)

// Retry returns the callback func result or an error if all retry attempts failed
func Retry(attempts int, delay time.Duration, retryCallbackFunc RetryFunc) (interface{}, error) {
	if attempts < 0 {
		return nil, errors.New("Retry attempts should be greater than 0")
	}
	var err error
	var res interface{}
	for i := 0; i < attempts; i++ {
		res, err = retryCallbackFunc()
		if err != nil {
			time.Sleep(delay)
			continue
		}
		return res, nil
	}
	return nil, err
}
