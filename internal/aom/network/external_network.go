// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"

	log "github.com/sirupsen/logrus"
)

// The network name is used by docker to create an internal network device.
// This can be used to post process additional settings on the network device like setting iptable entries.
const internalBridgeNetworkName = "docker1"

type networkNotFoundDockerdError struct {
}

func newNetworkNotFoundDockerdError() error {
	return &networkNotFoundDockerdError{}
}

func (e *networkNotFoundDockerdError) Error() string {
	return fmt.Sprintf("No such network: %s", model.InternalAddOnNetworkName)
}

func (e *networkNotFoundDockerdError) Is(target error) bool {
	return strings.Contains(target.Error(), e.Error())
}

// ExternalNetworkConnector manage connection with a specific external network
type ExternalNetworkConnector interface {
	// Initialize the external network connector
	Initialize() error

	// Return true if one of the add-on services uses the external network, otherwise false
	IsConnected(manifest *model.Root) bool

	// Reconnect all docker container to the external network, if they are connected to it.
	Reconnect(containers []types.Container) error
}

// dockerApiClient defines selected API methodes for docker
type dockerApiClient interface {
	client.NetworkAPIClient
	ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error)
	ContainerStop(ctx context.Context, container string, timeout *time.Duration) error
}

// Creates a new network connector instance of the internal-bridge connector
func NewInternalBridgeNetworkConnector(dockerApiClient client.APIClient) ExternalNetworkConnector {
	return &internalBridgeNetworkConnector{
		dockerApiClient:                dockerApiClient,
		newInitializedInternalBridgeId: "",
	}
}

type internalBridgeNetworkConnector struct {
	dockerApiClient                dockerApiClient
	newInitializedInternalBridgeId string
}

func (s *internalBridgeNetworkConnector) Initialize() error {
	if !s.isInitialized() {
		err := s.recreateInternalBridgeNetwork()
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *internalBridgeNetworkConnector) IsConnected(manifest *model.Root) bool {
	for _, e := range manifest.Environments {
		for _, networkConfig := range e.Config.Networks {
			if value, ok := networkConfig["name"]; ok {
				if value == model.InternalAddOnNetworkName {
					return true
				}
			}
		}
	}

	return false
}

func (s *internalBridgeNetworkConnector) Reconnect(containers []types.Container) error {
	for _, c := range containers {
		if s.isConnectedToInternalBridge(c) {
			err := s.reconnectContainer(c.ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *internalBridgeNetworkConnector) isInitialized() bool {
	return s.newInitializedInternalBridgeId != ""
}

func (s *internalBridgeNetworkConnector) isConnectedToInternalBridge(container types.Container) bool {
	if container.NetworkSettings == nil {
		return false
	}
	_, isConnected := container.NetworkSettings.Networks[model.InternalAddOnNetworkName]
	return isConnected
}

func (s *internalBridgeNetworkConnector) recreateInternalBridgeNetwork() error {
	log.Trace("recreateInternalBridgeNetwork")

	err := s.removeInternalBridgeNetworkIfExist()
	if err != nil {
		return err
	}

	s.newInitializedInternalBridgeId, err = s.createNewInternalBridgeNetwork()
	if err != nil {
		return err
	}

	log.Tracef("recreateInternalBridgeNetwork with ID %s", s.newInitializedInternalBridgeId)
	return nil
}

func (s *internalBridgeNetworkConnector) removeInternalBridgeNetworkIfExist() error {
	log.Trace("removeInternalBridgeNetworkIfExist")
	ctx := context.Background()
	networkResource, err := s.dockerApiClient.NetworkInspect(ctx, model.InternalAddOnNetworkName, types.NetworkInspectOptions{})
	if err != nil {
		// If we have a network not found error, we do not have to disconnect and remove the internal-bridge network.
		// This is the case when the device first boots up where the network has not yet been created.
		if errors.Is(newNetworkNotFoundDockerdError(), err) {
			return nil
		}
		return err
	}

	// NetworkResource contains only connected and started containers
	// Stopped containers are not included in this data structure, even if they are connected.
	// We need to stop all connected containers so we can safely remove the network.
	connectedAndStartedContainers := networkResource.Containers
	for containerID := range connectedAndStartedContainers {
		timeout := 1 * time.Second
		err := s.dockerApiClient.ContainerStop(ctx, containerID, &timeout)
		if err != nil {
			return err
		}
	}

	return s.dockerApiClient.NetworkRemove(ctx, model.InternalAddOnNetworkName)
}

func (s *internalBridgeNetworkConnector) createNewInternalBridgeNetwork() (string, error) {
	createOptions := types.NetworkCreate{
		Options: map[string]string{
			"com.docker.network.bridge.default_bridge":       "false",
			"com.docker.network.bridge.enable_icc":           "true",
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.bridge.host_binding_ipv4":    "0.0.0.0",
			"com.docker.network.bridge.name":                 internalBridgeNetworkName,
		},
	}

	ctx := context.Background()
	response, err := s.dockerApiClient.NetworkCreate(ctx, model.InternalAddOnNetworkName, createOptions)
	if err != nil {
		return "", err
	}
	return response.ID, nil
}

func (s *internalBridgeNetworkConnector) reconnectContainer(containerID string) error {

	aliases, err := s.getPreviousContainerNetworkAliases(containerID)
	if err != nil {
		return err
	}

	log.Tracef("reconnectContainer with ID %s and aliases %v", containerID, aliases)

	ctx := context.Background()
	return s.dockerApiClient.NetworkConnect(ctx, s.newInitializedInternalBridgeId, containerID, &network.EndpointSettings{
		Aliases: aliases,
	})
}

func (s *internalBridgeNetworkConnector) getPreviousContainerNetworkAliases(containerID string) ([]string, error) {
	containerJSON, err := s.dockerApiClient.ContainerInspect(context.Background(), containerID)
	if err != nil {
		return make([]string, 0), err
	}

	if containerJSON.NetworkSettings == nil {
		return make([]string, 0), nil
	}

	if previousInternalBridgeSettings, hasSettings := containerJSON.NetworkSettings.Networks[model.InternalAddOnNetworkName]; hasSettings {
		if previousInternalBridgeSettings.Aliases != nil {
			return previousInternalBridgeSettings.Aliases, nil
		}
	}

	return make([]string, 0), nil
}
