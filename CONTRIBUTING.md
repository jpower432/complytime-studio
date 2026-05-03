# Contributing to ComplyTime Studio

## Dev Environment

See [README.md](README.md) for prerequisites (Go, Node, Docker/Podman, kind, kubectl, Helm).

**Two modes:**

| Mode | Command | What you get |
|:--|:--|:--|
| Compose | `make compose-up` | Gateway + PostgreSQL + NATS + MCP servers. No agents. |
| Full stack | `make cluster-up && make deploy` | Kind cluster with kagent, agents, NATS, PostgreSQL |

Seed demo data after deploy:

```bash
make seed
```

When auth is enabled, seeding requires the API token: `STUDIO_API_TOKEN=dev-seed-token make seed`. See `README.md` for details.

## Code Contributions

**Branching:** Create feature branches from `main`. Keep PRs atomic — reviewable in one sitting.

**PR title format:** `<type>: <description>` per [Conventional Commits](https://www.conventionalcommits.org/).

**Commits:**

```bash
git commit -S -s -m "feat: add posture endpoint"
```

- `-S` GPG sign, `-s` Signed-off-by (both required)
- AI-assisted work adds an `Assisted-by: Cursor (<model>)` trailer

**CI gates** (must pass before merge):

```bash
go vet -tags dev ./...
go test -tags dev -race ./...
go build ./cmd/gateway/
golangci-lint run ./...
cd workbench && npx tsc --noEmit && npm run build
helm lint charts/complytime-studio/
```

**Review:** 2 maintainer approvals required. Exceptions for transient CI failures require maintainer consensus.

## Go Standards

Conventions are defined in [AGENTS.md](AGENTS.md). Key points:

- File names: lowercase with underscores (`my_file.go`)
- Package names: short, lowercase, no underscores
- Always check and return errors
- Format with `goimports` and `go fmt`
- SPDX header: `// SPDX-License-Identifier: Apache-2.0`
- Line length: 99 characters max
- Linter config: [`.golangci.yml`](.golangci.yml)

## Frontend Standards

Preact SPA in `workbench/`. TypeScript strict mode — `npx tsc --noEmit` must pass.

Do not run `npm install` or modify `package-lock.json` without maintainer approval. Dependency changes are a separate PR with justification.

## Agent and Skill Contributions

[AGENTS.md](AGENTS.md) describes the JTBD framework for agent design. Agents are deployed as kagent `Agent` CRDs with `type: BYO`. The platform is **framework-agnostic** — any framework (Google ADK, LangGraph, CrewAI, custom) works as long as the container speaks A2A.

**Modifying the assistant:**

| Artifact | Location |
|:--|:--|
| Agent spec | `agents/assistant/agent.yaml` |
| Prompt | `agents/assistant/prompt.md` |
| Skills | `skills/*/SKILL.md` (vendored to `agents/assistant/skills/`) |
| Helm template (Agent CRD) | `charts/complytime-studio/templates/byo-assistant.yaml` |

After changing prompts or skills:

```bash
make sync-prompts
make sync-skills
```

**Adding a new BYO agent:**

1. Build a container that serves A2A at `/.well-known/agent-card.json`
2. Create a Helm template rendering a kagent `Agent` CRD (`type: BYO`) in `charts/complytime-studio/templates/`
3. Add an entry to `agentDirectory` in `values.yaml` (name, description, url, skills)
4. Pass MCP server URLs as env vars in the CRD's `byo.deployment.env`

The agent appears in the kagent dashboard and A2A traffic routes through the kagent controller. See [AGENTS.md](AGENTS.md) for the full CRD template and checklist.

## OpenSpec Changes

Design changes follow a structured workflow:

```
proposal.md -> design.md -> specs/ -> tasks.md -> implementation
```

All artifacts live in `openspec/changes/<change-name>/`. Templates are in `openspec/schemas/unbound-force/templates/`.

## Testing

- Write tests for all code changes
- Go: table-driven tests, descriptive names, edge cases
- Run `make test` locally before submitting a PR
- Integration tests requiring PostgreSQL use `POSTGRES_TEST_URL`

## Reporting Issues

Use [GitHub Issues](https://github.com/complytime/complytime-studio/issues). For security vulnerabilities, follow [SECURITY.md](SECURITY.md).

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
