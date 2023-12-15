// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest_test

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/pkg/config"

	"github.com/stretchr/testify/assert"
)

type item struct {
	Title string `json:"title"`
}

func TestListManifestsDirectories(t *testing.T) {
	// arrange
	repository := manifest.NewRepository(os.ReadFile, filepath.WalkDir)
	expectedDirName := "test-uc-addon-status-running/@0.1.0"

	tmpDir := t.TempDir()
	type testFile struct {
		dir  string
		file string
	}
	testCases := []testFile{
		{
			dir:  expectedDirName,
			file: "manifest.json",
		},
		{
			dir:  config.DROP_IN_FOLDER_NAME,
			file: "manifest.json",
		},
		{
			dir:  "testFolder",
			file: "image-manifest.json",
		},
	}
	for _, testCase := range testCases {
		testDir := path.Join(tmpDir, testCase.dir)
		err := os.MkdirAll(testDir, os.ModePerm)
		assert.NoError(t, err)
		err = os.WriteFile(path.Join(testDir, testCase.file), make([]byte, 0), os.ModePerm)
		assert.NoError(t, err)
	}

	// act
	manifestDirectories, err := repository.GetManifestsDirectories(tmpDir)

	// assert
	if err != nil {
		t.Errorf("Failed getting manifests directories %v", err)
	}

	assert.Equal(t, []string{expectedDirName}, manifestDirectories)
}

type MockFileReader struct {
	Result      []byte
	ResultError error
}

type MockDirWalker struct {
	Result      []fs.DirEntry
	ResultError error
}

func (reader *MockFileReader) ReadFile(name string) ([]byte, error) {
	return reader.Result, reader.ResultError
}

func (reader *MockDirWalker) WalkDir(root string, fn fs.WalkDirFunc) error {
	for _, item := range reader.Result {
		err := fn(item.Name(), item, reader.ResultError)
		if err != nil {
			return err
		}
	}
	return reader.ResultError
}

type MockDirEntry struct {
	FileName    string
	IsDirectory bool
}

func (mfi MockDirEntry) Name() string               { return mfi.FileName }
func (mfi MockDirEntry) Size() int64                { return int64(8) }
func (mfi MockDirEntry) Mode() os.FileMode          { return os.ModePerm }
func (mfi MockDirEntry) ModTime() time.Time         { return time.Now() }
func (mfi MockDirEntry) IsDir() bool                { return mfi.IsDirectory }
func (mfi MockDirEntry) Sys() interface{}           { return nil }
func (mfi MockDirEntry) Info() (fs.FileInfo, error) { return mfi, nil }
func (mfi MockDirEntry) Type() fs.FileMode          { return mfi.Mode().Type() }

func TestReadManifestFromOK(t *testing.T) {
	// arrange
	var mockFileInfoAnyviz = MockDirEntry{FileName: filepath.Join(manifest.BASE_PATH, "anyviz", config.UcImageManifestFilename), IsDirectory: false}
	dirReaderResult := []fs.DirEntry{mockFileInfoAnyviz}
	testByte := []byte(`{ "version": "0.1", "title": "AnyViz Cloud Adapter", "description": "The AnyViz cloud solution allows you to remotely monitor, control and analyse industrial PLCs, sensors and meters.", "logo": "logo.png", "services": { "cloudadapter": { "type": "docker-compose", "config": { "image": "anyviz/cloudadapter", "restart": "always", "containerName": "anyviz", "ports": [ "8888:8888" ] } } }, "settings": { "environmentVariables": [ { "name": "SETTING_1", "label": "Setting 1", "default": "abc", "required": true }, { "name": "SETTING_2", "label": "Setting 2", "default": "xyz", "required": true, "pattern": "^[a-zA-Z]+$" } ] }, "vendor": { "name": "Abc", "url": "www.abc.de", "email": "contact@abc.de", "street": "alphabetstr. 7", "zip": "60329", "city": "Frankfurt a.M", "country": "Germany" } } `)
	mockFileReader := MockFileReader{Result: testByte, ResultError: nil}
	mockDirWalker := MockDirWalker{Result: dirReaderResult, ResultError: nil}
	uut := manifest.NewRepository(mockFileReader.ReadFile, mockDirWalker.WalkDir)
	title := "Non existing addon"

	// act
	manifest, err := uut.ReadManifestFrom(filepath.Join(manifest.BASE_PATH, title))

	// assert
	if err != nil {
		t.Error("Expect ReadManifestFrom() to succeed!")
	}
	if manifest.Title != "AnyViz Cloud Adapter" {
		t.Errorf("Expected '%s' but got '%s'", "AnyViz Cloud Adapter", manifest.Title)
	}
}

func TestReadManifestFromFailed(t *testing.T) {
	// arrange
	var mockFileInfoAnyviz = MockDirEntry{FileName: filepath.Join(manifest.BASE_PATH, "anyviz", config.UcImageManifestFilename), IsDirectory: false}
	dirReaderResult := []fs.DirEntry{mockFileInfoAnyviz}
	mockFileReader := MockFileReader{Result: nil, ResultError: fs.ErrNotExist}
	mockDirWalker := MockDirWalker{Result: dirReaderResult, ResultError: nil}
	uut := manifest.NewRepository(mockFileReader.ReadFile, mockDirWalker.WalkDir)
	title := "Non existing addon"

	// act
	_, err := uut.ReadManifestFrom(filepath.Join(manifest.BASE_PATH, title))

	// assert
	if err == nil {
		t.Error("Expect ReadManifestFrom() to fail!")
	}
}

type tarArchiveFile struct {
	Name    string
	Content []byte
}

func TestDecompress_ShallWriteFilesToLocalFileSystem(t *testing.T) {
	// arrange
	logoTarFile := createLogo("logo.png")
	manifestTarFile := createManifestWith(logoTarFile)

	reader := createManifestLogoTarReader(t, &manifestTarFile, &logoTarFile)
	tempDir := t.TempDir()
	decompressor := manifest.ManifestTarGzipDecompressor{}

	// act
	rc, err := decompressor.Decompress(reader)
	if err := manifest.WriteUcManifestContentToDestination(rc, tempDir); err != nil {
		t.Fatalf("Decompress() failed! %v", err)
	}

	// assert
	fileDirEntries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("os.ReadDir(tempDir) %v", err)
	}

	expectedNumberOfFiles := 2
	if len(fileDirEntries) != expectedNumberOfFiles {
		t.Errorf("Wrong number of files in destination directory, want %d, got: %d", expectedNumberOfFiles, len(fileDirEntries))
	}

}
