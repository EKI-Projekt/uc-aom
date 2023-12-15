// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package utils

import (
	"os"
	"path"
)

// Callback function to filter files
type FilesFilter func(fileName string) bool

// Copy files from the source to the destination directory.
func CopyFiles(sourceDirectory string, destinationDirectory string, filesFilterFunc FilesFilter) error {
	sourceFiles, err := os.ReadDir(sourceDirectory)
	if err != nil {
		return err
	}
	for _, file := range sourceFiles {
		if file.IsDir() || !filesFilterFunc(file.Name()) {
			continue
		}
		sourceFilePath := path.Join(sourceDirectory, file.Name())
		reader, err := os.ReadFile(sourceFilePath)
		if err != nil {
			return err
		}

		destinationFilePath := path.Join(destinationDirectory, file.Name())
		err = os.WriteFile(destinationFilePath, reader, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}
