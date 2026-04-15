// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"testing"

	gemara "github.com/gemaraproj/go-gemara"
)

func TestAssembleAndPush_NoArtifacts(t *testing.T) {
	_, err := AssembleAndPush(context.Background(), nil, "localhost:5000/test", "v1")
	if err == nil {
		t.Fatal("expected error for empty artifacts")
	}
}

func TestAssembleAndPush_UnknownArtifactType(t *testing.T) {
	artifacts := []ArtifactInput{
		{Type: gemara.InvalidArtifact, Content: []byte("content")},
	}
	_, err := AssembleAndPush(context.Background(), artifacts, "localhost:5000/test", "v1")
	if err == nil {
		t.Fatal("expected error for unknown artifact type")
	}
}
