// SPDX-License-Identifier: Apache-2.0

package consts

import "time"

const (
	// MaxRequestBody is the maximum allowed HTTP request body size (8 MiB).
	MaxRequestBody int64 = 8 << 20

	// HTTPClientTimeout is the default timeout for outbound HTTP clients.
	HTTPClientTimeout = 15 * time.Second

	// ProxyResponseTimeout is the timeout for A2A reverse proxy responses.
	ProxyResponseTimeout = 5 * time.Minute

	// RegistryPushTimeout is the timeout for OCI registry push operations.
	RegistryPushTimeout = 60 * time.Second

	// ServerReadTimeout is the HTTP server read timeout.
	ServerReadTimeout = 30 * time.Second

	// ServerWriteTimeout is the HTTP server write timeout (long for SSE).
	ServerWriteTimeout = 5 * time.Minute

	// ServerIdleTimeout is the HTTP server idle connection timeout.
	ServerIdleTimeout = 120 * time.Second

	// GemaraVersion is the default Gemara version stamped on OCI bundles.
	GemaraVersion = "v1.0.0"
)
