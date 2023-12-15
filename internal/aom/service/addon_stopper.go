// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/docker"

	log "github.com/sirupsen/logrus"
)

// StopInstalledAddOns stops all add-ons that are installed.
//
// Normally all add-on containers should be in stopped mode after boot-up.
// However, this might not always be the case e.g. if the uc-aom service is restart manually.
// In this scenario the running container would lead to an error while recreating the external network and reconnecting the container.
func StopInstalledAddOns(localCatalogue catalogue.LocalAddOnCatalogue, stackService docker.StackServiceAPI) error {
	log.Traceln("Stop all add-ons")
	addOns, err := localCatalogue.GetAddOns()
	if err != nil {
		return err
	}

	stop(addOns, stackService)

	return nil
}

func stop(addOns []*catalogue.CatalogueAddOn, stackService docker.StackServiceAPI) {

	for _, addOn := range addOns {
		err := stackService.StopStack(addOn.Name)
		if err != nil {
			log.Errorf("Error stop add-on stack '%s': %v", addOn.Name, err)
		}
	}
}
