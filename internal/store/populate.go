// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	sdk "github.com/gemaraproj/go-gemara"
	goyaml "github.com/goccy/go-yaml"

	gemarapkg "github.com/complytime/complytime-studio/internal/gemara"
)

// PopulateMappingEntries backfills the mapping_entries table from existing
// mapping_documents. Skips documents that already have entries. Safe to call
// on every startup.
func PopulateMappingEntries(ctx context.Context, s MappingStore) error {
	docs, err := s.ListAllMappings(ctx)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return nil
	}

	var totalInserted int
	for _, doc := range docs {
		count, err := s.CountMappingEntries(ctx, doc.MappingID)
		if err != nil {
			slog.Warn("count mapping entries failed, skipping", "mapping_id", doc.MappingID, "error", err)
			continue
		}
		if count > 0 {
			continue
		}

		entries, parseErr := gemarapkg.ParseMappingEntries(doc.Content, doc.MappingID, doc.PolicyID, doc.Framework)
		if parseErr != nil {
			slog.Warn("retroactive parse failed, skipping", "mapping_id", doc.MappingID, "error", parseErr)
			continue
		}
		if len(entries) == 0 {
			continue
		}

		if err := s.InsertMappingEntries(ctx, entries); err != nil {
			slog.Warn("retroactive insert failed", "mapping_id", doc.MappingID, "error", err)
			continue
		}
		totalInserted += len(entries)
	}

	if totalInserted > 0 {
		slog.Info("mapping entries backfilled", "entries", totalInserted, "documents", len(docs))
	}
	return nil
}

// PopulateControls backfills controls, assessment_requirements, and
// control_threats from stored ControlCatalog content. Safe to call on startup.
func PopulateControls(ctx context.Context, cs CatalogStore, ctrlS ControlStore) error {
	catalogs, err := cs.ListCatalogs(ctx)
	if err != nil {
		return err
	}

	var totalControls int
	for _, cat := range catalogs {
		if cat.CatalogType != "ControlCatalog" {
			continue
		}
		count, err := ctrlS.CountControls(ctx, cat.CatalogID)
		if err != nil {
			slog.Warn("count controls failed, skipping", "catalog_id", cat.CatalogID, "error", err)
			continue
		}
		if count > 0 {
			continue
		}

		full, err := cs.GetCatalog(ctx, cat.CatalogID)
		if err != nil {
			slog.Warn("get catalog content failed, skipping", "catalog_id", cat.CatalogID, "error", err)
			continue
		}

		controls, reqs, threats, parseErr := gemarapkg.ParseControlCatalog(full.Content, cat.CatalogID, cat.PolicyID)
		if parseErr != nil {
			slog.Warn("retroactive control catalog parse failed, skipping", "catalog_id", cat.CatalogID, "error", parseErr)
			continue
		}
		if len(controls) > 0 {
			if err := ctrlS.InsertControls(ctx, controls); err != nil {
				slog.Warn("retroactive controls insert failed", "catalog_id", cat.CatalogID, "error", err)
				continue
			}
		}
		if len(reqs) > 0 {
			if err := ctrlS.InsertAssessmentRequirements(ctx, reqs); err != nil {
				slog.Warn("retroactive assessment requirements insert failed", "catalog_id", cat.CatalogID, "error", err)
			}
		}
		if len(threats) > 0 {
			if err := ctrlS.InsertControlThreats(ctx, threats); err != nil {
				slog.Warn("retroactive control threats insert failed", "catalog_id", cat.CatalogID, "error", err)
			}
		}
		totalControls += len(controls)
	}

	if totalControls > 0 {
		slog.Info("controls backfilled", "controls", totalControls)
	}
	return nil
}

// PopulateThreats backfills the threats table from stored ThreatCatalog
// content. Safe to call on startup.
func PopulateThreats(ctx context.Context, cs CatalogStore, threatS ThreatStore) error {
	catalogs, err := cs.ListCatalogs(ctx)
	if err != nil {
		return err
	}

	var totalThreats int
	for _, cat := range catalogs {
		if cat.CatalogType != "ThreatCatalog" {
			continue
		}
		count, err := threatS.CountThreats(ctx, cat.CatalogID)
		if err != nil {
			slog.Warn("count threats failed, skipping", "catalog_id", cat.CatalogID, "error", err)
			continue
		}
		if count > 0 {
			continue
		}

		full, err := cs.GetCatalog(ctx, cat.CatalogID)
		if err != nil {
			slog.Warn("get catalog content failed, skipping", "catalog_id", cat.CatalogID, "error", err)
			continue
		}

		rows, parseErr := gemarapkg.ParseThreatCatalog(full.Content, cat.CatalogID, cat.PolicyID)
		if parseErr != nil {
			slog.Warn("retroactive threat catalog parse failed, skipping", "catalog_id", cat.CatalogID, "error", parseErr)
			continue
		}
		if len(rows) > 0 {
			if err := threatS.InsertThreats(ctx, rows); err != nil {
				slog.Warn("retroactive threats insert failed", "catalog_id", cat.CatalogID, "error", err)
				continue
			}
		}
		totalThreats += len(rows)
	}

	if totalThreats > 0 {
		slog.Info("threats backfilled", "threats", totalThreats)
	}
	return nil
}

