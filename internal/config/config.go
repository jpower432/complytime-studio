// SPDX-License-Identifier: Apache-2.0

package config

import (
	"net/http"

	"github.com/complytime/complytime-studio/internal/httputil"
)

// Options holds the platform configuration key-value pairs.
type Options struct {
	Values map[string]string
}

// Register mounts the /api/config endpoint on the mux.
func Register(mux *http.ServeMux, opts Options) {
	cfg := opts.Values
	if cfg == nil {
		cfg = map[string]string{}
	}
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		httputil.WriteJSON(w, http.StatusOK, cfg)
	})
}
