package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chinmay-sawant/gomindmapper/cmd/analyzer"
)

// cache holds the in-memory representation of relations + index for quick lookups.
type cache struct {
	mu        sync.RWMutex
	functions []analyzer.FunctionInfo         // raw filtered function infos (with Calls)
	relations []analyzer.OutRelation          // flattened relations list
	index     map[string]analyzer.OutRelation // composite key name|filePath
	roots     []analyzer.OutRelation          // relations whose function is not called by any other (entry points)
	loadedAt  time.Time
}

var global cache

func main() {
	var repoPath string
	var addr string
	flag.StringVar(&repoPath, "path", ".", "path to repository root")
	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.Parse()

	if err := load(repoPath); err != nil {
		log.Fatalf("initial load failed: %v", err)
	}

	http.HandleFunc("/api/relations", handleRelations)
	http.HandleFunc("/api/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := load(repoPath); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]any{"status": "reloaded", "loadedAt": global.loadedAt})
	})

	// Serve docs folder at root
	docsDir := filepath.Join(repoPath, "docs")
	http.Handle("/", http.FileServer(http.Dir(docsDir)))

	// React build served at /view/* (mind-map-react/build)
	reactBuild := filepath.Join(repoPath, "mind-map-react", "build")
	if st, err := os.Stat(reactBuild); err == nil && st.IsDir() {
		// Wrap to provide SPA fallback
		http.HandleFunc("/view/", func(w http.ResponseWriter, r *http.Request) {
			// Try to serve static asset
			// strip /view/
			rel := strings.TrimPrefix(r.URL.Path, "/view/")
			if rel == "" { // root of SPA
				http.ServeFile(w, r, filepath.Join(reactBuild, "index.html"))
				return
			}
			candidate := filepath.Join(reactBuild, rel)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				http.ServeFile(w, r, candidate)
				return
			}
			// fallback to index.html for client routing
			http.ServeFile(w, r, filepath.Join(reactBuild, "index.html"))
		})
		// Also serve static assets without /view prefix if CRA build placed hashed assets in root of build
		fileServer := http.FileServer(neuteredFileSystem{http.Dir(reactBuild)})
		http.Handle("/view/static/", http.StripPrefix("/view", fileServer))
	} else {
		log.Printf("react build not found at %s (run npm run build in mind-map-react)", reactBuild)
	}

	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, corsMiddleware(http.DefaultServeMux)))
}

// load (re)scans repository, rebuilds structures and populates cache.
func load(root string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	module, err := analyzer.GetModule(abs)
	if err != nil {
		return err
	}

	var functions []analyzer.FunctionInfo
	// Walk & extract
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fns, ferr := findFunctions(path, abs, module)
		if ferr != nil {
			return ferr
		}
		functions = append(functions, fns...)
		return nil
	})
	if err != nil {
		return err
	}

	// Filter calls like CLI does (reusing CreateJsonFile side effect free variant)
	// We temporarily copy functions then call CreateJsonFile to produce filtered Calls but ignore file writes.
	// Simpler: replicate minimal filtering here (to avoid writing files). We'll replicate logic from CreateJsonFile.
	filterCalls(functions)

	relations := analyzer.BuildRelations(functions)
	// stable sort by name then filePath
	sort.Slice(relations, func(i, j int) bool {
		if relations[i].Name == relations[j].Name {
			return relations[i].FilePath < relations[j].FilePath
		}
		return relations[i].Name < relations[j].Name
	})

	// composite key function
	ck := func(name, file string) string { return name + "|" + file }

	calledSet := make(map[string]bool)
	idx := make(map[string]analyzer.OutRelation, len(relations))
	internalPrefixes := []string{"analyzer.", "main.load", "main.findFunctions"}
	isInternal := func(name string) bool {
		for _, p := range internalPrefixes {
			if strings.HasPrefix(name, p) {
				return true
			}
		}
		return false
	}
	for _, r := range relations {
		for _, c := range r.Called {
			calledSet[ck(c.Name, c.FilePath)] = true
		}
		idx[ck(r.Name, r.FilePath)] = r
	}
	var roots []analyzer.OutRelation
	for _, r := range relations {
		if !calledSet[ck(r.Name, r.FilePath)] && !isInternal(r.Name) {
			roots = append(roots, r)
		}
	}
	sort.Slice(roots, func(i, j int) bool {
		if roots[i].Name == roots[j].Name {
			return roots[i].FilePath < roots[j].FilePath
		}
		return roots[i].Name < roots[j].Name
	})

	global.mu.Lock()
	global.functions = functions
	global.relations = relations
	global.index = idx
	global.roots = roots
	global.loadedAt = time.Now()
	global.mu.Unlock()
	return nil
}

