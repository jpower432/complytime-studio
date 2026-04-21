# SPDX-License-Identifier: Apache-2.0

"""Deterministic gates for the BYO gap analyst agent.

before_agent_callback: input validation — policy reference + audit timeline detection
after_agent_callback: output validation — AuditLog YAML extraction + save_artifact
before_tool_callback: SQL sanitization — DDL/DML deny-list for run_select_query
"""

import logging
import re
from typing import Any, Optional

import yaml

logger = logging.getLogger(__name__)

FORBIDDEN_SQL = re.compile(
    r"\b(INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|TRUNCATE|GRANT|REVOKE)\b",
    re.IGNORECASE,
)

POLICY_KEYWORDS = {"policy", "policy_id", "audit"}
TIMELINE_KEYWORDS = {"q1", "q2", "q3", "q4", "2025", "2026", "start", "end", "period"}


async def before_agent(callback_context) -> Optional[Any]:
    """Inspect inbound user message for policy reference and audit timeline."""
    user_content = getattr(callback_context, "user_content", None)
    if user_content is None:
        invocation = getattr(callback_context, "invocation_context", None)
        if invocation:
            user_content = getattr(invocation, "user_content", None)

    user_message = ""
    if user_content and hasattr(user_content, "parts"):
        for part in user_content.parts:
            if hasattr(part, "text") and part.text:
                user_message += part.text

    if not user_message.strip():
        logger.warning("Empty user message received")
        return None

    lower = user_message.lower()
    if not any(kw in lower for kw in POLICY_KEYWORDS):
        logger.info("No policy reference detected — agent will ask for it")
    if not any(kw in lower for kw in TIMELINE_KEYWORDS):
        logger.info("No audit timeline detected — agent will ask for it")

    return None


async def after_agent(callback_context) -> Optional[Any]:
    """Scan agent output for AuditLog YAML and save as artifact."""
    invocation = getattr(callback_context, "invocation_context", None)
    agent_output = ""

    if hasattr(callback_context, "state") and hasattr(callback_context.state, "get"):
        pass

    events = getattr(invocation, "intermediate_data", None) or []
    for event in events:
        content = getattr(event, "content", None)
        if content and hasattr(content, "parts"):
            for part in content.parts:
                if hasattr(part, "text") and part.text:
                    agent_output += part.text

    if not agent_output:
        return None

    yaml_blocks = _extract_yaml_blocks(agent_output)
    for i, block in enumerate(yaml_blocks):
        try:
            parsed = yaml.safe_load(block)
            if not isinstance(parsed, dict):
                continue

            is_audit_log = (
                parsed.get("metadata", {}).get("type") == "AuditLog"
                or "audit-results" in parsed
                or "results" in parsed
            )

            if is_audit_log:
                filename = f"audit-log-{i}.yaml"
                logger.info("Valid AuditLog detected, saving artifact: %s", filename)
                if hasattr(callback_context, "save_artifact"):
                    await callback_context.save_artifact(
                        filename=filename,
                        artifact=block.encode("utf-8"),
                        mime_type="application/yaml",
                    )
        except yaml.YAMLError as e:
            logger.warning("YAML parse error in output block %d: %s", i, e)

    return None


async def before_tool(
    tool: Any, args: dict[str, Any], tool_context: Any
) -> Optional[dict]:
    """SQL injection guard for ClickHouse run_select_query."""
    tool_name = getattr(tool, "name", str(tool))
    if tool_name != "run_select_query":
        return None

    query = args.get("query", "")
    if FORBIDDEN_SQL.search(query):
        logger.warning("Blocked forbidden SQL: %s", query[:200])
        return {
            "error": "Query rejected: only SELECT statements are allowed. "
            "DDL and DML operations are forbidden."
        }

    return None


def _extract_yaml_blocks(text: str) -> list[str]:
    """Extract YAML content from fenced code blocks or raw YAML."""
    fenced = re.findall(
        r"```(?:yaml|yml)?\s*\n(.*?)```",
        text,
        re.DOTALL,
    )
    if fenced:
        return fenced

    blocks: list[str] = []
    lines = text.split("\n")
    current: list[str] = []
    in_yaml = False
    for line in lines:
        if not in_yaml and re.match(
            r"^(metadata|title|groups|risks|results|audit-results):", line
        ):
            in_yaml = True
            current = [line]
        elif in_yaml:
            if line.strip() == "" and current and not current[-1].strip().endswith(":"):
                blocks.append("\n".join(current))
                current = []
                in_yaml = False
            else:
                current.append(line)
    if current:
        blocks.append("\n".join(current))
    return blocks
