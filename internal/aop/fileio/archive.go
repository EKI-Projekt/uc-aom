// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package fileio

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cavaliergopher/cpio"
)

type ArchiveFileEntry struct {
	Name string
	Body []byte
}

// Returns a tarball of a directory including any subdirectories or an error if it fails.
func Tarball(path string, predicate func(p string) bool, headerName HeaderNameFunc) ([]byte, error) {
	var buf bytes.Buffer

	if err := writeDirectoryAsTar(&buf, path, predicate, headerName); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

// Returns a gzipped tarball of path including any subdirectories or an error if it fails.
func GzipTarball(path string, predicate func(p string) bool) ([]byte, error) {
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)

	if err := writeDirectoryAsTar(gzipWriter, path, predicate, filepath.Rel); err != nil {
		return []byte{}, err
	}

	if err := gzipWriter.Close(); err != nil {
		return []byte{}, err
	}

	return buf.Bytes(), nil
}

func CreateCpioArchive(outputPath string, files []ArchiveFileEntry) error {
	buf := new(bytes.Buffer)
	w := cpio.NewWriter(buf)

	for _, file := range files {
		hdr := &cpio.Header{
			Name: file.Name,
			Mode: 0600,
			Size: int64(len(file.Body)),
		}
		if err := w.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := w.Write(file.Body); err != nil {
			return err
		}
	}

	if err := w.Close(); err != nil {
		return err
	}

	return ioutil.WriteFile(outputPath, buf.Bytes(), 0644)
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func UnGzipTarball(destinationPath string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()
	return UnTarball(destinationPath, gzr)
}

// Untar takes a destination path and a reader; a tar reader loops over the tarfile
// creating the file structure at 'dst' along the way, and writing any files
func UnTarball(destinationPath string, r io.Reader) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()

		switch {

		// if no more files are found return
		case err == io.EOF:
			return nil

		// return any other error
		case err != nil:
			return err

		// if the header is nil, just skip it (not sure how this happens)
		case header == nil:
			continue
		}

		// the target location where the dir/file should be created
		target := filepath.Join(destinationPath, header.Name)

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			f.Close()
		}
	}
}

// Writes the contents of the directory as a tarball into the provided writer
func writeDirectoryAsTar(writer io.Writer, inputDirectory string, predicate func(p string) bool, headerName HeaderNameFunc) error {
	inputDirectoryInfo, err := os.Stat(inputDirectory)
	if err != nil {
		return err
	}
	if !inputDirectoryInfo.IsDir() {
		return fs.ErrInvalid
	}

	tarWriter := tar.NewWriter(writer)

	filepath.WalkDir(inputDirectory, func(filePath string, directory fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fileInfo, err := directory.Info()
		if err != nil {
			return err
		}

		if !directory.IsDir() && !predicate(filePath) {
			return nil
		}

		header, err := tar.FileInfoHeader(fileInfo, filePath)
		if err != nil {
			return err
		}

		rel, err := headerName(inputDirectory, filePath)
		if err != nil {
			return err
		}
		header.Name = rel

		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if !directory.IsDir() {
			data, err := os.Open(filePath)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tarWriter, data); err != nil {
				return err
			}
		}
		return nil
	})

	if err := tarWriter.Close(); err != nil {
		return err
	}

	return nil
}
