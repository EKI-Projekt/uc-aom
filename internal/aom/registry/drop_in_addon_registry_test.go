// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aom/registry"
	"u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/opencontainers/go-digest"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/mock"
)

type mockProcessor struct {
	mock.Mock
}

func (p *mockProcessor) Filter(desc *ocispec.Descriptor) bool {
	args := p.Called(desc)
	return args.Bool(0)
}

func (p *mockProcessor) Action(src io.Reader, mediaType string) {
	p.Called(src, mediaType)
}

const (
	architecture = "arm"
	OS           = "linux"
)

var decompressor *registry.MockDecompressor

func createUut(testDir string) registry.AddOnRegistry {
	decompressor = registry.NewMockDecompressor()
	testDropInRegistry := registry.NewDropInAddOnRegistry(testDir, architecture, OS, decompressor)
	return testDropInRegistry
}

func createConfigJson(t *testing.T, addOnBasePath string) {
	t.Helper()
	configContent := fmt.Sprintf(`{"os":"%s","architecture": "%s"}`, OS, architecture)
	createFile(t, addOnBasePath, config.UcConfigAnnotationTitle, configContent)
}

func createImageManifestJson(t *testing.T, addOnBasePath string) {
	t.Helper()

	imageManifest := ocispec.Manifest{
		Versioned: specs.Versioned{},
		Config: ocispec.Descriptor{
			MediaType:   config.UcConfigMediaType,
			Digest:      digest.FromString("sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71dc"),
			Size:        123,
			Annotations: map[string]string{ocispec.AnnotationTitle: config.UcConfigAnnotationTitle},
		},
		Layers: []ocispec.Descriptor{
			{
				MediaType: config.UcImageLayerMediaType,
				Digest:    digest.FromString("sha256:7173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71de"),
				Size:      7154,
				Annotations: map[string]string{
					ocispec.AnnotationTitle:                    config.UcImageLayerAnnotationTitle,
					ocispec.AnnotationVersion:                  "0.1.0-1",
					config.UcImageLayerAnnotationSchemaVersion: manifest.ValidManifestVersion,
				},
			},
			{
				MediaType: ocispec.MediaTypeImageLayer,
				Digest:    digest.FromString("sha256:8173b809ca12ec5dee4506cd86be934c4596dd234ee82c0662eac04a8c2c71de"),
				Size:      7154,
				Annotations: map[string]string{
					ocispec.AnnotationTitle: "dockerFile",
				},
			},
		},
	}

	content, err := json.Marshal(imageManifest)
	if err != nil {
		t.Fatalf("Unable to create image-manifest.json at '%s': %v", addOnBasePath, err)
	}

	createFile(t, addOnBasePath, config.UcImageManifestDescriptorFilename, string(content))
}

func createDirectories(t *testing.T, basePath string, directoryNames ...string) {
	t.Helper()
	for _, directoryName := range directoryNames {
		err := os.MkdirAll(path.Join(basePath, directoryName), os.ModePerm)
		if err != nil {
			t.Fatalf("Unexpected error %s", err)
		}
	}

}

func createFile(t *testing.T, basePath string, filename string, content string) {
	t.Helper()
	err := os.MkdirAll(basePath, os.ModePerm)
	if err != nil {
		t.Fatalf("os.MkdirAll(): unexpected error %s", err)
	}
	err = os.WriteFile(path.Join(basePath, filename), []byte(content), os.ModePerm)
	if err != nil {
		t.Fatalf("os.WriteFile(): unexpected error %s", err)
	}
}

func areEquivalent(t *testing.T, expected []string, actual []string) bool {
	t.Helper()
	sort.Strings(expected)
	sort.Strings(actual)
	return reflect.DeepEqual(expected, actual)
}

func TestRepositories(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepositories := []string{"testRepository_0", "testRepository_1", "testRepository_2"}
	createDirectories(t, testDir, testRepositories...)
	testDropInRegistry := createUut(testDir)

	// Act
	repositories, err := testDropInRegistry.Repositories()

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if len(testRepositories) != len(repositories) {
		t.Fatalf("Expected %d repositories but got %d", len(testRepositories), len(repositories))
	}

	if !reflect.DeepEqual(testRepositories, repositories) {
		t.Fatalf("Expected repositories (%s) mismatch actual repositories %s", testRepositories, repositories)
	}
}

