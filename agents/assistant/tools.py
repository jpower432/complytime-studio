# SPDX-License-Identifier: Apache-2.0

"""LangChain tools for the ComplyTime Studio assistant.

publish_audit_log: persists a validated AuditLog YAML as a draft to the
Gateway's internal endpoint for human review and promotion.

SQL guard logic is applied as a pre-invocation check on query_database
via the MCP adapter's tool_filter mechanism.
"""

import json
import logging
import os
import re

import httpx
import yaml
from langchain_core.tools import tool

logger = logging.getLogger(__name__)

GATEWAY_URL = os.environ.get("GATEWAY_URL", "http://studio-gateway:8080")
INTERNAL_GATEWAY_URL = os.environ.get(
    "INTERNAL_GATEWAY_URL", "http://studio-gateway-internal:8081"
)

_SQL_WRITE = re.compile(
    r"\b(INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|TRUNCATE|GRANT|REVOKE|EXEC)\b",
    re.IGNORECASE,
)


def _slugify(text: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")[:64]


@tool
async def publish_audit_log(
    yaml_content: str,
    policy_id: str = "",
    reasoning: str = "",
) -> dict:
    """Publish a validated AuditLog YAML as a draft for human review.

    Call this AFTER validate_gemara_artifact succeeds. The AuditLog is persisted
    as a draft to the internal Gateway endpoint. A human reviewer must promote
    it to an official record via the workbench.

    Args:
        yaml_content: The complete, validated AuditLog YAML string.
        policy_id: The policy_id from the policies table (e.g. 'ampel-branch-protection').
        reasoning: JSON object mapping result IDs to classification reasoning.
    """
    try:
        doc = yaml.safe_load(yaml_content)
    except yaml.YAMLError as e:
        return {"error": f"Invalid YAML: {e}"}

    if not isinstance(doc, dict):
        return {"error": "YAML root must be a mapping"}

    metadata = doc.get("metadata", {})
    artifact_type = metadata.get("type", "")
    if artifact_type != "AuditLog":
        return {"error": f"Expected metadata.type=AuditLog, got '{artifact_type}'"}

    artifact_id = metadata.get("id", "audit-log")
    filename = f"{_slugify(artifact_id)}.yaml"

    model_name = os.environ.get("MODEL_NAME", "unknown")
    if not policy_id:
        policy_id = _extract_policy_id(doc)
    agent_reasoning = reasoning if reasoning else _extract_reasoning(doc)

    payload = {
        "policy_id": policy_id,
        "content": yaml_content,
        "agent_reasoning": agent_reasoning,
        "model": model_name,
        "prompt_version": "langgraph-v1",
    }

    try:
        async with httpx.AsyncClient(timeout=15.0) as client:
            resp = await client.post(
                f"{INTERNAL_GATEWAY_URL}/internal/draft-audit-logs",
                content=json.dumps(payload),
                headers={"Content-Type": "application/json"},
            )
        if resp.status_code == 201:
            result = resp.json()
            draft_id = result.get("draft_id", "")
            logger.info("persisted draft audit log %s (policy=%s)", draft_id, policy_id)
            return {
                "status": "drafted",
                "filename": filename,
                "draft_id": draft_id,
                "note": "Draft saved for human review. A reviewer must promote it to official record.",
            }
        else:
            logger.error("gateway rejected draft: %s %s", resp.status_code, resp.text[:200])
            return {"error": f"Gateway returned {resp.status_code}: {resp.text[:200]}"}
    except Exception as e:
        logger.error("failed to persist draft audit log: %s", e)
        return {"error": f"Failed to persist draft: {e}"}


def validate_sql_query(sql: str) -> str | None:
    """Return error message if SQL contains write statements, else None."""
    if _SQL_WRITE.search(sql):
        return "Only SELECT queries are allowed. Write operations are blocked."
    return None


def sql_guard_filter(tool_name: str, args: dict) -> dict | None:
    """Pre-invocation guard for query_database — blocks write SQL.

    Returns a dict error response if blocked, None to allow.
    Used as a tool call interceptor in the graph's tool node.
    """
    if tool_name != "query_database":
        return None
    sql = args.get("query", "") or args.get("sql", "")
    error = validate_sql_query(sql)
    if error:
        logger.warning("Blocked write SQL in query_database: %s", sql[:200])
        return {"error": error}
    return None


def build_tools() -> list:
    """Return the list of local LangChain tools for the assistant."""
    return [publish_audit_log]


def _extract_policy_id(doc: dict) -> str:
    scope = doc.get("scope", {})
    for key in ("policy-id", "policy_id"):
        val = scope.get(key, "")
        if val:
            return val

    metadata = doc.get("metadata", {})
    for key in ("policy-id", "policy_id"):
        val = metadata.get(key, "")
        if val:
            return val

    meta_id = metadata.get("id", "")
    known = _fetch_known_policy_ids()
    if known and meta_id:
        for pid in sorted(known, key=len, reverse=True):
            if pid in meta_id:
                return pid

    for entry in doc.get("criteria", []):
        ref = entry.get("reference-id", "")
        if ref:
            return ref

    return meta_id or "unknown"


def _fetch_known_policy_ids() -> list[str]:
    try:
        resp = httpx.get(f"{GATEWAY_URL}/api/policies", timeout=5.0)
        if resp.status_code == 200:
            return [p["policy_id"] for p in resp.json() if "policy_id" in p]
    except Exception:
        pass
    return []


def _extract_reasoning(doc: dict) -> str:
    reasons: list[str] = []
    for result in doc.get("results", []):
        rid = result.get("id", "")
        reasoning = result.get("agent-reasoning", "")
        if reasoning:
            reasons.append(f"{rid}: {reasoning}")
    return "\n".join(reasons)
