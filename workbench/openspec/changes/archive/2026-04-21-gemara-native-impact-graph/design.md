## Context

Studio stores mapping documents as YAML blobs in `mapping_documents.content`. The `importMappingHandler` writes `(mapping_id, policy_id, framework, content)` and nothing else. Evidence rows reference `control_id` and `policy_id` but have no join path to framework objectives like SOC 2 CC8.1.

The impact question — "which certifications are affected by this failure?" — requires parsing the mapping YAML, extracting `source.control-id → targets[].reference`, and joining against evidence. Today the assistant does this ad-hoc by reading the YAML blob and reasoning over it. This is fragile and non-deterministic.

ADR: [Impact Graph: Control Failure Blast Radius](../../docs/decisions/impact-graph.md)

## Goals / Non-Goals

**Goals:**

- Parse Gemara mapping YAML at import time into a structured `mapping_entries` table.
- Enable SQL joins between `evidence` and `mapping_entries` without runtime YAML parsing.
- Add impact query patterns to the assistant's evidence-schema skill.
- Retroactively populate `mapping_entries` for existing mapping documents on schema migration.

**Non-Goals:**

- Effective Policy resolution (blocked on [go-gemara#64](https://github.com/gemaraproj/go-gemara/issues/64)).
- Parsing Policy YAML into a `policy_criteria` table.
- Backfilling the `frameworks`/`requirements` Array columns on evidence rows.
- UI changes — impact is surfaced through the assistant, not a dedicated view.
- Drift detection or Grafana integration.

## Decisions

### 1. New `mapping_entries` table in ClickHouse

```sql
CREATE TABLE IF NOT EXISTS mapping_entries (
    mapping_id String,
    policy_id String,
    control_id String,
    requirement_id String DEFAULT '',
    framework String,
    reference String,
    strength UInt8 DEFAULT 0,
    confidence String DEFAULT '',
    imported_at DateTime64(3) DEFAULT now64(3)
) ENGINE = ReplacingMergeTree(imported_at)
ORDER BY (policy_id, framework, control_id, reference)
```

**Why `ReplacingMergeTree`:** Re-importing the same mapping document replaces old entries on merge (keyed by the ORDER BY tuple). No manual cleanup needed.

**Why not a materialized view:** The source data is a YAML blob, not structured columns. ClickHouse cannot parse YAML in a MV expression. Parsing must happen in Go at import time.

### 2. YAML parsing in the import handler

The `importMappingHandler` gains a second write step. After inserting into `mapping_documents`, it parses `content` as YAML and batch-inserts into `mapping_entries`.

The mapping YAML structure (from Gemara schema):

```yaml
mappings:
  - source:
      control-id: BP-4
      requirement-id: BP-4.01
    targets:
      - reference: CC8.1
        strength: 8
        confidence-level: High
      - reference: CC6.1
        strength: 7
        confidence-level: Medium
```

Each `(source, target)` pair becomes one `mapping_entries` row. For the demo mapping (5 controls, 7 total target refs), this produces 7 rows.

**Alternative considered:** Parse in a background job. Rejected — the mapping is small (tens of rows) and parsing is fast. Synchronous is simpler and guarantees consistency.

### 3. Store interface extension

Add `MappingEntryStore` interface and `MappingEntry` type to `internal/store/`. The `MappingStore` interface gains no new methods — `mapping_entries` is an implementation detail of `InsertMapping`, not a new domain concept.

**Alternative considered:** Separate `MappingEntryStore` interface exposed to handlers. Rejected — entries are always written as part of mapping import, never independently. Keep the seam internal to the store.

### 4. Impact query via skill, not MCP tool

The impact join is a single SQL query:

```sql
SELECT e.control_id, e.target_name, e.eval_result,
       m.framework, m.reference, m.strength, m.confidence
FROM evidence e
JOIN mapping_entries m
  ON e.policy_id = m.policy_id AND e.control_id = m.control_id
WHERE e.policy_id = ?
  AND e.eval_result IN ('Failed', 'Not Run')
  AND e.collected_at BETWEEN ? AND ?
ORDER BY m.framework, m.reference, m.strength DESC
```

This goes into `skills/evidence-schema/SKILL.md` as a query pattern. The assistant uses `run_select_query` — no new MCP tool.

**Alternative considered:** Dedicated `blast_radius` MCP tool. Rejected — adds a tool to maintain when a query pattern achieves the same result. The assistant already knows how to run SQL.

### 5. Retroactive population on schema migration

The ClickHouse init code already runs `CREATE TABLE IF NOT EXISTS` on startup. After creating `mapping_entries`, the gateway queries all existing `mapping_documents`, parses each, and inserts entries. This is idempotent (`ReplacingMergeTree` deduplicates).

**Alternative considered:** Manual migration script. Rejected — the gateway already owns schema init. Adding retroactive population there means every deploy is self-healing.

## Risks / Trade-offs

| Risk | Mitigation |
|:--|:--|
| YAML parsing failure blocks mapping import | Parse after blob insert. If parsing fails, log a warning and continue. The blob is always stored. `mapping_entries` is best-effort enrichment. |
| Mapping YAML schema evolves | Parse only the `mappings[].source` and `mappings[].targets` fields. Ignore unknown fields. Defensive parsing. |
| Large mapping documents (hundreds of entries) | Batch insert. ClickHouse handles thousands of rows per batch efficiently. Not a realistic concern for compliance mappings. |
| Retroactive population on every restart | Skip if `mapping_entries` already has rows for a given `mapping_id`. Check before parsing. |
