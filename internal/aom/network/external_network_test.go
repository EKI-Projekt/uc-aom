// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"errors"
	"testing"
	"time"
	model "u-control/uc-aom/internal/pkg/manifest"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	networktypes "github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var errorNotImplemented = errors.New("Not Implemented")

type mockDockerApiClient struct {
	mock.Mock
}

func (m *mockDockerApiClient) NetworkConnect(ctx context.Context, network, container string, config *networktypes.EndpointSettings) error {
	args := m.Called(ctx, network, container, config)
	return args.Error(0)
}

func (m *mockDockerApiClient) NetworkCreate(ctx context.Context, name string, options types.NetworkCreate) (types.NetworkCreateResponse, error) {
	args := m.Called(ctx, name, options)
	return args.Get(0).(types.NetworkCreateResponse), args.Error(1)
}

func (m *mockDockerApiClient) NetworkDisconnect(ctx context.Context, network, container string, force bool) error {
	args := m.Called(ctx, network, container, force)
	return args.Error(0)
}

func (m *mockDockerApiClient) NetworkInspect(ctx context.Context, network string, options types.NetworkInspectOptions) (types.NetworkResource, error) {
	args := m.Called(ctx, network, options)
	return args.Get(0).(types.NetworkResource), args.Error(1)
}

func (m *mockDockerApiClient) NetworkInspectWithRaw(ctx context.Context, network string, options types.NetworkInspectOptions) (types.NetworkResource, []byte, error) {
	return types.NetworkResource{}, []byte{}, errorNotImplemented
}

func (m *mockDockerApiClient) NetworkList(ctx context.Context, options types.NetworkListOptions) ([]types.NetworkResource, error) {
	return []types.NetworkResource{}, errorNotImplemented
}

func (m *mockDockerApiClient) NetworkRemove(ctx context.Context, network string) error {
	args := m.Called(ctx, network)
	return args.Error(0)
}

func (m *mockDockerApiClient) NetworksPrune(ctx context.Context, pruneFilter filters.Args) (types.NetworksPruneReport, error) {
	return types.NetworksPruneReport{}, errorNotImplemented
}

func (m *mockDockerApiClient) ContainerStop(ctx context.Context, container string, timeout *time.Duration) error {
	args := m.Called(ctx, container, timeout)
	return args.Error(0)
}

func (m *mockDockerApiClient) ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error) {
	args := m.Called(ctx, container)
	return args.Get(0).(types.ContainerJSON), args.Error(1)
}

