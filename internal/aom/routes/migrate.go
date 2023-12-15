// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package routes

import (
	"fmt"
	"u-control/uc-aom/internal/pkg/manifest"
)

type ReverseProxyMigrator interface {
	Migrate(name string, versionToMigrate string, title string, permissionId string, proxyRoute map[string]*manifest.ProxyRoute) error
}

func NewReverseProxyMigrator(reverseProxy ReverseProxyCreater) ReverseProxyMigrator {
	return &reverseProxyMigrator{reverseProxy: reverseProxy}

}

type reverseProxyMigrator struct {
	reverseProxy ReverseProxyCreater
}

func (m *reverseProxyMigrator) Migrate(name string, versionToMigrate string, title string, permissionId string, proxyRoute map[string]*manifest.ProxyRoute) error {

	switch versionToMigrate {
	case TemplateVersionV0_1_0:
		err := m.migrateV0_1_0(name, title, permissionId, proxyRoute)
		if err != nil {
			return err
		}

	case TemplateVersion:
		// nothing to do if version is the current template version

	default:
		return fmt.Errorf("Template version %s is unknown", versionToMigrate)
	}

	return nil
}

func (m *reverseProxyMigrator) migrateV0_1_0(name string, title string, permissionId string, proxyRoute map[string]*manifest.ProxyRoute) error {
	for id := range proxyRoute {
		prefixedId := CreatePrefixedRouteFilenameId(name, id)
		err := m.reverseProxy.Delete(prefixedId)
		if err != nil {
			return err
		}
	}
	for id, location := range proxyRoute {
		reverseProxyHttpConf := NewReverseProxyHttpConf(name, location)
		reverseProxyMap := &ReverseProxyMap{AddOnName: name, AddOnTitle: title, To: location.To, Id: permissionId}
		prefixedId := CreatePrefixedRouteFilenameId(name, id)
		err := m.reverseProxy.Create(prefixedId, reverseProxyMap, reverseProxyHttpConf)
		if err != nil {
			return err
		}
	}
	return nil
}
