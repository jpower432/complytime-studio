// SPDX-License-Identifier: Apache-2.0

package ingest

import (
	"reflect"
	"testing"
)

// OTel log attribute compliance.source.registry maps to ClickHouse column
// source_registry (see docs/design/evidence-semconv-alignment.md). The
// collector exporter in complytime-collector-components must use the same
// column name; this test locks the struct tag contract for that path.
func TestEvidenceRow_SourceRegistryCHTagMatchesSemconvColumn(t *testing.T) {
	t.Parallel()
	f, ok := reflect.TypeOf(EvidenceRow{}).FieldByName("SourceRegistry")
	if !ok {
		t.Fatal("SourceRegistry field missing")
	}
	tag := f.Tag.Get("ch")
	if tag != "source_registry" {
		t.Fatalf("ch tag %q, want source_registry", tag)
	}
}
