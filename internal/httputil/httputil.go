// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
)

// TokenProvider abstracts session-to-token extraction so modules can obtain
// the user's Bearer token without importing the auth package directly.
type TokenProvider interface {
	TokenFromRequest(r *http.Request) (token string, ok bool)
}

// WriteJSON encodes v as JSON and writes it with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// EnvOr returns the environment variable value or the fallback if empty.
func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ReadBody reads up to maxBytes from an io.Reader.
func ReadBody(body io.Reader, maxBytes int64) ([]byte, error) {
	return io.ReadAll(io.LimitReader(body, maxBytes))
}

// UnavailableHandler returns an http.HandlerFunc that responds with 503
// and the given message.
func UnavailableHandler(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusServiceUnavailable, map[string]string{"error": msg})
	}
}