// PopulateRisks backfills risks and risk_threats from stored RiskCatalog
// content. Safe to call on startup.
func PopulateRisks(ctx context.Context, cs CatalogStore, riskS RiskStore) error {
	catalogs, err := cs.ListCatalogs(ctx)
	if err != nil {
		return err
	}

	var totalRisks int
	for _, cat := range catalogs {
		if cat.CatalogType != "RiskCatalog" {
			continue
		}
		count, err := riskS.CountRisks(ctx, cat.CatalogID)
		if err != nil {
			slog.Warn("count risks failed, skipping", "catalog_id", cat.CatalogID, "error", err)
			continue
		}
		if count > 0 {
			continue
		}

		full, err := cs.GetCatalog(ctx, cat.CatalogID)
		if err != nil {
			slog.Warn("get catalog content failed, skipping", "catalog_id", cat.CatalogID, "error", err)
			continue
		}

		riskRows, linkRows, parseErr := gemarapkg.ParseRiskCatalog(full.Content, cat.CatalogID, cat.PolicyID)
		if parseErr != nil {
			slog.Warn("retroactive risk catalog parse failed, skipping", "catalog_id", cat.CatalogID, "error", parseErr)
			continue
		}
		if len(riskRows) > 0 {
			if err := riskS.InsertRisks(ctx, riskRows); err != nil {
				slog.Warn("retroactive risks insert failed", "catalog_id", cat.CatalogID, "error", err)
				continue
			}
		}
		if len(linkRows) > 0 {
			if err := riskS.InsertRiskThreats(ctx, linkRows); err != nil {
				slog.Warn("retroactive risk threats insert failed", "catalog_id", cat.CatalogID, "error", err)
			}
		}
		totalRisks += len(riskRows)
	}

	if totalRisks > 0 {
		slog.Info("risks backfilled", "risks", totalRisks)
	}
	return nil
}

// registryCatalog pairs a well-known OCI reference with the expected catalog type.
type registryCatalog struct {
	Repo string
	Tag  string
	Type string
}

var defaultSeedCatalogs = []registryCatalog{
	{Repo: "complytime-studio/samples/control-catalog", Tag: "v1.0.0", Type: "ControlCatalog"},
	{Repo: "complytime-studio/samples/threat-catalog", Tag: "v1.0.0", Type: "ThreatCatalog"},
}

