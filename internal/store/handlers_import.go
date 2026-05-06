// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	gemara "github.com/gemaraproj/go-gemara"
	gemarabundle "github.com/gemaraproj/go-gemara/bundle"
	sdk "github.com/gemaraproj/go-gemara"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	gemarapkg "github.com/complytime/complytime-studio/internal/gemara"
)

const maxUnifiedImportBytes = 10 << 20

func registerImportRoute(g *echo.Group, s Stores) {
	g.POST("/import", importArtifactHandler(s))
}

// importArtifactHandler supports two modes:
//  1. OCI reference: JSON body with {"reference": "ghcr.io/org/bundle:tag"}
//  2. Raw body: YAML or JSON artifact content with metadata.type auto-detection
func importArtifactHandler(s Stores) echo.HandlerFunc {
	return func(c echo.Context) error {
		ct := c.Request().Header.Get("Content-Type")

		if strings.HasPrefix(ct, "application/json") {
			body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxUnifiedImportBytes))
			if err != nil {
				return jsonError(c, http.StatusBadRequest, "read body failed")
			}
			var probe struct {
				Reference string `json:"reference"`
			}
			if json.Unmarshal(body, &probe) == nil && strings.TrimSpace(probe.Reference) != "" {
				return ociImport(c, s, strings.TrimSpace(probe.Reference))
			}
			return rawBodyImport(c, s, string(body))
		}

		body, err := io.ReadAll(io.LimitReader(c.Request().Body, maxUnifiedImportBytes))
		if err != nil {
			return jsonError(c, http.StatusBadRequest, "read body failed")
		}
		return rawBodyImport(c, s, string(body))
	}
}

// ── OCI reference import ────────────────────────────────────────────────────

type ociImportedArtifact struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type ociImportResponse struct {
	Imported []ociImportedArtifact `json:"imported"`
	Digest   string                `json:"digest,omitempty"`
}

func ociImport(c echo.Context, s Stores, ref string) error {
	if s.Registry == nil {
		return jsonError(c, http.StatusServiceUnavailable, "registry not configured")
	}

	repo, err := s.Registry.Repository(ref)
	if err != nil {
		return jsonError(c, http.StatusForbidden, err.Error())
	}

	ctx := c.Request().Context()
	bundle, err := gemarabundle.Unpack(ctx, repo, repo.Reference.Reference)
	if err != nil {
		slog.Error("oci import unpack failed", "reference", ref, "error", err)
		return jsonError(c, http.StatusBadGateway, "failed to pull bundle: "+err.Error())
	}

	var resp ociImportResponse
	resp.Digest = bundle.Etag

	allFiles := append(bundle.Files, bundle.Imports...)
	for _, f := range allFiles {
		art, err := storeArtifactFile(ctx, s, f)
		if err != nil {
			slog.Warn("import artifact failed", "name", f.Name, "error", err)
			continue
		}
		resp.Imported = append(resp.Imported, art)
	}

	if len(resp.Imported) == 0 {
		return jsonError(c, http.StatusBadRequest, "bundle contained no importable artifacts")
	}

	return c.JSON(http.StatusCreated, resp)
}

func storeArtifactFile(ctx context.Context, s Stores, f gemarabundle.File) (ociImportedArtifact, error) {
	content := string(f.Data)
	detected, err := gemara.DetectType(f.Data)
	if err != nil {
		return ociImportedArtifact{}, err
	}
	artType := detected.String()

	switch artType {
	case "Policy":
		return storePolicyFromContent(ctx, s.Policies, s.Controls, content)
	case "MappingDocument":
		return storeMappingFromContent(ctx, s.Mappings, content)
	case "ControlCatalog", "ThreatCatalog", "RiskCatalog", "GuidanceCatalog":
		return storeCatalogFromContent(ctx, s, artType, content)
	default:
		slog.Debug("skipping unsupported artifact type", "type", artType, "name", f.Name)
		return ociImportedArtifact{Type: artType, ID: "", Name: f.Name}, nil
	}
}

func storePolicyFromContent(ctx context.Context, ps PolicyStore, ctrlS ControlStore, content string) (ociImportedArtifact, error) {
	var pol sdk.Policy
	if err := gemarapkg.UnmarshalYAML([]byte(content), &pol); err != nil {
		return ociImportedArtifact{}, err
	}
	title := strings.TrimSpace(pol.Title)
	if title == "" {
		title = strings.TrimSpace(pol.Metadata.Id)
	}
	if title == "" {
		title = "Imported policy"
	}
	pid := strings.TrimSpace(pol.Metadata.Id)
	if pid == "" {
		pid = uuid.NewString()
	}
	p := Policy{
		PolicyID: pid,
		Title:    title,
		Version:  pol.Metadata.Version,
		Content:  content,
	}
	if err := ps.InsertPolicy(ctx, p); err != nil {
		return ociImportedArtifact{}, err
	}
	if ctrlS != nil {
		if _, err := ExtractPolicyCriteria(ctx, p.PolicyID, content, ctrlS); err != nil {
			slog.Warn("inline criteria extraction failed", "policy_id", p.PolicyID, "error", err)
		}
	}
	return ociImportedArtifact{Type: "Policy", ID: p.PolicyID, Name: title}, nil
}

