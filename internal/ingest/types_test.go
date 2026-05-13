// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"reflect"
	"testing"
)

// OTel log attribute compliance.source.registry maps to the
// source_registry column (see docs/design/evidence-semconv-alignment.md). The
// collector exporter in complytime-collector-components must use the same
// column name; this test locks the field contract for that path.
func TestEvidenceRow_SourceRegistryFieldExists(t *testing.T) {
	t.Parallel()
	_, ok := reflect.TypeOf(EvidenceRow{}).FieldByName("SourceRegistry")
	if !ok {
		t.Fatal("SourceRegistry field missing from EvidenceRow")
	}
}
