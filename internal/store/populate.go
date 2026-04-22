// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"log/slog"

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
