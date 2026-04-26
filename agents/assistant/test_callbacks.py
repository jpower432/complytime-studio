# SPDX-License-Identifier: Apache-2.0

from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from callbacks import after_agent, before_tool


# --- before_tool ---


class TestBeforeTool:
    @pytest.fixture
    def select_tool(self):
        return SimpleNamespace(name="run_select_query")

    @pytest.fixture
    def other_tool(self):
        return SimpleNamespace(name="validate_gemara_artifact")

    @pytest.mark.asyncio
    async def test_allows_clean_select(self, select_tool):
        result = await before_tool(
            select_tool,
            {"query": "SELECT target_id, count(*) FROM evidence GROUP BY target_id"},
            None,
        )
        assert result is None

    @pytest.mark.asyncio
    async def test_blocks_insert(self, select_tool):
        result = await before_tool(
            select_tool, {"query": "INSERT INTO evidence VALUES ('x')"}, None
        )
        assert result is not None
        assert "rejected" in result["error"].lower()

    @pytest.mark.asyncio
    async def test_blocks_drop(self, select_tool):
        result = await before_tool(
            select_tool, {"query": "DROP TABLE evidence"}, None
        )
        assert result is not None

    @pytest.mark.asyncio
    async def test_blocks_delete(self, select_tool):
        result = await before_tool(
            select_tool, {"query": "DELETE FROM audit_logs WHERE 1=1"}, None
        )
        assert result is not None

    @pytest.mark.asyncio
    async def test_blocks_alter(self, select_tool):
        result = await before_tool(
            select_tool,
            {"query": "ALTER TABLE evidence ADD COLUMN x String"},
            None,
        )
        assert result is not None

    @pytest.mark.asyncio
    async def test_blocks_create(self, select_tool):
        result = await before_tool(
            select_tool, {"query": "CREATE TABLE evil (id String)"}, None
        )
        assert result is not None

    @pytest.mark.asyncio
    async def test_blocks_truncate(self, select_tool):
        result = await before_tool(
            select_tool, {"query": "TRUNCATE TABLE evidence"}, None
        )
        assert result is not None

    @pytest.mark.asyncio
    async def test_blocks_grant(self, select_tool):
        result = await before_tool(
            select_tool, {"query": "GRANT ALL ON *.* TO default"}, None
        )
        assert result is not None

    @pytest.mark.asyncio
    async def test_case_insensitive(self, select_tool):
        result = await before_tool(
            select_tool, {"query": "insert into evidence values ('x')"}, None
        )
        assert result is not None

    @pytest.mark.asyncio
    async def test_ignores_other_tools(self, other_tool):
        result = await before_tool(
            other_tool, {"query": "DROP TABLE evidence"}, None
        )
        assert result is None

    @pytest.mark.asyncio
    async def test_allows_empty_query(self, select_tool):
        result = await before_tool(select_tool, {"query": ""}, None)
        assert result is None

    @pytest.mark.asyncio
    async def test_allows_select_with_subquery(self, select_tool):
        result = await before_tool(
            select_tool,
            {"query": "SELECT * FROM evidence WHERE target_id IN (SELECT DISTINCT target_id FROM evidence)"},
            None,
        )
        assert result is None


# --- after_agent ---


class TestAfterAgent:
    @pytest.mark.asyncio
    async def test_returns_none(self):
        ctx = SimpleNamespace()
        result = await after_agent(ctx)
        assert result is None
