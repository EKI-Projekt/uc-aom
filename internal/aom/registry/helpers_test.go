// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"
	aom_manifest "u-control/uc-aom/internal/aom/manifest"
	"u-control/uc-aom/internal/pkg/manifest/v0_1"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/mock"
)

type MockDecompressor struct {
	mock.Mock
	UcManifestBlob      []byte
	UcManifestLayerBlob []byte
}

func NewMockDecompressor() *MockDecompressor {
	decompressor := &MockDecompressor{}
	return decompressor
}

func (d *MockDecompressor) Decompress(layerReader io.ReadCloser) (io.ReadCloser, error) {
	args := d.Called()
	return args.Get(0).(io.ReadCloser), args.Error(1)
}

func (d *MockDecompressor) WithFetchContent(wantManifestContent []byte) *MockDecompressor {
	buf := bytes.NewBuffer(wantManifestContent)
	d.On("Decompress", mock.Anything).Return(io.NopCloser(buf), nil)
	return d
}

func (d *MockDecompressor) WithManifestV0_1(wantManifest *v0_1.Root) *MockDecompressor {
	d.UcManifestBlob, _ = json.Marshal(wantManifest)
	content := &aom_manifest.ManifestLayerContent{
		Manifest: d.UcManifestBlob,
	}
	buf := &bytes.Buffer{}
	json.NewEncoder(buf).Encode(content)
	d.UcManifestLayerBlob = buf.Bytes()
	d.On("Decompress", mock.Anything).Return(io.NopCloser(buf), nil)
	return d
}

type mockRepoWithMapDecriptorSize struct {
	mockRepo
}

func (r *mockRepoWithMapDecriptorSize) MapDescriptorSize(desc ocispec.Descriptor) int64 {
	return desc.Size + 1
}

func Test_cumulativeLayerSize(t *testing.T) {
	type args struct {
		repository Repository
		layers     []ocispec.Descriptor
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "Shall cumulative layer size for normal repository",
			args: args{
				repository: &mockRepo{},
				layers: []ocispec.Descriptor{
					{
						Size: 1,
					},
					{
						Size: 2,
					},
				},
			},
			want: 3,
		},
		{
			name: "Shall call MapDescriptorSize to cumulative layer size",
			args: args{
				repository: &mockRepoWithMapDecriptorSize{},
				layers: []ocispec.Descriptor{
					{
						Size: 1,
					},
					{
						Size: 2,
					},
				},
			},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cumulativeLayerSize(tt.args.repository, tt.args.layers); got != tt.want {
				t.Errorf("cumulativeLayerSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
