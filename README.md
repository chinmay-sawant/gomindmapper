<div align="center">

# GoMindMapper

Interactive function relationship visualization for Go codebases. Scan a repository, build a filtered call graph, and explore it as an expandable, pannable, zoomable mind map in the browser.

`Go (Analyzer + HTTP API)` + `React (Mind Map UI)` + `Notion‑style Overview`.

---

[Overview (/) Screenshot Placeholder]

</div>

## Table of Contents
1. [Overview & Motivation](#1-overview--motivation)
2. [Complete Feature List](#2-complete-feature-list)
3. [Architecture](#3-architecture)  
4. [Analyzer (CLI)](#4-analyzer-cli)  
5. [HTTP Server & API](#5-http-server--api)  
6. [React Mind Map UI](#6-react-mind-map-ui-view)  
7. [Building & Running](#7-building--running-end-to-end)  
8. [Data Model](#8-data-model-simplified)  
9. [Customization & Filtering](#9-customization--filtering)  
10. [Roadmap](#10-roadmap)  
11. [Contributing](#11-contributing)  
12. [License](#12-license)

---

## 1. Overview & Motivation
Large Go services quickly accumulate implicit structure: entrypoints, routers, middleware, domain handlers, config loaders. Reading raw source to understand call surfaces is slow. GoMindMapper parses the repository, extracts functions and user‑to‑user call edges, filters noise (stdlib/framework), and produces a navigable map so you can:

### Core Features
* **Interactive Function Visualization** - Navigate through Go codebases using an expandable, pannable, zoomable mind map interface
* **Smart Root Detection** - Automatically identify top‑level entry points (functions not called by any other user function)
* **Dual Data Modes** - Switch between offline JSON snapshots or live server API with real-time updates
* **Advanced Filtering** - Filter out stdlib, external libraries, and framework noise to focus on user-defined relationships
* **Fast Dependency Inspection** - Instantly explore function call closures and relationships
* **Pagination Support** - Handle large codebases efficiently with server-side pagination across function roots
* **External Library Toggle** - Choose to include or exclude external library calls in analysis using `--include-external` flag

### UI & User Experience Features
* **Custom Node Design** - Google NotebookLLM-inspired function nodes with color-coded types (main, handler, middleware, config, router)
* **Interactive Controls** - Pan (drag background), zoom (mouse wheel), expand/collapse nodes individually
* **Dark/Light Theme Support** - Toggle between themes with system preference detection and localStorage persistence
* **Drag & Drop JSON Upload** - Drop JSON files directly onto the interface for offline analysis
* **Real-time Search** - Search functions by name with debounced input and paginated results
* **Function Details Panel** - Click any node to view detailed information (file path, line numbers, called functions)
* **Responsive Design** - Works seamlessly across different screen sizes and devices

### Data Management Features
* **Live Server Integration** - Connect to running Go server for real-time function mapping
* **Hot Reload Capability** - Refresh data from repository without restarting (`POST /api/reload`)
* **Export Functionality** - Download function relations data as JSON for offline use
* **Multi-format Output** - Generate `functions.json`, `functionmap.json`, and `removed_calls.json` for analysis
* **Smart Call Resolution** - Resolve local function references and handle complex call patterns

---

## 2. Complete Feature List

### 🔍 Analysis & Parsing
• **AST-based Go Analysis** - Uses Go's built-in AST parsing for accurate function extraction  
• **Smart Module Detection** - Automatically detects Go module structure and package relationships  
• **Local Function Resolution** - Resolves function references within packages  
• **External Library Control** - Toggle inclusion/exclusion of external library calls with `--include-external` flag  
• **Test File Exclusion** - Automatically excludes `_test.go` files from analysis  
• **Call Pattern Recognition** - Identifies and categorizes different types of function calls  

### 🌐 Server & API
• **RESTful API** - Complete REST API with pagination, search, and data management  
• **Real-time Search** - `/api/search` endpoint with function name matching and pagination  
• **Hot Reload** - `/api/reload` endpoint for refreshing data without server restart  
• **Data Export** - `/api/download` endpoint for exporting function relations as JSON  
• **CORS Support** - Built-in CORS middleware for frontend integration  
• **Concurrent Safety** - Thread-safe operations with proper mutex handling  
• **Static File Serving** - Serves React frontend with SPA routing support  

### 🎨 Interactive UI & Visualization  
• **Interactive Mind Map** - Pannable, zoomable, expandable function relationship visualization  
• **Custom Node Design** - Google NotebookLLM-inspired nodes with color-coded function types  
• **Drag & Drop Upload** - Drop JSON files directly onto interface for offline analysis  
• **Dark/Light Theme Toggle** - Switch themes with system preference detection and persistence  
• **Real-time Search** - Debounced search with instant results across function names  
• **Function Details Panel** - Comprehensive information display on node selection  
• **Responsive Design** - Works seamlessly across different screen sizes  

### 🔧 User Experience
• **Dual Data Modes** - Switch between live server API and offline JSON file analysis  
• **Pagination Controls** - Handle large codebases efficiently with server-side pagination  
• **Keyboard Shortcuts** - Navigate and control the interface using keyboard  
• **Loading States** - Proper loading indicators for all async operations  
• **Error Handling** - User-friendly error messages and recovery options  
• **Auto-save Preferences** - Theme and settings persistence using localStorage  

### 📊 Data Management
• **Multiple Output Formats** - Generate `functions.json`, `functionmap.json`, and `removed_calls.json`  
• **Smart Filtering** - Filter out stdlib, external libraries, and framework noise  
• **Root Function Detection** - Automatically identify entry points and top-level functions  
• **Dependency Closure** - Complete dependency trees for selected function roots  
• **Call Relationship Mapping** - Detailed function-to-function call relationships  
• **Diagnostic Information** - Track and report filtered calls for analysis  

### 🚀 Performance & Scalability
• **In-Memory Caching** - Fast access to parsed function relationships  
• **Efficient Re-renders** - Optimized React components with proper memoization  
• **Lazy Loading** - Load function details on-demand  
• **Debounced Search** - Prevent excessive API calls during search typing  
• **Pagination Support** - Handle large codebases without performance issues  
• **Background Processing** - Non-blocking analysis and data processing  

### 🛠️ Developer Experience
• **CLI & Server Modes** - Flexible usage as CLI tool or web service  
• **Development Hot Reload** - Vite-powered frontend development with instant updates  
• **Comprehensive Documentation** - Detailed setup and usage instructions  
• **Cross-platform Support** - Works on Windows, macOS, and Linux  
• **Easy Setup** - Simple installation and configuration process  
• **Extensible Architecture** - Clean separation of concerns for easy modification  

---

## 3. Architecture
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

## 4. Analyzer (CLI)
Scans a path (default `.`) collecting:
* All Go functions (excluding `_test.go`)
* Raw call names inside each body  
* Filtered user‑only calls -> `functionmap.json`
* Optional external library call inclusion

### CLI Features:
* **Path Specification** - Analyze any Go repository directory with `-path` flag
* **External Library Control** - Use `--include-external` flag to include/exclude external library calls
* **Smart Module Detection** - Automatically detects Go module and package structure
* **AST-based Analysis** - Uses Go's AST parsing for accurate function extraction
* **Local Function Resolution** - Resolves local function references within packages

Run:
```cmd
cd /d "D:\GoMindMapper"
go run cmd/main.go -path . --include-external=false
```

Key outputs:
| File | Purpose | When Generated |
|------|---------|----------------|
| `functions.json` | All discovered functions + raw (unfiltered) calls | Always |
| `functionmap.json` | Reduced relationships (only user→user edges) | Always |
| `removed_calls.json` | Diagnostics: which calls were filtered out | Only when `--include-external=false` |

## 5. HTTP Server & API
`cmd/server/main.go` embeds the scan + an in‑memory cache with pagination across root functions.

### API Endpoints:
* **`GET /api/relations?page=1&pageSize=10`** – Returns paginated roots slice & full dependency closure
* **`GET /api/search?q=functionName&page=1&pageSize=10`** – Search functions by name with pagination
* **`POST /api/reload`** – Rescans repository (hot reload data without restart)
* **`GET /api/download`** – Download current function relations data as JSON

### Server Features:
* **In-Memory Caching** - Fast access to parsed function relationships
* **CORS Support** - Built-in CORS middleware for frontend integration
* **Concurrent Safety** - Thread-safe operations with proper mutex handling
* **External Library Toggle** - `--include-external` flag support for server mode
* **Static File Serving** - Serves built React frontend assets
* **Fallback Routing** - SPA routing support with proper fallbacks

### Static Routes:
* **`/`** – Overview site (dark, Notion‑style landing page)
* **`/view`** – React SPA (built mind map interface)
* **`/view/*`** – SPA fallback routing for React Router

Start server (after building frontend if you want the UI):
```cmd
cd /d "D:\GoMindMapper"
go run cmd/server/main.go -path . -addr :8080 --include-external=false
```

Browse:  
* **Overview:** http://localhost:8080/gomindmapper/
* **Mind Map:** http://localhost:8080/gomindmapper/view/

### Pagination & Search Semantics
* **Root Function** = user function not referenced by any other user function
* **Page Selection** - Returns root subset AND full closure of descendants for local expansion
* **Search Results** - Function name matching with dependency closure included
* **Real-time Updates** - Reload endpoint allows data refresh without server restart

## 6. React Mind Map UI (`/view`)
Location: `mind-map-react/` (Vite + React). Mounted under `/view` using `BrowserRouter` with `basename="/view"`.

### Interactive Features:
* **Pan & Zoom** - Drag background to pan, mouse wheel zoom (cursor‑centric)
* **Node Expansion** - Expand/collapse individual function nodes with smooth animations
* **Global Controls** - "Collapse All" and "Reset View" buttons for quick navigation
* **Dual Data Modes** - Toggle between live server API and offline JSON file upload
* **Drag & Drop Upload** - Drop JSON files directly onto the interface
* **Real-time Search** - Debounced search with instant results and pagination

### Visual Features:
* **Custom Node Design** - Google NotebookLLM-inspired function nodes
* **Color-coded Types** - Visual distinction for main, handler, middleware, config, router functions
* **Dynamic Sizing** - Responsive node sizing based on content
* **Curved Edges** - Smooth connecting lines between function calls
* **Glow Effects** - Visual highlights for selected nodes
* **Level-based Coloring** - Different colors for different call depth levels

### User Interface:
* **Dark/Light Theme Toggle** - Switch themes with system preference detection
* **Function Details Panel** - Detailed information on node click (name, file path, line numbers, called functions)
* **Navigation Controls** - Pagination controls for large codebases
* **Loading States** - Proper loading indicators for all async operations
* **Error Handling** - User-friendly error messages and recovery options

### Data Management:
* **Live Server Integration** - Real-time connection to Go server API
* **Offline Mode** - Upload and analyze JSON files without server
* **Hot Reload** - Refresh server data without page reload
* **Export Options** - Download current dataset as JSON
* **Search & Filter** - Find specific functions across large codebases

### Technical Features:
* **React Router** - SPA routing with proper navigation
* **Context API** - Theme management with localStorage persistence  
* **Ref Management** - Optimized DOM interactions and focus handling
* **Event Handling** - Keyboard shortcuts and mouse interactions
* **Performance** - Efficient re-renders and memoization

### Development:
Dev (hot reload):
```cmd
cd /d "D:\GoMindMapper\mind-map-react"
npm install
npm run dev
```
Then open: `http://localhost:5173/view` (Vite default port with `basename="/view"`).

For production build:
```cmd
npm run build
```
Server will serve the built output at `/view`.

## 7. Building & Running (End‑to‑End)

### Quick Start (Server with UI):
```cmd
:: 1. Build React frontend for /view
cd mind-map-react
npm install
npm run build

:: 2. Start Go server with external library filtering (from repo root)
cd ..
go run cmd/server/main.go -path . -addr :8080 --include-external=false

:: 3. Open browser
start http://localhost:8080/
```

### CLI-only Analysis:
```cmd
:: Generate JSON artifacts for offline analysis
go run cmd/main.go -path . --include-external=false

:: This creates:
:: - functions.json (all functions with raw calls)
:: - functionmap.json (filtered user-to-user relationships)  
:: - removed_calls.json (diagnostic info about filtered calls)
```

### Development Mode:
```cmd
:: 1. Start Go server
go run cmd/server/main.go -path . -addr :8080

:: 2. In another terminal, start React dev server
cd mind-map-react
npm run dev

:: 3. Open development UI
start http://localhost:5173/view
```

### Available Command-line Options:

**Analyzer CLI:**
- `-path <directory>` - Repository path to analyze (default: current directory)
- `--include-external` - Include external library calls in output (default: false)

**Server:**
- `-path <directory>` - Repository path to analyze (default: current directory)
- `-addr <address>` - Server listen address (default: :8080)
- `--include-external` - Include external library calls in relations (default: false)

## 8. Data Model (Simplified)
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

## 9. Customization & Filtering
* Edit `analyzer/utils.go` (`FindCalls`) to tweak exclusion heuristics (stdlib, sync helpers, etc.).
* Edit `analyzer/fileops.go` / server's `filterCalls` for user prefix logic (implement whitelists for frameworks if needed).
* Add flags (future) to include/exclude leaf functions, or to whitelist external packages.

## 10. Roadmap

### ✅ Completed Features:
- [x] Search endpoint (`/api/search?name=`) with pagination
- [x] Theming & light mode support with system preference detection
- [x] External library inclusion control with `--include-external` flag
- [x] Drag & drop JSON file upload functionality
- [x] Real-time search with debouncing
- [x] Custom node design inspired by Google NotebookLLM
- [x] Hot reload capability (`POST /api/reload`)
- [x] Function details panel with comprehensive information
- [x] Data export functionality (`GET /api/download`)

### 🚧 Planned Features:
- [ ] Incremental FS watcher to update cache automatically
- [ ] Graph export formats (GraphML / DOT / SVG)
- [ ] Configuration file support for whitelist/blacklist patterns
- [ ] Function metrics overlay (fan‑in / fan‑out counts, complexity metrics)
- [ ] Deploy container (multi‑stage: build React, embed assets)
- [ ] Code metrics integration (cyclomatic complexity, lines of code)
- [ ] Interactive filtering controls in UI
- [ ] Bookmarking and saved views
- [ ] Export to common graph formats
- [ ] Plugin system for custom analyzers

## 11. Contributing
PRs + issues welcome. Please:
1. Run `go fmt ./...` & `go vet ./...`
2. Keep analyzer + server filtering logic in sync
3. For UI changes include screenshot or short GIF

## 12. License
MIT (add a `LICENSE` file if distributing publicly).

---
Happy mapping! Open an issue for feature ideas or refinement suggestions.

