// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"encoding/json"
	"log/slog"
	"net/http"

	studiohttp "github.com/complytime/complytime-studio/internal/httputil"
)

// CardModel describes the LLM provider and model backing an agent.
type CardModel struct {
	Provider string `json:"provider,omitempty"`
	Name     string `json:"name,omitempty"`
}

// Card represents a specialist agent entry in the directory.
type Card struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	URL         string     `json:"-"`
	Role        string     `json:"role,omitempty"`
	Framework   string     `json:"framework,omitempty"`
	Status      string     `json:"status,omitempty"`
	Tools       []string   `json:"tools,omitempty"`
	Examples    []string   `json:"examples,omitempty"`
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

// RegisterDirectory mounts the agent card directory endpoint.
// Agents with status "hidden" are omitted from the public response.
func RegisterDirectory(mux *http.ServeMux, cards []Card) {
	visible := make([]Card, 0, len(cards))
	for _, c := range cards {
		if c.Status != "hidden" {
			visible = append(visible, c)
		}
	}

	mux.HandleFunc("/api/agents", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		studiohttp.WriteJSON(w, http.StatusOK, visible)
	})
	slog.Info("agent directory registered", "total", len(cards), "visible", len(visible))
}
