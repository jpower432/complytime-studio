# SPDX-License-Identifier: Apache-2.0

import asyncio
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from callbacks import _extract_yaml_blocks, after_agent, before_tool


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


# --- _extract_yaml_blocks ---


class TestExtractYamlBlocks:
    def test_fenced_yaml(self):
        text = "Here is output:\n```yaml\nmetadata:\n  type: AuditLog\nresults: []\n```\nDone."
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 1
        assert "metadata:" in blocks[0]
        assert "results: []" in blocks[0]

    def test_fenced_yml(self):
        text = "```yml\ntitle: test\n```"
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 1

    def test_fenced_no_lang_tag(self):
        text = "```\nmetadata:\n  type: AuditLog\n```"
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 1

    def test_multiple_fenced_blocks(self):
        text = "```yaml\nblock1: true\n```\ntext\n```yaml\nblock2: true\n```"
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 2

    def test_raw_yaml_metadata_trigger(self):
        text = "metadata:\n  type: AuditLog\nresults:\n  - finding: gap\n"
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) >= 1
        assert "metadata:" in blocks[0]

    def test_raw_yaml_results_trigger(self):
        text = "Some prose.\nresults:\n  - item: one\n  - item: two\n"
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 1
        assert "results:" in blocks[0]

    def test_raw_yaml_audit_results_trigger(self):
        text = "audit-results:\n  - finding: gap\n"
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 1

    def test_no_yaml(self):
        text = "This is plain text with no YAML content at all."
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 0

    def test_empty_string(self):
        blocks = _extract_yaml_blocks("")
        assert len(blocks) == 0

    def test_fenced_takes_priority_over_raw(self):
        text = "```yaml\nfenced: true\n```\nmetadata:\n  raw: true\n"
        blocks = _extract_yaml_blocks(text)
        assert len(blocks) == 1
        assert "fenced: true" in blocks[0]


# --- after_agent ---


class TestAfterAgent:
    def _make_context(self, agent_text: str, has_save=True):
        part = SimpleNamespace(text=agent_text)
        content = SimpleNamespace(parts=[part])
        event = SimpleNamespace(content=content)
        invocation = SimpleNamespace(intermediate_data=[event])
        ctx = SimpleNamespace(
            invocation_context=invocation,
            state=SimpleNamespace(get=lambda k, d=None: d),
        )
        if has_save:
            ctx.save_artifact = AsyncMock()
        return ctx

    @pytest.mark.asyncio
    async def test_detects_audit_log_by_metadata_type(self):
        yaml_text = "```yaml\nmetadata:\n  type: AuditLog\nresults:\n  - x\n```"
        ctx = self._make_context(yaml_text)
        await after_agent(ctx)
        ctx.save_artifact.assert_called_once()
        call_kwargs = ctx.save_artifact.call_args
        assert call_kwargs[1]["filename"].startswith("audit-log-")
        assert call_kwargs[1]["mime_type"] == "application/yaml"

    @pytest.mark.asyncio
    async def test_detects_audit_log_by_results_key(self):
        yaml_text = "```yaml\nresults:\n  - finding: gap\n```"
        ctx = self._make_context(yaml_text)
        await after_agent(ctx)
        ctx.save_artifact.assert_called_once()

    @pytest.mark.asyncio
    async def test_detects_audit_log_by_audit_results_key(self):
        yaml_text = "```yaml\naudit-results:\n  - finding: gap\n```"
        ctx = self._make_context(yaml_text)
        await after_agent(ctx)
        ctx.save_artifact.assert_called_once()

    @pytest.mark.asyncio
    async def test_ignores_non_audit_yaml(self):
        yaml_text = "```yaml\ntitle: Just a title\ndescription: Not an audit log\n```"
        ctx = self._make_context(yaml_text)
        await after_agent(ctx)
        ctx.save_artifact.assert_not_called()

    @pytest.mark.asyncio
    async def test_no_output(self):
        part = SimpleNamespace(text="")
        content = SimpleNamespace(parts=[part])
        event = SimpleNamespace(content=content)
        invocation = SimpleNamespace(intermediate_data=[event])
        ctx = SimpleNamespace(
            invocation_context=invocation,
            state=SimpleNamespace(get=lambda k, d=None: d),
            save_artifact=AsyncMock(),
        )
        await after_agent(ctx)
        ctx.save_artifact.assert_not_called()

    @pytest.mark.asyncio
    async def test_malformed_yaml_does_not_crash(self):
        yaml_text = "```yaml\n: :\n  bad:: yaml::\n[invalid\n```"
        ctx = self._make_context(yaml_text)
        await after_agent(ctx)
        ctx.save_artifact.assert_not_called()

    @pytest.mark.asyncio
    async def test_no_save_artifact_method(self):
        yaml_text = "```yaml\nresults:\n  - finding: gap\n```"
        ctx = self._make_context(yaml_text, has_save=False)
        await after_agent(ctx)

    @pytest.mark.asyncio
    async def test_yaml_list_not_dict_ignored(self):
        yaml_text = "```yaml\n- item1\n- item2\n```"
        ctx = self._make_context(yaml_text)
        await after_agent(ctx)
        ctx.save_artifact.assert_not_called()
