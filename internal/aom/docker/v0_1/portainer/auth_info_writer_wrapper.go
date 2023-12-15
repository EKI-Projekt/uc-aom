package portainer

import (
	"context"
	"time"

	"u-control/uc-aom/internal/aom/docker/v0_1/portainer/client/auth"
	"u-control/uc-aom/internal/aom/docker/v0_1/portainer/models"

	httptransport "github.com/go-openapi/runtime/client"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/golang-jwt/jwt/v4"
)

type ClientAuthInfoWriterWrapper struct {
	authInfoWriter    runtime.ClientAuthInfoWriter
	credentials       *PortainerUserCredentials
	expTime           time.Time
	authUsersCallback authUsersCallback
}

type authUsersCallback func(*auth.AuthenticateUserParams, ...auth.ClientOption) (*auth.AuthenticateUserOK, error)

//Function to create a new NewClientAuthInfoWriterWrapper
//The new wrapper will be authenticated with the provided credentials
func NewClientAuthInfoWriterWrapper(credentials *PortainerUserCredentials, authUsersCallback authUsersCallback) (*ClientAuthInfoWriterWrapper, error) {
	ClientAuthInfoWriterWrapper := &ClientAuthInfoWriterWrapper{authInfoWriter: nil, credentials: credentials, expTime: time.Now(), authUsersCallback: authUsersCallback}
	err := ClientAuthInfoWriterWrapper.authenticate()
	return ClientAuthInfoWriterWrapper, err
}

//AuthenticateRequest is a wrapper that checks if the token is expired/will expire soon. If thats the case the token is refreshed.
func (c *ClientAuthInfoWriterWrapper) AuthenticateRequest(req runtime.ClientRequest, reg strfmt.Registry) error {
	if c.isTokenExpired() {
		err := c.authenticate()
		if err != nil {
			return err
		}
	}
	return c.authInfoWriter.AuthenticateRequest(req, reg)
}

//Function checks if the token is expired/will expire soon.
func (c *ClientAuthInfoWriterWrapper) isTokenExpired() bool {
	const timeWindow = time.Minute * 30
	expiryTime := time.Now().Add(timeWindow)

	return c.expTime.Before(expiryTime)
}

func (c *ClientAuthInfoWriterWrapper) authenticate() error {
	body := models.AuthAuthenticatePayload{
		Username: &c.credentials.Username,
		Password: &c.credentials.Password,
	}
	client_data := auth.NewAuthenticateUserParamsWithContext(context.Background())
	client_data.WithBody(&body)
	auth_response, err := c.authUsersCallback(client_data)

	if err != nil {
		return err
	}

	tokenString := auth_response.GetPayload().Jwt

	claims := jwt.RegisteredClaims{}
	_, _, err = new(jwt.Parser).ParseUnverified(tokenString, &claims)
	if err != nil {
		return err
	}

	c.authInfoWriter = httptransport.BearerToken(tokenString)
	c.expTime = claims.ExpiresAt.Time
	return nil
}