// PopulateCatalogsFromRegistry fetches well-known seed catalogs from an
// in-cluster OCI registry and inserts them into the catalogs table if
// they don't already exist. Safe to call on every startup.
func PopulateCatalogsFromRegistry(ctx context.Context, cs CatalogStore, ctrlS ControlStore, threatS ThreatStore, riskS RiskStore, registryAddr string) error {
	if registryAddr == "" {
		return nil
	}
	existing, err := cs.ListCatalogs(ctx)
	if err != nil {
		return fmt.Errorf("list catalogs: %w", err)
	}
	if len(existing) > 0 {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	var imported int

	for _, sc := range defaultSeedCatalogs {
		content, err := fetchOCILayer(ctx, client, registryAddr, sc.Repo, sc.Tag)
		if err != nil {
			slog.Warn("seed catalog fetch failed", "repo", sc.Repo, "error", err)
			continue
		}

		catalogID := detectMetadataID(content)
		if catalogID == "" {
			slog.Warn("seed catalog has no metadata.id, skipping", "repo", sc.Repo)
			continue
		}

		title := detectTitle(content)
		if err := cs.InsertCatalog(ctx, Catalog{
			CatalogID:   catalogID,
			CatalogType: sc.Type,
			Title:       title,
			Content:     content,
		}); err != nil {
			slog.Warn("seed catalog insert failed", "catalog_id", catalogID, "error", err)
			continue
		}

		parseCatalogStructuredRows(ctx, sc.Type, content, catalogID, "", ctrlS, threatS, riskS)
		imported++
		slog.Info("seed catalog imported from registry", "catalog_id", catalogID, "type", sc.Type)
	}

	if imported > 0 {
		slog.Info("seed catalogs imported", "count", imported)
	}
	return nil
}

// PopulateEffectiveControls resolves each stored policy's catalog imports
// against the catalogs table, applies policy-level overlays (exclusions,
// AR modifications), and inserts the effective controls. Runs after
// PopulateCatalogsFromRegistry. Safe on every startup.
func PopulateEffectiveControls(ctx context.Context, ps PolicyStore, cs CatalogStore, ctrlS ControlStore) error {
	policies, err := ps.ListPolicies(ctx)
	if err != nil {
		return fmt.Errorf("list policies: %w", err)
	}
	if len(policies) == 0 {
		return nil
	}

	catalogs, err := cs.ListCatalogs(ctx)
	if err != nil {
		return fmt.Errorf("list catalogs: %w", err)
	}

	var sdkCatalogs []sdk.ControlCatalog
	for _, cat := range catalogs {
		if cat.CatalogType != "ControlCatalog" {
			continue
		}
		full, err := cs.GetCatalog(ctx, cat.CatalogID)
		if err != nil {
			slog.Warn("load catalog for effective resolution", "catalog_id", cat.CatalogID, "error", err)
			continue
		}
		var cc sdk.ControlCatalog
		if err := goyaml.Unmarshal([]byte(full.Content), &cc); err != nil {
			slog.Warn("parse catalog for effective resolution", "catalog_id", cat.CatalogID, "error", err)
			continue
		}
		sdkCatalogs = append(sdkCatalogs, cc)
	}

	if len(sdkCatalogs) == 0 {
		return nil
	}

	var totalControls int
	for _, p := range policies {
		var pol sdk.Policy
		if err := goyaml.Unmarshal([]byte(p.Content), &pol); err != nil {
			slog.Warn("parse policy for effective resolution", "policy_id", p.PolicyID, "error", err)
			continue
		}

		if len(pol.Imports.Catalogs) == 0 {
			continue
		}

		effective, err := gemarapkg.ResolveEffectiveControls(pol, sdkCatalogs)
		if err != nil {
			slog.Warn("effective control resolution failed", "policy_id", p.PolicyID, "error", err)
			continue
		}

		controls, reqs, threats := gemarapkg.ExtractControlRows(effective, p.PolicyID)
		if len(controls) == 0 {
			continue
		}

		first := controls[0].CatalogID
		count, _ := ctrlS.CountControls(ctx, first)
		if count > 0 {
			continue
		}

		if err := ctrlS.InsertControls(ctx, controls); err != nil {
			slog.Warn("effective controls insert failed", "policy_id", p.PolicyID, "error", err)
			continue
		}
		if len(reqs) > 0 {
			if err := ctrlS.InsertAssessmentRequirements(ctx, reqs); err != nil {
				slog.Warn("effective ARs insert failed", "policy_id", p.PolicyID, "error", err)
			}
		}
		if len(threats) > 0 {
			if err := ctrlS.InsertControlThreats(ctx, threats); err != nil {
				slog.Warn("effective threats insert failed", "policy_id", p.PolicyID, "error", err)
			}
		}
		totalControls += len(controls)
		slog.Info("effective controls resolved", "policy_id", p.PolicyID, "controls", len(controls), "requirements", len(reqs))
	}

	if totalControls > 0 {
		slog.Info("effective controls backfilled", "total_controls", totalControls)
	}
	return nil
}

// PopulatePolicyCriteria extracts criteria and assessment-requirements
// directly from each policy's YAML content, inserting rows into the
// controls and assessment_requirements tables. This handles policies
// that embed criteria inline (no external catalog import needed).
// Safe to call on every startup; skips policies whose criteria have
// already been populated.
func PopulatePolicyCriteria(ctx context.Context, ps PolicyStore, ctrlS ControlStore) error {
	policies, err := ps.ListPolicies(ctx)
	if err != nil {
		return fmt.Errorf("list policies: %w", err)
	}

	type parsedCriteria struct {
		Criteria []struct {
			ID                     string `yaml:"id"`
			Title                  string `yaml:"title"`
			Description            string `yaml:"description"`
			CatalogRef             string `yaml:"catalog-ref"`
			AssessmentRequirements []struct {
				ID          string `yaml:"id"`
				Description string `yaml:"description"`
			} `yaml:"assessment-requirements"`
		} `yaml:"criteria"`
	}

	var totalControls int
	for _, p := range policies {
		full, err := ps.GetPolicy(ctx, p.PolicyID)
		if err != nil {
			slog.Warn("get policy content for criteria", "policy_id", p.PolicyID, "error", err)
			continue
		}

		var pol parsedCriteria
		if err := goyaml.Unmarshal([]byte(full.Content), &pol); err != nil {
			slog.Warn("parse policy criteria", "policy_id", p.PolicyID, "error", err)
			continue
		}
		if len(pol.Criteria) == 0 {
			continue
		}

		catalogID := p.PolicyID
		count, _ := ctrlS.CountControls(ctx, catalogID)
		if count > 0 {
			continue
		}

		var controls []gemarapkg.ControlRow
		var reqs []gemarapkg.AssessmentRequirementRow

		for _, c := range pol.Criteria {
			controls = append(controls, gemarapkg.ControlRow{
				CatalogID: catalogID,
				ControlID: c.ID,
				Title:     c.Title,
				Objective: c.Description,
				State:     "Active",
				PolicyID:  p.PolicyID,
			})
			for _, ar := range c.AssessmentRequirements {
				reqs = append(reqs, gemarapkg.AssessmentRequirementRow{
					CatalogID:     catalogID,
					ControlID:     c.ID,
					RequirementID: ar.ID,
					Text:          ar.Description,
					State:         "Active",
				})
			}
		}

		if len(controls) > 0 {
			if err := ctrlS.InsertControls(ctx, controls); err != nil {
				slog.Warn("policy criteria controls insert failed", "policy_id", p.PolicyID, "error", err)
				continue
			}
		}
		if len(reqs) > 0 {
			if err := ctrlS.InsertAssessmentRequirements(ctx, reqs); err != nil {
				slog.Warn("policy criteria ARs insert failed", "policy_id", p.PolicyID, "error", err)
			}
		}
		totalControls += len(controls)
		slog.Info("policy criteria extracted", "policy_id", p.PolicyID, "controls", len(controls), "requirements", len(reqs))
	}

	if totalControls > 0 {
		slog.Info("policy criteria backfilled", "total_controls", totalControls)
	}
	return nil
}

// fetchOCILayer retrieves the first layer blob from an OCI manifest
// at the given registry/repo:tag. Suitable for single-layer artifacts.
func fetchOCILayer(ctx context.Context, client *http.Client, registryAddr, repo, tag string) (string, error) {
	manifestURL := fmt.Sprintf("http://%s/v2/%s/manifests/%s", registryAddr, repo, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json, application/vnd.docker.distribution.manifest.v2+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch manifest: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("manifest %s: %s", manifestURL, resp.Status)
	}

	var manifest struct {
		Layers []struct {
			Digest    string `json:"digest"`
			MediaType string `json:"mediaType"`
		} `json:"layers"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&manifest); err != nil {
		return "", fmt.Errorf("decode manifest: %w", err)
	}
	if len(manifest.Layers) == 0 {
		return "", fmt.Errorf("no layers in manifest for %s/%s:%s", registryAddr, repo, tag)
	}

	digest := manifest.Layers[0].Digest
	blobURL := fmt.Sprintf("http://%s/v2/%s/blobs/%s", registryAddr, repo, digest)
	blobReq, err := http.NewRequestWithContext(ctx, http.MethodGet, blobURL, nil)
	if err != nil {
		return "", err
	}

	blobResp, err := client.Do(blobReq)
	if err != nil {
		return "", fmt.Errorf("fetch blob: %w", err)
	}
	defer func() { _ = blobResp.Body.Close() }()
	if blobResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("blob %s: %s", blobURL, blobResp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(blobResp.Body, 8<<20))
	if err != nil {
		return "", fmt.Errorf("read blob: %w", err)
	}
	return string(data), nil
}

func detectMetadataID(content string) string {
	var meta struct {
		Metadata struct {
			ID string `yaml:"id"`
		} `yaml:"metadata"`
	}
	if err := goyaml.Unmarshal([]byte(content), &meta); err != nil {
		return ""
	}
	return meta.Metadata.ID
}

func detectTitle(content string) string {
	var meta struct {
		Title string `yaml:"title"`
	}
	if err := goyaml.Unmarshal([]byte(content), &meta); err != nil {
		return ""
	}
	return meta.Title
}
