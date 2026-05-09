# SPDX-License-Identifier: Apache-2.0

"""ComplyTime Studio assistant — LangGraph agent with kagent checkpointing.

Defines the StateGraph, MCP tool integration, and KAgentApp bootstrap.
A2A protocol serving is handled by kagent-langgraph (KAgentApp).
"""

import logging
import os
from typing import Annotated, Sequence, TypedDict

from a2a.types import AgentCapabilities, AgentCard, AgentSkill
from kagent.core import KAgentConfig
from kagent.langgraph import KAgentApp
from langchain_core.messages import BaseMessage
from langchain_mcp_adapters.client import MultiServerMCPClient
from langgraph.graph import StateGraph
from langgraph.graph.message import add_messages
from langgraph.prebuilt import ToolNode

from prompt import load_system_prompt
from tools import build_tools, publish_audit_log, sql_guard_filter

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

MODEL_NAME = os.environ.get("MODEL_NAME", "claude-opus-4-6")
MODEL_PROVIDER = os.environ.get("MODEL_PROVIDER", "AnthropicVertexAI")
PORT = int(os.environ.get("PORT", "8080"))

GEMARA_MCP_URL = os.environ.get("GEMARA_MCP_URL", "")
POSTGRES_MCP_URL = os.environ.get("POSTGRES_MCP_URL", "")
AGENT_ID = os.environ.get("AGENT_ID", "studio-assistant")
KAGENT_URL = os.environ.get("KAGENT_URL", "http://kagent-controller:8083")
APP_NAME = os.environ.get("APP_NAME", "studio_assistant")


class State(TypedDict):
    messages: Annotated[Sequence[BaseMessage], add_messages]


def _create_model():
    """Instantiate the chat model based on MODEL_PROVIDER."""
    if MODEL_PROVIDER == "AnthropicVertexAI":
        from langchain_google_vertexai.model_garden import ChatAnthropicVertex

        return ChatAnthropicVertex(
            model_name=MODEL_NAME,
            project=os.environ.get("GOOGLE_CLOUD_PROJECT", ""),
            location=os.environ.get("GOOGLE_CLOUD_LOCATION", "us-east5"),
            max_tokens=16384,
        )

    if MODEL_PROVIDER == "GeminiVertexAI":
        from langchain_google_vertexai import ChatVertexAI

        return ChatVertexAI(
            model_name=MODEL_NAME,
            project=os.environ.get("GOOGLE_CLOUD_PROJECT", ""),
            location=os.environ.get("GOOGLE_CLOUD_LOCATION", "us-east5"),
        )

    from langchain_anthropic import ChatAnthropic

    return ChatAnthropic(model=MODEL_NAME, max_tokens=16384)


def _build_mcp_tools() -> dict:
    """Connect to MCP servers via langchain-mcp-adapters."""
    headers = {"X-Agent-ID": AGENT_ID}
    servers = {}
    if GEMARA_MCP_URL:
        servers["gemara-mcp"] = {
            "transport": "streamable_http",
            "url": GEMARA_MCP_URL,
            "headers": headers,
        }
    if POSTGRES_MCP_URL:
        servers["postgres-mcp"] = {
            "transport": "streamable_http",
            "url": POSTGRES_MCP_URL,
            "headers": headers,
        }
    if not servers:
        logger.warning("No MCP servers configured — agent running without MCP tools")
    return servers


def build_graph() -> StateGraph:
    """Construct the StateGraph with agent + tool nodes."""
    system_prompt = load_system_prompt()
    model = _create_model()
    mcp_servers = _build_mcp_tools()

    all_tools = build_tools()

    async def agent_node(state: State, config):
        """Invoke the LLM with tools bound."""
        messages = state["messages"]
        system_messages = [{"role": "system", "content": system_prompt}]

        tools_for_model = all_tools
        if mcp_servers:
            client = MultiServerMCPClient(mcp_servers)
            mcp_tools = await client.get_tools()
            tools_for_model = all_tools + mcp_tools

        bound_model = model.bind_tools(tools_for_model)
        response = await bound_model.ainvoke(system_messages + list(messages))

        return {"messages": [response]}

    async def tool_node_fn(state: State, config):
        """Execute tool calls (MCP + local) with SQL write guard."""
        last_msg = state["messages"][-1]
        if hasattr(last_msg, "tool_calls"):
            for tc in last_msg.tool_calls:
                blocked = sql_guard_filter(tc.get("name", ""), tc.get("args", {}))
                if blocked:
                    from langchain_core.messages import ToolMessage

                    return {"messages": [ToolMessage(
                        content=str(blocked),
                        tool_call_id=tc.get("id", ""),
                    )]}

        tools_for_node = all_tools
        if mcp_servers:
            client = MultiServerMCPClient(mcp_servers)
            mcp_tools = await client.get_tools()
            tools_for_node = all_tools + mcp_tools
        node = ToolNode(tools_for_node, handle_tool_errors=True)
        return await node.ainvoke(state, config)

    def should_use_tool(state: State):
        """Route to tools if the last message has tool_calls."""
        last = state["messages"][-1]
        if hasattr(last, "tool_calls") and last.tool_calls:
            return "tools"
        return "__end__"

    builder = StateGraph(State)
    builder.add_node("agent", agent_node)
    builder.add_node("tools", tool_node_fn)
    builder.add_edge("__start__", "agent")
    builder.add_conditional_edges("agent", should_use_tool, {"tools": "tools", "__end__": "__end__"})
    builder.add_edge("tools", "agent")
    return builder


_builder = build_graph()
_compiled_graph = _builder.compile()

_agent_card = AgentCard(
    name="studio_assistant",
    description=(
        "ComplyTime Studio assistant — audit preparation, evidence synthesis, "
        "cross-framework coverage analysis, and compliance guidance"
    ),
    version="0.1.0",
    url=f"http://localhost:{os.getenv('PORT', '8080')}",
    capabilities=AgentCapabilities(streaming=True),
    default_input_modes=["text"],
    default_output_modes=["text"],
    skills=[
        AgentSkill(
            id="compliance-assistant",
            name="Studio Assistant",
            description=(
                "Audit preparation, evidence synthesis, cross-framework coverage "
                "analysis, policy guidance, and AuditLog generation."
            ),
            tags=["assistant", "audit", "compliance", "coverage", "evidence"],
        )
    ],
)

_config = KAgentConfig(
    url=KAGENT_URL,
    name=APP_NAME,
    namespace=os.getenv("KAGENT_NAMESPACE", "kagent"),
)

_app = KAgentApp(
    graph=_compiled_graph,
    agent_card=_agent_card,
    config=_config,
)

graph = _app.build()
