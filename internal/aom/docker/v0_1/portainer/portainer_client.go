package portainer

import (
	"errors"
	"fmt"
	"io/fs"
	"time"
	portainer_api "u-control/uc-aom/internal/aom/docker/v0_1/portainer/client"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer/client/auth"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer/client/stacks"

	"github.com/go-openapi/runtime"

	log "github.com/sirupsen/logrus"
)

type PortainerClientService interface {
	DeleteAddOnStack(string) error
	Logout() error
}

// Create a new instance of the PortainerClientWrapper.
func NewPortainerClientService(
	portainerClient *portainer_api.PortainerCEAPI,
	clientAuthInfoWriter runtime.ClientAuthInfoWriter,
	endpointId int64, timeout time.Duration) PortainerClientService {
	return &portainerClientService{portainerClient, clientAuthInfoWriter, endpointId, timeout}
}

// portainerClientService wraps the Portainer/docker HTTP v2 APIs
type portainerClientService struct {
	portainerClient      *portainer_api.PortainerCEAPI
	clientAuthInfoWriter runtime.ClientAuthInfoWriter
	endpointID           int64
	timeout              time.Duration
}

// Logout from portainer
func (c *portainerClientService) Logout() error {
	_, err := c.portainerClient.Auth.Logout(auth.NewLogoutParams(), c.clientAuthInfoWriter)
	return err
}

// Delete the portainer stack by the given stackName.
// Note: stackName is normalized based on the portainer rules.
func (c *portainerClientService) DeleteAddOnStack(stackName string) error {
	id, err := c.getPortainerStackId(stackName)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Debug("No Portainer stack to delete, skipping.")
			return nil
		}
		return err
	}

	params := stacks.NewStackDeleteParamsWithTimeout(c.timeout).WithID(id).WithEndpointID(&c.endpointID)
	_, err = c.portainerClient.Stacks.StackDelete(params, c.clientAuthInfoWriter)
	return err
}

func (c *portainerClientService) getPortainerStackId(name string) (int64, error) {
	filter := fmt.Sprintf(`{"EndpointID":%d}`, c.endpointID)
	params := stacks.NewStackListParams().WithFilters(&filter)
	resp, noContent, err := c.portainerClient.Stacks.StackList(params, c.clientAuthInfoWriter)
	if err != nil {
		return -1, err
	}
	if noContent != nil {
		return -1, fs.ErrNotExist
	}

	normalizedStackName := NormalizeName(name)
	for _, item := range resp.Payload {
		if item.Name == normalizedStackName {
			return item.ID, nil
		}
	}

	return -1, fs.ErrNotExist
}
