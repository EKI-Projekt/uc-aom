// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package fileio

import (
	"io"
	"io/fs"
)

// Reads the contents of the file at path and returns the result or an error.
type ReadFileFunc func(path string) ([]byte, error)

// Checks if the given path exits.
type FileExistsFunc func(path string) (fs.FileInfo, error)

// Creates an cpio-archive-file at specified path with the specified files
type CreateCpioArchiveFunc func(outputPath string, files []ArchiveFileEntry) error

// Given the path this function returns a tarball as byte array or an error.
// Only those visited filepath (directories are not considered) that pass the predicate
// are included in the final tarball.
type TarballFunc func(path string, predicate func(filepath string) bool, headerName HeaderNameFunc) ([]byte, error)

// Given the path this function returns a gzipped tarball as byte array or an error.
// Only those visited filepath (directories are not considered) that pass the predicate
// are included in the final tarball.
type GzipTarballFunc func(path string, predicate func(filepath string) bool) ([]byte, error)

// Given the path this function and fileReader to the gzipped tarball.
// All parts of of the tarball will be extract to the given path.
type UnGzipTarballFunc func(path string, fileReader io.Reader) error

// Given a basepath and a filepath this function can return an
// appropriate Header Name for a TAR or CPIO archive.
type HeaderNameFunc func(basepath string, filepath string) (string, error)
