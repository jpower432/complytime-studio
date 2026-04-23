# SPDX-License-Identifier: Apache-2.0

import asyncio
import hashlib
import logging
import os
from contextlib import asynccontextmanager
from pathlib import Path

import uvicorn
import yaml
from a2a.server.apps import A2AStarletteApplication
from a2a.server.events import InMemoryQueueManager
from a2a.server.request_handlers import DefaultRequestHandler
from a2a.server.tasks import InMemoryPushNotificationConfigStore, InMemoryTaskStore
from a2a.types import AgentCapabilities
from google.adk.agents import LlmAgent
from google.adk.a2a.executor.a2a_agent_executor import A2aAgentExecutor
from google.adk.a2a.executor.config import A2aAgentExecutorConfig
from google.adk.a2a.utils.agent_card_builder import AgentCardBuilder
from google.adk.artifacts.in_memory_artifact_service import InMemoryArtifactService
from google.adk.auth.credential_service.in_memory_credential_service import (
    InMemoryCredentialService,
)
from google.adk.memory.in_memory_memory_service import InMemoryMemoryService
from google.adk.runners import Runner
from google.adk.sessions.in_memory_session_service import InMemorySessionService
from google.adk.tools.mcp_tool import McpToolset
from google.adk.tools.mcp_tool.mcp_session_manager import (
    StreamableHTTPConnectionParams,
)
from starlette.applications import Starlette

from callbacks import after_agent, before_agent, before_tool
from event_converter import convert_event_with_yaml_metadata

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

MODEL_NAME = os.environ.get("MODEL_NAME", "gemini-2.5-pro")
PORT = int(os.environ.get("PORT", "8080"))

GEMARA_MCP_URL = os.environ.get("GEMARA_MCP_URL", "")
CLICKHOUSE_MCP_URL = os.environ.get("CLICKHOUSE_MCP_URL", "")


def load_skills() -> str:
    skills_dir = Path("/app/skills")
    if not skills_dir.exists():
        return ""
    parts = []
    for skill_file in sorted(skills_dir.glob("*/SKILL.md")):
        parts.append(skill_file.read_text())
    return "\n\n---\n\n".join(parts)


def load_few_shot_examples() -> str:
    """Read YAML few-shot files and format into a prompt section."""
    few_shot_dir = Path("/app/prompts/few-shot")
    if not few_shot_dir.exists():
        return ""
    parts: list[str] = []
    for f in sorted(few_shot_dir.glob("*.yaml")):
        try:
            examples = yaml.safe_load(f.read_text())
        except yaml.YAMLError:
            logger.warning("Skipping malformed few-shot file: %s", f.name)
            continue
        if not isinstance(examples, list):
            continue
        for ex in examples:
            if not isinstance(ex, dict) or "scenario" not in ex:
                continue
            lines = [f"**Scenario:** {ex['scenario']}"]
            if ex.get("evidence"):
                lines.append(f"**Evidence:** {ex['evidence']}")
            elif ex.get("audit_result_type"):
                lines.append(f"**AuditResult type:** {ex['audit_result_type']}")
                if ex.get("mapping"):
                    lines.append(f"**Mapping:** {ex['mapping']}")
            for key in ("classification", "determination", "coverage"):
                if key in ex:
                    lines.append(f"**{key.title()}:** {ex[key]}")
            if ex.get("reasoning"):
                lines.append(f"**Reasoning:** {ex['reasoning'].strip()}")
            parts.append("\n".join(lines))
    if not parts:
        return ""
    return "## Classification Examples\n\n" + "\n\n---\n\n".join(parts)


def load_prompt() -> str:
    prompt_path = Path("/app/prompt.md")
    base_prompt = prompt_path.read_text()
    skills_text = load_skills()
    if skills_text:
        base_prompt = f"{base_prompt}\n\n## Loaded Skills\n\n{skills_text}"
    few_shot_text = load_few_shot_examples()
    if few_shot_text:
        base_prompt = f"{base_prompt}\n\n{few_shot_text}"
    return base_prompt


