// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

type fakeChatStore struct {
	chat   *ChatSession
	getErr error
	putErr error
}

func (f *fakeChatStore) PutChat(_ context.Context, _ string, _ ChatSession) error {
	return f.putErr
}

func (f *fakeChatStore) GetChat(_ context.Context, _ string) (*ChatSession, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.chat == nil {
		return nil, ErrChatNotFound
	}
	return f.chat, nil
}

func (f *fakeChatStore) DeleteChat(_ context.Context, _ string) error {
	return nil
}

func chatEcho(h *Handler, cs ChatStore) *echo.Echo {
	e := echo.New()
	e.Use(h.Middleware())
	g := e.Group("/api")
	h.RegisterChatHistory(g, cs)
	return e
}

func authedReq(method, path, body string) *http.Request {
	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("X-Forwarded-Email", "alice@example.com")
	req.Header.Set("X-Forwarded-Preferred-Username", "alice")
	return req
}

func TestGetChatHistory_Unauthenticated(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/chat/history", nil)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGetChatHistory_NotFound(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{getErr: ErrChatNotFound})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, authedReq(http.MethodGet, "/api/chat/history", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["messages"] != nil {
		t.Fatalf("messages = %v, want nil", body["messages"])
	}
}

func TestGetChatHistory_Expired(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{getErr: ErrChatExpired})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, authedReq(http.MethodGet, "/api/chat/history", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body["taskId"] != "" {
		t.Fatalf("taskId = %q, want empty", body["taskId"])
	}
}

func TestGetChatHistory_Success(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	cs := &fakeChatStore{
		chat: &ChatSession{
			Messages: json.RawMessage(`[{"role":"user","content":"hello"}]`),
			TaskID:   "task-123",
		},
	}
	e := chatEcho(h, cs)

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, authedReq(http.MethodGet, "/api/chat/history", ""))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body ChatSession
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if body.TaskID != "task-123" {
		t.Fatalf("taskId = %q, want task-123", body.TaskID)
	}
}

func TestGetChatHistory_StoreError(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{getErr: errors.New("db connection lost")})

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, authedReq(http.MethodGet, "/api/chat/history", ""))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestPutChatHistory_Unauthenticated(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/chat/history",
		strings.NewReader(`{"messages":[],"taskId":"t1"}`))
	req.Header.Set("Content-Type", "application/json")
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestPutChatHistory_InvalidBody(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{})

	rec := httptest.NewRecorder()
	req := authedReq(http.MethodPut, "/api/chat/history", "not json at all {{{")
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestPutChatHistory_Success(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{})

	rec := httptest.NewRecorder()
	req := authedReq(http.MethodPut, "/api/chat/history", `{"messages":[{"role":"user","content":"hi"}],"taskId":"t1"}`)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

func TestPutChatHistory_StoreError(t *testing.T) {
	h, _ := testHandlerWithStore(t)
	e := chatEcho(h, &fakeChatStore{putErr: errors.New("disk full")})

	rec := httptest.NewRecorder()
	req := authedReq(http.MethodPut, "/api/chat/history", `{"messages":[],"taskId":"t1"}`)
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
