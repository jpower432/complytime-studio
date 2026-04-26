# SPDX-License-Identifier: Apache-2.0

"""Deterministic gates for the BYO gap analyst agent.

before_agent_callback: input validation — policy reference + audit timeline detection
after_agent_callback: reserved for future post-processing
before_tool_callback: SQL sanitization — DDL/DML deny-list for run_select_query
"""

import logging
import re
from typing import Any, Optional

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
    """Post-processing hook. AuditLogs are published via the publish_audit_log tool."""
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


