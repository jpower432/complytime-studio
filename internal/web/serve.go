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

// RegisterSPA configures Echo's error handler to serve the embedded SPA for
// any non-API 404. Static assets are served directly; all other paths get
// index.html (history-mode routing).
func RegisterSPA(e *echo.Echo, assets fs.FS) {
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
		if strings.HasPrefix(c.Request().URL.Path, "/api/") {
			defaultErrHandler(err, c)
			return
		}
		_ = serveSPA(c)
	}
}
