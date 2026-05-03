// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net/http"
	"strings"
)

// CORSOptions configures the CORS middleware.
//
// Deprecated: the gateway uses Echo's CORS middleware. Retained for tests
// and internal tooling that embed net/http directly.
type CORSOptions struct {
	// AllowedOrigins is the set of origins permitted to make requests.
	// An empty slice rejects all cross-origin requests.
	AllowedOrigins []string
}

// CORS returns middleware that sets Access-Control headers. Preflight
// OPTIONS requests are handled and short-circuited.
//
// Deprecated: the gateway uses Echo's CORS middleware.
func CORS(opts CORSOptions) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(opts.AllowedOrigins))
	for _, o := range opts.AllowedOrigins {
		allowed[strings.TrimRight(o, "/")] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.Header().Set("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
