// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"testing"
)

func TestUnmarshalManifestVersionFrom(t *testing.T) {
	type args struct {
		manifestRawByteContent []byte
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "shall return the manifest version",
			args: args{
				manifestRawByteContent: func() []byte {
					model := Root{
						ManifestVersion: ValidManifestVersion,
					}
					result, _ := model.ToBytes()
					return result
				}(),
			},
			want: ValidManifestVersion,
		},
		{
			name: "shall return error if empty string is set",
			args: args{
				manifestRawByteContent: func() []byte {
					model := Root{
						ManifestVersion: "",
					}
					result, _ := model.ToBytes()
					return result
				}(),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "shall return error object is empty",
			args: args{
				manifestRawByteContent: func() []byte {
					model := Root{}
					result, _ := model.ToBytes()
					return result
				}(),
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "shall return empty string if not set",
			args: args{
				manifestRawByteContent: []byte{},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalManifestVersionFrom(tt.args.manifestRawByteContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalManifestVersionFrom() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UnmarshalManifestVersionFrom() = %v, want %v", got, tt.want)
			}
		})
	}
}
