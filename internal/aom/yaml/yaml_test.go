// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package yaml_test

import (
	"testing"
	"u-control/uc-aom/internal/aom/yaml"
)

var compose = `
version: '3'

services:
  roach1:
    image: cockroachdb/cockroach:latest
    command: "start --insecure"
    deploy:
      replicas: 1
    ports:
       - 8080:1234
       - 9000:4567
       - 7000

  roachN:
    image: cockroachdb/cockroach:latest
    command: "start --insecure --join=roach1,roachN"
    deploy:
      mode: global
`

func TestReadingPorts(t *testing.T) {
	services, err := yaml.GetComposeServicesPublicPortsFrom(compose)
	if err != nil {
		t.Errorf("error reading compose services %v", err)
	}

	t.Log(len(services))

	if len(services) != 1 {
		t.Errorf("expect only 1 service with ports")
	}

	if len(services[0].Ports) != 2 {
		t.Errorf("expect 2 ports for this service.")
		t.Log(len(services[0].Ports))
	}

	s := services[0]
	if s.Ports[0] != "8080" || s.Ports[1] != "9000" {
		t.Errorf("wrong port values")
	}
}
