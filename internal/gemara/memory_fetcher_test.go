// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"context"
	"io"
	"testing"
)

func TestMemoryFetcher_Fetch(t *testing.T) {
	content := []byte("hello: world")
	f := NewMemoryFetcher(map[string][]byte{"test.yaml": content})

	rc, err := f.Fetch(context.Background(), "test.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(data) != "hello: world" {
		t.Errorf("got %q, want %q", string(data), "hello: world")
	}
}

func TestMemoryFetcher_NotFound(t *testing.T) {
	f := NewMemoryFetcher(map[string][]byte{})
	_, err := f.Fetch(context.Background(), "missing.yaml")
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}