func storeMappingFromContent(ctx context.Context, ms MappingStore, content string) (ociImportedArtifact, error) {
	var doc sdk.MappingDocument
	if err := gemarapkg.UnmarshalYAML([]byte(content), &doc); err != nil {
		return ociImportedArtifact{}, err
	}
	src := strings.TrimSpace(doc.SourceReference.ReferenceId)
	tgt := strings.TrimSpace(doc.TargetReference.ReferenceId)
	mid := strings.TrimSpace(doc.Metadata.Id)
	if mid == "" {
		mid = uuid.NewString()
	}
	m := MappingDocument{
		MappingID:       mid,
		SourceCatalogID: src,
		TargetCatalogID: tgt,
		Framework:       strings.TrimSpace(doc.Title),
		Content:         content,
	}
	if err := ms.InsertMapping(ctx, m); err != nil {
		return ociImportedArtifact{}, err
	}
	entries, parseErr := gemarapkg.ParseMappingEntries(content, mid, src, tgt, m.Framework)
	if parseErr != nil {
		slog.Warn("mapping parse failed", "mapping_id", mid, "error", parseErr)
	} else if len(entries) > 0 {
		if err := ms.InsertMappingEntries(ctx, entries); err != nil {
			slog.Warn("insert mapping entries failed", "mapping_id", mid, "error", err)
		}
	}
	return ociImportedArtifact{Type: "MappingDocument", ID: mid, Name: m.Framework}, nil
}

func storeCatalogFromContent(ctx context.Context, s Stores, artType, content string) (ociImportedArtifact, error) {
	_, title := detectCatalogType(content)
	catalogID := detectCatalogID(content)
	if catalogID == "" {
		catalogID = uuid.NewString()
	}
	if s.Catalogs != nil {
		if err := s.Catalogs.InsertCatalog(ctx, Catalog{
			CatalogID:   catalogID,
			CatalogType: artType,
			Title:       title,
			Content:     content,
		}); err != nil {
			return ociImportedArtifact{}, err
		}
	}
	parseCatalogStructuredRows(ctx, artType, content, catalogID, "", s.Controls, s.Threats, s.Risks, s.Guidance)
	return ociImportedArtifact{Type: artType, ID: catalogID, Name: title}, nil
}

// ── Raw body import (file drop from ImportOverlay) ──────────────────────────

func rawBodyImport(c echo.Context, s Stores, content string) error {
	if strings.TrimSpace(content) == "" {
		return jsonError(c, http.StatusBadRequest, "empty body")
	}
	typ, err := detectGemaraArtifactMetadataType(content)
	if err != nil || typ == "" {
		return jsonError(c, http.StatusBadRequest, "could not detect metadata.type")
	}
	switch typ {
	case "Policy":
		return importPolicyFromArtifactBody(c, s.Policies, s.Controls, content)
	case "MappingDocument":
		return importMappingFromArtifactBody(c, s.Mappings, content)
	case "ControlCatalog", "ThreatCatalog", "RiskCatalog", "GuidanceCatalog":
		return importCatalogFromArtifactBody(c, s, content)
	default:
		return jsonError(c, http.StatusBadRequest, "unsupported metadata.type: "+typ)
	}
}

func detectGemaraArtifactMetadataType(content string) (string, error) {
	trim := strings.TrimSpace(content)
	var meta struct {
		Metadata struct {
			Type string `json:"type" yaml:"type"`
		} `json:"metadata" yaml:"metadata"`
	}
	if strings.HasPrefix(trim, "{") {
		if err := json.Unmarshal([]byte(trim), &meta); err != nil {
			return "", err
		}
		return meta.Metadata.Type, nil
	}
	if err := gemarapkg.UnmarshalYAML([]byte(content), &meta); err != nil {
		return "", err
	}
	return meta.Metadata.Type, nil
}