async def _fetch_gemara_resources(url: str) -> str:
    """Preload all Gemara MCP resources and format them for prompt injection."""
    ts = McpToolset(
        connection_params=StreamableHTTPConnectionParams(url=url),
        use_mcp_resources=True,
    )
    parts: list[str] = []
    try:
        names = await ts.list_resources()
        for name in names:
            try:
                contents = await ts.read_resource(name)
                for item in contents:
                    uri = str(item.uri) if hasattr(item, "uri") else name
                    text = item.text if hasattr(item, "text") else str(item)
                    parts.append(f"### Resource: `{uri}`\n\n```\n{text}\n```")
                    logger.info("preloaded resource %s (%d chars)", uri, len(text))
            except Exception as e:
                logger.warning("failed to read resource %s: %s", name, e)
    except Exception as e:
        logger.warning("failed to list gemara resources: %s", e)
    finally:
        await ts.close()
    if not parts:
        return ""
    return "## Gemara Schema Reference\n\n" + "\n\n".join(parts)


def build_tools() -> list:
    tools = []

    if GEMARA_MCP_URL:
        tools.append(
            McpToolset(
                connection_params=StreamableHTTPConnectionParams(url=GEMARA_MCP_URL),
                tool_filter=["validate_gemara_artifact", "migrate_gemara_artifact"],
            )
        )
        logger.info("gemara-mcp toolset registered (http: %s)", GEMARA_MCP_URL)

    if CLICKHOUSE_MCP_URL:
        tools.append(
            McpToolset(
                connection_params=StreamableHTTPConnectionParams(
                    url=CLICKHOUSE_MCP_URL
                ),
                tool_filter=["run_select_query", "list_databases", "list_tables"],
            )
        )
        logger.info("clickhouse-mcp toolset registered (http: %s)", CLICKHOUSE_MCP_URL)

    if not tools:
        logger.warning("No MCP URLs configured — agent running without tools")

    return tools


def _build_instruction() -> str:
    base = load_prompt()
    if GEMARA_MCP_URL:
        try:
            resources_text = asyncio.run(_fetch_gemara_resources(GEMARA_MCP_URL))
            if resources_text:
                base = f"{base}\n\n{resources_text}"
        except Exception as e:
            logger.warning("resource preload failed: %s", e)
    return base


_INSTRUCTION = _build_instruction()
PROMPT_VERSION = hashlib.sha256(_INSTRUCTION.encode()).hexdigest()[:12]
logger.info("prompt_version=%s model=%s", PROMPT_VERSION, MODEL_NAME)

root_agent = LlmAgent(
    name="studio_assistant",
    model=MODEL_NAME,
    instruction=_INSTRUCTION,
    tools=build_tools(),
    description=(
        "ComplyTime Studio assistant — audit preparation, evidence synthesis, "
        "cross-framework coverage analysis, and compliance guidance"
    ),
    before_agent_callback=before_agent,
    after_agent_callback=after_agent,
    before_tool_callback=before_tool,
)


# --- Manual A2aAgentExecutor setup (replaces to_a2a()) ---
# Shared services must be singletons so multi-turn conversations
# within the same task retain session and memory state.

_session_service = InMemorySessionService()
_artifact_service = InMemoryArtifactService()
_memory_service = InMemoryMemoryService()
_credential_service = InMemoryCredentialService()


async def create_runner() -> Runner:
    return Runner(
        app_name="studio_assistant",
        agent=root_agent,
        artifact_service=_artifact_service,
        session_service=_session_service,
        memory_service=_memory_service,
        credential_service=_credential_service,
    )


config = A2aAgentExecutorConfig(
    adk_event_converter=convert_event_with_yaml_metadata,
)

agent_executor = A2aAgentExecutor(
    runner=create_runner,
    config=config,
)

task_store = InMemoryTaskStore()
push_config_store = InMemoryPushNotificationConfigStore()
queue_manager = InMemoryQueueManager()

request_handler = DefaultRequestHandler(
    agent_executor=agent_executor,
    task_store=task_store,
    push_config_store=push_config_store,
    queue_manager=queue_manager,
)

rpc_url = f"http://0.0.0.0:{PORT}/"
card_builder = AgentCardBuilder(
    agent=root_agent,
    rpc_url=rpc_url,
    capabilities=AgentCapabilities(streaming=True),
)


@asynccontextmanager
async def lifespan(app: Starlette):
    agent_card = await card_builder.build()
    a2a_app = A2AStarletteApplication(
        agent_card=agent_card,
        http_handler=request_handler,
    )
    a2a_app.add_routes_to_app(app)
    logger.info("A2A routes registered, agent card built for %s", root_agent.name)
    yield


app = Starlette(lifespan=lifespan)

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=PORT)
