// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"

	"github.com/complytime/complytime-studio/internal/consts"
	studiohttp "github.com/complytime/complytime-studio/internal/httputil"
)

// CardModel describes the LLM provider and model backing an agent.
type CardModel struct {
	Provider string `json:"provider,omitempty"`
	Name     string `json:"name,omitempty"`
}

// Card represents a specialist agent entry in the directory.
type Card struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	URL         string     `json:"url"`
	Skills      []Skill    `json:"skills"`
	Model       *CardModel `json:"model,omitempty"`
}

// Skill describes one A2A skill exposed by a specialist agent.
type Skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// Options configures the agents module.
type Options struct {
	Cards         []Card
	TokenProvider studiohttp.TokenProvider
	// KagentA2AURL is the base URL for the kagent controller's A2A endpoint,
	// e.g. "http://kagent-controller.kagent:8083/api/a2a". When set, all A2A
	// requests are routed through the controller using the pattern:
	//   {KagentA2AURL}/{AgentNamespace}/{agent-name}/
	// When empty, falls back to direct per-agent URLs from Card.URL.
	KagentA2AURL string
	// AgentNamespace is the Kubernetes namespace where agents are deployed.
	// Used with KagentA2AURL to build the controller proxy path.
	AgentNamespace string
}

// ParseDirectory parses the AGENT_DIRECTORY JSON into a slice of Cards.
func ParseDirectory(raw string) []Card {
	if raw == "" {
		return []Card{}
	}
	var cards []Card
	if err := json.Unmarshal([]byte(raw), &cards); err != nil {
		slog.Warn("AGENT_DIRECTORY parse error", "error", err)
		return []Card{}
	}
	return cards
}

// RegisterDirectory mounts the agent card directory endpoint on the Echo group.
func RegisterDirectory(g *echo.Group, cards []Card) {
	g.GET("/agents", func(c echo.Context) error {
		return c.JSON(http.StatusOK, cards)
	})
	slog.Info("agent directory registered", "count", len(cards))
}

// RegisterA2AProxy mounts the A2A reverse proxy routes on the Echo group.
func RegisterA2AProxy(g *echo.Group, opts Options) {
	allowedAgents := make(map[string]string, len(opts.Cards))
	for _, c := range opts.Cards {
		allowedAgents[c.Name] = c.URL
	}

	transport := &http.Transport{
		ResponseHeaderTimeout: consts.ProxyResponseTimeout,
	}

	handler := func(c echo.Context) error {
		agentName := c.Param("*")
		agentName = strings.SplitN(agentName, "/", 2)[0]
		if agentName == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing agent name"})
		}
		fallbackURL, ok := allowedAgents[agentName]
		if !ok {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "unknown agent"})
		}

		var targetURL string
		var targetPath string
		if opts.KagentA2AURL != "" {
			ns := opts.AgentNamespace
			if ns == "" {
				ns = "default"
			}
			targetURL = opts.KagentA2AURL
			targetPath = fmt.Sprintf("/api/a2a/%s/%s/", ns, agentName)
		} else if fallbackURL != "" {
			targetURL = fallbackURL
			targetPath = "/"
		} else {
			targetURL = fmt.Sprintf("http://%s:8080", agentName)
			targetPath = "/"
		}

		target, err := url.Parse(targetURL)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid target URL"})
		}

		slog.Debug("a2a proxy", "agent", agentName, "target", target.String()+targetPath)

		rp := &httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = target.Scheme
				req.URL.Host = target.Host
				req.URL.Path = targetPath
				req.Host = target.Host

				if opts.TokenProvider != nil {
					if token, ok := opts.TokenProvider.TokenFromRequest(req); ok {
						req.Header.Set("Authorization", "Bearer "+token)
					}
				}
			},
			Transport:     transport,
			FlushInterval: -1,
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				slog.Error("a2a proxy error", "agent", agentName, "target", target.String()+targetPath, "error", err)
				studiohttp.WriteJSON(rw, http.StatusBadGateway, map[string]string{
					"error": fmt.Sprintf("agent %s unreachable: %v", agentName, err),
				})
			},
		}

		rp.ServeHTTP(c.Response(), c.Request())
		return nil
	}

	g.Any("/a2a/*", handler)

	mode := "direct (per-agent URL)"
	if opts.KagentA2AURL != "" {
		mode = fmt.Sprintf("controller (%s)", opts.KagentA2AURL)
	}
	slog.Info("a2a proxy registered", "route", "/api/a2a/{agent-name}", "mode", mode)
}

// RegisterA2AForward mounts a thin pass-through that forwards all /a2a/*
// requests to the standalone A2A proxy service.
func RegisterA2AForward(g *echo.Group, proxyURL string) {
	target, err := url.Parse(proxyURL)
	if err != nil {
		slog.Error("invalid A2A_PROXY_URL", "url", proxyURL, "error", err)
		return
	}
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
		Transport: &http.Transport{
			ResponseHeaderTimeout: consts.ProxyResponseTimeout,
		},
		FlushInterval: -1,
		ErrorHandler: func(rw http.ResponseWriter, _ *http.Request, err error) {
			slog.Error("a2a forward error", "target", proxyURL, "error", err)
			studiohttp.WriteJSON(rw, http.StatusBadGateway, map[string]string{
				"error": fmt.Sprintf("a2a proxy unreachable: %v", err),
			})
		},
	}
	g.Any("/a2a/*", echo.WrapHandler(rp))
	slog.Info("a2a forwarding registered", "target", proxyURL)
}
