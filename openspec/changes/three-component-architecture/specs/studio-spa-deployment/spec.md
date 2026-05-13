## ADDED Requirements

### Requirement: Studio SPA runs as standalone container
The Studio SPA SHALL be deployable as an independent Nginx container that serves static assets without any dependency on the Go gateway binary.

#### Scenario: Studio container starts without platform
- **WHEN** the Studio container starts with `PLATFORM_URL` set to a valid URL
- **THEN** Nginx serves `index.html` for all non-asset paths (history-mode fallback)
- **THEN** static assets (JS, CSS) are served with appropriate cache headers

#### Scenario: Studio container starts with unreachable platform
- **WHEN** the Studio container starts and the platform is unreachable
- **THEN** the SPA loads in the browser and displays a connection error to the user
- **THEN** the Nginx container itself remains healthy (no crash loop)

### Requirement: Runtime config injection via env.js
The Studio container SHALL inject `PLATFORM_URL` at runtime via `envsubst` on an `env.js.template` file, producing a `env.js` that sets `window.__STUDIO_CONFIG__`.

#### Scenario: Same image deployed to different environments
- **WHEN** the same Studio container image is deployed with `PLATFORM_URL=https://api.staging.example.com`
- **THEN** the SPA makes API calls to `https://api.staging.example.com/api/*`

#### Scenario: env.js is not cached by browser
- **WHEN** a browser requests `/env.js`
- **THEN** Nginx serves it with `Cache-Control: no-cache` header

### Requirement: SPA API client uses configurable base URL
All API calls from the SPA SHALL prepend the platform URL from `window.__STUDIO_CONFIG__.platformUrl` to request paths. When the value is empty string, same-origin behavior is preserved.

#### Scenario: Cross-origin API call
- **WHEN** `platformUrl` is `http://platform:8080` and the SPA calls `/api/policies`
- **THEN** the fetch request targets `http://platform:8080/api/policies`

#### Scenario: Same-origin fallback for local dev
- **WHEN** `platformUrl` is `""` (empty string)
- **THEN** the fetch request targets `/api/policies` (relative, same-origin)

### Requirement: A2A SSE connections use platform URL
The A2A streaming client SHALL use the same `platformUrl` base for `/a2a/*` connections.

#### Scenario: Chat connects to platform A2A endpoint
- **WHEN** user sends a message in the chat assistant
- **THEN** the SSE connection opens to `${platformUrl}/a2a/studio-assistant`

### Requirement: Studio Dockerfile produces minimal image
The Studio Dockerfile SHALL use a multi-stage build: Node stage for `vite build`, Nginx stage for serving.

#### Scenario: Image size
- **WHEN** the Studio image is built
- **THEN** the final image contains only Nginx, the built `dist/` assets, `nginx.conf`, and `env.js.template`
- **THEN** no Node.js runtime, `node_modules`, or source files are present in the final stage

### Requirement: Nginx history-mode fallback
Nginx SHALL serve `index.html` for any request path that does not match a physical file in `dist/`.

#### Scenario: Deep link to policy view
- **WHEN** a browser requests `/policies/ac-1`
- **THEN** Nginx serves `dist/index.html` (SPA handles client-side routing)

#### Scenario: Static asset request
- **WHEN** a browser requests `/assets/index-abc123.js`
- **THEN** Nginx serves the file directly with long-lived cache headers
