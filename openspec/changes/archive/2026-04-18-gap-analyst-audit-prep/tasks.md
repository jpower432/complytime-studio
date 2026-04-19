## 1. Agent Prompt & Spec

- [x] 1.1 Rewrite `agents/gap-analyst/prompt.md` with 4-phase combined audit workflow (scope & inventory, evidence assessment with cadence validation, cross-framework coverage, multi-doc output)
- [x] 1.2 Update `agents/gap-analyst/agent.yaml` description to reflect combined audit preparation role
- [x] 1.3 Update A2A skill definition (id: gap-analysis) with combined audit description and tags

## 2. Helm Chart Sync

- [x] 2.1 Run `make sync-prompts` to copy updated `prompt.md` into `charts/complytime-studio/agents/gap-analyst/`
- [x] 2.2 Update gap-analyst `spec.description` in `charts/complytime-studio/templates/agent-specialists.yaml` to match `agent.yaml`
- [x] 2.3 Update gap-analyst `a2aConfig.skills` in `agent-specialists.yaml` to match `agent.yaml` A2A skill definition

## 3. Verification

- [x] 3.1 `helm template` renders valid Agent CRD with updated description, prompt, and A2A skill
- [x] 3.2 Verify prompt content in rendered ConfigMap matches `agents/gap-analyst/prompt.md`
- [x] 3.3 Verify no references to "single-target" or old gap-analyst description remain in chart templates
