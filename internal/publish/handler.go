// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/complytime/complytime-studio/internal/httputil"
)

// Options configures the publish module.
type Options struct {
	TokenProvider      httputil.TokenProvider
	InsecureRegistries []string
}

// Register mounts the /api/publish endpoint on the mux.
func Register(mux *http.ServeMux, opts Options) {
	mux.HandleFunc("/api/publish", handler(opts))
	slog.Info("publish endpoint registered", "route", "/api/publish")
}

func handler(opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Artifacts []string `json:"artifacts"`
			Target    string   `json:"target"`
			Tag       string   `json:"tag"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 8<<20)).Decode(&req); err != nil {
			httputil.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}

		var yamlContents [][]byte
		for _, raw := range req.Artifacts {
			yamlContents = append(yamlContents, []byte(raw))
		}

		pushOpts := PushOptions{}
		if opts.TokenProvider != nil {
			if token, ok := opts.TokenProvider.TokenFromRequest(r); ok {
				pushOpts.Token = token
			}
		}
		for _, insecure := range opts.InsecureRegistries {
			if strings.HasPrefix(req.Target, insecure) {
				pushOpts.PlainHTTP = true
				break
			}
		}

		result, err := AssembleAndPush(r.Context(), yamlContents, req.Target, req.Tag, pushOpts)
		if err != nil {
			httputil.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		httputil.WriteJSON(w, http.StatusOK, result)
	}
}
