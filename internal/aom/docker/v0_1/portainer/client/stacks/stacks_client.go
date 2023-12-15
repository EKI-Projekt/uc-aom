// Copyright 2023 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

// Code generated by go-swagger; DO NOT EDIT.

package stacks

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// New creates a new stacks API client.
func New(transport runtime.ClientTransport, formats strfmt.Registry) ClientService {
	return &Client{transport: transport, formats: formats}
}

/*
Client for stacks API
*/
type Client struct {
	transport runtime.ClientTransport
	formats   strfmt.Registry
}

// ClientOption is the option for Client methods
type ClientOption func(*runtime.ClientOperation)

// ClientService is the interface for Client methods
type ClientService interface {
	StackDelete(params *StackDeleteParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*StackDeleteNoContent, error)

	StackList(params *StackListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*StackListOK, *StackListNoContent, error)

	SetTransport(transport runtime.ClientTransport)
}

/*
  StackDelete removes a stack

  Remove a stack.
**Access policy**: restricted
*/
func (a *Client) StackDelete(params *StackDeleteParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*StackDeleteNoContent, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewStackDeleteParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "StackDelete",
		Method:             "DELETE",
		PathPattern:        "/stacks/{id}",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &StackDeleteReader{formats: a.formats},
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
	success, ok := result.(*StackDeleteNoContent)
	if ok {
		return success, nil
	}
	// unexpected success response
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for StackDelete: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

/*
  StackList lists stacks

  List all stacks based on the current user authorizations.
Will return all stacks if using an administrator account otherwise it
will only return the list of stacks the user have access to.
**Access policy**: restricted
*/
func (a *Client) StackList(params *StackListParams, authInfo runtime.ClientAuthInfoWriter, opts ...ClientOption) (*StackListOK, *StackListNoContent, error) {
	// TODO: Validate the params before sending
	if params == nil {
		params = NewStackListParams()
	}
	op := &runtime.ClientOperation{
		ID:                 "StackList",
		Method:             "GET",
		PathPattern:        "/stacks",
		ProducesMediaTypes: []string{"application/json"},
		ConsumesMediaTypes: []string{"application/json"},
		Schemes:            []string{"http", "https"},
		Params:             params,
		Reader:             &StackListReader{formats: a.formats},
		AuthInfo:           authInfo,
		Context:            params.Context,
		Client:             params.HTTPClient,
	}
	for _, opt := range opts {
		opt(op)
	}

	result, err := a.transport.Submit(op)
	if err != nil {
		return nil, nil, err
	}
	switch value := result.(type) {
	case *StackListOK:
		return value, nil, nil
	case *StackListNoContent:
		return nil, value, nil
	}
	// safeguard: normally, absent a default response, unknown success responses return an error above: so this is a codegen issue
	msg := fmt.Sprintf("unexpected success response for stacks: API contract not enforced by server. Client expected to get an error, but got: %T", result)
	panic(msg)
}

// SetTransport changes the transport on the client
func (a *Client) SetTransport(transport runtime.ClientTransport) {
	a.transport = transport
}
