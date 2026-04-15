// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"encoding/json"
	"fmt"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/opencontainers/image-spec/specs-go"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
)

// ArtifactInput represents a single Gemara artifact to include in an OCI bundle.
// TODO(go-gemara): This type is a candidate for replacement by the go-gemara SDK's
// bundle types once the SDK ships its OCI packing API.
type ArtifactInput struct {
	Type    gemara.ArtifactType
	Content []byte
}

// BundleResult holds the outcome of a push operation.
type BundleResult struct {
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
	Tag       string `json:"tag"`
}

// AssembleAndPush builds an OCI manifest from the given artifacts and pushes
// the bundle to the target OCI reference.
//
// TODO(go-gemara): This function and the local OCI packing logic are candidates
// for replacement by the go-gemara SDK's upcoming packing API. Track progress at
// https://github.com/gemaraproj/go-gemara for bundle/pack functionality.
func AssembleAndPush(ctx context.Context, artifacts []ArtifactInput, target, tag string) (*BundleResult, error) {
	if len(artifacts) == 0 {
		return nil, fmt.Errorf("no artifacts provided")
	}

	store := memory.New()

	var layers []ocispec.Descriptor
	for _, a := range artifacts {
		mt, err := MediaTypeForArtifact(a.Type)
		if err != nil {
			return nil, fmt.Errorf("artifact %q: %w", a.Type, err)
		}

		desc := ocispec.Descriptor{
			MediaType: mt,
			Size:      int64(len(a.Content)),
		}
		desc.Digest = digestBytes(a.Content)

		if err := store.Push(ctx, desc, newBytesReader(a.Content)); err != nil {
			return nil, fmt.Errorf("store layer %q: %w", a.Type, err)
		}
		layers = append(layers, desc)
	}

	configContent := []byte("{}")
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageConfig,
		Size:      int64(len(configContent)),
		Digest:    digestBytes(configContent),
	}
	if err := store.Push(ctx, configDesc, newBytesReader(configContent)); err != nil {
		return nil, fmt.Errorf("store config: %w", err)
	}

	manifest := ocispec.Manifest{
		Versioned: specs.Versioned{SchemaVersion: 2},
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    layers,
		Annotations: map[string]string{
			ocispec.AnnotationCreated:        nowUTC(),
			"org.opencontainers.image.title": "gemara-bundle",
		},
	}

	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}

	manifestDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Size:      int64(len(manifestJSON)),
		Digest:    digestBytes(manifestJSON),
	}
	if err := store.Push(ctx, manifestDesc, newBytesReader(manifestJSON)); err != nil {
		return nil, fmt.Errorf("store manifest: %w", err)
	}

	if tag != "" {
		if err := store.Tag(ctx, manifestDesc, tag); err != nil {
			return nil, fmt.Errorf("tag manifest: %w", err)
		}
	}

	repo, err := remote.NewRepository(target)
	if err != nil {
		return nil, fmt.Errorf("parse target %q: %w", target, err)
	}

	resolvedTag := tag
	if resolvedTag == "" {
		resolvedTag = "latest"
	}

	if err := copyToRemote(ctx, store, repo, resolvedTag, manifestDesc); err != nil {
		return nil, fmt.Errorf("push to %s:%s: %w", target, resolvedTag, err)
	}

	return &BundleResult{
		Reference: fmt.Sprintf("%s@%s", target, manifestDesc.Digest.String()),
		Digest:    manifestDesc.Digest.String(),
		Tag:       resolvedTag,
	}, nil
}
