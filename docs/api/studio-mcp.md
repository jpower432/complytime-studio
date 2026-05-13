<!-- SPDX-License-Identifier: Apache-2.0 -->

# studio-mcp — MCP resources and tools

`studio-mcp` (`cmd/studio-mcp`) exposes ComplyTime Platform data to agents as MCP resources (`studio://…`) and tools. All resource payloads use `Content-Type: application/json` (text JSON in MCP resource contents).

## CLI usage

**Flags:** `--transport stdio|http`, `--port` (HTTP only), `--postgres-url <URL>`.

**Environment:** `POSTGRES_URL` is used when `--postgres-url` is omitted.

```bash
# stdio (typical sidecar)
studio-mcp --transport stdio --postgres-url postgres://user:pass@postgres:5432/studio?sslmode=disable

# HTTP listener
studio-mcp --transport http --port 3000 --postgres-url "$POSTGRES_URL"
```

## Resources

| URI / template | Params | Returns |
|:--|:--|:--|
| `studio://policies` | — | JSON array of policy list rows (metadata columns). |
| `studio://policies/{id}` | Path `id` = policy id | Single policy object (full content/YAML payload per store). `404`-style MCP not-found if missing. |
| `studio://evidence{?policy_id,limit,offset}` | `policy_id` optional filter; `limit` default 100; `offset` default 0 (clamped like gateway limits, max 1000) | JSON array of evidence records. |
| `studio://posture{?policy_id}` | `policy_id` optional | JSON array of posture aggregate rows; filtered client-side when `policy_id` set. |
| `studio://audit-logs{?policy_id,limit}` | **`policy_id` required**; `limit` optional (default 100 via clamp) | JSON array of audit logs for the policy. |
| `studio://mappings{?source_catalog}` | `source_catalog` optional (matches mapping framework / source catalog) | JSON array of mapping documents. |
| `studio://catalogs` | — | JSON array of catalog index rows (not full artifact blobs). |
| `studio://threats{?catalog_id}` | `catalog_id` optional | JSON array of threat rows (default limit 100). |
| `studio://risks{?catalog_id}` | `catalog_id` optional | JSON array of risk rows (default limit 100). |

## Tools

### `ingest_evidence`

Insert evidence rows into PostgreSQL (via the platform store).

| Field | Type | Required |
|:--|:--|:--|
| Arguments | JSON array of evidence objects **or** `{"records":[…]}` | Yes |

Each record must include `policy_id`, `target_id`, `control_id`, and `collected_at` (RFC3339 timestamp).

**Success:** structured content `{"inserted": <int>}`.

**Error:** `IsError` tool result with JSON text `{"error":"…"}` or `{"errors":["row N: …"]}` for validation failures.

### `save_draft_audit_log`

Parse Gemara AuditLog YAML and persist a draft for human review.

| Field | Type | Required |
|:--|:--|:--|
| `policy_id` | string | Yes |
| `yaml` | string | Yes |
| `agent_reasoning` | string | No |
| `model` | string | No |
| `prompt_version` | string | No |

**Success:** `{"status":"drafted","draft_id":"<id>"}`.

**Failure:** tool error (invalid YAML, missing fields, store error).

## Example snippets

Read policies list (conceptual MCP JSON-RPC / client-specific):

```json
{"method":"resources/read","params":{"uri":"studio://policies"}}
```

Evidence page for one policy:

```
studio://evidence?policy_id=policy-ampel&limit=50&offset=0
```

Promote-shaped workflow stays on Platform REST; agents draft via tool:

```json
{
  "name": "save_draft_audit_log",
  "arguments": {
    "policy_id": "policy-ampel",
    "yaml": "metadata:\n  kind: AuditLog\n..."
  }
}
```

Bulk ingest:

```json
{
  "name": "ingest_evidence",
  "arguments": {
    "records": [
      {
        "policy_id": "policy-ampel",
        "target_id": "repo-1",
        "control_id": "CC6.1",
        "collected_at": "2026-05-01T12:00:00Z"
      }
    ]
  }
}
```
