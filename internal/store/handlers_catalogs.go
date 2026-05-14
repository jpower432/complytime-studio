// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	gemarapkg "github.com/complytime/complytime-studio/internal/gemara"
)

func registerCatalogRoutes(g *echo.Group, s Stores) {
	g.GET("/catalogs", listCatalogsHandler(s.Catalogs))
	g.POST("/catalogs/import", importCatalogHandler(s.Catalogs, s.Controls, s.Threats, s.Risks, s.Guidance))
}

func listCatalogsHandler(cs CatalogStore) echo.HandlerFunc {
	type catalogLite struct {
		CatalogID   string    `json:"catalog_id"`
		CatalogType string    `json:"catalog_type"`
		Title       string    `json:"title"`
		PolicyID    string    `json:"policy_id,omitempty"`
		ImportedAt  time.Time `json:"imported_at"`
	}
	return func(c echo.Context) error {
		if cs == nil {
			return c.JSON(http.StatusOK, []catalogLite{})
		}
		rows, err := cs.ListCatalogs(c.Request().Context())
		if err != nil {
			slog.Error("list catalogs failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "query failed")
		}
		filter := c.QueryParam("type")
		out := make([]catalogLite, 0, len(rows))
		for _, row := range rows {
			if filter != "" && row.CatalogType != filter {
				continue
			}
			out = append(out, catalogLite{
				CatalogID:   row.CatalogID,
				CatalogType: row.CatalogType,
				Title:       row.Title,
				PolicyID:    row.PolicyID,
				ImportedAt:  row.ImportedAt,
			})
		}
		return c.JSON(http.StatusOK, out)
	}
}

func importCatalogHandler(
	cs CatalogStore, ctrlS ControlStore, threatS ThreatStore, riskS RiskStore, guidanceS GuidanceStore,
) echo.HandlerFunc {
	type importReq struct {
		CatalogID string `json:"catalog_id"`
		PolicyID  string `json:"policy_id"`
		Content   string `json:"content"`
	}
	return func(c echo.Context) error {
		var req importReq
		if err := c.Bind(&req); err != nil {
			return jsonError(c, http.StatusBadRequest, "invalid json")
		}
		if req.Content == "" {
			return jsonError(c, http.StatusBadRequest, "content required")
		}

		catalogType, title := detectCatalogType(req.Content)
		if catalogType == "" {
			return jsonError(c, http.StatusBadRequest,
				"could not detect catalog type from content (expected ControlCatalog, ThreatCatalog, RiskCatalog, or GuidanceCatalog)")
		}

		catalogID := req.CatalogID
		if catalogID == "" {
			catalogID = detectCatalogID(req.Content)
		}

		if cs != nil {
			if err := cs.InsertCatalog(c.Request().Context(), Catalog{
				CatalogID:   catalogID,
				CatalogType: catalogType,
				Title:       title,
				Content:     req.Content,
				PolicyID:    req.PolicyID,
			}); err != nil {
				slog.Error("insert catalog failed", "error", err)
				return jsonError(c, http.StatusInternalServerError, "insert failed")
			}
		}

		parseCatalogStructuredRows(
			c.Request().Context(), catalogType, req.Content, catalogID, req.PolicyID, ctrlS, threatS, riskS, guidanceS,
		)

		return c.JSON(http.StatusCreated, map[string]string{
			"status":       "imported",
			"catalog_id":   catalogID,
			"catalog_type": catalogType,
		})
	}
}

func parseCatalogStructuredRows(
	ctx context.Context, catalogType, content, catalogID, policyID string,
	ctrlS ControlStore, threatS ThreatStore, riskS RiskStore, guidanceS GuidanceStore,
) {
	switch catalogType {
	case "ControlCatalog":
		if ctrlS == nil {
			return
		}
		controls, reqs, threats, err := gemarapkg.ParseControlCatalog(ctx, content, catalogID, policyID)
		if err != nil {
			slog.Warn("control catalog parse failed, structured rows skipped", "catalog_id", catalogID, "error", err)
			return
		}
		if len(controls) > 0 {
			if err := ctrlS.InsertControls(ctx, controls); err != nil {
				slog.Warn("insert controls failed", "catalog_id", catalogID, "error", err)
			}
		}
		if len(reqs) > 0 {
			if err := ctrlS.InsertAssessmentRequirements(ctx, reqs); err != nil {
				slog.Warn("insert assessment requirements failed", "catalog_id", catalogID, "error", err)
			}
		}
		if len(threats) > 0 {
			if err := ctrlS.InsertControlThreats(ctx, threats); err != nil {
				slog.Warn("insert control threats failed", "catalog_id", catalogID, "error", err)
			}
		}
		slog.Info("control catalog indexed", "catalog_id", catalogID, "controls", len(controls), "requirements", len(reqs), "control_threats", len(threats))

	case "ThreatCatalog":
		if threatS == nil {
			return
		}
		rows, err := gemarapkg.ParseThreatCatalog(ctx, content, catalogID, policyID)
		if err != nil {
			slog.Warn("threat catalog parse failed, structured rows skipped", "catalog_id", catalogID, "error", err)
			return
		}
		if len(rows) > 0 {
			if err := threatS.InsertThreats(ctx, rows); err != nil {
				slog.Warn("insert threats failed", "catalog_id", catalogID, "error", err)
			}
		}
		slog.Info("threat catalog indexed", "catalog_id", catalogID, "threats", len(rows))

	case "RiskCatalog":
		if riskS == nil {
			return
		}
		riskRows, linkRows, err := gemarapkg.ParseRiskCatalog(ctx, content, catalogID, policyID)
		if err != nil {
			slog.Warn("risk catalog parse failed, structured rows skipped", "catalog_id", catalogID, "error", err)
			return
		}
		if len(riskRows) > 0 {
			if err := riskS.InsertRisks(ctx, riskRows); err != nil {
				slog.Warn("insert risks failed", "catalog_id", catalogID, "error", err)
			}
		}
		if len(linkRows) > 0 {
			if err := riskS.InsertRiskThreats(ctx, linkRows); err != nil {
				slog.Warn("insert risk threats failed", "catalog_id", catalogID, "error", err)
			}
		}
		slog.Info("risk catalog indexed", "catalog_id", catalogID, "risks", len(riskRows), "risk_threats", len(linkRows))

	case "GuidanceCatalog":
		if guidanceS == nil {
			return
		}
		entries, err := gemarapkg.ParseGuidanceCatalog(ctx, content, catalogID)
		if err != nil {
			slog.Warn("guidance catalog parse failed, structured rows skipped", "catalog_id", catalogID, "error", err)
			return
		}
		if len(entries) > 0 {
			if err := guidanceS.InsertGuidanceEntries(ctx, entries); err != nil {
				slog.Warn("insert guidance entries failed", "catalog_id", catalogID, "error", err)
			}
		}
		slog.Info("guidance catalog indexed", "catalog_id", catalogID, "guidelines", len(entries))
	}
}

func detectCatalogType(content string) (catalogType, title string) {
	var meta struct {
		Title    string `json:"title" yaml:"title"`
		Metadata struct {
			Type string `json:"type" yaml:"type"`
		} `json:"metadata" yaml:"metadata"`
	}
	trim := strings.TrimSpace(content)
	var err error
	if strings.HasPrefix(trim, "{") {
		err = json.Unmarshal([]byte(trim), &meta)
	} else {
		err = gemarapkg.UnmarshalYAML([]byte(content), &meta)
	}
	if err != nil {
		return "", ""
	}
	switch meta.Metadata.Type {
	case "ControlCatalog", "ThreatCatalog", "RiskCatalog", "GuidanceCatalog":
		return meta.Metadata.Type, meta.Title
	default:
		return "", ""
	}
}

func detectCatalogID(content string) string {
	var meta struct {
		Metadata struct {
			ID string `json:"id" yaml:"id"`
		} `json:"metadata" yaml:"metadata"`
	}
	trim := strings.TrimSpace(content)
	var err error
	if strings.HasPrefix(trim, "{") {
		err = json.Unmarshal([]byte(trim), &meta)
	} else {
		err = gemarapkg.UnmarshalYAML([]byte(content), &meta)
	}
	if err != nil {
		return ""
	}
	return meta.Metadata.ID
}
