// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

//go:build dev
// +build dev

package routes_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"testing"
	"time"
	"u-control/uc-aom/internal/aom/dbus"
	"u-control/uc-aom/internal/aom/routes"
	"u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
)

type TestCase struct {
	Description   string
	ReloadMessage string
	ExpectError   bool
}

func init() {
	log.SetLevel(log.TraceLevel)
}

func mock(reloadMessage string, fakeWork time.Duration) dbus.Factory {
	return func(context.Context) (dbus.DBusConnection, error) {
		log.Tracef("Creating Mock with %s %v", reloadMessage, fakeWork)
		return dbus.NewDBusConnectionMock(reloadMessage, fakeWork), nil
	}
}

func setUp(reloadReturnMessage string, fakeWork time.Duration, reloadTimeout ...time.Duration) (*routes.ReverseProxy, *bytes.Buffer) {
	var timeout = 300 * time.Millisecond
	if len(reloadTimeout) > 0 {
		timeout = reloadTimeout[0]
	}

	var buf bytes.Buffer
	writeToBuffer := func(name string, writeContent func(io.Writer) error) error {
		log.Tracef("Created buffer for: '%s'", name)
		return writeContent(&buf)
	}

	deleteFile := func(name string) error {
		log.Tracef("Deleting file: '%s'", name)
		return nil
	}

	createSymbolicLink := func(target string, linkname string) error {
		log.Tracef("ln -s %s %s", target, linkname)
		return nil
	}

	removeSymbolicLink := func(linkname string) error {
		log.Tracef("unlink %s", linkname)
		return nil
	}

	uut := routes.NewReverseProxy(mock(reloadReturnMessage, fakeWork), routes.SITES_AVAILABLE_PATH, routes.SITES_ENABLED_PATH, routes.ROUTES_MAP_AVAILABLE_PATH, routes.ROUTES_MAP_ENABLED_PATH, writeToBuffer, deleteFile, createSymbolicLink, removeSymbolicLink, timeout)
	return uut, &buf
}

func TestReverseProxyRouteCreate(t *testing.T) {
	// Arrange
	uut, buf := setUp("done", 0, time.Second)
	expected := `
# {"com.weidmueller.uc.aom.reverse-proxy.version":"0.2.0","com.weidmueller.uc.aom.version":"0.5.3"}
location /add-on-id/test/addon {

    return 301 $scheme://$host/add-on-id/test/addon/$request_uri;

    location /add-on-id/test/addon/ {
        proxy_pass http://localhost:8189/;

        proxy_http_version 1.1;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        client_max_body_size 0;
        client_body_timeout 30m;
    }
}

# {"com.weidmueller.uc.aom.reverse-proxy.version":"0.2.0","com.weidmueller.uc.aom.version":"0.5.3"}
# TestAddOn Add-on UI
~^/+add-on-id/+test/+addon.*$    test-addon.access;

`
	// Act and assert
	reverseProxyHttpConf := routes.NewReverseProxyHttpConf(
		"add-on-id",
		&manifest.ProxyRoute{From: "http://localhost:8189", To: "/test/addon"},
	)
	reverseProxyMap := &routes.ReverseProxyMap{
		AddOnName:  "add-on-id",
		AddOnTitle: "TestAddOn",
		To:         reverseProxyHttpConf.To,
		Id:         "test-addon",
	}
	err := uut.Create("id", reverseProxyMap, reverseProxyHttpConf)
	if err != nil {
		t.Errorf("Received unexpected error: '%+v'", err)
	}

	actual := fmt.Sprint(buf)
	if actual != expected {
		t.Errorf("Not Equal. Expected '%+v' Actual '%+v'", expected, actual)
	}

}

func TestReverseProxyReloadTimeout(t *testing.T) {
	// Arrange
	uut, _ := setUp("done", time.Second)

	// Act and assert
	err := uut.Delete("")
	if err == nil {
		t.Error("Expecting error none returned.")
	} else {
		if err.Error() != "context deadline exceeded" {
			t.Errorf("Received unexpected error: '%+v'", err)
		}
	}
}

func TestReverseProxyReloadParametized(t *testing.T) {
	testCases := []TestCase{
		{`Reload returns "Done"`, "done", false},
		{`Reload returns "canceled"`, "canceled", true},
		{`Reload returns "timeout"`, "timeout", true},
		{`Reload returns "failed"`, "failed", true},
		{`Reload returns "dependency"`, "dependency", true},
	}

	for _, data := range testCases {
		t.Run(data.Description, func(t *testing.T) {
			// Arrange
			uut, _ := setUp(data.ReloadMessage, 10*time.Millisecond, time.Second)

			// Act and Assert
			err := uut.Delete("")
			actualIsError := err != nil
			if actualIsError != data.ExpectError {
				t.Errorf("Received unexpected error: '%+v'", err)
			}
		})
	}
}

func TestCreatePrefixedRouteFilenameId(t *testing.T) {
	type args struct {
		addOnName string
		routeId   string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Combinde name and id",
			args: args{
				addOnName: "name",
				routeId:   "ui",
			},
			want: "name-ui",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := routes.CreatePrefixedRouteFilenameId(tt.args.addOnName, tt.args.routeId); got != tt.want {
				t.Errorf("CreatePrefixedRouteFilenameId() = %v, want %v", got, tt.want)
			}
		})
	}
}
