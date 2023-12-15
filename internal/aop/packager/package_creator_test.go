// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package packager_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"strings"
	"testing"
	"u-control/uc-aom/internal/aop/manifest"
	"u-control/uc-aom/internal/aop/packager"
	"u-control/uc-aom/internal/aop/registry"
	"u-control/uc-aom/internal/pkg/config"

	model "u-control/uc-aom/internal/pkg/manifest"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/mock"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	orasRegistry "oras.land/oras-go/v2/registry"
)

type ioStubPackageCreator struct {
	dockerTarballPayload []byte
	gzipContentsError    error
	gzipTarballPayload   []byte
	gzipTarballError     error
}

func (r *ioStubPackageCreator) exportDockerImageFuncMock(registry orasRegistry.Registry, ref string, platform *ocispec.Platform) ([]byte, error) {
	return r.dockerTarballPayload, r.gzipContentsError
}

func (r *ioStubPackageCreator) gzipTarballFuncMock(path string, predicate func(string) bool) ([]byte, error) {
	return r.gzipTarballPayload, r.gzipTarballError
}

type mockManifestReader struct {
	mock.Mock

	logoErr error
}

func (r *mockManifestReader) ReadManifestFrom(directoryOfManifest string) (*model.Root, error) {
	args := r.Called(directoryOfManifest)
	return args.Get(0).(*model.Root), args.Error(1)
}

func (r *mockManifestReader) fileExistsFunc(path string) (fs.FileInfo, error) {
	return nil, r.logoErr
}

func TestAddOnOciImageCreationPass(t *testing.T) {
	// Arrange
	testCases := []struct {
		Name       string
		ImageIndex string
	}{
		{
			Name:       "Image Index with single image Manifest",
			ImageIndex: ImageIndexWithSingleImageManifestJSON,
		},
		{
			Name: "Image Index with attestation manifest",
			ImageIndex: `{
		"mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
		"schemaVersion": 2,
		"manifests": [
		  {
			"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
			"digest": "sha256:d8a6c150cc9ec2dbc8b3b6c676d90046d6d3e820a6cb1f2ff93a24cc6273207e",
			"size": 528,
			"platform": {
			  "architecture": "arm",
			  "os": "linux",
			  "variant": "v7"
			}
		  },
		  {
			"mediaType": "application/vnd.oci.image.manifest.v1+json",
			"digest": "sha256:5b6eb6c6190f2773e3160addb7c804a0c275688489d7954bd225ff8efb537b3d",
			"size": 566,
			"annotations": {
				"vnd.docker.reference.digest": "sha256:d8a6c150cc9ec2dbc8b3b6c676d90046d6d3e820a6cb1f2ff93a24cc6273207e",
				"vnd.docker.reference.type": "attestation-manifest"
			},
			"platform": {
				"architecture": "unknown",
				"os": "unknown"
			}
		  }
		]
	  }`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {

			ctx := context.Background()
			manifestGzipTarballPayload := "manifest-logo-dir"
			dockerTarballPayload := "docker-image-path"
			tag := "0.42.0-1"

			ioStubPackageCreator := ioStubPackageCreator{
				dockerTarballPayload: []byte(dockerTarballPayload),
				gzipTarballPayload:   []byte(manifestGzipTarballPayload),
			}
			pkg := packager.NewPackageCreator(ioStubPackageCreator.gzipTarballFuncMock, ioStubPackageCreator.exportDockerImageFuncMock)
			root := &model.Root{
				ManifestVersion: "0.1",
				Version:         tag,
				Logo:            "logo.png",
				Title:           "add-on's title",
				Description:     "Describes this test add-on",
				Services: map[string]*model.Service{
					"ucAddonTestService": {
						Type:   "docker-compose",
						Config: map[string]interface{}{"image": "test/docker-image:v1.1.1-alpha"},
					},
				},
				Platform: []string{"ucm", "ucg"},
				Vendor: &model.Vendor{
					Name:    "abc",
					Url:     "https://www.abc.de",
					Email:   "email@abc.de",
					Street:  "street",
					Zip:     "12345",
					City:    "City",
					Country: "Country",
				},
			}

			mockManifest := &mockManifestReader{logoErr: nil}
			mockManifest.On("ReadManifestFrom", mock.AnythingOfType("string")).Return(root, nil)

			// Act
			addOnManifest, err := manifest.ParseAndValidate(mockManifest, mockManifest.fileExistsFunc, "")
			if err != nil {
				t.Fatal(err)
			}

			rc := nopCloser{bytes.NewBufferString(testCase.ImageIndex)}
			desc := ocispec.Descriptor{}

			mockOrasRegistry := &mockOrasRegistry{}
			mockRepo := &mockRepo{}
			mockOrasRegistry.On("Repository", mock.Anything, mock.AnythingOfType("string")).Return(mockRepo, nil)
			mockRepo.On("Resolve", mock.Anything, mock.AnythingOfType("string")).Return(desc, nil)
			mockRepo.On("Fetch", mock.Anything, mock.Anything).Return(rc, nil)

			memoryStore := memory.New()

			addOnTarget, err := pkg.CreateOciArtifact(ctx, addOnManifest, mockOrasRegistry)
			if err != nil {
				t.Fatal("[CreateOciArtifact] Unexpected error = ", err)
			}

			_, err = registry.Copy(ctx, addOnTarget, memoryStore, oras.DefaultCopyOptions)
			if err != nil {
				t.Fatal("[Push] Unexpected error = ", err)
			}
			validator, err := packager.NewPackageValidator(ctx, addOnTarget)
			if err != nil {
				t.Fatal("packager.NewPackageValidator() Unexpected error = ", err)
			}

			// Act & Assert
			expectedManifestAnnotations := []*packager.AnnotationPair{
				{ocispec.AnnotationTitle, config.UcImageLayerAnnotationTitle},
				{ocispec.AnnotationVersion, tag},
				{config.UcImageLayerAnnotationSchemaVersion, "0.1"},
			}
			expectedDockerAnnotations := []*packager.AnnotationPair{
				{ocispec.AnnotationTitle, "test/docker-image:v1.1.1-alpha"},
			}

			err = validator.ValidateLayerDescriptorAnnotations(expectedManifestAnnotations, expectedDockerAnnotations)
			if err != nil {
				t.Fatal("[validator.ValidateLayerDescriptorAnnotations()] Unexpected error = ", err)
			}

			expectedLayerContent := []string{manifestGzipTarballPayload, dockerTarballPayload}
			err = validator.ValidateLayerContent(func(content io.ReadCloser, descriptor ocispec.Descriptor, index int) bool {
				stringBuffer := new(strings.Builder)
				io.Copy(stringBuffer, content)
				return stringBuffer.String() == expectedLayerContent[index]
			})
			if err != nil {
				t.Fatal("[validator.ValidateLayerContent()] Unexpected error = ", err)
			}
		})

	}

}

