<div align="center">

# GoMindMapper

🚀 **Advanced Go Function Relationship Visualizer** 🚀

Interactive function relationship visualization for Go codebases with intelligent type resolution, interface implementation detection, and external module analysis. Scan any Go repository and explore it through an expandable, pannable, zoomable mind map.

`Go (AST Analyzer + HTTP API)` + `React (Interactive Mind Map)` + `Notion‑style UI`

[![GitHub Stars](https://img.shields.io/github/stars/chinmay-sawant/gomindmapper?style=social)](https://github.com/chinmay-sawant/gomindmapper) 
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/chinmay-sawant/gomindmapper)](https://golang.org/)

---

</div>

## 📋 Table of Contents
1. [🚀 Quick Start](#-quick-start)
2. [✨ Features Overview](#-features-overview)
3. [🏗️ Architecture](#️-architecture)
4. [⚙️ Installation & Setup](#️-installation--setup)
5. [🔧 Development](#-development)
6. [📖 Usage Guide](#-usage-guide)
7. [🎯 Advanced Features](#-advanced-features)
8. [🔍 API Reference](#-api-reference)
9. [📊 Data Models](#-data-models)
10. [🎨 Customization](#-customization)
11. [🗺️ Roadmap](#️-roadmap)
12. [🤝 Contributing](#-contributing)
13. [📄 License](#-license)

---

## 🚀 Quick Start

Get started with GoMindMapper in under 2 minutes:

### Single Command Deployment
```bash
# Clone and run (example analyzing the 'gopdfsuit' subdirectory)
git clone https://github.com/chinmay-sawant/gomindmapper.git
cd gomindmapper
go run cmd/server/main.go -path gopdfsuit -addr :8080 --include-external=true --skip-folders="golang.org,gin-gonic,bytedance,ugorji,go-playground"
```

**Command Flags:**
- `-path <dir>`: Repository/subfolder to analyze (e.g., `gopdfsuit`)
- `-addr <addr>`: HTTP server address (default `:8080`)
- `--include-external`: Include external module functions in analysis
- `--skip-folders`: Comma-separated dependency prefixes to skip during external scanning

**Access Points:**
- 🌐 **Overview**: http://localhost:8080/gomindmapper/
- 🗺️ **Mind Map**: http://localhost:8080/gomindmapper/view/

> **Note**: Production React assets are automatically served by the Go server — no separate frontend setup required!

### Makefile Shortcuts
```bash
make ui-build   # Build React frontend
make server     # Start Go server
make ui         # Start React dev server
make run        # Run CLI analyzer
```

---

## ✨ Features Overview

GoMindMapper goes beyond simple function visualization with advanced Go code analysis capabilities:

### 🎯 Core Analysis Engine
* **🧠 AST-based Go Analysis** - Uses Go's built-in AST parsing for accurate function extraction
* **🔍 Smart Root Detection** - Automatically identify top-level entry points (functions not called by any other user function)
* **🏗️ Interface Implementation Detection** - Discover concrete implementations of interfaces and add them to call graphs
* **🔗 Type Resolution Engine** - Resolve method calls through comprehensive type analysis
* **📦 External Module Scanning** - Recursively scan external dependencies with intelligent filtering
* **🎛️ Advanced Filtering** - Multi-layer filtering: stdlib, external libraries, framework noise, custom patterns
* **⚡ Performance Optimization** - Parallel processing, in-memory caching, and efficient data structures

### 🎨 Interactive UI & Visualization
* **🗺️ Google NotebookLLM-inspired Nodes** - Custom-designed function nodes with color-coded types (main, handler, middleware, config, router)
* **🖱️ Intuitive Controls** - Pan (drag background), zoom (mouse wheel), expand/collapse nodes individually
* **🌓 Advanced Theming** - Dark/light theme with system preference detection and localStorage persistence
* **📤 Drag & Drop Upload** - Drop JSON files directly onto interface for offline analysis
* **🔎 Real-time Search** - Debounced search with instant results and pagination
* **📋 Function Details Panel** - Comprehensive information display on node selection (file path, line numbers, calls)
* **📱 Responsive Design** - Works seamlessly across desktop, tablet, and mobile devices
* **🎞️ Screenshot Slideshow** - Interactive feature showcase with auto-play and navigation
* **📊 Comparison Table** - Built-in comparison with other Go visualization tools

### 🔧 Data Management & Integration
* **🔄 Dual Data Modes** - Switch between offline JSON snapshots or live server API
* **🔥 Hot Reload Capability** - Refresh data from repository without restarting (`POST /api/reload`)
* **💾 Multi-format Export** - Download as JSON, with planned support for GraphML/DOT/SVG
* **📊 Multiple Output Formats** - Generate `functions.json`, `functionmap.json`, and `removed_calls.json`
* **🌐 Live Server Integration** - RESTful API with pagination, search, and real-time updates
* **🔒 Concurrent Safety** - Thread-safe operations with proper mutex handling

---

## 🏗️ Architecture

GoMindMapper follows a modern 3-tier architecture with intelligent caching and real-time capabilities:

```
┌────────────────────────────┐    JSON Artifacts    ┌──────────────────────────────┐
│ 🔍 Go Analyzer (CLI)        │ ──────────────────▶ │ 📄 functionmap.json         │
│ • AST Parsing               │                      │ 📄 functions.json           │
│ • Type Resolution           │                      │ 📄 removed_calls.json       │
│ • Interface Detection       │                      └─────────┬────────────────────┘
│ • External Module Scanning  │                                │
└──────────┬─────────────────┘                                │ Consumed by
           │ In-process Reuse                                   ▼
           ▼                                   ┌──────────────────────────────┐
┌────────────────────────────┐   REST API + WebSockets   │ ⚛️ React Mind Map UI         │
│ 🌐 Go HTTP Server           │ ◀────────────────────────▶ │ • Interactive Visualization  │
│ • RESTful API               │                             │ • Theme Management          │
│ • Real-time Updates         │                             │ • Search & Filter           │
│ • Pagination Engine         │                             │ • Drag & Drop               │
│ • Concurrent Safety         │                             │ • Responsive Design         │
│ • Static Asset Serving      │                             └──────────────────────────────┘
└────────────────────────────┘                             
```

### Key Components:
- **📁 `cmd/main.go`** - CLI analyzer with interface detection and type resolution
- **📁 `cmd/server/main.go`** - HTTP server with in-memory caching and parallel processing
- **📁 `cmd/analyzer/*`** - Core analysis engine (types, relations, utils, external modules)
- **📁 `mind-map-react/`** - Vite+React SPA with advanced UI components
- **📁 `docs/`** - Production build output served by Go server

## ⚙️ Installation & Setup

### Prerequisites
- **Go 1.23** - [Download Go](https://golang.org/dl/)
- **Node.js 16+** & **npm** - [Download Node.js](https://nodejs.org/) (only for development)
- **Git** - [Download Git](https://git-scm.com/)

### Installation Options

#### Option 1: Direct Git Clone (Recommended)
```bash
# Clone the repository
git clone https://github.com/chinmay-sawant/gomindmapper.git
cd gomindmapper

# Run immediately (production-ready)
go run cmd/server/main.go -path . -addr :8080
```

#### Option 2: Go Install (Coming Soon)
```bash
# Future release
go install github.com/chinmay-sawant/gomindmapper@latest
gomindmapper --help
```

### Build from Source
```bash
# Clone and build
git clone https://github.com/chinmay-sawant/gomindmapper.git
cd gomindmapper

# Build frontend (optional - for latest UI changes)
cd mind-map-react
npm install && npm run build
cd ..

# Build Go binary
go build -o gomindmapper cmd/server/main.go

# Run
./gomindmapper -path /path/to/your/go/project -addr :8080
```

---

## 🔧 Development

### Development Environment Setup
```bash
# 1. Clone repository
git clone https://github.com/chinmay-sawant/gomindmapper.git
cd gomindmapper

# 2. Start backend server
go run cmd/server/main.go -path . -addr :8080

# 3. In another terminal, start frontend dev server
cd mind-map-react
npm install
npm run dev

# 4. Access development UI
# Frontend dev server: http://localhost:5173/gomindmapper/view
# Backend API: http://localhost:8080/api/relations
```

### Development Workflow
- **Backend changes**: Restart `go run cmd/server/main.go`
- **Frontend changes**: Auto-reload via Vite dev server
- **Build for production**: `make ui-build` then `make server`

### Project Structure
```
gomindmapper/
├── 📁 cmd/                    # Go applications
│   ├── 📄 main.go             # CLI analyzer
│   ├── 📁 analyzer/           # Core analysis engine
│   │   ├── 📄 types.go        # Data structures
│   │   ├── 📄 relations.go    # Relationship building
│   │   ├── 📄 utils.go        # Function call extraction
│   │   ├── 📄 types_resolver.go # Type resolution & interface detection
│   │   └── 📄 external.go     # External module scanning
│   └── 📁 server/
│       └── 📄 main.go          # HTTP server
├── 📁 mind-map-react/         # React frontend
│   ├── 📄 vite.config.js      # Build config (outputs to ../docs)
│   ├── 📄 package.json        # Dependencies
│   └── 📁 src/
│       ├── 📄 App.jsx          # Main app component
│       ├── 📁 components/      # UI components
│       └── 📁 contexts/        # Theme management
├── 📁 docs/                   # Production build output
├── 📄 makefile               # Development shortcuts
└── 📄 README.md              # This file
```

---

## 📖 Usage Guide

### CLI Analyzer Mode
Generate JSON artifacts for offline analysis:

```bash
# Basic analysis (user functions only)
go run cmd/main.go -path . --include-external=false

# Advanced analysis (includes external dependencies)
go run cmd/main.go -path . --include-external=true --skip-folders="golang.org,google.golang.org"

# Analyze specific project
go run cmd/main.go -path /path/to/your/go/project --include-external=true
```

**Generated Files:**
| File | Purpose | Content |
|------|---------|--------|
| `functions.json` | Raw function data | All discovered functions + unfiltered calls |
| `functionmap.json` | Filtered relationships | User→user function relationships only |
| `removed_calls.json` | Diagnostics | Calls filtered out during analysis |

### Server Mode (Recommended)
Start the HTTP server with live analysis and web UI:

```bash
# Basic server
go run cmd/server/main.go -path . -addr :8080

# Advanced with external libraries
go run cmd/server/main.go -path . -addr :8080 --include-external=true --skip-folders="golang.org,gin-gonic"

# Analyze external project
go run cmd/server/main.go -path /path/to/project -addr :8080
```

**Access Points:**
- 🌐 **Overview**: http://localhost:8080/gomindmapper/ 
- 🗺️ **Mind Map**: http://localhost:8080/gomindmapper/view/
- 📡 **API Docs**: http://localhost:8080/api/relations

### Command Line Options
| Flag | Description | Default | Example |
|------|-------------|---------|--------|
| `-path <dir>` | Repository to analyze | `.` (current) | `-path ./myproject` |
| `-addr <address>` | Server listen address | `:8080` | `-addr :3000` |
| `--include-external` | Include external modules | `false` | `--include-external=true` |
| `--skip-folders <patterns>` | Skip dependency patterns | `""` | `--skip-folders="golang.org,gin-gonic"` |

---

## 🎯 Advanced Features

GoMindMapper includes several advanced features that set it apart from other Go visualization tools:

### 🧠 Interface Implementation Detection
Automatically discovers concrete implementations of interfaces and includes them in the call graph:

```go
// Example: Interface definition
type UserService interface {
    CreateUser(user User) error
    GetUser(id string) (*User, error)
}

// Implementation detection finds:
type DatabaseUserService struct { /* ... */ }
func (d *DatabaseUserService) CreateUser(user User) error { /* ... */ }
func (d *DatabaseUserService) GetUser(id string) (*User, error) { /* ... */ }
```

**Benefits:**
- 🎯 **Precise Call Resolution**: Method calls resolve to actual implementations
- 🔗 **Complete Dependency Trees**: See full call chains through interface boundaries
- 📊 **Better Visualization**: Understand polymorphic relationships in your code

### 🔍 Advanced Type Resolution Engine
Intelligent type resolution handles complex Go patterns:

- **Struct Field Method Calls**: `svc.UserService.CreateUser()` → `DatabaseUserService.CreateUser`
- **Import Alias Resolution**: Resolves through import aliases and package names
- **External Type Mapping**: Maps external types to their actual implementations
- **Recursive Method Discovery**: Finds methods called within implementations

### 📦 External Module Intelligence
Comprehensive external dependency analysis:

```bash
# Scans all go.mod files recursively
# Filters by relevance (only modules actually called)
# Applies intelligent skip patterns
go run cmd/server/main.go --include-external=true --skip-folders="golang.org,google.golang.org"
```

**Features:**
- 🔄 **Recursive go.mod Discovery**: Finds all modules in monorepos
- 🎛️ **Smart Filtering**: Only scans modules actually used by your code
- ⚡ **Performance Optimized**: Parallel processing with timeout protection
- 🎯 **Relevance Scoring**: Prioritizes frequently-used external functions

### ⚡ Performance Optimizations
- **Parallel Processing**: Multi-core function analysis and relation building
- **In-Memory Caching**: Fast access to parsed relationships
- **Lazy Loading**: Load function details on-demand
- **Efficient Data Structures**: Optimized for large codebases
- **Memory Management**: Automatic garbage collection and memory monitoring

### 🎨 Advanced UI Components
- **Screenshot Slideshow**: Interactive feature showcase
- **Comparison Table**: Built-in comparison with other Go tools
- **Theme Context**: System preference detection with localStorage
- **Responsive Grid**: Adaptive layouts for all screen sizes
- **GitHub Integration**: Live star count and repository linking

---

## 🔍 API Reference

Complete REST API documentation for integration and automation:

### Core Endpoints

#### `GET /api/relations`
Retrieve paginated function relationships with full dependency closure.

**Parameters:**
- `page` (int): Page number (1-based, default: 1)
- `pageSize` (int): Items per page (max: 200, default: 10)
- `includeInternals` (bool): Include internal analyzer functions

**Response:**
```json
{
  "page": 1,
  "pageSize": 10,
  "totalRoots": 45,
  "roots": [/* root function objects */],
  "data": [/* complete dependency closure */],
  "loadedAt": "2024-01-15T10:30:00Z"
}
```

#### `GET /api/search`
Search functions by name with pagination.

**Parameters:**
- `q` (string): Search query (required)
- `page` (int): Page number (default: 1)
- `pageSize` (int): Results per page (default: 10)

**Response:**
```json
{
  "query": "CreateUser",
  "page": 1,
  "totalResults": 3,
  "matchingFunctions": [/* matching functions */],
  "data": [/* dependency closure for matches */]
}
```

#### `POST /api/reload`
Trigger repository rescan without server restart.

**Response:**
```json
{
  "status": "reloaded",
  "loadedAt": "2024-01-15T10:35:00Z"
}
```

#### `GET /api/download`
Download complete function relations as JSON.

**Headers:**
- `Content-Type: application/json`
- `Content-Disposition: attachment; filename=function_relations.json`

### Static Routes
- **`/`** - Overview page (Notion-style landing)
- **`/gomindmapper/`** - Base application route
- **`/gomindmapper/view`** - Mind map interface
- **`/gomindmapper/view/*`** - SPA routing fallbacks
- **`/docs/*`** - Static assets (CSS, JS, images)

### Authentication & CORS
- **CORS**: Enabled for all origins (`*`)
- **Authentication**: Currently none (designed for local/internal use)
- **Rate Limiting**: None (add reverse proxy for production)

---

## 📊 Data Models

Understand the internal data structures for integration and customization:

### Core Types

```go
// FunctionInfo - Raw function data from AST parsing
type FunctionInfo struct {
    Name     string   // Fully qualified name (package.function)
    Line     int      // Line number in source file
    FilePath string   // Relative file path
    Calls    []string // Function calls made within this function
}

// OutRelation - Processed relationship for JSON output
type OutRelation struct {
    Name     string      `json:"name"`
    Line     int         `json:"line"`
    FilePath string      `json:"filePath"`
    Called   []OutCalled `json:"called,omitempty"`
}

// OutCalled - Called function reference
type OutCalled struct {
    Name     string `json:"name"`
    Line     int    `json:"line"`
    FilePath string `json:"filePath"`
}
```

### Advanced Types (Type Resolution)

```go
// TypeInfo - Comprehensive type information
type TypeInfo struct {
    Name        string
    Package     string
    IsInterface bool
    IsStruct    bool
    Fields      map[string]string // field name → type
    Methods     []string
    ImportPath  string // for external types
}

// InterfaceImplementation - Concrete interface implementation
type InterfaceImplementation struct {
    InterfaceName string
    StructName    string
    PackageName   string
    FilePath      string
    Methods       map[string]MethodImplementation
}
```

### File Formats

#### `functionmap.json` Structure
```json
[
  {
    "name": "main.main",
    "line": 10,
    "filePath": "main.go",
    "called": [
      {
        "name": "config.LoadConfig",
        "line": 25,
        "filePath": "internal/config/config.go"
      },
      {
        "name": "server.StartServer",
        "line": 45,
        "filePath": "internal/server/server.go"
      }
    ]
  }
]
```

#### `functions.json` Structure
```json
[
  {
    "name": "main.main",
    "line": 10,
    "filePath": "main.go",
    "calls": ["config.LoadConfig", "server.StartServer", "log.Println"]
  }
]
```

---

## 🎨 Customization

### Filtering & Analysis Customization

Modify analysis behavior by editing key files:

#### `cmd/analyzer/utils.go` - Call Extraction Rules
```go
// Add custom exclusion patterns in FindCalls()
standardPackages := map[string]bool{
    "fmt":     true,
    "os":      true,
    // Add your exclusions here
    "mycorp.internal": true,
}

// Add custom regex exclusions
regexFunctions := map[string]bool{
    "FindAllSubmatch": true,
    // Add patterns to ignore
    "MyCustomPattern": true,
}
```

#### `cmd/analyzer/relations.go` - Relationship Building
```go
// Modify BuildRelations() to change output format
// Add custom matching logic for external functions
// Implement whitelist/blacklist patterns
```

### UI Theming & Styling

#### Theme Variables (`mind-map-react/src/App.css`)
```css
:root {
  /* Customize color scheme */
  --bg-primary: #0f0f0f;
  --text-primary: #ffffff;
  --accent-color: #3b82f6;
  
  /* Add custom variables */
  --node-primary: #1e40af;
  --node-secondary: #059669;
}
```

#### Component Customization
- **Node Styles**: Modify `mind-map-react/src/components/Node.jsx`
- **Theme Logic**: Edit `mind-map-react/src/contexts/ThemeContext.jsx`
- **Layout**: Update `mind-map-react/src/components/Overview.css`

### Adding New Features

#### 1. New API Endpoint
```go
// In cmd/server/main.go
router.GET("/api/metrics", handleMetrics)

func handleMetrics(c *gin.Context) {
    // Your custom endpoint logic
}
```

#### 2. New UI Component
```jsx
// In mind-map-react/src/components/
import React from 'react';

const MyComponent = () => {
    return (
        <div className="my-component">
            {/* Your component JSX */}
        </div>
    );
};

export default MyComponent;
```

### Configuration Files

Create configuration files for advanced customization:

#### `gomindmapper.json` (Future)
```json
{
  "analysis": {
    "excludePatterns": ["*_test.go", "vendor/*"],
    "includeExternalByDefault": false,
    "maxExternalDepth": 3
  },
  "server": {
    "defaultPort": 8080,
    "enableCORS": true,
    "maxPageSize": 200
  },
  "ui": {
    "defaultTheme": "dark",
    "enableAnimations": true,
    "nodeColors": {
      "main": "#3b82f6",
      "handler": "#059669"
    }
  }
}
```

---

## 🗺️ Roadmap

### ✅ Advanced Features Completed:
- [x] **Interface Implementation Detection** - Automatic discovery of concrete interface implementations
- [x] **Advanced Type Resolution Engine** - Complex type resolution with import alias handling
- [x] **External Module Intelligence** - Recursive go.mod scanning with relevance filtering
- [x] **Performance Optimization** - Parallel processing, in-memory caching, lazy loading
- [x] **Search API** (`/api/search`) with pagination and dependency closure
- [x] **Advanced Theming** - System preference detection with localStorage persistence
- [x] **Custom Node Design** - Google NotebookLLM-inspired UI components
- [x] **Screenshot Slideshow** - Interactive feature showcase with auto-navigation
- [x] **Comparison Table** - Built-in comparison with other Go visualization tools
- [x] **Hot Reload** (`POST /api/reload`) - Live repository rescanning
- [x] **Drag & Drop Upload** - Offline JSON analysis capability
- [x] **Multi-format Export** - JSON download with planned GraphML/DOT/SVG support

### � Next Major Features:
- [ ] **Real-time Code Analysis** - FS watcher for automatic updates as code changes
- [ ] **Function Metrics Dashboard** - Complexity, fan-in/fan-out, LOC, cyclomatic complexity
- [ ] **Call Path Analysis** - Trace execution paths between functions
- [ ] **Performance Profiling Integration** - Overlay runtime performance data
- [ ] **Collaborative Features** - Share and bookmark specific views
- [ ] **VS Code Extension** - Inline function relationship viewer

### 🔮 Advanced Roadmap:
- [ ] **AI-Powered Analysis** - Semantic understanding of function purposes
- [ ] **Architecture Pattern Detection** - Identify common patterns (MVC, hexagonal, etc.)
- [ ] **Microservices Visualization** - Cross-service dependency mapping
- [ ] **Security Analysis** - Data flow analysis for security vulnerabilities
- [ ] **Configuration Management** - Project-specific analysis profiles
- [ ] **Plugin System** - Custom analyzers and visualizers
- [ ] **Graph Database Integration** - Neo4j backend for complex queries
- [ ] **Export Formats** - Mermaid, PlantUML, GraphML, GEXF support

### 🐳 Infrastructure & Distribution:
- [ ] **Docker Images** - Multi-stage containerized builds
- [ ] **Kubernetes Helm Charts** - Enterprise deployment support  
- [ ] **Go Install Support** - Direct installation via `go install`
- [ ] **GitHub Actions Integration** - CI/CD pipeline integration
- [ ] **Documentation Site** - Comprehensive docs with interactive examples
- [ ] **Performance Benchmarks** - Automated performance regression testing

### 🎯 Community & Integration:
- [ ] **Language Server Protocol** - IDE integration support
- [ ] **GitHub App** - Repository analysis bot
- [ ] **Slack/Teams Integration** - Team collaboration features
- [ ] **API Client Libraries** - Go, Python, JavaScript clients
- [ ] **Template Gallery** - Pre-configured analysis templates
- [ ] **Community Plugins** - Marketplace for custom analyzers

## 11. Contributing
PRs + issues welcome. Please:
1. Run `go fmt ./...` & `go vet ./...`
2. Keep analyzer + server filtering logic in sync
3. For UI changes include screenshot or short GIF

## 12. License
MIT (add a `LICENSE` file if distributing publicly).

---
Happy mapping! Open an issue for feature ideas or refinement suggestions.

