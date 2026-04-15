// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"bytes"
	"context"
	"io"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/memory"
	"oras.land/oras-go/v2/registry/remote"
)

func digestBytes(b []byte) digest.Digest {
	return digest.FromBytes(b)
}

func newBytesReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// TODO(go-gemara): copyToRemote and the OCI helper functions in this file are
// candidates for consolidation once the go-gemara SDK ships its packing API.
func copyToRemote(ctx context.Context, src *memory.Store, dst *remote.Repository, tag string, desc ocispec.Descriptor) error {
	_, err := oras.Copy(ctx, src, tag, dst, tag, oras.CopyOptions{})
	if err != nil {
		return err
	}
	_ = desc // used for reference consistency; oras.Copy resolves by tag
	return nil
}