func importPolicyFromArtifactBody(
	c echo.Context,
	ps PolicyStore,
	ctrlS ControlStore,
	content string,
) error {
	var pol sdk.Policy
	trim := strings.TrimSpace(content)
	var err error
	if strings.HasPrefix(trim, "{") {
		err = json.Unmarshal([]byte(trim), &pol)
	} else {
		err = gemarapkg.UnmarshalYAML([]byte(content), &pol)
	}
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid policy document")
	}
	title := strings.TrimSpace(pol.Title)
	if title == "" {
		title = strings.TrimSpace(pol.Metadata.Id)
	}
	if title == "" {
		title = "Imported policy"
	}
	pid := strings.TrimSpace(pol.Metadata.Id)
	if pid == "" {
		pid = uuid.NewString()
	}
	p := Policy{
		PolicyID:     pid,
		Title:        title,
		Version:      pol.Metadata.Version,
		OCIReference: "",
		Content:      content,
	}
	if err := ps.InsertPolicy(c.Request().Context(), p); err != nil {
		slog.Error("insert policy failed", "error", err)
		return jsonError(c, http.StatusInternalServerError, "insert failed")
	}
	if ctrlS != nil {
		if _, err := ExtractPolicyCriteria(c.Request().Context(), p.PolicyID, content, ctrlS); err != nil {
			slog.Warn("inline criteria extraction failed", "policy_id", p.PolicyID, "error", err)
		}
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "imported", "policy_id": p.PolicyID})
}

func importMappingFromArtifactBody(c echo.Context, ms MappingStore, content string) error {
	var doc sdk.MappingDocument
	trim := strings.TrimSpace(content)
	var err error
	if strings.HasPrefix(trim, "{") {
		err = json.Unmarshal([]byte(trim), &doc)
	} else {
		err = gemarapkg.UnmarshalYAML([]byte(content), &doc)
	}
	if err != nil {
		return jsonError(c, http.StatusBadRequest, "invalid mapping document")
	}
	src := strings.TrimSpace(doc.SourceReference.ReferenceId)
	tgt := strings.TrimSpace(doc.TargetReference.ReferenceId)
	if src == "" || tgt == "" {
		return jsonError(c, http.StatusBadRequest,
			"source-reference and target-reference reference-id required")
	}
	mid := strings.TrimSpace(doc.Metadata.Id)
	if mid == "" {
		mid = uuid.NewString()
	}
	m := MappingDocument{
		MappingID:       mid,
		SourceCatalogID: src,
		TargetCatalogID: tgt,
		Framework:       strings.TrimSpace(doc.Title),
		Content:         content,
	}
	if err := ms.InsertMapping(c.Request().Context(), m); err != nil {
		slog.Error("insert mapping failed", "error", err)
		return jsonError(c, http.StatusInternalServerError, "insert failed")
	}
	entries, parseErr := gemarapkg.ParseMappingEntries(
		content, m.MappingID, m.SourceCatalogID, m.TargetCatalogID, m.Framework,
	)
	if parseErr != nil {
		slog.Warn("mapping YAML parse failed, structured entries skipped",
			"mapping_id", m.MappingID, "error", parseErr)
	} else if len(entries) > 0 {
		if err := ms.InsertMappingEntries(c.Request().Context(), entries); err != nil {
			slog.Warn("insert mapping entries failed", "mapping_id", m.MappingID, "error", err)
		} else {
			slog.Info("mapping entries stored", "mapping_id", m.MappingID, "count", len(entries))
		}
	}
	return c.JSON(http.StatusCreated, map[string]string{"status": "imported", "mapping_id": m.MappingID})
}

func importCatalogFromArtifactBody(c echo.Context, s Stores, content string) error {
	cs := s.Catalogs
	catalogType, title := detectCatalogType(content)
	if catalogType == "" {
		return jsonError(c, http.StatusBadRequest,
			"could not detect catalog type from content (expected ControlCatalog, ThreatCatalog, RiskCatalog, or GuidanceCatalog)")
	}
	catalogID := detectCatalogID(content)
	if catalogID == "" {
		catalogID = uuid.NewString()
	}
	if cs != nil {
		if err := cs.InsertCatalog(c.Request().Context(), Catalog{
			CatalogID:   catalogID,
			CatalogType: catalogType,
			Title:       title,
			Content:     content,
			PolicyID:    "",
		}); err != nil {
			slog.Error("insert catalog failed", "error", err)
			return jsonError(c, http.StatusInternalServerError, "insert failed")
		}
	}
	parseCatalogStructuredRows(
		c.Request().Context(), catalogType, content, catalogID, "", s.Controls, s.Threats, s.Risks, s.Guidance,
	)
	return c.JSON(http.StatusCreated, map[string]string{
		"status":       "imported",
		"catalog_id":   catalogID,
		"catalog_type": catalogType,
	})
}
