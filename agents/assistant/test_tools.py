# SPDX-License-Identifier: Apache-2.0

"""Tests for tools.py — SQL guard, slugify, extract helpers."""

import sys
from unittest.mock import MagicMock

import pytest

# Mock langchain_core before importing tools.py (not installed in test env)
lc_mock = MagicMock()
lc_mock.tools.tool = lambda f: f
sys.modules.setdefault("langchain_core", lc_mock)
sys.modules.setdefault("langchain_core.tools", lc_mock.tools)

from tools import (  # noqa: E402
    _extract_policy_id,
    _extract_reasoning,
    _slugify,
    sql_guard_filter,
    validate_sql_query,
)


class TestValidateSqlQuery:
    @pytest.mark.parametrize("sql", [
        "SELECT * FROM policies",
        "select count(*) from evidence where policy_id = 'abc'",
        "SELECT p.name, e.status FROM policies p JOIN evidence e ON p.id = e.policy_id",
        "WITH cte AS (SELECT 1) SELECT * FROM cte",
    ])
    def test_allows_select_queries(self, sql):
        assert validate_sql_query(sql) is None

    @pytest.mark.parametrize("keyword", [
        "INSERT", "UPDATE", "DELETE", "DROP", "ALTER",
        "CREATE", "TRUNCATE", "GRANT", "REVOKE", "EXEC",
    ])
    def test_blocks_write_keywords(self, keyword):
        sql = f"{keyword} INTO policies VALUES ('x')"
        result = validate_sql_query(sql)
        assert result is not None
        assert "Only SELECT" in result

    def test_case_insensitive(self):
        assert validate_sql_query("insert into foo values (1)") is not None
        assert validate_sql_query("Drop Table policies") is not None

    def test_keyword_in_string_literal_still_blocks(self):
        # Regex-based guard is intentionally conservative
        assert validate_sql_query("SELECT 'DELETE' FROM foo") is not None

    def test_empty_query_allowed(self):
        assert validate_sql_query("") is None


class TestSqlGuardFilter:
    def test_blocks_query_database_with_write(self):
        result = sql_guard_filter("query_database", {"query": "DELETE FROM policies"})
        assert result is not None
        assert "error" in result

    def test_allows_query_database_with_select(self):
        result = sql_guard_filter("query_database", {"query": "SELECT 1"})
        assert result is None

    def test_ignores_other_tools(self):
        result = sql_guard_filter("validate_gemara_artifact", {"query": "DROP TABLE x"})
        assert result is None

    def test_checks_sql_arg_key(self):
        result = sql_guard_filter("query_database", {"sql": "INSERT INTO foo VALUES (1)"})
        assert result is not None

    def test_empty_args(self):
        result = sql_guard_filter("query_database", {})
        assert result is None


class TestSlugify:
    def test_basic(self):
        assert _slugify("Audit Log 2024") == "audit-log-2024"

    def test_special_chars(self):
        assert _slugify("foo@bar.baz!") == "foo-bar-baz"

    def test_empty(self):
        assert _slugify("") == ""

    def test_length_cap(self):
        long_text = "a" * 100
        assert len(_slugify(long_text)) <= 64

    def test_strips_leading_trailing_hyphens(self):
        assert _slugify("---hello---") == "hello"


class TestExtractPolicyId:
    def test_from_scope_policy_id(self):
        doc = {"scope": {"policy-id": "ampel-branch"}, "metadata": {"id": "fallback"}}
        assert _extract_policy_id(doc) == "ampel-branch"

    def test_from_scope_underscore(self):
        doc = {"scope": {"policy_id": "kube-sec"}, "metadata": {}}
        assert _extract_policy_id(doc) == "kube-sec"

    def test_from_metadata_policy_id(self):
        doc = {"scope": {}, "metadata": {"policy-id": "soc2-prod"}}
        assert _extract_policy_id(doc) == "soc2-prod"

    def test_from_criteria_reference_id(self):
        doc = {"metadata": {"id": ""}, "scope": {}, "criteria": [{"reference-id": "iso-27001"}]}
        assert _extract_policy_id(doc) == "iso-27001"

    def test_fallback_to_metadata_id(self):
        doc = {"metadata": {"id": "my-audit-log"}, "scope": {}}
        assert _extract_policy_id(doc) == "my-audit-log"

    def test_fallback_unknown(self):
        doc = {"metadata": {}, "scope": {}}
        assert _extract_policy_id(doc) == "unknown"


class TestExtractReasoning:
    def test_extracts_reasoning(self):
        doc = {"results": [
            {"id": "r1", "agent-reasoning": "Evidence covers all controls"},
            {"id": "r2", "agent-reasoning": "Missing attestation"},
        ]}
        result = _extract_reasoning(doc)
        assert "r1: Evidence covers all controls" in result
        assert "r2: Missing attestation" in result

    def test_skips_empty_reasoning(self):
        doc = {"results": [
            {"id": "r1", "agent-reasoning": ""},
            {"id": "r2"},
        ]}
        assert _extract_reasoning(doc) == ""

    def test_no_results(self):
        assert _extract_reasoning({}) == ""
