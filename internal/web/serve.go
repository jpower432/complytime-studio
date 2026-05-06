// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// Register mounts the SPA file server with history-mode fallback on the mux.
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

// RegisterEchoWithMux sets Echo's fallback handlers to serve the SPA for
// non-API paths and delegate unmatched /api/* paths to the legacy ServeMux.
// This avoids e.Any("/*") which conflicts with apiGroup route priority.
func RegisterEchoWithMux(e *echo.Echo, assets fs.FS, apiMux *http.ServeMux) {
	sub, err := fs.Sub(assets, "dist")
	if err != nil {
		slog.Error("embed workbench failed", "error", err)
		os.Exit(1)
	}
	fileServer := http.FileServer(http.FS(sub))

	serveSPA := func(c echo.Context) error {
		path := c.Request().URL.Path
		if path == "/" || path == "" {
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}
		if _, readErr := sub.(fs.ReadFileFS).ReadFile(strings.TrimPrefix(path, "/")); readErr != nil {
			c.Request().URL.Path = "/"
			fileServer.ServeHTTP(c.Response(), c.Request())
			return nil
		}
		fileServer.ServeHTTP(c.Response(), c.Request())
		return nil
	}

	defaultErrHandler := e.HTTPErrorHandler
	if defaultErrHandler == nil {
		defaultErrHandler = e.DefaultHTTPErrorHandler
	}

	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		he, ok := err.(*echo.HTTPError)
		if !ok || he.Code != http.StatusNotFound {
			defaultErrHandler(err, c)
			return
		}
		path := c.Request().URL.Path
		if strings.HasPrefix(path, "/api/") {
			apiMux.ServeHTTP(c.Response(), c.Request())
			return
		}
		_ = serveSPA(c)
	}
}
