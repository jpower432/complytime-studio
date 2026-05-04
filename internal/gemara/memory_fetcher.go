// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"bytes"
	"context"
	"fmt"
	"io"
)

// MemoryFetcher satisfies the gemara.Fetcher interface using in-memory content.
// Useful for feeding already-loaded YAML (e.g. from a database query or request body)
// into the sdk.Load[T] pipeline without hitting the filesystem or network.
type MemoryFetcher struct {
	content map[string][]byte
}

// NewMemoryFetcher returns a Fetcher backed by the provided name→content map.
func NewMemoryFetcher(content map[string][]byte) *MemoryFetcher {
	return &MemoryFetcher{content: content}
}

// Fetch returns an io.ReadCloser for the named source.
func (m *MemoryFetcher) Fetch(_ context.Context, source string) (io.ReadCloser, error) {
	data, ok := m.content[source]
	if !ok {
		return nil, fmt.Errorf("source %q not found in memory fetcher", source)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}
