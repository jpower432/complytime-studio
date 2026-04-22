// SPDX-License-Identifier: Apache-2.0

// Package gemara provides schema-aware YAML parsing for Gemara artifacts.
// It uses go-gemara types and goccy/go-yaml to extract structured data
// from Policy, MappingDocument, and AuditLog YAML content.
package gemara

import goyaml "github.com/goccy/go-yaml"

// UnmarshalYAML exposes the goccy/go-yaml unmarshaller so callers can do
// lightweight type detection without importing the YAML library directly.
func UnmarshalYAML(data []byte, v any) error {
	return goyaml.Unmarshal(data, v)
}
