// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"fmt"
	"net/http"

	gemara "github.com/gemaraproj/go-gemara"
	gemarabundle "github.com/gemaraproj/go-gemara/bundle"

	"github.com/complytime/complytime-studio/internal/consts"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

var registryClient = &http.Client{Timeout: consts.RegistryPushTimeout}

// BundleResult holds the outcome of a push operation.
type BundleResult struct {
	Reference string `json:"reference"`
	Digest    string `json:"digest"`
	Tag       string `json:"tag"`
}

// PushOptions configures the push operation.
type PushOptions struct {
	// Token is a Bearer token for registry authentication (e.g., GitHub OAuth token for GHCR).
	Token string
	// PlainHTTP allows pushing to insecure (non-TLS) registries.
	PlainHTTP bool
}

// AssembleAndPush builds an OCI bundle from raw YAML artifacts and pushes it
// to the target registry reference using the go-gemara bundle SDK.
func AssembleAndPush(ctx context.Context, yamlContents [][]byte, target, tag string, opts PushOptions) (*BundleResult, error) {
	if len(yamlContents) == 0 {
		return nil, fmt.Errorf("no artifacts provided")
	}

	var files []gemarabundle.File
	for i, raw := range yamlContents {
		at, err := gemara.DetectType(raw)
		if err != nil {
			return nil, fmt.Errorf("artifact[%d]: %w", i, err)
		}
		files = append(files, gemarabundle.File{
			Name: at.String() + ".yaml",
			Data: raw,
		})
	}

	b := &gemarabundle.Bundle{
		Manifest: gemarabundle.Manifest{
			BundleVersion: "1",
			GemaraVersion: consts.GemaraVersion,
		},
		Files: files,
	}

	store := memory.New()
	desc, err := gemarabundle.Pack(ctx, store, b)
	if err != nil {
		return nil, fmt.Errorf("pack bundle: %w", err)
	}

	resolvedTag := tag
	if resolvedTag == "" {
		resolvedTag = "latest"
	}
	if err := store.Tag(ctx, desc, resolvedTag); err != nil {
		return nil, fmt.Errorf("tag manifest: %w", err)
	}

	repo, err := remote.NewRepository(target)
	if err != nil {
		return nil, fmt.Errorf("parse target %q: %w", target, err)
	}
	repo.PlainHTTP = opts.PlainHTTP

	if opts.Token != "" {
		repo.Client = &auth.Client{
			Client: registryClient,
			Credential: auth.StaticCredential(repo.Reference.Host(), auth.Credential{
				Username: "oauth2",
				Password: opts.Token,
			}),
		}
	}

	if _, err := oras.Copy(ctx, store, resolvedTag, repo, resolvedTag, oras.CopyOptions{}); err != nil {
		return nil, fmt.Errorf("push to %s:%s: %w", target, resolvedTag, err)
	}

	return &BundleResult{
		Reference: fmt.Sprintf("%s@%s", target, desc.Digest.String()),
		Digest:    desc.Digest.String(),
		Tag:       resolvedTag,
	}, nil
}
