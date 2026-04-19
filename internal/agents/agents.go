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
	"time"

	studiohttp "github.com/complytime/complytime-studio/internal/httputil"
)

// Card represents a specialist agent entry in the directory.
type Card struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	URL         string  `json:"url"`
	Skills      []Skill `json:"skills"`
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

// Register mounts agent directory and A2A proxy routes on the mux.
func Register(mux *http.ServeMux, opts Options) {
	registerDirectory(mux, opts.Cards)
	registerA2AProxy(mux, opts)
}

func registerDirectory(mux *http.ServeMux, cards []Card) {
	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		studiohttp.WriteJSON(w, http.StatusOK, cards)
	})
	slog.Info("agent directory registered", "count", len(cards))
}

func registerA2AProxy(mux *http.ServeMux, opts Options) {
	allowedAgents := make(map[string]string, len(opts.Cards))
	for _, c := range opts.Cards {
		allowedAgents[c.Name] = c.URL
	}

	mux.HandleFunc("/api/a2a/", func(w http.ResponseWriter, r *http.Request) {
		agentName := strings.TrimPrefix(r.URL.Path, "/api/a2a/")
		agentName = strings.SplitN(agentName, "/", 2)[0]
		if agentName == "" {
			studiohttp.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "missing agent name"})
			return
		}
		fallbackURL, ok := allowedAgents[agentName]
		if !ok {
			studiohttp.WriteJSON(w, http.StatusForbidden, map[string]string{"error": "unknown agent"})
			return
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
			studiohttp.WriteJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid target URL"})
			return
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
			Transport: &http.Transport{
				ResponseHeaderTimeout: 5 * time.Minute,
			},
			FlushInterval: -1,
			ErrorHandler: func(rw http.ResponseWriter, req *http.Request, err error) {
				slog.Error("a2a proxy error", "agent", agentName, "target", target.String()+targetPath, "error", err)
				studiohttp.WriteJSON(rw, http.StatusBadGateway, map[string]string{
					"error": fmt.Sprintf("agent %s unreachable: %v", agentName, err),
				})
			},
		}

		rp.ServeHTTP(w, r)
	})

	mode := "direct (per-agent URL)"
	if opts.KagentA2AURL != "" {
		mode = fmt.Sprintf("controller (%s)", opts.KagentA2AURL)
	}
	slog.Info("a2a proxy registered", "route", "/api/a2a/{agent-name}", "mode", mode)
}
