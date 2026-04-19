## 1. Helm Chart Updates

- [x] 1.1 Change `runtime: go` to `runtime: python` for all three agents in `agent-specialists.yaml`
- [x] 1.2 Restore `allowedHeaders: [Authorization]` on every `studio-github-mcp` tool reference in `agent-specialists.yaml`
- [x] 1.3 Verify `tokenSecret` / `secretRefs` fallback remains intact in `mcp-servers.yaml`

## 2. Validation

- [ ] 2.1 Deploy to Kind cluster and confirm all three agent pods start with Python runtime
- [ ] 2.2 Verify `allowedHeaders` warning is gone from `helm upgrade` output
- [ ] 2.3 Submit a job through the workbench and confirm the agent can call GitHub MCP tools
- [ ] 2.4 Test with static `tokenSecret` (no OBO) to confirm fallback still works

## 3. Upstream Tracking

- [ ] 3.1 Monitor [kagent#1679](https://github.com/kagent-dev/kagent/issues/1679) for Go runtime fix
- [ ] 3.2 When fixed upstream, revert `runtime` to `go` and re-test OBO flow
