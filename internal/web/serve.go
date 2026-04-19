// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

// Register mounts the SPA file server with history-mode fallback on the mux.
// The assets FS should contain the built frontend files (e.g., workbench/dist).
func Register(mux *http.ServeMux, assets fs.FS) {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		slog.Error("embed workbench failed", "error", err)
		os.Exit(1)
	}

	fileServer := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}
		if _, err := sub.(fs.ReadFileFS).ReadFile(strings.TrimPrefix(path, "/")); err != nil {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}
