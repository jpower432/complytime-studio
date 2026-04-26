// SPDX-License-Identifier: Apache-2.0

package gemara

import "strings"

// ExtractAccountableContact parses a Policy YAML and returns the name of the
// first contact with an "accountable" RACI role. Returns empty string if none.
func ExtractAccountableContact(content string) string {
	var doc struct {
		Contacts []struct {
			Name string `yaml:"name"`
			Role string `yaml:"role"`
		} `yaml:"contacts"`
	}
	if err := UnmarshalYAML([]byte(content), &doc); err != nil {
		return ""
	}
	for _, c := range doc.Contacts {
		if strings.EqualFold(c.Role, "accountable") {
			return c.Name
		}
	}
	return ""
}
