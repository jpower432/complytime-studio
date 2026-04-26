## MODIFIED Requirements

### Requirement: React SPA replaces vanilla JS workbench
The workbench SHALL be a React single-page application built to `workbench/dist/` and embedded in the gateway binary via `go:embed`. All view components SHALL use semantic HTML elements (`<section>`, `<article>`, `<table>`, `<header>`) and include `data-*` attributes as defined in the agent DOM contract.

#### Scenario: SPA build and embed
- **WHEN** the React app is built (`npm run build` or equivalent)
- **THEN** static assets are output to `workbench/dist/`
- **THEN** the gateway embeds and serves them at `/` with SPA fallback routing

#### Scenario: Semantic structure in all views
- **WHEN** any view component renders
- **THEN** the root element is a `<section>` with an `<h2>` heading
- **THEN** tabular data uses `<table>` with `<thead>` and `<th>` headers
- **THEN** card groups use `<article>` elements
