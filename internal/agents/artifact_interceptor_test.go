// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/complytime/complytime-studio/internal/store"
)

// mockAuditLogStore captures InsertAuditLog calls for test assertions.
type mockAuditLogStore struct {
	mu      sync.Mutex
	logs    []store.AuditLog
	err     error
	called  chan struct{}
}

func newMockStore() *mockAuditLogStore {
	return &mockAuditLogStore{called: make(chan struct{}, 10)}
}

func (m *mockAuditLogStore) InsertAuditLog(_ context.Context, a store.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, a)
	m.called <- struct{}{}
	return m.err
}

func (m *mockAuditLogStore) ListAuditLogs(_ context.Context, _ string, _, _ time.Time) ([]store.AuditLog, error) {
	return nil, nil
}

func (m *mockAuditLogStore) GetAuditLog(_ context.Context, _ string) (*store.AuditLog, error) {
	return nil, nil
}

func (m *mockAuditLogStore) wait(t *testing.T) {
	t.Helper()
	select {
	case <-m.called:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for store call")
	}
}

const validAuditLogYAML = `metadata:
  id: audit-test
  type: AuditLog
  gemara-version: "1.0.0"
  description: Test audit
  date: "2026-04-16T10:00:00Z"
  author:
    id: test
    name: Test
    type: Software Assisted
  mapping-references:
    - id: ref
      title: TestFramework
      version: "1.0"
target:
  id: test-policy
  name: test
  type: Software
summary: Test
criteria:
  - reference-id: ref
results:
  - id: r1
    title: Check
    type: Strength
    description: Passed
    criteria-reference:
      reference-id: ref
      entries:
        - reference-id: ref
    evidence:
      - type: EvaluationLog
        collected: "2026-04-07T10:00:00Z"
        location:
          reference-id: ref
        description: eval`

func buildSSE(eventType, data string) string {
	return fmt.Sprintf("event: %s\ndata:%s\n\n", eventType, data)
}

func buildArtifactPayload(content string, metadata map[string]string) string {
	metaJSON := "{"
	first := true
	for k, v := range metadata {
		if !first {
			metaJSON += ","
		}
		metaJSON += fmt.Sprintf(`"%s":"%s"`, k, v)
		first = false
	}
	metaJSON += "}"

	escaped := strings.ReplaceAll(content, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)

	return fmt.Sprintf(`{"result":{"artifact":{"parts":[{"text":"%s","metadata":%s}]}}}`, escaped, metaJSON)
}

func TestInterceptor_ValidArtifact(t *testing.T) {
	ms := newMockStore()
	rec := httptest.NewRecorder()
	ai := newArtifactInterceptor(rec, ms)

	payload := buildArtifactPayload(validAuditLogYAML, map[string]string{
		"mimeType":      "application/yaml",
		"model":         "gemini-2.5-pro",
		"promptVersion": "abc123",
	})
	sse := buildSSE("artifact", payload)

	_, _ = ai.Write([]byte(sse))
	ai.Close()
	ms.wait(t)

	ms.mu.Lock()
	defer ms.mu.Unlock()

	if len(ms.logs) != 1 {
		t.Fatalf("expected 1 insert, got %d", len(ms.logs))
	}
	a := ms.logs[0]
	if a.PolicyID != "test-policy" {
		t.Errorf("policy_id = %q, want test-policy", a.PolicyID)
	}
	if a.Model != "gemini-2.5-pro" {
		t.Errorf("model = %q, want gemini-2.5-pro", a.Model)
	}
	if a.PromptVersion != "abc123" {
		t.Errorf("prompt_version = %q, want abc123", a.PromptVersion)
	}
	if a.CreatedBy != "auto-persist" {
		t.Errorf("created_by = %q, want auto-persist", a.CreatedBy)
	}
}

func TestInterceptor_ForwardsAllBytes(t *testing.T) {
	ms := newMockStore()
	rec := httptest.NewRecorder()
	ai := newArtifactInterceptor(rec, ms)

	data := "event: message\ndata:{\"text\":\"hello\"}\n\n"
	n, err := ai.Write([]byte(data))
	ai.Close()

	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	if n != len(data) {
		t.Fatalf("wrote %d bytes, want %d", n, len(data))
	}
	if rec.Body.String() != data {
		t.Fatalf("forwarded body mismatch:\n  got:  %q\n  want: %q", rec.Body.String(), data)
	}
}

func TestInterceptor_NonYAMLIgnored(t *testing.T) {
	ms := newMockStore()
	rec := httptest.NewRecorder()
	ai := newArtifactInterceptor(rec, ms)

	payload := buildArtifactPayload("some text content", map[string]string{
		"mimeType": "text/plain",
	})
	sse := buildSSE("artifact", payload)

	_, _ = ai.Write([]byte(sse))
	ai.Close()

	time.Sleep(100 * time.Millisecond)
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if len(ms.logs) != 0 {
		t.Fatalf("expected 0 inserts for non-YAML, got %d", len(ms.logs))
	}
}

func TestInterceptor_MalformedSSEPassesThrough(t *testing.T) {
	ms := newMockStore()
	rec := httptest.NewRecorder()
	ai := newArtifactInterceptor(rec, ms)

	malformed := "event: artifact\ndata:not-json-at-all\n\n"
	n, err := ai.Write([]byte(malformed))
	ai.Close()

	if err != nil {
		t.Fatalf("write error: %v", err)
	}
	if n != len(malformed) {
		t.Fatalf("wrote %d bytes, want %d", n, len(malformed))
	}

	time.Sleep(100 * time.Millisecond)
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if len(ms.logs) != 0 {
		t.Fatalf("expected 0 inserts for malformed data, got %d", len(ms.logs))
	}
}

func TestInterceptor_StoreErrorDoesNotBlock(t *testing.T) {
	ms := newMockStore()
	ms.err = fmt.Errorf("clickhouse unavailable")
	rec := httptest.NewRecorder()
	ai := newArtifactInterceptor(rec, ms)

	payload := buildArtifactPayload(validAuditLogYAML, map[string]string{
		"mimeType": "application/yaml",
	})
	sse := buildSSE("artifact", payload)

	_, _ = ai.Write([]byte(sse))
	ai.Close()
	ms.wait(t)

	if rec.Body.String() != sse {
		t.Fatal("stream body should be forwarded unchanged despite store error")
	}
}

func TestInterceptor_ContentAddressedID(t *testing.T) {
	h := sha256.Sum256([]byte(validAuditLogYAML))
	expected := fmt.Sprintf("%x", h[:8])

	ms := newMockStore()
	rec := httptest.NewRecorder()
	ai := newArtifactInterceptor(rec, ms)

	payload := buildArtifactPayload(validAuditLogYAML, map[string]string{
		"mimeType": "application/yaml",
	})
	sse := buildSSE("artifact", payload)

	_, _ = ai.Write([]byte(sse))
	ai.Close()
	ms.wait(t)

	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.logs[0].AuditID != expected {
		t.Errorf("audit_id = %q, want %q", ms.logs[0].AuditID, expected)
	}
}

func TestInterceptor_DisabledWhenNilStore(t *testing.T) {
	rec := httptest.NewRecorder()

	data := "event: artifact\ndata:{}\n\n"
	_, _ = rec.Write([]byte(data))

	if rec.Body.String() != data {
		t.Fatal("without interceptor, data should pass through directly")
	}
}
