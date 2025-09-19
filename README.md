# GoMindMapper `cmd` analyzer

This folder contains a small Go-based code analyzer used to scan a repository and produce two JSON artifacts:

- `functions.json` — full list of discovered functions and the raw calls found inside each function body (before user-call filtering). This preserves the original `Calls` arrays for debugging.
- `functionmap.json` — cleaned function relationship map containing only user-defined call edges. Entries that do not call any user-defined functions are omitted.
- `removed_calls.json` — report listing calls removed from `functions.json` when filtering out non-user calls. Useful for tuning exclusion/whitelisting rules.

Files
- `main.go` — orchestrates scanning the repository, extracting functions, and building the function map.
- `analyzer/utils.go` — contains `FindFunctionBody`, `FindCalls`, and `GetModule`.
- `analyzer/fileops.go` — writes `functions.json` and `removed_calls.json`.
- `analyzer/types.go` — data structures used by the analyzer.

## Usage (CLI Analyzer)

From the repository root run (Windows `cmd.exe` / PowerShell):

```cmd
cd /d "D:\Chinmay_Personal_Projects\GoMindMapper"
go run cmd/main.go
```

Optional: analyze a specific subpath

```cmd
go run cmd/main.go -path ./EmployeeApp
```

Outputs
- `functions.json` — full details of discovered functions (kept as-is for debugging and further processing).
- `functionmap.json` — simplified graph of user-defined function call relations (only functions that call other user functions are included).
- `removed_calls.json` — report of which calls were filtered out when producing `functions.json`.

Customization
- To change which low-level or framework calls are excluded, edit `cmd/analyzer/utils.go`:
  - The `FindCalls` function contains regex/filter rules for excluding regex helpers, error handling, wait-group methods, and standard library packages.
- To change how non-user calls are identified, edit `cmd/analyzer/fileops.go` where user package prefixes are computed and used to filter calls. You can implement a whitelist for framework prefixes (e.g., `c` for Gin context) by modifying the `userPrefixes` logic or by adding a `whitelist` map.

Notes
- `functionmap.json` intentionally omits functions that don't call other user-defined functions to keep the graph focused on actual function-to-function relationships.
- `functions.json` still includes all discovered functions and the raw set of calls (useful for debugging and tuning filters).

## HTTP Server with In‑Memory Cache & Pagination

A lightweight HTTP server is included at `cmd/server/main.go` to serve the function relationship graph with:

* In‑memory cached scan results (loaded at startup and on demand via reload endpoint)
* Pagination across root (entry point) functions
* Dependency closure expansion per returned page (so each page includes all descendants of the selected root set)
* Simple CORS enabled for local frontend development

### Endpoints

`GET /api/relations?page=1&pageSize=10`
Returns JSON:
```json
{
  "page": 1,
  "pageSize": 10,
  "totalRoots": 3,
  "roots": [ { "name": "main.main", "line": 9, "filePath": "EmployeeApp\\main.go", "called": [...] } ],
  "data": [ { "name": "main.main", "line": 9, "filePath": "...", "called": [ {"name": "routes.SetupRouter", ...} ] } ],
  "loadedAt": "2025-09-19T10:00:00Z"
}
```

`POST /api/reload`
Triggers a full rescan of the repository and refreshes the cache.

### Run the server
```cmd
cd /d "D:\Chinmay_Personal_Projects\GoMindMapper"
go run cmd/server/main.go -path . -addr :8080
```

### Pagination semantics
* Roots = functions not called by any other user function (entry points)
* Each page chooses a slice of these roots and returns the full transitive closure of their calls in `data`.
* This ensures the frontend can expand dependency chains without additional round trips for that page.

## React Frontend (Mind Map)

Located in `mind-map-react/`.

Features:
* Toggle between a static uploaded JSON file and the live server (`Use Live Server` checkbox).
* Pagination controls (page forward/back, page size selection) appear when live mode is enabled.
* Zoom, pan (drag background), collapse/reset, and node highlighting.

### Start frontend (after installing dependencies)
```cmd
cd /d "D:\Chinmay_Personal_Projects\GoMindMapper\mind-map-react"
npm install
npm run dev
```

Then open the printed local URL (typically `http://localhost:5173`) and optionally enable `Use Live Server` (expects backend on `http://localhost:8080`).

### Uploading a JSON file
You can still drag & drop or choose a previously generated `functionmap.json` if you prefer offline usage.

## Development Notes
* The server duplicates a small portion of the CLI logic (function scanning) to remain standalone.
* Call filtering logic is mirrored (see `filterCalls` in `cmd/server/main.go`). Future improvement: refactor shared filtering into the analyzer package to avoid drift.
* `analyzer.BuildRelations` now centralizes relation construction for both CLI and server.

## Next steps (suggested)
- Add a CLI flag to control whether `functionmap.json` should omit functions with no calls.
- Use `go list` to map module/package paths for more robust user-call detection instead of the simple prefix heuristic.
- Export the function map in CSV or GraphML for import into visualization tools.
 - Add search endpoint (`/api/search?name=...`) to fetch a specific function's neighborhood lazily.
 - Incremental scan / file watcher to update cache without full rebuild.
 - Authentication / rate limiting if deployed.

If you'd like, I can add a `-whitelist` flag to keep certain prefixes (for example `c` or `router`) in the `functionmap.json` output.