// replicate minimal call filtering from CreateJsonFile without file writes
func filterCalls(functions []analyzer.FunctionInfo) {
	userPrefixes := make(map[string]bool)
	for _, f := range functions {
		if dot := strings.Index(f.Name, "."); dot != -1 {
			userPrefixes[f.Name[:dot]] = true
		}
	}
	for i := range functions {
		if len(functions[i].Calls) == 0 {
			continue
		}
		var filtered []string
		for _, c := range functions[i].Calls {
			if !strings.Contains(c, ".") {
				continue
			}
			parts := strings.Split(c, ".")
			if len(parts) > 0 && userPrefixes[parts[0]] {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			functions[i].Calls = nil
		} else {
			functions[i].Calls = filtered
		}
	}
}

// handleRelations returns paginated root relations with full dependency closure for each root on the page.
// Query params: page (1-based), pageSize
// Response: { page, pageSize, totalRoots, roots: [...root names...], data: [OutRelation ...] }
func handleRelations(w http.ResponseWriter, r *http.Request) {
	global.mu.RLock()
	defer global.mu.RUnlock()
	page := parseInt(r.URL.Query().Get("page"), 1)
	pageSize := parseInt(r.URL.Query().Get("pageSize"), 10)
	includeInternals := strings.EqualFold(r.URL.Query().Get("includeInternals"), "true")
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 10
	}

	totalRoots := len(global.roots)
	start := (page - 1) * pageSize
	if start > totalRoots {
		start = totalRoots
	}
	end := start + pageSize
	if end > totalRoots {
		end = totalRoots
	}
	selectedRoots := global.roots[start:end]

	// Collect dependency closure using composite keys
	closureMap := make(map[string]analyzer.OutRelation)
	ck := func(name, file string) string { return name + "|" + file }
	var collect func(name, file string)
	collect = func(name, file string) {
		k := ck(name, file)
		if _, exists := closureMap[k]; exists {
			return
		}
		rel, ok := global.index[k]
		if !ok {
			return
		}
		if !includeInternals && (strings.HasPrefix(name, "analyzer.") || name == "main.findFunctions" || name == "main.load") {
			for _, c := range rel.Called {
				collect(c.Name, c.FilePath)
			}
			return
		}
		closureMap[k] = rel
		for _, c := range rel.Called {
			collect(c.Name, c.FilePath)
		}
	}
	for _, r := range selectedRoots {
		collect(r.Name, r.FilePath)
	}

	var closure []analyzer.OutRelation
	for _, v := range closureMap {
		closure = append(closure, v)
	}
	sort.Slice(closure, func(i, j int) bool {
		if closure[i].Name == closure[j].Name {
			return closure[i].FilePath < closure[j].FilePath
		}
		return closure[i].Name < closure[j].Name
	})

	writeJSON(w, map[string]any{
		"page":             page,
		"pageSize":         pageSize,
		"totalRoots":       totalRoots,
		"roots":            selectedRoots,
		"data":             closure,
		"loadedAt":         global.loadedAt,
		"includeInternals": includeInternals,
	})
}

// Helpers --------------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return def
	}
	return v
}

// Simplified duplicate of CLI findFunctions (cannot import from main package) ----------------------------------------
func findFunctions(filePath, absPath, module string) ([]analyzer.FunctionInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	var funcs []analyzer.FunctionInfo
	re := regexp.MustCompile(`^\s*func\s+(\w+)`)
	relPath, err := filepath.Rel(absPath, filePath)
	if err != nil {
		return nil, err
	}
	var packageName string
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			packageName = strings.TrimSpace(strings.TrimPrefix(line, "package "))
			break
		}
	}
	for i, line := range lines {
		if matches := re.FindStringSubmatch(line); matches != nil {
			fi := analyzer.FunctionInfo{Name: packageName + "." + matches[1], Line: i + 1, FilePath: relPath}
			start, end := analyzer.FindFunctionBody(lines, i)
			if start != -1 && end != -1 && start+1 < end && end < len(lines) {
				fi.Calls = analyzer.FindCalls(lines[start+1 : end])
			}
			funcs = append(funcs, fi)
		}
	}
	return funcs, nil
}

// Basic CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

// neuteredFileSystem prevents directory listing (optional hardening for static assets)
type neuteredFileSystem struct{ fs http.FileSystem }

func (nfs neuteredFileSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}
	if stat, err := f.Stat(); err == nil && stat.IsDir() {
		// If directory, look for index.html else block listing
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			return nil, fs.ErrPermission
		}
	}
	return f, nil
}
