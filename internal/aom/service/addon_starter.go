// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/aom/network"

	log "github.com/sirupsen/logrus"
)

// AddOnStarter is responsible for starting all the add-on related services.
type AddOnStarter struct {
	localCatalogue           catalogue.LocalAddOnCatalogue
	stackService             docker.StackServiceAPI
	externalNetworkConnector network.ExternalNetworkConnector
}

func NewAddOnStarter(
	localCatalogue catalogue.LocalAddOnCatalogue,
	stackService docker.StackServiceAPI,
	externalNetworkConnector network.ExternalNetworkConnector) *AddOnStarter {
	return &AddOnStarter{localCatalogue: localCatalogue, stackService: stackService, externalNetworkConnector: externalNetworkConnector}
}

// Start all AddOns that are installed.
func (s *AddOnStarter) StartInstalledAddOns() error {
	addOns, err := s.localCatalogue.GetAddOns()
	if err != nil {
		return err
	}

	err = s.externalNetworkConnector.Initialize()
	if err != nil {
		return err
	}

	s.prepareAddOnsBeforeStart(addOns)
	s.start(addOns)
	return nil
}

func (s *AddOnStarter) prepareAddOnsBeforeStart(addOns []*catalogue.CatalogueAddOn) {
	for _, addOn := range addOns {
		if s.externalNetworkConnector.IsConnected(&addOn.Manifest) {
			addOnContainers, err := s.stackService.ListAllStackContainers(addOn.Name)
			if err != nil {
				log.Errorf("stackService.ListAllStackContainers(%s): Unexpected error %v", addOn.Name, err)
				continue
			}
			err = s.externalNetworkConnector.Reconnect(addOnContainers)
			if err != nil {
				log.Errorf("externalNetworkConnector.Reconnect(): Unexpected error %v", err)
				continue
			}
		}
	}
}

func (s *AddOnStarter) start(addOns []*catalogue.CatalogueAddOn) {
	for _, addOn := range addOns {
		err := s.stackService.StartupStackNonBlocking(addOn.Name)
		if err != nil {
			log.Errorf("Error starting App '%s': %v", addOn.Name, err)
		}
	}
}
