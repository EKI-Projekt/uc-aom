// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package service

import (
	"errors"
	"testing"
	"u-control/uc-aom/internal/aom/catalogue"
	"u-control/uc-aom/internal/aom/docker"
	"u-control/uc-aom/internal/pkg/manifest"
)

func newArgs() *args {
	return &args{
		localCatalogue: &catalogue.CatalogueMock{},
		stackService:   &docker.MockStackService{},
	}
}

type args struct {
	localCatalogue *catalogue.CatalogueMock
	stackService   *docker.MockStackService
}

func (a *args) withInstalledAddOns(installedAddOns []*catalogue.CatalogueAddOn) *args {
	a.localCatalogue.On("GetAddOns").Return(installedAddOns, nil)
	for _, addOn := range installedAddOns {
		a.stackService.On("StopStack", addOn.Name).Return(nil).Once()
	}
	return a
}

func (a *args) withGetAddOnsError(err error) *args {
	a.localCatalogue.On("GetAddOns").Return([]*catalogue.CatalogueAddOn{}, err)
	return a
}

func (a *args) assertExpectations(t *testing.T) {
	a.localCatalogue.AssertExpectations(t)
	a.stackService.AssertExpectations(t)
}

func TestStopInstalledAddOns(t *testing.T) {

	tests := []struct {
		name    string
		args    *args
		wantErr bool
	}{
		{
			name: "shall stop installed add-ons",
			args: newArgs().withInstalledAddOns([]*catalogue.CatalogueAddOn{
				{
					Name: "firstAddOn", Manifest: manifest.Root{
						Version:  "1.0.0-1",
						Title:    "addOn title",
						Platform: []string{"ucm"},
					},
					Version: "1.0.0-1",
				},
				{
					Name: "secondAddOn", Manifest: manifest.Root{
						Version:  "1.0.0-1",
						Title:    "addOn title",
						Platform: []string{"ucm"},
					},
					Version: "1.0.0-1",
				},
			}),
			wantErr: false,
		},
		{
			name:    "shall not call stop if no apps are installed",
			args:    newArgs().withInstalledAddOns([]*catalogue.CatalogueAddOn{}),
			wantErr: false,
		},
		{
			name:    "shall return error if GetAddOns return an error",
			args:    newArgs().withGetAddOnsError(errors.New("GetAddOn error")),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := StopInstalledAddOns(tt.args.localCatalogue, tt.args.stackService); (err != nil) != tt.wantErr {
				t.Errorf("StopInstalledAddOns() error = %v, wantErr %v", err, tt.wantErr)
			}
			tt.args.assertExpectations(t)
		})
	}
}
