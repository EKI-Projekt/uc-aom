// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"

	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// PortainerPair portainer pair
//
// swagger:model portainer.Pair
type PortainerPair struct {

	// name
	// Example: name
	Name string `json:"name,omitempty"`

	// value
	// Example: value
	Value string `json:"value,omitempty"`
}

// Validate validates this portainer pair
func (m *PortainerPair) Validate(formats strfmt.Registry) error {
	return nil
}

// ContextValidate validates this portainer pair based on context it is used
func (m *PortainerPair) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *PortainerPair) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *PortainerPair) UnmarshalBinary(b []byte) error {
	var res PortainerPair
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}