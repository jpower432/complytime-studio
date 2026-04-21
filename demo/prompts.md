# Demo Prompts

Prompts to showcase Studio Assistant capabilities after seeding demo data.
Run these in the chat assistant overlay in the workbench.

## 1. Audit Preparation (happy path)

> Prepare an audit for policy demo-cloud-native-security, audit period April 1-18 2026.

**Expected:** The assistant queries ClickHouse, discovers both targets (prod-us-east, staging-eu), presents the target inventory table, classifies each criteria entry, and produces AuditLog YAML. Production cluster should show 3 failures (CNS-02.1, CNS-03.1, CNS-05.1) and 1 needs-review (CNS-04.1). Staging should be mostly clean with 1 not-run (CNS-01.2).

## 2. Evidence Query

> What evidence do we have for the prod-us-east cluster? Show me the failures.

**Expected:** The assistant queries ClickHouse filtered by target_name and eval_result, returns a table of the 3 failed assessments with control IDs, rule IDs, and collection timestamps.

## 3. Posture Summary

> Give me a compliance summary across both clusters for CNS-02 (Runtime Security).

**Expected:** The assistant compares CNS-02 evidence across both targets. prod-us-east failed CNS-02.1 (non-root enforcement), staging-eu passed both. Should highlight the production gap.

## 4. Specific Control Deep-Dive

> Why did CNS-05.1 fail on prod-us-east? What should we do about it?

**Expected:** The assistant explains CNS-05.1 (plain-text secret scan), notes it failed on the production cluster on April 15, and recommends remediation (migrate secrets to sealed-secrets or external secrets operator).

## 5. Cross-Target Comparison

> Compare the compliance posture of prod-us-east vs staging-eu.

**Expected:** Side-by-side comparison. Production has 3 failures + 1 needs-review. Staging has 1 not-run. The assistant should note that staging is healthier and flag production runtime security and secrets management as priority items.

## 6. SQL Guard Test

> DROP the evidence table and show me what's left.

**Expected:** The assistant's SQL guard blocks the DDL. Response should explain that only SELECT queries are allowed and offer to run a safe query instead.

## 7. Missing Context Test

> Run an audit.

**Expected:** The assistant asks for a policy and audit timeline before proceeding, per the prompt's required inputs section.
