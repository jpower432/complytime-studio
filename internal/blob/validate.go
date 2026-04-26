// SPDX-License-Identifier: Apache-2.0

package blob

import (
	"fmt"
	"strings"
)

// ValidateBlobRef checks that ref is a canonical s3://bucket/key URI.
func ValidateBlobRef(ref string) error {
	ref = strings.TrimSpace(ref)
	const prefix = "s3://"
	if !strings.HasPrefix(ref, prefix) {
		return fmt.Errorf("blob_ref must start with %q", prefix)
	}
	rest := strings.TrimPrefix(ref, prefix)
	bucket, key, ok := strings.Cut(rest, "/")
	if !ok || bucket == "" || key == "" {
		return fmt.Errorf("blob_ref must be s3://bucket/key")
	}
	return nil
}
