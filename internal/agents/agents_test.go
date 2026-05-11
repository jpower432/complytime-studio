// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestParseDirectory_Empty(t *testing.T) {
	cards := ParseDirectory("")
	if len(cards) != 0 {
		t.Fatalf("expected empty slice, got %d cards", len(cards))
	}
}

func TestParseDirectory_InvalidJSON(t *testing.T) {
	cards := ParseDirectory("not-json")
	if len(cards) != 0 {
		t.Fatalf("expected empty slice for invalid JSON, got %d", len(cards))
	}
}

func TestParseDirectory_ValidJSON(t *testing.T) {
	raw := `[{"name":"agent-a","description":"desc","url":"http://a:8080","skills":[]}]`
	cards := ParseDirectory(raw)
	if len(cards) != 1 {
		t.Fatalf("expected 1 card, got %d", len(cards))
	}
	if cards[0].Name != "agent-a" {
		t.Fatalf("name = %q, want agent-a", cards[0].Name)
	}
}

func TestRegisterDirectory_GET(t *testing.T) {
	e := echo.New()
	g := e.Group("/api")
	cards := []Card{
		{Name: "studio-threat-modeler", Description: "STRIDE analysis"},
		{Name: "studio-assistant", Description: "Compliance assistant"},
	}
	RegisterDirectory(g, cards)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []Card
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d agents, want 2", len(got))
	}
}

func TestRegisterDirectory_MethodNotAllowed(t *testing.T) {
	e := echo.New()
	g := e.Group("/api")
	RegisterDirectory(g, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agents", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestA2AProxy_UnknownAgent(t *testing.T) {
	e := echo.New()
	g := e.Group("/api")
	RegisterDirectory(g, []Card{{Name: "known-agent", URL: "http://known:8080"}})
	RegisterA2AProxy(g, Options{
		Cards: []Card{{Name: "known-agent", URL: "http://known:8080"}},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/a2a/unknown-agent", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 for unknown agent", rec.Code)
	}
}

func TestA2AProxy_MissingAgentName(t *testing.T) {
	e := echo.New()
	g := e.Group("/api")
	RegisterA2AProxy(g, Options{Cards: []Card{}})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/a2a/", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for missing agent name", rec.Code)
	}
}
