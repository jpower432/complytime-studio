// SPDX-License-Identifier: Apache-2.0

package httputil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	handler := CORS(CORSOptions{AllowedOrigins: []string{"https://studio.example.com"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.Header.Set("Origin", "https://studio.example.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://studio.example.com" {
		t.Fatalf("ACAO = %q, want allowed origin", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("ACAC = %q, want true", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	handler := CORS(CORSOptions{AllowedOrigins: []string{"https://allowed.com"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	req.Header.Set("Origin", "https://evil.com")
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("ACAO = %q, want empty for disallowed origin", got)
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	handler := CORS(CORSOptions{AllowedOrigins: []string{"https://studio.example.com"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("inner handler should not be called for preflight")
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/agents", nil)
	req.Header.Set("Origin", "https://studio.example.com")
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 for preflight", rec.Code)
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	handler := CORS(CORSOptions{AllowedOrigins: []string{"https://allowed.com"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("ACAO = %q, want empty when no Origin header", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
