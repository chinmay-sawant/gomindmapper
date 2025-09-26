Repository: GoMindMapper — quick reference for AI coding agents

Purpose
- Help an AI agent become productive quickly in this repository: how to run, where to make changes, and project-specific conventions to preserve behavior.

Big picture
- Two main parts:
  1. Analyzer (Go CLI) — cmd/main.go and cmd/analyzer/* scan a Go codebase, extract functions and raw calls, and emit JSON artifacts: `functions.json`, `functionmap.json`, `removed_calls.json`.
  2. Server + UI — cmd/server/main.go hosts an HTTP API and serves the React SPA produced by `mind-map-react` (Vite build outputs to `../docs` per `mind-map-react/vite.config.js`).

Key files & where to look for common tasks
- Run & dev:
  - Start server (production): `go run cmd/server/main.go -path gopdfsuit -addr :8080 --include-external=true --skip-folders="golang.org,gin-gonic,bytedance,ugorji,go-playground"` (example used in README/overview).
  - Start analyzer (CLI): `go run cmd/main.go -path . --include-external=true`
  - Frontend dev: `cd mind-map-react && npm run dev` (hot reload). To produce production assets: `cd mind-map-react && npm run build` (builds into `../docs`).
  - Makefile shortcuts: `make ui-build` (build UI), `make server` (run server).

- Analyzer internals (where to change parsing/filtering):
  - `cmd/analyzer/*` — types, relations, utils, find-calls, and file operations. `analyzer/utils.go:FindCalls` contains the main heuristics (standard package list, regex exclusions) — edit this to change what is considered noise.
  - `analyzer/relations.go` — conversion from raw `FunctionInfo` to JSON `OutRelation`; contains matching rules (exact lookup, suffix matching, partial matches) used for external functions.
  - `cmd/main.go` — project scanning entrypoint and `scanExternalModules` logic for walking go.mod files and deciding which external modules to scan.

- Server & runtime behavior:
  - `cmd/server/main.go` — loader, in-memory cache, endpoints, SPA routing and static serving. Important functions: `load()` (scanning or loading `functionmap.json`), `buildRelationsParallel()` and `filterCallsParallel()`.
  - Endpoints you will use or modify: `GET /api/relations`, `GET /api/search`, `POST /api/reload`, `GET /api/download`.
  - SPA routing: server serves `docs/index.html` at `/`, `/gomindmapper` and `/gomindmapper/view/*`.

Project-specific conventions & gotchas
- Test files are excluded automatically: any file ending with `_test.go` is ignored when collecting functions.
- Production React assets are built into `docs/` (Vite outDir set to `../docs`). When changing the UI, remember to run `npm run build` and commit the resulting `docs/` files if you want the server to serve the updated SPA.
- External modules are scanned only when `--include-external=true`. The analyzer will also attempt flexible matching for external calls; see `analyzer/relations.go` for suffix/partial matching behavior — change here with caution.
- External functions are annotated with `FilePath` starting with `external:` when scanned from modules; some lookup code relies on that.
- The `FindCalls` heuristics include an explicit list of standard packages and exclusion lists (regex functions, wait-group helpers, etc.). Adjust there rather than sprinkling ad-hoc filters across the codebase.

Developer workflows (short):
- Quick run (example):
  go run cmd/server/main.go -path gopdfsuit -addr :8080 --include-external=true --skip-folders="golang.org,gin-gonic,bytedance,ugorji,go-playground"
  - Use `-path` to point to a specific project directory; `--skip-folders` reduces scanning work for large vendor sets.
- Build UI & server for a production demo:
  cd mind-map-react && npm ci && npm run build
  cd .. && go run cmd/server/main.go -path . -addr :8080
- Development (iterate UI + server):
  - Terminal 1: `cd mind-map-react && npm run dev` (hot reload)
  - Terminal 2: `go run cmd/server/main.go -path . -addr :8080` (serve live API)

Where to make safe changes
- To change filtering: update `analyzer/utils.go:FindCalls` and tests around it (no unit tests currently present — add small unit tests for `FindCalls` and `FindCallsWithLines` to lock behaviour).
- To change relation building or external matching: edit `analyzer/relations.go` and add targeted tests for suffix-matching and partial-match fallbacks.
- To change SPA routing or where files are served from: edit `cmd/server/main.go` (search for `docsDir` discovery and static routes) and `mind-map-react/vite.config.js` (base/outDir).

Useful examples (copyable)
- Run scanner for repo root, exclude stdlib scanning:
  go run cmd/main.go -path . --include-external=false
- Build and serve the bundled UI:
  cd mind-map-react && npm run build
  go run cmd/server/main.go -path . -addr :8080

If unsure, open these files first: `README.md`, `cmd/main.go`, `cmd/server/main.go`, `cmd/analyzer/utils.go`, `cmd/analyzer/relations.go`, `mind-map-react/vite.config.js`, and `makefile`.

Behavior to preserve
- Pagination semantics: `GET /api/relations?page=1&pageSize=10` returns a page of root functions and the full dependency closure for those roots.
- Interface-implementation expansion: analyzer attempts to add concrete implementation functions when interface method calls are found — see `enhanceProjectFunctionsWithTypeInfo` / `enhanceProjectFunctionsWithInterfaceDetection` in `cmd/main.go` / `cmd/server/main.go`.

If you need more context or to perform a specific change (e.g., add a flag, change matching rules, or optimize external scanning), state the exact goal and the agent will produce a focused patch and unit tests where applicable.