func TestRepositoriesShallOnlyReturnDirectories(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepositories := []string{"testRepository_0", "testRepository_1", "testRepository_2"}
	createDirectories(t, testDir, testRepositories...)
	createFile(t, testDir, "aTestFile", "")
	createFile(t, testDir, "zTestFile", "")

	testDropInRegistry := createUut(testDir)

	// Act
	repositories, err := testDropInRegistry.Repositories()

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if len(testRepositories) != len(repositories) {
		t.Fatalf("Expected %d repositories but got %d", len(testRepositories), len(repositories))
	}

	if !reflect.DeepEqual(testRepositories, repositories) {
		t.Fatalf("Expected repositories (%s) mismatch actual repositories %s", testRepositories, repositories)
	}
}

func TestRepositoriesWithSubfolder(t *testing.T) {
	// Arrange
	testDir := t.TempDir()

	testRepositories := []string{"testRepository_0", "testRepository_1", "testRepository_2"}
	createDirectories(t, testDir, testRepositories...)
	testSubfolder := []string{"aSubfolder", "zSubfolder"}
	testSubfolderPath := path.Join(testDir, testRepositories[0])
	createDirectories(t, testSubfolderPath, testSubfolder...)

	testDropInRegistry := createUut(testDir)

	// act
	repositories, err := testDropInRegistry.Repositories()

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if len(testRepositories) != len(repositories) {
		t.Fatalf("Expected %d repositories but got %d", len(testRepositories), len(repositories))
	}

	if !reflect.DeepEqual(testRepositories, repositories) {
		t.Fatalf("Expected repositories (%s) mismatch actual repositories %s", testRepositories, repositories)
	}
}

func TestRepositoriesEmptyDirectory(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testDropInRegistry := createUut(testDir)

	// Act
	repositories, err := testDropInRegistry.Repositories()

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	if len(repositories) != 0 {
		t.Fatalf("Expected length %d repositories but got %d", 0, len(repositories))
	}

}

func TestTag(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	createDirectories(t, testDir, testRepository)

	testTags := []string{"0.1.0-1", "0.1.0-2", "0.2.0-1", "0.2.0-2", "0.2.0-3"}
	repositoryPath := path.Join(testDir, testRepository)
	createDirectories(t, repositoryPath, testTags...)
	testDropInRegistry := createUut(testDir)

	// act
	tags, err := testDropInRegistry.Tags(testRepository)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}
	if !reflect.DeepEqual(testTags, tags) {
		t.Fatalf("Expected tags (%s) mismatch actual tags %s", testTags, tags)
	}
}

func TestDelete(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	createDirectories(t, testDir, path.Join(testRepository, testTag))
	testDropInRegistry := createUut(testDir)

	// Act
	err := testDropInRegistry.Delete(testRepository, testTag)

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	_, err = os.Stat(path.Join(testDir, testRepository, testTag))
	if err == nil {
		t.Fatalf("Deleting failed: AddOn still exists.")
	}
}

func TestDeleteEmptyDirectory(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testDropInRegistry := createUut(testDir)

	// Act
	err := testDropInRegistry.Delete("", "")

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}
}

func TestDeleteNotRemoveRoot(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testDropInRegistry := createUut(testDir)

	// Act
	err := testDropInRegistry.Delete("", "")

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}
	_, err = os.Stat(testDir)
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}
}

func TestDeleteNotRemoveRootIfRepositoryDoesNotExist(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testDropInRegistry := createUut(testDir)

	// Act
	err := testDropInRegistry.Delete("test", "1.0")

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}
	_, err = os.Stat(testDir)
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}
}