func Test_internalBridgeNetworkConnector_IsConnected(t *testing.T) {
	type args struct {
		manifest *model.Root
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "shall return true if connected",
			args: args{
				manifest: &model.Root{
					ManifestVersion: model.ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Vendor: &model.Vendor{
						Name: "name",
						Url:  "abcd@vendor.com",
					},
					Services: map[string]*model.Service{
						"testservice": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"networks": []string{"external"},
							},
						},
					},
					Environments: map[string]*model.Environment{
						"testservice": model.NewEnvironment("docker-compose").WithNetworks(
							map[string]map[string]interface{}{"external": {"external": true, "name": model.InternalAddOnNetworkName}},
						),
					},
				},
			},
			want: true,
		},
		{
			name: "shall return false if not connected",
			args: args{
				manifest: &model.Root{
					ManifestVersion: model.ValidManifestVersion,
					Version:         "1.0.0-1",
					Title:           "title",
					Description:     "Description",
					Logo:            "logo.png",
					Platform:        []string{"ucg"},
					Vendor: &model.Vendor{
						Name: "name",
						Url:  "abcd@vendor.com",
					},
					Services: map[string]*model.Service{
						"testservice": {
							Type: "docker-compose",
							Config: map[string]interface{}{
								"networks": []string{"internal"},
							},
						},
					},
					Environments: map[string]*model.Environment{
						"testservice": model.NewEnvironment("docker-compose").WithNetworks(
							map[string]map[string]interface{}{"internal": {}},
						),
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &internalBridgeNetworkConnector{}
			if got := s.IsConnected(tt.args.manifest); got != tt.want {
				t.Errorf("internalBridgeNetworkConnector.IsConnected() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_internalBridgeNetworkConnector_Reconnect(t *testing.T) {
	type args struct {
		connectedContainers    []types.Container
		notConnectedContainers []types.Container
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "shall reconnect container if it is connected to internal-bridge",
			args: args{
				connectedContainers: []types.Container{{
					ID: "1",
					NetworkSettings: &types.SummaryNetworkSettings{
						Networks: map[string]*networktypes.EndpointSettings{
							model.InternalAddOnNetworkName: {},
						},
					},
				}},
			},
		},
		{
			name: "shall reconnect container if it is connected to internal-bridge with aliases",
			args: args{
				connectedContainers: []types.Container{{
					ID: "1",
					NetworkSettings: &types.SummaryNetworkSettings{
						Networks: map[string]*networktypes.EndpointSettings{
							model.InternalAddOnNetworkName: {
								Aliases: []string{"container-name"},
							},
						},
					},
				}},
			},
		},
		{
			name: "shall not reconnect container if it is not connected to internal-bridge",
			args: args{
				notConnectedContainers: []types.Container{{
					ID: "1",
				}},
			},
		},
		{
			name: "shall not reconnect container if it is not connected to internal-bridge (empty NetworkSettings)",
			args: args{
				notConnectedContainers: []types.Container{{
					ID:              "1",
					NetworkSettings: &types.SummaryNetworkSettings{},
				}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			mockDockerApiClient := &mockDockerApiClient{}
			assignPreexistingNetworkCallWithConnectedContainers(mockDockerApiClient, tt.args.connectedContainers...)
			networkCreateResponse := assignCreateAndRemoveNetworkCalls(mockDockerApiClient)
			assignReconnectContainerCalls(mockDockerApiClient, tt.args.connectedContainers, networkCreateResponse)
			assignStopContainerCalls(mockDockerApiClient, tt.args.connectedContainers)
			uut := &internalBridgeNetworkConnector{
				dockerApiClient:                mockDockerApiClient,
				newInitializedInternalBridgeId: "",
			}

			err := uut.Initialize()
			assert.NoError(t, err)

			// Act
			containers := append(tt.args.connectedContainers, tt.args.notConnectedContainers...)
			if err := uut.Reconnect(containers); err != nil {
				t.Errorf("internalBridgeNetworkConnector.Reconnect() unexpected error = %v", err)
			}

			// assert
			mockDockerApiClient.AssertExpectations(t)
		})
	}
}

func TestShallNotRemoveOnInitialStart(t *testing.T) {
	// Arrange
	connectedContainers :=
		[]types.Container{{
			ID: "1",
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*networktypes.EndpointSettings{
					model.InternalAddOnNetworkName: {},
				},
			},
		}}

	mockDockerApiClient := &mockDockerApiClient{}
	networkCreateResponse := assignCreateNetworkCalls(mockDockerApiClient)
	assignNoneExistingNetworkCall(mockDockerApiClient)
	assignReconnectContainerCalls(mockDockerApiClient, connectedContainers, networkCreateResponse)

	uut := &internalBridgeNetworkConnector{
		dockerApiClient:                mockDockerApiClient,
		newInitializedInternalBridgeId: "",
	}
	err := uut.Initialize()
	assert.NoError(t, err)

	// act
	if err := uut.Reconnect(connectedContainers); err != nil {
		t.Errorf("internalBridgeNetworkConnector.Reconnect() unexpected error = %v", err)
	}

	// assert
	mockDockerApiClient.AssertExpectations(t)
}

func TestShallReturnErrorIfErrorIsUnknownWhileNetworkInspect(t *testing.T) {
	// Arrange
	mockDockerApiClient := &mockDockerApiClient{}
	wantErr := errors.New("Unknown network inspect error")
	mockDockerApiClient.On("NetworkInspect", context.Background(), model.InternalAddOnNetworkName, types.NetworkInspectOptions{}).Return(types.NetworkResource{}, wantErr).Once()
	uut := &internalBridgeNetworkConnector{
		dockerApiClient:                mockDockerApiClient,
		newInitializedInternalBridgeId: "",
	}

	// Act
	gotErr := uut.Initialize()

	// Assert
	assert.ErrorIs(t, gotErr, wantErr)
	mockDockerApiClient.AssertExpectations(t)
}
func TestShallReturnErrorOnContainerStop(t *testing.T) {
	// Arrange
	mockDockerApiClient := &mockDockerApiClient{}
	connectedContainer := types.Container{
		ID: "12345",
	}

	assignPreexistingNetworkCallWithConnectedContainers(mockDockerApiClient, connectedContainer)

	wantErr := errors.New("dummyError")
	timeout := 1 * time.Second
	mockDockerApiClient.On("ContainerStop", context.Background(), connectedContainer.ID, &timeout).Return(wantErr)

	uut := &internalBridgeNetworkConnector{
		dockerApiClient:                mockDockerApiClient,
		newInitializedInternalBridgeId: "",
	}

	// Act
	gotErr := uut.Initialize()

	// Assert
	assert.ErrorIs(t, gotErr, wantErr)
	mockDockerApiClient.AssertExpectations(t)
}

func assignPreexistingNetworkCallWithConnectedContainers(mock *mockDockerApiClient, connectedContainers ...types.Container) {
	containers := map[string]types.EndpointResource{}

	for _, c := range connectedContainers {
		containers[c.ID] = types.EndpointResource{}
	}

	networkInspectResult := types.NetworkResource{
		Containers: containers,
	}
	mock.On("NetworkInspect", context.Background(), model.InternalAddOnNetworkName, types.NetworkInspectOptions{}).Return(networkInspectResult, nil).Once()
}

func assignNoneExistingNetworkCall(mock *mockDockerApiClient) {
	err := newNetworkNotFoundDockerdError()
	mock.On("NetworkInspect", context.Background(), model.InternalAddOnNetworkName, types.NetworkInspectOptions{}).Return(types.NetworkResource{}, err).Once()
}

func assignCreateAndRemoveNetworkCalls(mock *mockDockerApiClient) types.NetworkCreateResponse {

	networkCreateResponse := assignCreateNetworkCalls(mock)
	mock.On("NetworkRemove", context.Background(), model.InternalAddOnNetworkName).Return(nil).Once()
	return networkCreateResponse
}

func assignCreateNetworkCalls(mock *mockDockerApiClient) types.NetworkCreateResponse {
	createOptions := types.NetworkCreate{
		Options: map[string]string{
			"com.docker.network.bridge.default_bridge":       "false",
			"com.docker.network.bridge.enable_icc":           "true",
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.bridge.host_binding_ipv4":    "0.0.0.0",
			"com.docker.network.bridge.name":                 "docker1",
		},
	}

	networkCreateResponse := types.NetworkCreateResponse{
		ID: "abdcd",
	}

	mock.On("NetworkCreate", context.Background(), model.InternalAddOnNetworkName, createOptions).Return(networkCreateResponse, nil).Once()
	return networkCreateResponse
}

func assignReconnectContainerCalls(mockDockerApiClient *mockDockerApiClient, connectedContainers []types.Container, networkCreateResponse types.NetworkCreateResponse) {
	for _, c := range connectedContainers {

		aliases := []string{}

		if networkSettings, hasSettings := c.NetworkSettings.Networks[model.InternalAddOnNetworkName]; hasSettings {
			aliases = append(aliases, networkSettings.Aliases...)
		}

		mockDockerApiClient.On("ContainerInspect", context.Background(), c.ID).Return(types.ContainerJSON{
			NetworkSettings: &types.NetworkSettings{
				Networks: c.NetworkSettings.Networks,
			},
		}, nil)
		mockDockerApiClient.On("NetworkConnect", context.Background(), networkCreateResponse.ID, c.ID, &networktypes.EndpointSettings{
			Aliases: aliases,
		}).Return(nil)
	}
}

func assignStopContainerCalls(mockContainerClient *mockDockerApiClient, connectedContainers []types.Container) {
	for _, c := range connectedContainers {
		timeout := 1 * time.Second
		mockContainerClient.On("ContainerStop", context.Background(), c.ID, &timeout).Return(nil)
	}
}

func createMockNetworkAPIClientWithError(connectedContainers []types.Container) *mockDockerApiClient {
	mock := &mockDockerApiClient{}
	c := connectedContainers[0]

	networkInspectResult := types.NetworkResource{
		Containers: map[string]types.EndpointResource{
			c.ID: {},
		},
	}
	mock.On("NetworkInspect", context.Background(), model.InternalAddOnNetworkName, types.NetworkInspectOptions{}).Return(networkInspectResult, nil).Once()

	err := errors.New("dummyError")
	mock.On("NetworkDisconnect", context.Background(), model.InternalAddOnNetworkName, c.ID, true).Return(err)

	return mock
}
