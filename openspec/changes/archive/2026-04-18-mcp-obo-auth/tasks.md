## 1. Helm Chart — MCPServer CRD Transport

- [x] 1.1 In `mcp-servers.yaml`, change `studio-github-mcp` `spec.transportType` from `stdio` to `streamablehttp`
- [x] 1.2 Remove `stdioTransport: {}` from `studio-github-mcp`
- [x] 1.3 Change `spec.deployment.args` from `["stdio", "--toolsets=repos,code_security"]` to `["http", "--port", "3000", "--toolsets=repos,code_security"]`
- [x] 1.4 Remove the `$ghSecret` lookup block, `env` block with `GITHUB_PERSONAL_ACCESS_TOKEN`, and the conditional around it

## 2. Helm Chart — Values Cleanup

- [x] 2.1 Remove `secretName` and `secretKey` fields from `mcpServers.github` in `values.yaml`
- [x] 2.2 Remove the comment referencing `GITHUB_PERSONAL_ACCESS_TOKEN` and unauthenticated mode from `values.yaml`

## 3. Deploy Scripts

- [x] 3.1 Remove the `studio-github-token` Secret creation block from `setup.sh`
- [x] 3.2 Remove the `GITHUB_TOKEN` variable declaration and the warning about unauthenticated mode from `setup.sh`

## 4. Documentation

- [x] 4.1 Update `README.md` to remove `GITHUB_TOKEN` from env vars and document that GitHub OAuth provides agent access to private repos
- [x] 4.2 Update the `mcp-servers.yaml` file header comment to reflect github-mcp using streamablehttp

## 5. Verification

- [x] 5.1 Run `helm template` and verify `studio-github-mcp` renders with `streamablehttp` transport, `http` command args, and no `GITHUB_PERSONAL_ACCESS_TOKEN` env
- [x] 5.2 Run `helm template` and verify gemara-mcp, oras-mcp, and clickhouse-mcp are unchanged
