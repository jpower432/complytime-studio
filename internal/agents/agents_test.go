// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestParseDirectory_URLOmittedFromJSON(t *testing.T) {
	cards := []Card{{Name: "test", Description: "d", URL: "http://internal:8080"}}
	data, err := json.Marshal(cards)
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded[0]["url"]; ok {
		t.Fatal("URL should be omitted from JSON output (json:\"-\")")
	}
}

func TestRegisterDirectory_GET(t *testing.T) {
	mux := http.NewServeMux()
	cards := []Card{
		{Name: "studio-threat-modeler", Description: "STRIDE analysis"},
		{Name: "studio-assistant", Description: "Compliance assistant"},
	}
	RegisterDirectory(mux, cards)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	mux.ServeHTTP(rec, req)

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
	mux := http.NewServeMux()
	RegisterDirectory(mux, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/agents", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

func TestRegisterDirectory_FiltersHidden(t *testing.T) {
	mux := http.NewServeMux()
	cards := []Card{
		{ID: "visible-agent", Name: "Visible", Status: "active"},
		{ID: "hidden-agent", Name: "Hidden", Status: "hidden"},
	}
	RegisterDirectory(mux, cards)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var got []Card
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d agents, want 1 (hidden should be filtered)", len(got))
	}
	if got[0].ID != "visible-agent" {
		t.Fatalf("got id = %q, want visible-agent", got[0].ID)
	}
}
