// Copyright 2022 Weidmueller Interface GmbH & Co. KG <oss@weidmueller.com>
//
// SPDX-License-Identifier: MIT

package manifest

import (
	"encoding/json"
	"fmt"
	"u-control/uc-aom/api"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// Validator validates addon manifest based on the manifest schema
type Validator interface {
	Validate(manifest *Root) error
}

// Represents the addon manifest validator
type ManifestValidator struct {
	schema *jsonschema.Schema
}

// Represents manifest schema validation errors.
type SchemaValidationError struct {
	message string
}

func (s *SchemaValidationError) Error() string {
	return s.message
}

// NewValidator returns a new addon manifest validator
func NewValidator() (*ManifestValidator, error) {
	schemaStr := api.ManifestSchema
	schema, err := jsonschema.CompileString("/schema", schemaStr)
	if err != nil {
		return nil, err
	}
	return &ManifestValidator{
		schema: schema,
	}, nil
}

// Validate validates a given manifest and returns an error if not confirm with the schema
func (s *ManifestValidator) Validate(manifest *Root) error {
	manifestBytes, err := manifest.ToBytes()
	if err != nil {
		return err
	}
	var convertedManifest map[string]interface{}
	if err := json.Unmarshal(manifestBytes, &convertedManifest); err != nil {
		return err
	}

	if err := s.schema.Validate(convertedManifest); err != nil {
		msg := fmt.Sprintf("'%s' has an invalid manifest. %s", manifest.Title, err.Error())
		return &SchemaValidationError{message: msg}
	}
	return nil
}
