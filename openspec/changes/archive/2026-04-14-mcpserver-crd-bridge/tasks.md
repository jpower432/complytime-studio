## 1. Spike: Verify KMCP-generated Service Convention

- [x] 1.1 Create a throwaway `MCPServer` CRD (`test-stdio-mcp`) with `transportType: stdio` using a simple image (e.g., `busybox` or the gemara-mcp image) and inspect what the KMCP controller generates (Service name, port, path, RemoteMCPServer URL)
- [x] 1.2 Document the generated Service port, endpoint path, and any labels/annotations the KMCP controller applies
- [x] 1.3 Delete the test MCPServer resource

## 2. Rewrite MCP Server Templates

- [x] 2.1 Replace the gemara-mcp Deployment+Service in `mcp-servers.yaml` with an `MCPServer` CRD (`studio-gemara-mcp`, `transportType: stdio`, `cmd: serve`)
- [x] 2.2 Replace the oras-mcp Deployment+Service with an `MCPServer` CRD (`studio-oras-mcp`, `transportType: stdio`, `cmd: serve`)
- [x] 2.3 Replace the github-mcp Deployment+Service with an `MCPServer` CRD (`studio-github-mcp`, `transportType: stdio`, `cmd: stdio`, `args: [--toolsets=repos,code_security]`, `env` with `GITHUB_PERSONAL_ACCESS_TOKEN` from secret)
- [x] 2.4 Update `values.yaml` MCP server config to align with MCPServer CRD fields (remove stale Deployment-era config)

## 3. Update Orchestrator URL Env Vars

- [x] 3.1 Update `GEMARA_MCP_URL`, `ORAS_MCP_URL`, and `GITHUB_MCP_URL` in `agent-orchestrator.yaml` to match the KMCP-generated Service endpoints (name, port, path) discovered in task 1.2

## 4. Validate End-to-End

- [x] 4.1 Run `helm upgrade --install` and verify all three MCPServer resources reconcile to Ready
- [x] 4.2 Confirm orchestrator pod starts without "connection refused" or "proxy disabled" warnings for gemara-mcp and oras-mcp
- [x] 4.3 Verify `helm template` output contains zero `apps/v1 Deployment` or `v1 Service` resources for the three MCP servers

## 5. Cleanup

- [x] 5.1 Remove the `stdin: true` / `stdinOnce: false` workaround from any remaining templates
- [x] 5.2 Update chart comments and `NOTES.txt` to reflect MCPServer CRD architecture
