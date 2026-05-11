// SPDX-License-Identifier: Apache-2.0

package publish

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/complytime/complytime-studio/internal/httputil"
)

// Options configures the publish module.
type Options struct {
	TokenProvider      httputil.TokenProvider
	InsecureRegistries []string
}

// Register mounts the /publish endpoint on the Echo group.
func Register(g *echo.Group, opts Options) {
	g.POST("/publish", publishHandler(opts))
	slog.Info("publish endpoint registered", "route", "/api/publish")
}

func publishHandler(opts Options) echo.HandlerFunc {
	return func(c echo.Context) error {
		var req struct {
			Artifacts []string `json:"artifacts"`
			Target    string   `json:"target"`
			Tag       string   `json:"tag"`
		}
		if err := c.Bind(&req); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		var yamlContents [][]byte
		for _, raw := range req.Artifacts {
			yamlContents = append(yamlContents, []byte(raw))
		}

		pushOpts := PushOptions{}
		if opts.TokenProvider != nil {
			if token, ok := opts.TokenProvider.TokenFromRequest(c.Request()); ok {
				pushOpts.Token = token
			}
		}
		for _, insecure := range opts.InsecureRegistries {
			if strings.HasPrefix(req.Target, insecure) {
				pushOpts.PlainHTTP = true
				break
			}
		}

		result, err := AssembleAndPush(c.Request().Context(), yamlContents, req.Target, req.Tag, pushOpts)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return c.JSON(http.StatusOK, result)
	}
}