func TestAddOnOciImageCreationFail(t *testing.T) {
	// Arrange
	ctx := context.Background()

	ioStubPackageCreator := ioStubPackageCreator{
		gzipContentsError: fs.ErrNotExist,
		gzipTarballError:  fs.ErrPermission,
	}
	pkg := packager.NewPackageCreator(ioStubPackageCreator.gzipTarballFuncMock, ioStubPackageCreator.exportDockerImageFuncMock)

	root := &model.Root{
		ManifestVersion: "0.1",
		Version:         "0.17.0-1",
		Logo:            "logo.png",
		Services: map[string]*model.Service{
			"ucAddonTestService": {
				Type:   "docker-compose",
				Config: map[string]interface{}{"image": "test/image:0.42"},
			},
		},
		Platform: []string{"ucm", "ucg"},
		Vendor: &model.Vendor{
			Name:    "abc",
			Url:     "https://www.abc.de",
			Email:   "email@abc.de",
			Street:  "street",
			Zip:     "12345",
			City:    "City",
			Country: "Country",
		},
	}

	mockManifest := &mockManifestReader{logoErr: nil}
	mockManifest.On("ReadManifestFrom", mock.AnythingOfType("string")).Return(root, nil)

	// Act
	addOnManifest, err := manifest.ParseAndValidate(mockManifest, mockManifest.fileExistsFunc, "")
	if err != nil {
		t.Fatal(err)
	}

	// Act & Assert
	_, err = pkg.CreateOciArtifact(ctx, addOnManifest, nil)
	if err == nil {
		t.Fatal("[CreateOciArtifact] Expected error was not thrown")
	}
}
