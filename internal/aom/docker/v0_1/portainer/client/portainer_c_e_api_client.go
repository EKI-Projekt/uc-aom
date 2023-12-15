package client

import (
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer/client/auth"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer/client/stacks"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer/client/status"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

const (
	// DefaultBasePath is the default BasePath
	DefaultBasePath string = "/api"
)

func New(transport runtime.ClientTransport) *PortainerCEAPI {
	formats := strfmt.Default

	cli := new(PortainerCEAPI)
	cli.Transport = transport
	cli.Auth = auth.New(transport, formats)
	cli.Stacks = stacks.New(transport, formats)
	cli.Status = status.New(transport, formats)

	return cli
}

type PortainerCEAPI struct {
	Auth      auth.ClientService
	Stacks    stacks.ClientService
	Status    status.ClientService
	Transport runtime.ClientTransport
}
