// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package fileserver

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/registry"
	"u-control/uc-aom/internal/aom/service"
	"u-control/uc-aom/internal/pkg/manifest"

	log "github.com/sirupsen/logrus"
)

// Interface to have a create add-on routinge
type AddOnCreator interface {
	CreateAddOnRoutine(repository string, version string) error
	UpdateAddOnRoutine(repository string, version string, settings ...*manifest.Setting) error
}

type addOnCreator struct {
	createService        *service.Service
	transactionScheduler *service.TransactionScheduler
}

func (a *addOnCreator) CreateAddOnRoutine(repository string, version string) error {
	tx, err := a.transactionScheduler.CreateTransaction(context.Background(), a.createService)
	if err != nil {
		return err
	}

	defer tx.Rollback()
	err = tx.CreateAddOnRoutine(repository, version)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (a *addOnCreator) UpdateAddOnRoutine(repository string, version string, settings ...*manifest.Setting) error {
	tx, err := a.transactionScheduler.CreateTransaction(context.Background(), a.createService)
	if err != nil {
		return err
	}

	defer tx.Rollback()
	err = tx.ReplaceAddOnRoutine(repository, version, settings...)
	if err != nil {
		return err
	}

	return tx.Commit()
}

type FileServer struct {
	addOnCreator        AddOnCreator
	swUpdateWatcher     SWUpdateWatcher
	dropInAddOnRegistry registry.AddOnRegistry
	localCatalogue      catalogue.LocalAddOnCatalogue
	stopChan            chan struct{}
}

// Create a new instance of FileServer based on the TransactionSchedular
func NewFileServerWithTransactionSchedular(
	transactionScheduler *service.TransactionScheduler,
	swUpdateWatcher SWUpdateWatcher,
	service *service.Service,
	dropInAddOnRegistry registry.AddOnRegistry,
	localCatalogue catalogue.LocalAddOnCatalogue) *FileServer {
	creator := addOnCreator{service, transactionScheduler}

	return NewFileServer(&creator, swUpdateWatcher, dropInAddOnRegistry, localCatalogue)
}

// reate a new instance of FileServer
func NewFileServer(creator AddOnCreator, swUpdateWatcher SWUpdateWatcher, dropInAddOnRegistry registry.AddOnRegistry, localCatalogue catalogue.LocalAddOnCatalogue) *FileServer {
	sc := make(chan struct{})
	return &FileServer{
		addOnCreator:        creator,
		swUpdateWatcher:     swUpdateWatcher,
		dropInAddOnRegistry: dropInAddOnRegistry,
		localCatalogue:      localCatalogue,
		stopChan:            sc,
	}
}

// Start the SWUpdateWatcher
// The SWUpdateWatcher connects to the provided SWUpdate socket path
// Reading a 'SUCCESS' message from the socket start the installation of all drop-in add-ons
func (s *FileServer) StartSWUpdateWatcher() (*sync.WaitGroup, error) {
	log.Trace("StartSWUpdateWatcher()")
	err := s.swUpdateWatcher.Connect()
	if err != nil {
		return nil, err
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		statusChan, errChan := s.swUpdateWatcher.ListenOnStatus()
		for {
			select {
			case status := <-statusChan:
				if status == SUCCESS {
					err := s.InstallAllDropInAddOns()
					if err != nil {
						log.Error(err)
					}
				}
			case <-errChan:
				// lost socket connection, try to reconnect once. If the connection fails
				// again then the offline installation feature would be unavailable.
				retryError := s.swUpdateWatcher.Connect()
				if retryError != nil {
					log.Errorf("Could not reconnect to swupdate socket. Offline installation feature is unavailable for now, to enable it again, try rebooting the device.")
					log.Error(retryError)
					return
				}
			case <-s.stopChan:
				return
			}
		}
	}()
	return wg, nil
}

// stop watching swupdate
func (s *FileServer) StopSWUpdateWatcher() {
	s.stopChan <- struct{}{}
}

// Install all AddOns from the drop-in folder
func (s *FileServer) InstallAllDropInAddOns() error {
	log.Trace("InstallAllDropInAddOns()")
	addOnsRepositoriesInDropInFolder, err := s.dropInAddOnRegistry.Repositories()
	if err != nil {
		return err
	}

	if len(addOnsRepositoriesInDropInFolder) == 0 {
		log.Infof("Nothing to install, drop-in registry is empty.")
		return nil
	}

	for _, repository := range addOnsRepositoriesInDropInFolder {
		tags, err := s.dropInAddOnRegistry.Tags(repository)
		if err != nil {
			log.Errorf("Failed to read tags: %v", err)
			continue
		}
		if len(tags) == 0 {
			log.Infof("No tags found for repository: %s", repository)
			continue
		}

		sort.Sort(manifest.ByAddOnVersion(tags))
		version := tags[len(tags)-1]

		defer s.dropInAddOnRegistry.Delete(repository, version)

		err = s.addOnCreator.CreateAddOnRoutine(repository, version)
		if errors.Is(err, service.ErrorAddOnAlreadyInstalled) {
			err = s.tryToUpdateAddon(repository, version)
		}

		if err != nil {
			// in case of an error we do not handle it because user will not see this error.
			// we just remove the files from the drop-in registry
			log.Errorf("Failed addon routine: %v", err)
		}
	}

	return nil
}

func (s *FileServer) tryToUpdateAddon(repository string, version string) error {
	err := s.isValidUpdate(repository, version)
	if err != nil {
		return err
	}

	return s.addOnCreator.UpdateAddOnRoutine(repository, version)
}

func (s *FileServer) isValidUpdate(repository string, version string) error {
	currentAddOn, err := s.localCatalogue.GetAddOn(repository)
	if err != nil {
		return err
	}

	if !manifest.GreaterThan(version, currentAddOn.Version) {
		err := fmt.Errorf("Requested version %s needs to be greater than the current version %s.", version, currentAddOn.Version)
		return err
	}

	return nil
}
