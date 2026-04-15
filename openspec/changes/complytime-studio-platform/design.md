## Context

ComplyTime Studio is a rebrand and evolution of the GIDE prototype (`github.com/jpower432/gide`). GIDE demonstrated a multi-agent Gemara artifact authoring platform using Google ADK Go, A2A protocol, and kagent BYO deployment. It has four agents (Orchestrator, Threat Modeler, Mapper, Policy Composer), an embedded workbench SPA, and MCP integrations for gemara-mcp, oras-mcp, and github-mcp.

Studio replaces the Mapper with a Gap Analyst, adds native OCI publishing with signing, and promotes the workbench to a first-class UI. The compliance workflow follows the Fox model: agents author criteria-phase artifacts, the orchestrator publishes signed OCI bundles, downstream tools consume them for evaluation and enforcement.

**Stakeholders**: ComplyTime maintainers, compliance teams using complyctl, Gemara specification community.

**Constraints**: Go shop (ADK Go, not Python). Apache 2.0 license. No npm install (embedded SPA pattern). kagent BYO for Kubernetes deployment.

## Goals / Non-Goals

**Goals:**
- Rebrand GIDE → ComplyTime Studio with clean module path and naming
- Replace Mapper agent with Gap Analyst (MappingDocument input → AuditLog output)
- Automate OCI bundle publishing with signing as a native Go tool on the orchestrator
- Deliver a first-class workbench UI with missions, chat, YAML editing, publishing, and registry browsing
- Maintain deployment parity: local Go binary, Docker Compose, and kagent BYO on Kubernetes
- Multi-provider LLM support: Gemini, Anthropic (Vertex + direct), extensible

**Non-Goals:**
- L1 Guidance Author agent (future scope)
- L5/L6 runtime evaluation (that's complyctl/Lula)
- Custom session persistence layer (kagent handles K8s; in-memory for local)
- Standalone MappingDocument authoring (MappingDocuments are always provided as input)
- Mobile or responsive UI (desktop-first workbench)

## Decisions

### D1: Gap Analyst replaces Mapper

**Decision**: Drop the Mapper agent entirely. Introduce a Gap Analyst that takes a MappingDocument as input and produces an AuditLog.

**Rationale**: The Mapper produced crosswalks. Compliance officers need gap reports — coverage classification per reference framework entry. The Gemara `#AuditLog` schema already has `ResultType: "Gap" | "Finding" | "Observation" | "Strength"` with evidence and recommendations. The MappingDocument is a prerequisite, not something Studio needs to author.

**Alternatives considered**:
- Extend Mapper with a `gap-analysis` skill → Overloads one agent with two distinct jobs
- Keep Mapper and add Gap Analyst → Two agents for one workflow, unnecessary complexity

### D2: Native Go `publish_bundle` tool via oras-go

**Decision**: Implement OCI push as a native Go function registered as an ADK `tool.Func` on the orchestrator. Use `oras-go` for push and `notation-go` for signing. Do not depend on oras-mcp for writes.

**Rationale**: oras-mcp is read-only (list, fetch, parse). No push tool exists or is planned. Adding a Go function avoids an external dependency for the critical publishing path. `oras-go` is the same library oras-mcp is built on. Signing with `notation-go` keeps the supply chain tamper-evident. Auth reuses existing `docker login` / `oras login` credential stores.

**Alternatives considered**:
- Wait for oras-mcp push tool → External dependency, unknown timeline
- Shell out to `oras` CLI → Requires CLI binary in container, fragile parsing
- Custom MCP server for push → Extra process, extra failure mode, unnecessary indirection

### D3: Embedded SPA with build step

**Decision**: Replace the vanilla JS workbench with a framework-based SPA. Keep the `go:embed` pattern — built frontend assets are embedded in the Go binary at compile time.

**Rationale**: The current ~1000 lines of vanilla JS with localStorage and CDN imports cannot support the five required capabilities (missions, chat, YAML editor, publishing, registry browser) without becoming unmaintainable. A component framework provides routing, state management, and testability. Embedding preserves the single-binary deployment model.

**UI framework**: To be decided during implementation. Candidates: React (kagent uses it), Svelte (smaller bundle, simpler mental model), or Preact (React API, smaller footprint). Decision criteria: bundle size (embedded), maintainer familiarity, kagent alignment.

### D4: Stateless binary, platform-managed sessions

**Decision**: Studio binary owns no persistence. Use `session.InMemoryService()` for local dev. On Kubernetes, kagent's controller manages sessions via `KAgentSessionService` backed by the controller's database.

**Rationale**: kagent already provides session CRUD (create, get, list, delete), event storage, and task lifecycle via its Go controller HTTP API. Building a custom persistence layer duplicates infrastructure. The Go ADK also has a GORM-backed `DatabaseSessionService` if local persistence is ever needed, but it's not required for the initial release.

### D5: Agent topology — single binary, multi-port A2A

**Decision**: Carry forward GIDE's architecture. All agents run in one Go binary. Each specialist gets its own A2A server on a dedicated port. The orchestrator delegates via ADK sub-agent mechanism.

**Rationale**: Simple deployment (one container), simple scaling (one replica set), specialists are independently addressable via A2A for testing and kagent invocation. The pattern already works.

| Agent | Port | Role |
|:------|:-----|:-----|
| Orchestrator | 8080 | Routing, publishing, workbench serving |
| Threat Modeler | 8001 | L2 artifacts (ThreatCatalog, CapabilityCatalog, ControlCatalog) |
| Gap Analyst | 8002 | L7 artifacts (AuditLog from MappingDocument input) |
| Policy Composer | 8003 | L3 artifacts (RiskCatalog, Policy) |

### D6: Bundle versioning and tagging

**Decision**: Support both user-specified and metadata-derived OCI references. User-specified takes priority. Fall back to `metadata.id` for repo path and `metadata.version` for tag. Content-addressed digest is always returned.

**Rationale**: Automated pipelines need deterministic tagging from artifact metadata. Interactive users need to override. Both paths produce immutable digests.

## Risks / Trade-offs

**[Risk] `adk-anthropic-go` is a community adapter** → It could lag behind ADK releases. Mitigation: Pin version, monitor upstream, Gemini works natively as fallback.

**[Risk] UI framework choice affects long-term maintenance** → Wrong pick creates drag. Mitigation: Defer final choice to implementation, evaluate against bundle size and kagent alignment. The `go:embed` boundary means the frontend is replaceable without changing the backend.

**[Risk] `notation-go` adds signing complexity (key management, trust policies)** → Users need to configure signing keys. Mitigation: Support keyless signing via OIDC (Sigstore-style) for development, require explicit key config for production. Document both paths.

**[Risk] Gap Analyst depends on MappingDocument quality** → Garbage crosswalk in, garbage audit out. Mitigation: Validate MappingDocument via gemara-mcp before analysis. Surface validation errors to the user.

**[Trade-off] Dropping the Mapper means Studio can't create crosswalks** → Users must bring their own MappingDocuments or use external tooling. This is intentional — crosswalk authoring is a distinct domain with its own expertise. Studio focuses on what happens after the crosswalk exists.

**[Trade-off] Single-binary multi-port means all agents share resource limits** → One slow specialist can starve others. Acceptable for the initial release. Future: per-agent containers behind a service mesh if scaling demands it.
