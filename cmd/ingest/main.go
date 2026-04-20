// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/fetcher"
	"github.com/goccy/go-yaml"

	"github.com/complytime/complytime-studio/internal/httputil"
	"github.com/complytime/complytime-studio/internal/ingest"
)

func main() {
	ctx := context.Background()

	source, f, err := resolveInput()
	if err != nil {
		log.Fatalf("resolve input: %v", err)
	}

	artifactType, err := detectType(ctx, f, source)
	if err != nil {
		log.Fatalf("detect artifact type: %v", err)
	}

	cfg := writerConfig()
	w, err := ingest.NewWriter(cfg)
	if err != nil {
		log.Fatalf("connect to clickhouse: %v", err)
	}
	defer w.Close()

	switch artifactType {
	case gemara.EvaluationLogArtifact:
		if err := ingestEvaluationLog(ctx, f, source, w); err != nil {
			log.Fatalf("ingest evaluation log: %v", err)
		}
	case gemara.EnforcementLogArtifact:
		if err := ingestEnforcementLog(ctx, f, source, w); err != nil {
			log.Fatalf("ingest enforcement log: %v", err)
		}
	default:
		log.Fatalf("unsupported artifact type: %s (expected EvaluationLog or EnforcementLog)", artifactType)
	}
}

func ingestEvaluationLog(ctx context.Context, f gemara.Fetcher, source string, w *ingest.Writer) error {
	evalLog, err := gemara.Load[gemara.EvaluationLog](ctx, f, source)
	if err != nil {
		return fmt.Errorf("load EvaluationLog: %w", err)
	}

	policyID := derivePolicyID(evalLog.Metadata.MappingReferences)

	rows, err := ingest.FlattenEvaluationLog(evalLog, policyID)
	if err != nil {
		return err
	}

	if err := w.InsertEvidenceRows(ctx, rows); err != nil {
		return err
	}

	log.Printf("ingested %d evidence rows from %s", len(rows), evalLog.Metadata.Id)
	return nil
}

func ingestEnforcementLog(ctx context.Context, f gemara.Fetcher, source string, w *ingest.Writer) error {
	enfLog, err := gemara.Load[gemara.EnforcementLog](ctx, f, source)
	if err != nil {
		return fmt.Errorf("load EnforcementLog: %w", err)
	}

	policyID := derivePolicyID(enfLog.Metadata.MappingReferences)

	rows, err := ingest.FlattenEnforcementLog(enfLog, policyID)
	if err != nil {
		return err
	}

	if err := w.InsertEvidenceRows(ctx, rows); err != nil {
		return err
	}

	log.Printf("ingested %d evidence rows from %s", len(rows), enfLog.Metadata.Id)
	return nil
}

// resolveInput returns a source path and Fetcher for the input.
// File-path arguments use the SDK's file fetcher; stdin uses a
// bytes-backed fetcher with a synthetic ".yaml" source for format detection.
func resolveInput() (string, gemara.Fetcher, error) {
	if len(os.Args) > 1 && os.Args[1] != "-" {
		return os.Args[1], &fetcher.File{}, nil
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", nil, fmt.Errorf("read stdin: %w", err)
	}
	return "stdin.yaml", &bytesFetcher{data: data}, nil
}

// bytesFetcher satisfies gemara.Fetcher by returning in-memory bytes.
type bytesFetcher struct {
	data []byte
}

func (b *bytesFetcher) Fetch(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(b.data)), nil
}

// detectType performs a lightweight header parse to determine the artifact type
// without fully decoding the document. Falls back to goccy/go-yaml for the
// header-only parse since gemara.Load requires a full decode.
func detectType(ctx context.Context, f gemara.Fetcher, source string) (gemara.ArtifactType, error) {
	rc, err := f.Fetch(ctx, source)
	if err != nil {
		return gemara.InvalidArtifact, fmt.Errorf("fetch source: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return gemara.InvalidArtifact, fmt.Errorf("read source: %w", err)
	}

	var hdr struct {
		Metadata gemara.Metadata `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(data, &hdr); err != nil {
		return gemara.InvalidArtifact, fmt.Errorf("parse YAML header: %w", err)
	}
	if hdr.Metadata.Type == gemara.InvalidArtifact {
		return gemara.InvalidArtifact, fmt.Errorf("missing or invalid metadata.type field")
	}
	return hdr.Metadata.Type, nil
}

// derivePolicyID scans mapping-references for a Policy-type reference.
// Falls back to the first reference ID if none is explicitly typed as Policy.
func derivePolicyID(refs []gemara.MappingReference) string {
	for _, r := range refs {
		if r.Title == "Policy" || r.Id == "policy" {
			return r.Id
		}
	}
	if len(refs) > 0 {
		return refs[0].Id
	}
	return ""
}

func writerConfig() ingest.WriterConfig {
	port, _ := strconv.Atoi(httputil.EnvOr("CLICKHOUSE_PORT", "9000"))
	return ingest.WriterConfig{
		Host:     httputil.EnvOr("CLICKHOUSE_HOST", "localhost"),
		Port:     port,
		User:     httputil.EnvOr("CLICKHOUSE_USER", "default"),
		Password: httputil.EnvOr("CLICKHOUSE_PASSWORD", ""),
		Database: httputil.EnvOr("CLICKHOUSE_DATABASE", "default"),
	}
}
