<div align="center">

# GoMindMapper

Interactive function relationship visualization for Go codebases. Scan a repository, build a filtered call graph, and explore it as an expandable, pannable, zoomable mind map in the browser.

`Go (Analyzer + HTTP API)` + `React (Mind Map UI)` + `Notion‑style Overview`.

---

[Overview (/) Screenshot Placeholder]

</div>

## Table of Contents
1. Overview & Motivation  
2. Architecture  
3. Analyzer (CLI)  
4. HTTP Server & API  
5. React Mind Map UI (`/view`)  
6. Building & Running  
7. Data Model  
8. Customization & Filtering  
9. Roadmap  
10. Contributing  
11. License

---

## 1. Overview & Motivation
Large Go services quickly accumulate implicit structure: entrypoints, routers, middleware, domain handlers, config loaders. Reading raw source to understand call surfaces is slow. GoMindMapper parses the repository, extracts functions and user‑to‑user call edges, filters noise (stdlib/framework), and produces a navigable map so you can:
* Identify top‑level roots (functions not called by any other user function)
* Inspect dependency closures fast
* Page across many roots without loading the entire graph at once
* Upload an offline JSON snapshot or stream directly from a live local scan

## 2. Architecture
```
┌────────────────────────────┐        build (JSON)        ┌──────────────────────────┐
│ Go Analyzer (cmd/main.go)  │ ─────────────────────────▶ │ functionmap.json         │
│ + filtering (analyzer/*)   │                            │ functions.json           │
└──────────┬─────────────────┘                            │ removed_calls.json       │
           │                                               └─────────┬────────────────┘
           │ in‑process reuse (server)                               │ consumed
           ▼                                                         ▼
┌────────────────────────────┐   /api/relations,pagination   ┌──────────────────────────┐
│ Go HTTP Server             │──────────────────────────────▶│ React Mind Map (/view)  │
│ cmd/server/main.go         │◀────────── optional reload ───│ drag/zoom/paginate       │
└────────────────────────────┘                                └──────────────────────────┘
```
Additionally a static, Notion‑inspired overview page ( `mind-map-react/public/overview.html` ) is served at `/` summarizing the project and linking into `/view`.

## 3. Analyzer (CLI)
Scans a path (default `.`) collecting:
* All Go functions (excluding `_test.go`)
* Raw call names inside each body
* Filtered user‑only calls -> `functionmap.json`

Run:
```cmd
cd /d "D:\Chinmay_Personal_Projects\GoMindMapper"
go run cmd/main.go -path .
```
Key outputs:
| File | Purpose |
|------|---------|
| `functions.json` | All discovered functions + raw (unfiltered) calls |
| `functionmap.json` | Reduced relationships (only user→user edges) |
| `removed_calls.json` | Diagnostics: which calls were filtered out |

## 4. HTTP Server & API
`cmd/server/main.go` embeds the scan + an in‑memory cache with pagination across root functions.

Endpoints:
* `GET /api/relations?page=1&pageSize=10` – returns roots slice & full dependency closure for that slice.
* `POST /api/reload` – rescans repository (hot reload data).

Static:
* `/` – overview site (dark, Notion‑style)
* `/view` – React SPA (built assets). Any unknown `/view/*` path falls back to SPA `index.html`.

Start server (after building frontend if you want the UI):
```cmd
cd /d "D:\Chinmay_Personal_Projects\GoMindMapper"
go run cmd/server/main.go -path . -addr :8080
```
Browse:  
* Overview: http://localhost:8080/  
* Mind Map: http://localhost:8080/view/

### Pagination Semantics
* Root = user function not referenced by any other user function.
* Selecting page N returns its root subset AND the full closure of their descendants so the UI can expand locally without extra round trips.

## 5. React Mind Map UI (`/view`)
Location: `mind-map-react/` (Create React App). Now mounted under `/view` using `BrowserRouter` with `basename="/view"`.

Features:
* Drag background to pan, mouse wheel zoom (cursor‑centric)
* Expand/collapse per function node (curved edges, colored by inferred type)
* Collapse All / Reset view controls
* Live vs Offline: toggle between uploaded JSON & server API with pagination controls
* Node detail side panel (name, line, file, called functions)
* Dark UI styling

Dev (hot reload):
```cmd
cd /d "D:\Chinmay_Personal_Projects\GoMindMapper\mind-map-react"
npm install
npm start
```
Then open: `http://localhost:3000/view` (because the app uses `basename="/view"`).  
Optionally comment out the `basename` in `src/index.js` while developing to use root `/`.  
For production / integrated mode run a build:
```cmd
npm run build
```
Server will serve the output at `/view`.

## 6. Building & Running (End‑to‑End)
```cmd
:: 1. (Optional) Regenerate JSON artifacts manually
go run cmd/main.go -path .

:: 2. Build React for /view
cd mind-map-react
npm install
npm run build

:: 3. Start Go server (from repo root)
cd ..
go run cmd/server/main.go -path . -addr :8080

:: 4. Open browser
start http://localhost:8080/
```

## 7. Data Model (Simplified)
```go
type FunctionInfo struct {
  Name    string   // package.func
  Line    int
  FilePath string
  Calls   []string // raw call names extracted (unfiltered)
}

type OutRelation struct {
  Name    string
  Line    int
  FilePath string
  Called  []struct { Name string; Line int; FilePath string }
}
```
`functionmap.json` = slice of `OutRelation` where `Called` only contains user‑scoped edges.

## 8. Customization & Filtering
* Edit `analyzer/utils.go` (`FindCalls`) to tweak exclusion heuristics (stdlib, sync helpers, etc.).
* Edit `analyzer/fileops.go` / server's `filterCalls` for user prefix logic (implement whitelists for frameworks if needed).
* Add flags (future) to include/exclude leaf functions, or to whitelist external packages.

## 9. Roadmap
- [ ] Search endpoint (`/api/search?name=`)
- [ ] Incremental FS watcher to update cache
- [ ] Graph export (GraphML / DOT)
- [ ] Whitelist / blacklist configuration file
- [ ] Function metrics overlay (fan‑in / fan‑out counts)
- [ ] Theming & light mode
- [ ] Deploy container (multi‑stage: build React, embed assets)

## 10. Contributing
PRs + issues welcome. Please:
1. Run `go fmt ./...` & `go vet ./...`
2. Keep analyzer + server filtering logic in sync
3. For UI changes include screenshot or short GIF

## 11. License
MIT (add a `LICENSE` file if distributing publicly).

---
Happy mapping! Open an issue for feature ideas or refinement suggestions.

