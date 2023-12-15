// Copyright 2022 - 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"
	"u-control/uc-aom/internal/aom/config"
	"u-control/uc-aom/internal/aom/dbus"
	"u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
)

const done = "done"
const reverseProxyConfigTemplate = `
# {{.Metadata}}
location /{{.AddOnName}}{{.To}} {

    return 301 $scheme://$host/{{.AddOnName}}{{.To}}/$request_uri;

    location /{{.AddOnName}}{{.To}}/ {
        proxy_pass {{.From}}/;

        proxy_http_version 1.1;

        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        client_max_body_size 0;
        client_body_timeout 30m;
    }
}
`
const reverseProxyMapTemplate = `
# {{ .Metadata }}
# {{.AddOnTitle}} Add-on UI
~^/+{{.AddOnName}}{{.To}}.*$    {{.Id}}.access;

`

// Create a prefixed route filename id which can be used for create and delete reverse proxy routes.
func CreatePrefixedRouteFilenameId(addOnName string, routeId string) string {
	prefixedId := fmt.Sprintf("%s-%s", addOnName, routeId)
	return prefixedId
}

type ReverseProxyCreater interface {
	Create(filenameId string, reverseProxyMap *ReverseProxyMap, reverseProxyHttpConf *ReverseProxyHttpConf) error
	Delete(filenameId string) error
}

type ReverseProxy struct {
	factory                dbus.Factory
	configTemplate         *template.Template
	mapTemplate            *template.Template
	sitesAvailablePath     string
	sitesEnabledPath       string
	routesMapAvailablePath string
	routesMapEnabledPath   string
	writeToFile            WriteToFileCmd
	deleteFile             DeleteFileCmd
	createSymbolicLink     CreateSymbolicLinkCmd
	removeSymbolicLink     DeleteFileCmd
	reloadTimeout          time.Duration
}

type ReverseProxyMap struct {
	AddOnName  string
	AddOnTitle string
	To         string
	Id         string
}

type ReverseProxyHttpConf struct {
	AddOnName string
	*manifest.ProxyRoute
}

type metadata map[string]string

func newMetaData() metadata {
	return metadata{
		config.UcAomVersionLabel: config.UcAomVersion,
		TemplateVersionLabel:     TemplateVersion,
	}

}

func (m *metadata) ToString() string {
	jsonData, _ := json.Marshal(m)
	return string(jsonData)
}

type reverseProxyMapWithMetadata struct {
	*ReverseProxyMap
	metadata
}

func (r *reverseProxyMapWithMetadata) Metadata() string {
	return r.metadata.ToString()
}

type ReverseProxyHttpConfWithMetadata struct {
	*ReverseProxyHttpConf
	metadata
}

func (r *ReverseProxyHttpConfWithMetadata) Metadata() string {
	return r.metadata.ToString()
}

// Given a filepath, an io.Writer instance will be created and passed
// to the function which is responsible for creating and saving the file's contents.
type WriteToFileCmd func(filepath string, factory func(io.Writer) error) error

// Deletes the file at the given filepath.
type DeleteFileCmd func(filepath string) error

// Creates a symbolic link with the given linkname which points to target.
type CreateSymbolicLinkCmd func(target string, linkname string) error

func (r *ReverseProxy) getSitesAvailableFilepath(id string) string {
	return getFilepath(r.sitesAvailablePath, id)
}

func (r *ReverseProxy) getSitesEnabledFilepath(id string) string {
	return getFilepath(r.sitesEnabledPath, id)
}

func getFilepath(parent string, name string) string {
	return fmt.Sprintf("%s/%s.http.conf", parent, name)
}

func (r *ReverseProxy) getRoutesMapAvailableFilepath(id string) string {
	return getMapFilepath(r.routesMapAvailablePath, id)
}

func (r *ReverseProxy) getRoutesMapEnabledFilepath(id string) string {
	return getMapFilepath(r.routesMapEnabledPath, id)
}

func getMapFilepath(parent string, name string) string {
	return fmt.Sprintf("%s/%s-proxy.map", parent, name)
}

func NewReverseProxyHttpConf(name string, location *manifest.ProxyRoute) *ReverseProxyHttpConf {
	return &ReverseProxyHttpConf{name, location}
}

