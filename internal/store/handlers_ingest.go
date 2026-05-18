// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/complytime-labs/complytime-core/internal/consts"
	"github.com/complytime-labs/complytime-core/internal/httputil"
)

func registerIngestRoutes(g *echo.Group, s Stores) {
	g.POST("/ingest", echo.WrapHandler(IngestAsyncHandler(s.IngestPublisher, s.IngestTracker)))
	g.GET("/ingest/jobs/:job_id", IngestJobStatusHandler(s.IngestTracker))
}

// IngestRawPublisher publishes raw YAML for async processing via NATS.
type IngestRawPublisher interface {
	PublishIngestRaw(jobID string, yaml []byte) error
}

// IngestAsyncHandler returns an http.HandlerFunc that accepts raw Gemara
// YAML, assigns a job ID, publishes it to NATS for async processing, and
// returns 202 Accepted with the job ID for polling.
func IngestAsyncHandler(pub IngestRawPublisher, tracker *IngestTracker) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, consts.MaxRequestBody))
		if err != nil {
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			httputil.WriteJSON(w, http.StatusBadRequest, map[string]any{
				"errors": []string{"request body is empty — expected Gemara YAML"},
			})
			return
		}

		jobID := generateJobID()
		tracker.Create(jobID)

		if err := pub.PublishIngestRaw(jobID, body); err != nil {
			tracker.Fail(jobID, fmt.Sprintf("publish failed: %v", err))
			slog.Error("async ingest publish failed", "job_id", jobID, "error", err)
			httputil.WriteJSON(w, http.StatusServiceUnavailable, map[string]any{
				"errors": []string{"event bus unavailable — try again later"},
			})
			return
		}

		httputil.WriteJSON(w, http.StatusAccepted, map[string]any{
			"job_id": jobID,
			"status": "pending",
		})
	}
}

// IngestJobStatusHandler returns an echo handler for polling async ingest jobs.
func IngestJobStatusHandler(tracker *IngestTracker) echo.HandlerFunc {
	return func(c echo.Context) error {
		jobID := c.Param("job_id")
		if jobID == "" {
			return jsonError(c, http.StatusBadRequest, "missing job_id")
		}
		status := tracker.Get(jobID)
		if status == nil {
			return jsonError(c, http.StatusNotFound, "job not found")
		}
		return c.JSON(http.StatusOK, status)
	}
}

func generateJobID() string {
	return uuid.New().String()
}
