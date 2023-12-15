// Code generated by go-swagger; DO NOT EDIT.

package auth

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new auth API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for auth API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	AuthenticateUser(params *AuthenticateUserParams, opts ...ClientOption) (*AuthenticateUserOK, error)

	Logout(params *LogoutParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*LogoutNoContent, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  AuthenticateUser authenticates

  Use this endpoint to authenticate against Portainer using a username and password.
*/
func (a *Client) AuthenticateUser(params *AuthenticateUserParams, opts ...ClientOption) (*AuthenticateUserOK, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewAuthenticateUserParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "AuthenticateUser",
		Method:             "POST",
		PathPattern:        "/auth",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &AuthenticateUserReader{formats: a.formats},
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*AuthenticateUserOK)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for AuthenticateUser: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  Logout logouts
*/
func (a *Client) Logout(params *LogoutParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*LogoutNoContent, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewLogoutParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "logout",
		Method:             "POST",
		PathPattern:        "/auth/logout",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &LogoutReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, err
	}
	success, ok := result.(*LogoutNoContent)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for logout: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
