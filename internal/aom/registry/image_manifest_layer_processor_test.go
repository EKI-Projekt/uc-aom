// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package registry

import (
	"testing"
	"u-control/uc-aom/internal/pkg/config"
	"u-control/uc-aom/internal/pkg/manifest"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
)

type args struct {
	desc *ocispec.Descriptor
}

var testCases = []struct {
	name string
	p    *ucImageLayerProcessor
	args args
	want bool
}{
	{
		name: "Shall return true for UcImageLayerMediaType mediatype and with right annotion schema version",
		args: args{
			desc: &ocispec.Descriptor{
				MediaType:   config.UcImageLayerMediaType,
				Annotations: map[string]string{config.UcImageLayerAnnotationSchemaVersion: manifest.ValidManifestVersion},
			},
		},
		want: true,
	},
	{
		name: "Shall return false for UcImageLayerMediaType mediatype but with wrong annotion schema verion",
		args: args{
			desc: &ocispec.Descriptor{
				MediaType:   config.UcImageLayerMediaType,
				Annotations: map[string]string{config.UcImageLayerAnnotationSchemaVersion: manifest.ValidManifestVersion + "1"},
			},
		},
		want: false,
	},
	{
		name: "Shall return false for UcImageLayerMediaType mediatype but empty annotions",
		args: args{
			desc: &ocispec.Descriptor{
				MediaType:   config.UcImageLayerMediaType,
				Annotations: map[string]string{},
			},
		},
		want: false,
	},
	{
		name: "Shall return false for Unkown mediatype",
		args: args{
			desc: &ocispec.Descriptor{
				MediaType: "UNKOWN",
			},
		},
		want: false,
	},
}

func TestUcImageLayerProcessor_Apply(t *testing.T) {
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			p := NewUcImageLayerProcessor(nil)
			if got := p.Filter(tt.args.desc); got != tt.want {
				t.Errorf("ucImageLayerProcessor.Apply() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAllExceptUcImageLayerProcessor_Apply(t *testing.T) {
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			p := NewAllExceptUcImageLayerProcessor(nil)
			// We are testing the antagonist.
			want := !tt.want

			if got := p.Filter(tt.args.desc); got != want {
				t.Errorf("allExceptUcImageLayerProcessor.Apply() = %v, want %v", got, want)
			}
		})
	}
}
