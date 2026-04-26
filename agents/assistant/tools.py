# SPDX-License-Identifier: Apache-2.0

"""Custom tools for the ComplyTime Studio assistant.

publish_audit_log: persists a validated AuditLog YAML as a draft to the
Gateway's internal endpoint for human review and promotion.
"""

import json
import logging
import os
import re

import httpx
import yaml
from google.adk.tools import ToolContext
from google.genai import types

logger = logging.getLogger(__name__)

GATEWAY_URL = os.environ.get("GATEWAY_URL", "http://studio-gateway:8080")


def _slugify(text: str) -> str:
    return re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")[:64]


async def publish_audit_log(
    yaml_content: str,
    tool_context: ToolContext,
    policy_id: str = "",
) -> dict:
    """Publish a validated AuditLog YAML as a draft for human review.

    Call this AFTER validate_gemara_artifact succeeds. The AuditLog is persisted
    as a draft to the internal Gateway endpoint. A human reviewer must promote
    it to an official record via the workbench.

    Args:
        yaml_content: The complete, validated AuditLog YAML string.
        tool_context: Injected by ADK — provides save_artifact.
        policy_id: The policy_id from the policies table (e.g. 'ampel-branch-protection').
                   When provided, used directly instead of guessing from YAML metadata.

    Returns:
        dict with status, filename, and draft_id.
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

    from main import MODEL_NAME, PROMPT_VERSION

    if not policy_id:
        policy_id = _extract_policy_id(doc)
    reasoning = _extract_reasoning(doc)
    payload = {
        "policy_id": policy_id,
        "content": yaml_content,
        "agent_reasoning": reasoning,
        "model": MODEL_NAME,
        "prompt_version": PROMPT_VERSION,
    }

    draft_id = ""
    try:
        async with httpx.AsyncClient(timeout=15.0) as client:
            resp = await client.post(
                f"{GATEWAY_URL}/internal/draft-audit-logs",
                content=json.dumps(payload),
                headers={"Content-Type": "application/json"},
            )
        if resp.status_code == 201:
            result = resp.json()
            draft_id = result.get("draft_id", "")
            logger.info("persisted draft audit log %s (policy=%s)", draft_id, policy_id)
        else:
            logger.error("gateway rejected draft: %s %s", resp.status_code, resp.text[:200])
            return {"error": f"Gateway returned {resp.status_code}: {resp.text[:200]}"}
    except Exception as e:
        logger.error("failed to persist draft audit log: %s", e)
        return {"error": f"Failed to persist draft: {e}"}

    part = types.Part(text=yaml_content)
    try:
        version = await tool_context.save_artifact(filename, part)
    except Exception as e:
        logger.warning("save_artifact failed (draft already persisted): %s", e)
        version = -1

    logger.info("published draft %s (version %s, draft_id=%s)", filename, version, draft_id)
    return {
        "status": "drafted",
        "filename": filename,
        "version": version,
        "draft_id": draft_id,
        "note": "Draft saved for human review. A reviewer must promote it to official record.",
    }


def _extract_policy_id(doc: dict) -> str:
    """Best-effort extraction of policy_id from parsed AuditLog YAML.

    Checks explicit fields first, then matches the metadata.id against
    known policy_ids from the gateway.
    """
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
    """Fetch policy_ids from the gateway. Returns empty list on failure."""
    try:
        resp = httpx.get(f"{GATEWAY_URL}/api/policies", timeout=5.0)
        if resp.status_code == 200:
            return [p["policy_id"] for p in resp.json() if "policy_id" in p]
    except Exception:
        pass
    return []


def _extract_reasoning(doc: dict) -> str:
    """Collect agent-reasoning fields from all results into a summary."""
    reasons: list[str] = []
    for result in doc.get("results", []):
        rid = result.get("id", "")
        reasoning = result.get("agent-reasoning", "")
        if reasoning:
            reasons.append(f"{rid}: {reasoning}")
    return "\n".join(reasons)
