package main

import (
	"flag"
	"fmt"
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
	"github.com/gin-gonic/gin"
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

	// Create Gin router
	router := gin.Default()

	// Add CORS middleware
	router.Use(corsMiddleware())

	// API routes
	router.GET("/api/relations", handleRelations)
	router.GET("/api/search", handleSearch)
	router.POST("/api/reload", func(c *gin.Context) {
		if err := load(repoPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "reloaded", "loadedAt": global.loadedAt})
	})

	router.GET("/api/download", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=function_relations.json")
		c.JSON(http.StatusOK, global.relations)
	})

	// Serve docs folder at /docs
	docsDir := filepath.Join(repoPath, "docs")
	router.Static("/docs", docsDir)

	// Serve docs files at root
	router.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(docsDir, "index.html"))
	})

	// Serve static assets
	router.Static("/assets", filepath.Join(docsDir, "assets"))

	// Serve assets at /gomindmapper/assets/ path for HTML compatibility
	router.Static("/gomindmapper/assets", filepath.Join(docsDir, "assets"))

	// Add routes for /gomindmapper/ path - serve docs content
	router.GET("/gomindmapper", func(c *gin.Context) {
		c.File(filepath.Join(docsDir, "index.html"))
	})
	router.GET("/gomindmapper/", func(c *gin.Context) {
		c.File(filepath.Join(docsDir, "index.html"))
	})
	router.GET("/gomindmapper/view", func(c *gin.Context) {
		c.File(filepath.Join(docsDir, "index.html"))
	})
	router.GET("/gomindmapper/view/*path", func(c *gin.Context) {
		c.File(filepath.Join(docsDir, "index.html"))
	})

	log.Printf("server listening on %s", addr)
	log.Fatal(router.Run(addr))
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
func handleRelations(c *gin.Context) {
	global.mu.RLock()
	defer global.mu.RUnlock()
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("pageSize"), 10)
	includeInternals := strings.EqualFold(c.Query("includeInternals"), "true")
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

	c.JSON(http.StatusOK, gin.H{
		"page":             page,
		"pageSize":         pageSize,
		"totalRoots":       totalRoots,
		"roots":            selectedRoots,
		"data":             closure,
		"loadedAt":         global.loadedAt,
		"includeInternals": includeInternals,
	})
}

// handleSearch searches for functions by name and returns their dependency closure with pagination
// Query params: q (search query), page (1-based), pageSize
// Response: { query, page, pageSize, totalResults, matchingFunctions: [...], data: [OutRelation ...] }
func handleSearch(c *gin.Context) {
	global.mu.RLock()
	defer global.mu.RUnlock()

	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "search query 'q' is required"})
		return
	}

	// Parse pagination parameters
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("pageSize"), 10)
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 10
	}

	// Convert query to lowercase for case-insensitive search
	lowerQuery := strings.ToLower(query)

	// First, try to find functions by name only (prioritized search)
	var matchingFunctions []analyzer.OutRelation
	for _, rel := range global.relations {
		lowerName := strings.ToLower(rel.Name)
		if strings.Contains(lowerName, lowerQuery) {
			matchingFunctions = append(matchingFunctions, rel)
		}
	}

	// If no results found in function names, search in both function names and file paths
	if len(matchingFunctions) == 0 {
		for _, rel := range global.relations {
			lowerName := strings.ToLower(rel.Name)
			lowerPath := strings.ToLower(rel.FilePath)
			if strings.Contains(lowerName, lowerQuery) || strings.Contains(lowerPath, lowerQuery) {
				matchingFunctions = append(matchingFunctions, rel)
			}
		}
	}

	// Sort matching functions for consistent pagination
	sort.Slice(matchingFunctions, func(i, j int) bool {
		if matchingFunctions[i].Name == matchingFunctions[j].Name {
			return matchingFunctions[i].FilePath < matchingFunctions[j].FilePath
		}
		return matchingFunctions[i].Name < matchingFunctions[j].Name
	})

	// Apply pagination to matching functions
	totalResults := len(matchingFunctions)
	start := (page - 1) * pageSize
	if start > totalResults {
		start = totalResults
	}
	end := start + pageSize
	if end > totalResults {
		end = totalResults
	}
	paginatedMatches := matchingFunctions[start:end]

	// Build dependency closure for paginated matching functions
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
		// Exclude internal functions from search results
		if strings.HasPrefix(name, "analyzer.") || name == "main.findFunctions" || name == "main.load" {
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

	// Collect closure for each paginated matching function
	for _, match := range paginatedMatches {
		collect(match.Name, match.FilePath)
	}

	// Convert to slice and sort
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

	c.JSON(http.StatusOK, gin.H{
		"query":             query,
		"page":              page,
		"pageSize":          pageSize,
		"totalResults":      totalResults,
		"matchingFunctions": paginatedMatches,
		"data":              closure,
		"loadedAt":          global.loadedAt,
	})
}

// Helpers --------------------------------------------------------------------------------

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

// Basic CORS middleware for Gin
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	}
}
