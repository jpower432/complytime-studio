# SPDX-License-Identifier: Apache-2.0

"""Graph nodes for the Studio assistant that are not LLM invocations.

These are deterministic functions called as LangGraph nodes:
- publish_draft_node: persists validated AuditLog after human approval
- halt_node: terminal node emitting accumulated errors
- clarify_node: asks user to disambiguate intent
"""

import logging

from langchain_core.messages import AIMessage

from tools import publish_audit_log

logger = logging.getLogger(__name__)


async def publish_draft_node(state: dict) -> dict:
    """Graph node: publish the validated draft AuditLog.

    Reached only after validation gate passes AND human approves
    at the interrupt gate. Calls the publish function and emits
    a confirmation message.
    """
    draft = state.get("draft_yaml", "")
    if not draft:
        msg = AIMessage(content="Error: no draft to publish. Validation gate should have caught this.")
        return {"messages": [msg]}

    policy_id = ""
    import re
    match = re.search(r"policy[_-]id:\s*(\S+)", draft)
    if match:
        policy_id = match.group(1)

    reasoning = state.get("_reasoning", "")
    result = await publish_audit_log(draft, policy_id=policy_id, reasoning=reasoning)

    if "error" in result:
        msg = AIMessage(content=f"Failed to publish draft: {result['error']}")
    else:
        draft_id = result.get("draft_id", "unknown")
        msg = AIMessage(
            content=(
                f"Draft AuditLog saved for review (draft_id: {draft_id}). "
                "A reviewer must promote it to the official audit history."
            )
        )

    return {"messages": [msg]}


async def halt_node(state: dict) -> dict:
    """Graph node: terminal halt after retry exhaustion or infra failure.

    Emits all accumulated validation errors to the user.
    """
    result = state.get("validation_result", {})
    errors = result.get("errors", ["Unknown validation failure"])
    attempts = state.get("validation_attempts", 0)

    error_list = "\n".join(f"- {e}" for e in errors)
    msg = AIMessage(
        content=(
            f"Validation failed after {attempts} attempts. "
            "Human intervention required.\n\n"
            f"**Errors:**\n{error_list}"
        )
    )
    return {"messages": [msg]}


async def clarify_node(state: dict) -> dict:
    """Graph node: ask user to disambiguate intent."""
    msg = AIMessage(
        content=(
            "Do you want a **posture check** (readiness overview — "
            "is your evidence current and from the right sources?) "
            "or a **full audit** (AuditLog production with classifications)?"
        )
    )
    return {"messages": [msg]}
