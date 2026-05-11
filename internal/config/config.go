// SPDX-License-Identifier: Apache-2.0

package config

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Options holds the platform configuration key-value pairs.
type Options struct {
	Values map[string]string
}

// Register mounts the /config endpoint on the Echo group.
func Register(g *echo.Group, opts Options) {
	cfg := opts.Values
	if cfg == nil {
		cfg = map[string]string{}
	}
	g.GET("/config", func(c echo.Context) error {
		return c.JSON(http.StatusOK, cfg)
	})
}