func TestPull(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	addOnBasePath := path.Join(testDir, testRepository, testTag)
	os.MkdirAll(addOnBasePath, os.ModePerm)

	manifestFileContent := "manifestFile content"
	createFile(t, addOnBasePath, config.UcImageLayerAnnotationTitle, manifestFileContent)

	dockerFileName := "dockerFile"
	dockerFileContent := "dockerFile content"
	createFile(t, addOnBasePath, dockerFileName, dockerFileContent)

	createConfigJson(t, addOnBasePath)
	createImageManifestJson(t, addOnBasePath)

	expectedManifestLayerMediaTypes := []string{ocispec.MediaTypeImageLayer, config.UcImageLayerMediaType}

	mediaTypeToLayerContentMap := make(map[string]string)

	action := func(src io.Reader, mediaType string) {
		buf := new(strings.Builder)
		io.Copy(buf, src)
		mediaTypeToLayerContentMap[mediaType] = buf.String()
	}

	testDropInRegistry := createUut(testDir)
	decompressor.WithFetchContent([]byte(manifestFileContent))

	// Act
	_, err := testDropInRegistry.Pull(testRepository, testTag, registry.NewAcceptAllManifestLayerProcessor(action))

	// Assert
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	actualManifestLayerMediaTypes := make([]string, 0, len(mediaTypeToLayerContentMap))
	for mediaType := range mediaTypeToLayerContentMap {
		actualManifestLayerMediaTypes = append(actualManifestLayerMediaTypes, mediaType)
	}

	if !areEquivalent(t, expectedManifestLayerMediaTypes, actualManifestLayerMediaTypes) {
		t.Errorf("Manifest layer MediaType mismatch: want %v, but got %v", expectedManifestLayerMediaTypes, actualManifestLayerMediaTypes)
	}

	if mediaTypeToLayerContentMap[config.UcImageLayerMediaType] != manifestFileContent {
		t.Errorf("Wrong manifest content: want %s, but got %s", manifestFileContent, mediaTypeToLayerContentMap[config.UcImageLayerMediaType])
	}
}

func TestPullMissingFolder(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testDropInRegistry := createUut(testDir)

	testRepository := "testRepository"
	testTag := "0.1.0-1"

	action := func(src io.Reader, mediaType string) {}

	// Act
	_, err := testDropInRegistry.Pull(testRepository, testTag, registry.NewAcceptAllManifestLayerProcessor(action))

	// Assert
	if err == nil {
		t.Fatalf("Expected error.")
	}
}

func TestPullShouldNotCallActionIfPredicateReturnsFalse(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	createDirectories(t, testDir, path.Join(testRepository, testTag))

	addOnBasePath := path.Join(testDir, testRepository, testTag)
	createFile(t, addOnBasePath, config.UcImageLayerAnnotationTitle, "")

	absCountOfActionCalls := 0
	action := func(src io.Reader, mediaType string) {
		absCountOfActionCalls++
	}

	testDropInRegistry := createUut(testDir)
	createConfigJson(t, addOnBasePath)
	createImageManifestJson(t, addOnBasePath)

	// Act
	_, err := testDropInRegistry.Pull(testRepository, testTag, registry.NewAcceptNoneManifestLayerProcessor(action))

	// Assert
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	const expectedActionCalls = 0
	if absCountOfActionCalls != expectedActionCalls {
		t.Errorf("countOfActionCalls: got %d, but want %d", absCountOfActionCalls, expectedActionCalls)
	}

}

func TestPullShouldNotCallActionForAddOnManifestIfPredicateReturnsFalse(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	createDirectories(t, testDir, path.Join(testRepository, testTag))
	addOnBasePath := path.Join(testDir, testRepository, testTag)
	manifestFileContent := "manifestFile content"
	createFile(t, addOnBasePath, config.UcImageLayerAnnotationTitle, manifestFileContent)

	dockerFileName := "dockerFile"
	dockerFileContent := "dockerFile content"
	createFile(t, addOnBasePath, dockerFileName, dockerFileContent)

	countOfManifestActionCalls := 0
	countOfDockerActionCalls := 0
	action := func(src io.Reader, mediaType string) {
		if registry.IsUcImageLayerMediaType(mediaType) {
			countOfManifestActionCalls++
			return
		}
		if mediaType == ocispec.MediaTypeImageLayer {
			countOfDockerActionCalls++
		}
	}

	testDropInRegistry := createUut(testDir)
	createConfigJson(t, addOnBasePath)
	createImageManifestJson(t, addOnBasePath)

	// Act
	_, err := testDropInRegistry.Pull(testRepository, testTag, registry.NewAllExceptUcImageLayerProcessor(action))

	// Assert
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	const expectedCountOfManifestActionCalls = 0
	if countOfManifestActionCalls != expectedCountOfManifestActionCalls {
		t.Errorf("countOfManifestActionCalls: got %d, but want %d", countOfManifestActionCalls, expectedCountOfManifestActionCalls)
	}

	const expectedCountOfDockerActionCalls = 1
	if countOfDockerActionCalls != expectedCountOfDockerActionCalls {
		t.Errorf("countOfDockerActionCalls: got %d, but want %d", countOfDockerActionCalls, expectedCountOfDockerActionCalls)
	}

}

