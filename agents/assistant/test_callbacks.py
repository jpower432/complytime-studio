# SPDX-License-Identifier: Apache-2.0

from types import SimpleNamespace

import pytest

from callbacks import after_agent, before_tool


# --- before_tool ---


class TestBeforeTool:
    @pytest.fixture
    def query_tool(self):
        return SimpleNamespace(name="query_database")

    @pytest.fixture
    def other_tool(self):
        return SimpleNamespace(name="validate_gemara_artifact")

    @pytest.mark.asyncio
    async def test_allows_select(self, query_tool):
        result = await before_tool(
            query_tool,
            {"query": "SELECT * FROM evidence WHERE policy_id = 'ampel'"},
            None,
        )
        assert result is None

    @pytest.mark.asyncio
    async def test_blocks_insert(self, query_tool):
        result = await before_tool(
            query_tool,
            {"query": "INSERT INTO evidence (evidence_id) VALUES ('hack')"},
            None,
        )
        assert result is not None
        assert result["error"] == "Only SELECT queries are allowed."

    @pytest.mark.asyncio
    async def test_blocks_drop(self, query_tool):
        result = await before_tool(
            query_tool,
            {"query": "DROP TABLE evidence"},
            None,
        )
        assert result is not None
        assert result["error"] == "Only SELECT queries are allowed."

    @pytest.mark.asyncio
    async def test_blocks_update(self, query_tool):
        result = await before_tool(
            query_tool,
            {"query": "UPDATE evidence SET certified = true"},
            None,
        )
        assert result is not None
        assert result["error"] == "Only SELECT queries are allowed."

    @pytest.mark.asyncio
    async def test_blocks_delete(self, query_tool):
        result = await before_tool(
            query_tool,
            {"query": "DELETE FROM evidence WHERE 1=1"},
            None,
        )
        assert result is not None
        assert result["error"] == "Only SELECT queries are allowed."

    @pytest.mark.asyncio
    async def test_blocks_truncate(self, query_tool):
        result = await before_tool(
            query_tool,
            {"query": "TRUNCATE TABLE evidence"},
            None,
        )
        assert result is not None
        assert result["error"] == "Only SELECT queries are allowed."

    @pytest.mark.asyncio
    async def test_blocks_case_insensitive(self, query_tool):
        result = await before_tool(
            query_tool,
            {"query": "dRoP TABLE evidence"},
            None,
        )
        assert result is not None
        assert result["error"] == "Only SELECT queries are allowed."

    @pytest.mark.asyncio
    async def test_ignores_other_tools(self, other_tool):
        result = await before_tool(
            other_tool,
            {"query": "DROP TABLE evidence"},
            None,
        )
        assert result is None

    @pytest.mark.asyncio
    async def test_allows_empty_query(self, query_tool):
        result = await before_tool(query_tool, {"query": ""}, None)
        assert result is None


# --- after_agent ---


class TestAfterAgent:
    @pytest.mark.asyncio
    async def test_returns_none(self):
        ctx = SimpleNamespace()
        result = await after_agent(ctx)
        assert result is None
