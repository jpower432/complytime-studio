// SPDX-License-Identifier: Apache-2.0

package agents

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/complytime/complytime-studio/internal/gemara"
	"github.com/complytime/complytime-studio/internal/store"
)

// artifactInterceptor wraps an http.ResponseWriter to tee SSE data through
// a line scanner that detects TaskArtifactUpdateEvent payloads. Valid AuditLog
// YAML artifacts are persisted asynchronously without blocking the stream.
type artifactInterceptor struct {
	http.ResponseWriter
	store store.AuditLogStore
	pw    *io.PipeWriter
}

// newArtifactInterceptor creates an interceptor that scans SSE events in a
// background goroutine and persists AuditLog artifacts to the given store.
func newArtifactInterceptor(w http.ResponseWriter, s store.AuditLogStore) *artifactInterceptor {
	pr, pw := io.Pipe()
	ai := &artifactInterceptor{
		ResponseWriter: w,
		store:          s,
		pw:             pw,
	}
	go ai.scan(pr)
	return ai
}

func (ai *artifactInterceptor) Write(p []byte) (int, error) {
	_, _ = ai.pw.Write(p)
	return ai.ResponseWriter.Write(p)
}

// Close signals the scanner goroutine to stop.
func (ai *artifactInterceptor) Close() {
	_ = ai.pw.Close()
}

// Flush supports streaming — delegates to the underlying ResponseWriter if
// it implements http.Flusher.
func (ai *artifactInterceptor) Flush() {
	if f, ok := ai.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (ai *artifactInterceptor) scan(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)

	var isArtifactEvent bool
	var dataBuf bytes.Buffer

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			eventType := strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			isArtifactEvent = strings.Contains(eventType, "artifact")
			dataBuf.Reset()
			continue
		}

		if strings.HasPrefix(line, "data:") && isArtifactEvent {
			dataBuf.WriteString(strings.TrimPrefix(line, "data:"))
			continue
		}

		if line == "" && isArtifactEvent && dataBuf.Len() > 0 {
			ai.processEvent(dataBuf.Bytes())
			isArtifactEvent = false
			dataBuf.Reset()
		}
	}
}

// artifactEvent is the minimal structure of a TaskArtifactUpdateEvent.
type artifactEvent struct {
	Result struct {
		Artifact struct {
			Parts []artifactPart `json:"parts"`
		} `json:"artifact"`
	} `json:"result"`
}

type artifactPart struct {
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata"`
}

func (ai *artifactInterceptor) processEvent(data []byte) {
	var evt artifactEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return
	}

	for _, part := range evt.Result.Artifact.Parts {
		mime := part.Metadata["mimeType"]
		if mime != "application/yaml" {
			continue
		}

		content := part.Text
		if content == "" {
			continue
		}

		summary, err := gemara.ParseAuditLog(content)
		if err != nil {
			slog.Warn("auto-persist: artifact is not a valid AuditLog", "error", err)
			continue
		}

		h := sha256.Sum256([]byte(content))
		auditID := fmt.Sprintf("%x", h[:8])

		policyID := summary.TargetID
		if policyID == "" {
			policyID = "unassigned"
			slog.Warn("auto-persist: could not derive policy_id, using 'unassigned'")
		}

		counts := fmt.Sprintf("Strengths: %d, Findings: %d, Gaps: %d, Observations: %d",
			summary.Strengths, summary.Findings, summary.Gaps, summary.Observations)

		a := store.AuditLog{
			AuditID:       auditID,
			PolicyID:      policyID,
			AuditStart:    summary.AuditStart,
			AuditEnd:      summary.AuditEnd,
			Framework:     summary.Framework,
			CreatedBy:     "auto-persist",
			Content:       content,
			Summary:       counts,
			Model:         part.Metadata["model"],
			PromptVersion: part.Metadata["promptVersion"],
		}

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := ai.store.InsertAuditLog(ctx, a); err != nil {
				slog.Error("auto-persist: store insert failed", "audit_id", a.AuditID, "error", err)
			} else {
				slog.Info("auto-persist: artifact saved", "audit_id", a.AuditID, "policy_id", a.PolicyID)
			}
		}()
	}
}
