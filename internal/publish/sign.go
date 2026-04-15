// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"context"
	"fmt"
)

// SigningConfig holds configuration for artifact signing.
type SigningConfig struct {
	Enabled bool
	// KeyRef is the signing key reference (e.g. cosign key path or KMS URI).
	KeyRef string
}

// SignBundle signs an OCI manifest digest at the given reference.
// This is a placeholder that returns an error when signing is requested
// but not yet configured. The concrete implementation will use
// notation-go or cosign-go once a signing provider is selected.
func SignBundle(ctx context.Context, cfg SigningConfig, reference, manifestDigest string) error {
	if !cfg.Enabled {
		return nil
	}
	if cfg.KeyRef == "" {
		return fmt.Errorf("signing enabled but no key reference configured")
	}
	// TODO: integrate notation-go or cosign-go signing provider
	// See design.md Decision 2 for signing strategy
	return fmt.Errorf("signing not yet implemented (key=%s, ref=%s, digest=%s)", cfg.KeyRef, reference, manifestDigest)
}
