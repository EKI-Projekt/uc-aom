// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
)

// MkDirAll Creates a directory and set the permissions additionally
// https://github.com/golang/go/issues/15210
func MkDirAll(path string, permission os.FileMode) error {
	err := os.MkdirAll(path, os.ModeDir)
	if err != nil {
		return err
	}
	return os.Chmod(path, permission)
}

// WriteFileToDestination write a file content in to the destination and
// set the permissions additionally
// https://github.com/golang/go/issues/15210
func WriteFileToDestination(fileName string, fileData []byte, destination string) error {
	target := filepath.Join(destination, fileName)
	fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}

	if err := os.Chmod(target, 0644); err != nil {
		return err
	}

	fileReader := bytes.NewReader(fileData)
	if _, err := io.Copy(fileToWrite, fileReader); err != nil {
		return err
	}
	return fileToWrite.Close()
}
