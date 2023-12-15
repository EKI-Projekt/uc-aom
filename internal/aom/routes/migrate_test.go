// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package routes

import (
	"errors"
	"fmt"
	"testing"
	"u-control/uc-aom/internal/pkg/manifest"

	"github.com/stretchr/testify/mock"
)

func Test_reverseProxyMigrator_Migrate(t *testing.T) {
	type fields struct {
		reverseProxy ReverseProxyCreater
	}
	type args struct {
		name             string
		versionToMigrate string
		title            string
		permissionId     string
		proxyRoute       map[string]*manifest.ProxyRoute
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Shall not migrate for current version",
			args: args{
				versionToMigrate: TemplateVersion,
			},
			fields: fields{
				reverseProxy: func() ReverseProxyCreater {
					return &MockReverseProxyCreater{}
				}(),
			},
		},
		{
			name: "Shall not migrate if version is unkown",
			fields: fields{
				reverseProxy: func() ReverseProxyCreater {
					return &MockReverseProxyCreater{}
				}(),
			},
			args: args{
				versionToMigrate: "unknown",
			},
			wantErr: true,
		},
		{
			name: fmt.Sprintf("Shall migrate from %s", TemplateVersionV0_1_0),
			args: args{
				name:             "app-name",
				versionToMigrate: TemplateVersionV0_1_0,
				title:            "app-title",
				permissionId:     "permission-id",
				proxyRoute: map[string]*manifest.ProxyRoute{
					"ui": {
						From: "/ui",
						To:   "localhost:5000",
					},
				},
			},
			fields: fields{
				reverseProxy: func() ReverseProxyCreater {
					mockProxy := &MockReverseProxyCreater{}

					expectedFilenameId := CreatePrefixedRouteFilenameId("app-name", "ui")
					mockProxy.On("Delete", expectedFilenameId).Return(nil)

					expectedReverseProxyMap := &ReverseProxyMap{
						AddOnName:  "app-name",
						AddOnTitle: "app-title",
						To:         "localhost:5000",
						Id:         "permission-id",
					}

					expectedReverseProxyHttpConf := &ReverseProxyHttpConf{
						AddOnName: "app-name",
						ProxyRoute: &manifest.ProxyRoute{
							From: "/ui",
							To:   "localhost:5000",
						},
					}

					mockProxy.On("Create", expectedFilenameId, expectedReverseProxyMap, expectedReverseProxyHttpConf).Return(nil)

					return mockProxy
				}(),
			},
		},
		{
			name: "Shall not migrate if proxy routes are emptry",
			args: args{
				name:             "app-name",
				versionToMigrate: TemplateVersionV0_1_0,
				title:            "app-title",
				permissionId:     "permission-id",
				proxyRoute:       map[string]*manifest.ProxyRoute{},
			},
			fields: fields{
				reverseProxy: func() ReverseProxyCreater {
					mockProxy := &MockReverseProxyCreater{}
					return mockProxy
				}(),
			},
		},
		{
			name: "Shall return error if migration delete return error.",
			args: args{
				name:             "app-name",
				versionToMigrate: TemplateVersionV0_1_0,
				title:            "app-title",
				permissionId:     "permission-id",
				proxyRoute: map[string]*manifest.ProxyRoute{
					"ui": {
						From: "/ui",
						To:   "localhost:5000",
					},
				},
			},
			fields: fields{
				reverseProxy: func() ReverseProxyCreater {
					mockProxy := &MockReverseProxyCreater{}

					expectedFilenameId := CreatePrefixedRouteFilenameId("app-name", "ui")
					mockProxy.On("Delete", expectedFilenameId).Return(errors.New("error"))

					return mockProxy
				}(),
			},
			wantErr: true,
		},
		{
			name: "Shall return error if migration create return error.",
			args: args{
				name:             "app-name",
				versionToMigrate: TemplateVersionV0_1_0,
				title:            "app-title",
				permissionId:     "permission-id",
				proxyRoute: map[string]*manifest.ProxyRoute{
					"ui": {
						From: "/ui",
						To:   "localhost:5000",
					},
				},
			},
			fields: fields{
				reverseProxy: func() ReverseProxyCreater {
					mockProxy := &MockReverseProxyCreater{}

					expectedFilenameId := CreatePrefixedRouteFilenameId("app-name", "ui")
					mockProxy.On("Delete", expectedFilenameId).Return(nil)

					expectedReverseProxyMap := &ReverseProxyMap{
						AddOnName:  "app-name",
						AddOnTitle: "app-title",
						To:         "localhost:5000",
						Id:         "permission-id",
					}

					expectedReverseProxyHttpConf := &ReverseProxyHttpConf{
						AddOnName: "app-name",
						ProxyRoute: &manifest.ProxyRoute{
							From: "/ui",
							To:   "localhost:5000",
						},
					}

					mockProxy.On("Create", expectedFilenameId, expectedReverseProxyMap, expectedReverseProxyHttpConf).Return(errors.New("error"))

					return mockProxy
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &reverseProxyMigrator{
				reverseProxy: tt.fields.reverseProxy,
			}
			if err := m.Migrate(tt.args.name, tt.args.versionToMigrate, tt.args.title, tt.args.permissionId, tt.args.proxyRoute); (err != nil) != tt.wantErr {
				t.Errorf("reverseProxyMigrator.Migrate() error = %v, wantErr %v", err, tt.wantErr)
			}

			mock.AssertExpectationsForObjects(t, tt.fields.reverseProxy)
		})
	}
}