func TestPullEmptyRepository(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	createDirectories(t, testDir, path.Join(testRepository, testTag))
	action := func(src io.Reader, mediaType string) {}

	testDropInRegistry := createUut(testDir)

	// Act
	_, err := testDropInRegistry.Pull(testRepository, testTag, registry.NewAcceptNoneManifestLayerProcessor(action))

	// Assert
	if err == nil {
		t.Fatalf("Expected error.")
	}
}

func TestPullCantOpenFile(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	addOnBasePath := path.Join(testDir, testRepository, testTag)
	createDirectories(t, testDir, path.Join(testRepository, testTag))

	openFileName := "dockerFile"
	openFileContent := "unreadable content"
	createFile(t, addOnBasePath, openFileName, openFileContent)

	testDropInRegistry := createUut(testDir)

	// Act
	_, err := os.Open(path.Join(addOnBasePath, openFileName))
	if err != nil {
		t.Fatalf("Unexpected error %s", err)
	}

	processor := &mockProcessor{}

	processor.On("Filter", mock.AnythingOfType("*ocispec.Descriptor")).Run(func(args mock.Arguments) {
		os.Remove(path.Join(addOnBasePath, openFileName))
	}).Return(true)
	processor.On("Action", mock.Anything, mock.AnythingOfType("string"))

	_, err = testDropInRegistry.Pull(testRepository, testTag, processor)

	// Assert
	if err == nil {
		t.Fatalf("Expected an error.")
	}
}

func TestPullOnlyValidArchitectureAndOS(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	addOnBasePath := path.Join(testDir, testRepository, testTag)
	os.MkdirAll(addOnBasePath, os.ModePerm)

	manifestFileContent := "manifestFile content"
	createFile(t, addOnBasePath, config.UcImageLayerAnnotationTitle, manifestFileContent)

	dockerFileName := "dockerFile"
	dockerFileContent := "dockerFile content"
	createFile(t, addOnBasePath, dockerFileName, dockerFileContent)

	otherArchitecture := fmt.Sprintf("OtherArchThan_%s", architecture)
	configContent := fmt.Sprintf(`{"os":"%s","architecture": "%s"}`, OS, otherArchitecture)
	configFileName := config.UcConfigAnnotationTitle
	createFile(t, addOnBasePath, configFileName, configContent)
	createImageManifestJson(t, addOnBasePath)

	testMap := make(map[string]string)

	action := func(src io.Reader, mediaType string) {
		buf := new(strings.Builder)
		io.Copy(buf, src)
		testMap[mediaType] = buf.String()
	}

	testDropInRegistry := createUut(testDir)

	// Act
	_, err := testDropInRegistry.Pull(testRepository, testTag, registry.NewAcceptAllManifestLayerProcessor(action))

	// Assert
	if !errors.Is(err, registry.ErrWrongArchOrOS) {
		t.Fatalf("DropInRegistry.Pull() = %v, want %v", err, registry.ErrWrongArchOrOS)
	}
}

func TestPullShallReturnErrorIfConfigIsInvalid(t *testing.T) {
	// Arrange
	testDir := t.TempDir()
	testRepository := "testRepository"
	testTag := "0.1.0-1"
	addOnBasePath := path.Join(testDir, testRepository, testTag)
	os.MkdirAll(addOnBasePath, os.ModePerm)

	configContent := "NotaJSONContent"
	configFileName := config.UcConfigAnnotationTitle
	createFile(t, addOnBasePath, configFileName, configContent)
	createImageManifestJson(t, addOnBasePath)

	action := func(src io.Reader, mediaType string) {
	}

	testDropInRegistry := createUut(testDir)

	// Act
	_, err := testDropInRegistry.Pull(testRepository, testTag, registry.NewAcceptAllManifestLayerProcessor(action))

	// Assert
	if !errors.Is(err, registry.ErrReadImageConfig) {
		t.Fatalf("DropInRegistry.Pull() = %v, want %v", err, registry.ErrReadImageConfig)
	}
}
