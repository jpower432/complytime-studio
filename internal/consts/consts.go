// SPDX-License-Identifier: Apache-2.0

package consts

import "time"

const (
	// MaxRequestBody is the maximum allowed HTTP request body size (8 MiB).
	MaxRequestBody int64 = 8 << 20

	// MaxInternalRequestBody is the body limit for the internal (agent-only)
	// listener. AuditLog drafts include full YAML with evidence and can
	// exceed the public API limit.
	MaxInternalRequestBody int64 = 64 << 20

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

	// EvalMessageWarnBytes triggers a warning when eval_message exceeds this
	// length, indicating the field may contain raw data rather than a summary.
	EvalMessageWarnBytes = 4096

	// DefaultQueryLimit is the default row limit for list endpoints when
	// the caller omits the limit parameter.
	DefaultQueryLimit = 100

	// MaxQueryLimit is the maximum allowed limit for list endpoints.
	// Requests exceeding this are silently clamped.
	MaxQueryLimit = 1000

	// DefaultInternalPort is the default port for the internal (agent-only)
	// HTTP listener. Override via INTERNAL_PORT env var.
	DefaultInternalPort = "8081"

	// DefaultDevAPIToken is the well-known dev seed token shipped in
	// values.yaml. The gateway warns at startup if this value is still in use.
	DefaultDevAPIToken = "dev-seed-token"

	// RoleAdmin is the admin role value stored in the users table.
	RoleAdmin = "admin"

	// RoleReviewer is the default role for new users.
	RoleReviewer = "reviewer"
)

// ClampLimit applies the default and max query limit policy.
// Zero or negative values get DefaultQueryLimit; values above MaxQueryLimit
// are silently clamped.
func ClampLimit(n int) int {
	if n <= 0 {
		return DefaultQueryLimit
	}
	if n > MaxQueryLimit {
		return MaxQueryLimit
	}
	return n
}

// Blob / manual evidence enrichment
const (
	// MsgBlobStorageNotConfigured is returned when a request includes a file
	// attachment but BLOB_ENDPOINT is not configured.
	MsgBlobStorageNotConfigured = "file upload not supported: blob storage not configured"
)
