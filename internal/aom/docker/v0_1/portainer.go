package v0_1

import (
	"time"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer"
	portainer_api "u-control/uc-aom/internal/aom/docker/v0_1/portainer/client"
	portainer_status "u-control/uc-aom/internal/aom/docker/v0_1/portainer/client/status"
	"u-control/uc-aom/internal/pkg/utils"

	httptransport "github.com/go-openapi/runtime/client"
	log "github.com/sirupsen/logrus"
)

const (
	StackVersion = "0.1"

	// The default portainer portainerTimeout is set to 60 minutes
	// as an installation of node-red-minimal took roughly 20 minutes on a UC20-WL2000-AC
	portainerTimeout = 60 * time.Minute
)

func ConnectToPortainer() (portainer.PortainerClientService, error) {

	portainerHost := utils.GetEnv("PORTAINER_CE_URI", "portainer-service:9000")
	portainerApiClient := createPortainerAPIClientAt(portainerHost)

	portainerStatus, errStatus := tryGetPortainerStatus(portainerApiClient, 3000)
	if errStatus != nil {
		log.Errorf("Get portainer status failed %v", errStatus)

	}
	log.Infof("Portainer version: %v", portainerStatus.GetPayload().Version)

	portainerCredentials, errCreds := portainer.GetPortainerCredentials()
	if errCreds != nil {
		log.Errorln("Get portainer credentials failed")
		log.Errorln(errCreds.Error())
		return nil, errCreds
	}

	clientAuthInfoWrapper, err := portainer.NewClientAuthInfoWriterWrapper(portainerCredentials, portainerApiClient.Auth.AuthenticateUser)

	if err != nil {
		log.Errorln("Failed to Authenticate")
		log.Errorln(err.Error())
		return nil, err
	}
	portainerClient := portainer.NewPortainerClientService(portainerApiClient, clientAuthInfoWrapper, 1, portainerTimeout)
	return portainerClient, nil
}

func createPortainerAPIClientAt(host string) *portainer_api.PortainerCEAPI {
	transport := httptransport.New(host, portainer_api.DefaultBasePath, []string{"http"})
	return portainer_api.New(transport)
}

// tryGetPortainerStatus will call the portainers service status with the portainer api
// and keep trying to get the status within an interval of retrySleepMilliseconds for 5 times
// if the service is not initially available
func tryGetPortainerStatus(api *portainer_api.PortainerCEAPI, retrySleepMilliseconds int) (*portainer_status.StatusInspectOK, error) {
	var err error
	var numberOfTries = 5
	for ; numberOfTries > 0; numberOfTries-- {
		status, errStatus := api.Status.StatusInspect(nil)
		if errStatus == nil {
			return status, nil
		}
		err = errStatus
		time.Sleep(time.Duration(retrySleepMilliseconds) * time.Millisecond)
	}
	return nil, err
}