func NewReverseProxy(
	factory dbus.Factory,
	sitesAvailablePath string,
	sitesEnabledPath string,
	routesMapAvailablePath string,
	routesMapEnabledPath string,
	writeToFile WriteToFileCmd,
	deleteFile DeleteFileCmd,
	createSymbolicLink CreateSymbolicLinkCmd,
	removeSymbolicLink DeleteFileCmd,
	timeout ...time.Duration) *ReverseProxy {

	configTmpl, err := template.New("ProxyRoute").Parse(reverseProxyConfigTemplate)
	if err != nil {
		panic(err)
	}

	mapTmpl, err := template.New("ProxyRouteMap").Parse(reverseProxyMapTemplate)
	if err != nil {
		panic(err)
	}

	var reloadTimeout = 5 * time.Second
	if len(timeout) > 0 {
		reloadTimeout = timeout[0]
	}

	return &ReverseProxy{
		factory:                factory,
		configTemplate:         configTmpl,
		mapTemplate:            mapTmpl,
		sitesAvailablePath:     sitesAvailablePath,
		sitesEnabledPath:       sitesEnabledPath,
		routesMapAvailablePath: routesMapAvailablePath,
		routesMapEnabledPath:   routesMapEnabledPath,
		writeToFile:            writeToFile,
		deleteFile:             deleteFile,
		createSymbolicLink:     createSymbolicLink,
		removeSymbolicLink:     removeSymbolicLink,
		reloadTimeout:          reloadTimeout}
}

func (r *ReverseProxy) Create(filenameId string, reverseProxyMap *ReverseProxyMap, reverseProxyHttpConf *ReverseProxyHttpConf) error {
	log.Tracef("ReverseProxy/Create")
	configFilepath := r.getSitesAvailableFilepath(filenameId)
	mapFilepath := r.getRoutesMapAvailableFilepath(filenameId)

	executeConfigTemplate := func(writer io.Writer) error {
		reverseProxyHttpConfWithMetadata := ReverseProxyHttpConfWithMetadata{
			ReverseProxyHttpConf: reverseProxyHttpConf,
			metadata:             newMetaData(),
		}
		err := r.configTemplate.Execute(writer, &reverseProxyHttpConfWithMetadata)
		return err
	}

	executeMapTemplate := func(writer io.Writer) error {
		reverseProxyMap.To = strings.ReplaceAll(reverseProxyMap.To, "/", "/+")
		reverseProxyMapWithMetadata := reverseProxyMapWithMetadata{
			ReverseProxyMap: reverseProxyMap,
			metadata:        newMetaData(),
		}
		err := r.mapTemplate.Execute(writer, &reverseProxyMapWithMetadata)
		return err
	}

	err := r.writeToFile(configFilepath, executeConfigTemplate)
	if err != nil {
		return err
	}

	err = r.createSymbolicLink(configFilepath, r.getSitesEnabledFilepath(filenameId))
	if err != nil {
		return err
	}

	err = r.writeToFile(mapFilepath, executeMapTemplate)
	if err != nil {
		return err
	}

	err = r.createSymbolicLink(mapFilepath, r.getRoutesMapEnabledFilepath(filenameId))
	if err != nil {
		return err
	}

	return r.reload()
}

func (r *ReverseProxy) Delete(id string) error {
	log.Tracef("ReverseProxy/Delete")
	path := r.getSitesEnabledFilepath(id)
	err := r.removeSymbolicLink(path)
	if err != nil {
		return err
	}

	path = r.getSitesAvailableFilepath(id)
	err = r.deleteFile(path)
	if err != nil {
		return err
	}

	path = r.getRoutesMapEnabledFilepath(id)
	err = r.removeSymbolicLink(path)
	if err != nil {
		return err
	}

	path = r.getRoutesMapAvailableFilepath(id)
	err = r.deleteFile(path)
	if err != nil {
		return err
	}

	return r.reload()
}

func (r *ReverseProxy) reload() error {
	log.Tracef("ReverseProxy/Reload")
	ctx, cancel := context.WithTimeout(context.Background(), r.reloadTimeout)
	defer cancel()

	sdc, err := r.factory(ctx)
	if err != nil {
		return err
	}
	defer sdc.Close()

	c := make(chan string)
	err = sdc.ReloadUnitContext(ctx, c)
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		// Timeout reached.
		return ctx.Err()
	case result := <-c:
		log.Debugf("Read from channel: %+v", result)
		// result string, is one of done, canceled, timeout, failed, dependency, skipped
		if result != done {
			return errors.New(fmt.Sprintf("Restart of nginx.service returned: %s", result))
		}
		return nil
	}
}
