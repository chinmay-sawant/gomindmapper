package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/chinmay-sawant/gomindmapper/cmd/analyzer"
	"github.com/chinmay-sawant/gomindmapper/utils"
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
	var includeExternal bool
	var skipFolders string
	flag.StringVar(&repoPath, "path", ".", "path to repository root")
	flag.StringVar(&addr, "addr", ":8080", "listen address")
	flag.BoolVar(&includeExternal, "include-external", false, "include external library calls in relations (store all calls in memory)")
	flag.StringVar(&skipFolders, "skip-folders", "", "comma-separated list of folder patterns to skip when scanning external dependencies (e.g., 'golang.org,google.golang.org')")
	flag.Parse()

	// Parse skip patterns
	var skipPatterns []string
	if skipFolders != "" {
		skipPatterns = strings.Split(skipFolders, ",")
		for i, pattern := range skipPatterns {
			skipPatterns[i] = strings.TrimSpace(pattern)
		}
		log.Printf("Skipping external dependency folders matching: %v", skipPatterns)
	}

	if err := load(repoPath, includeExternal, skipPatterns); err != nil {
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
		log.Printf("Reloading data from repository: %s", repoPath)
		if err := load(repoPath, includeExternal, skipPatterns); err != nil {
			log.Printf("Reload failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("Data reload completed successfully")
		c.JSON(http.StatusOK, gin.H{"status": "reloaded", "loadedAt": global.loadedAt})
	})

	router.GET("/api/download", func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Header("Content-Disposition", "attachment; filename=function_relations.json")
		// Pretty-print the in-memory relations for download
		data, err := json.MarshalIndent(global.relations, "", "  ")
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("failed to format JSON: %v", err))
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	})

	// Get the absolute path to the executable to locate static files
	execDir, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable path: %v", err)
	}
	execDir = filepath.Dir(execDir)

	// Try to find the docs directory - look in multiple possible locations
	var docsDir string
	possibleDocsPaths := []string{
		filepath.Join(execDir, "docs"),             // next to executable
		filepath.Join(execDir, "..", "..", "docs"), // from cmd/server/ to root
		"docs", // relative to current working directory
	}

	for _, path := range possibleDocsPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(filepath.Join(absPath, "index.html")); err == nil {
				docsDir = absPath
				break
			}
		}
	}

	if docsDir == "" {
		log.Fatalf("could not find docs directory with index.html in any of the expected locations")
	}

	log.Printf("Serving frontend from: %s", docsDir)

	// Serve docs folder at /docs
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

// loadExistingFunctionMap tries to load existing functionmap.json file for consistency
func loadExistingFunctionMap(functionMapPath string) ([]analyzer.OutRelation, error) {
	data, err := os.ReadFile(functionMapPath)
	if err != nil {
		return nil, err
	}

	var relations []analyzer.OutRelation
	if err := json.Unmarshal(data, &relations); err != nil {
		return nil, err
	}

	return relations, nil
}

// load (re)scans repository, rebuilds structures and populates cache.
func load(root string, includeExternal bool, skipPatterns []string) error {
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}

	log.Printf("Scanning repository: %s", abs)

	// First, try to load existing functionmap.json if it exists
	var relations []analyzer.OutRelation
	var functions []analyzer.FunctionInfo
	functionMapPath := filepath.Join(abs, "functionmap.json")

	// Always try to load functionmap.json if it exists, regardless of includeExternal flag
	if stat, err := os.Stat(functionMapPath); err == nil && !stat.IsDir() {
		log.Printf("Found existing functionmap.json, attempting to load...")
		if loadedRelations, err := loadExistingFunctionMap(functionMapPath); err == nil {
			log.Printf("Successfully loaded %d relations from functionmap.json", len(loadedRelations))
			relations = loadedRelations
		} else {
			log.Printf("Failed to load functionmap.json: %v, falling back to scanning", err)
		}
	}

	// If we couldn't load the existing file, scan and generate relations
	if len(relations) == 0 {
		module, err := analyzer.GetModule(abs)
		if err != nil {
			return err
		}

		log.Println("Scanning Go files for functions...")
		// Walk & extract
		err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fns, ferr := analyzer.FindFunctions(path, abs, module)
			if ferr != nil {
				return ferr
			}
			functions = append(functions, fns...)
			return nil
		})
		if err != nil {
			return err
		}

		log.Printf("Found %d functions in local repository", len(functions))

		// If include-external is true, scan external modules (same as CLI)
		var externalFunctions []analyzer.FunctionInfo
		if includeExternal {
			log.Println("Scanning external modules...")
			if len(skipPatterns) > 0 {
				log.Printf("Skipping external dependency folders matching: %v", skipPatterns)
			}

			// Add memory monitoring for large datasets
			var m runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m)
			log.Printf("Memory before external scanning: %.2f MB", float64(m.Alloc)/1024/1024)

			extFuncs, err := analyzer.ScanExternalModules(abs, functions, skipPatterns)
			if err != nil {
				log.Printf("Warning: failed to scan external modules: %v", err)
			} else {
				externalFunctions = extFuncs
				log.Printf("Successfully scanned external modules and found %d external functions", len(externalFunctions))

				// Memory check after scanning
				runtime.ReadMemStats(&m)
				log.Printf("Memory after external scanning: %.2f MB", float64(m.Alloc)/1024/1024)

				functions = append(functions, externalFunctions...)
			}
		}

		// Add interface implementation detection for better call resolution
		if !includeExternal {
			log.Println("Detecting interface implementations...")
			functions = analyzer.EnhanceProjectFunctionsWithTypeInfo(functions, abs)
		}

		// Optimize performance for large datasets with parallel processing
		log.Printf("Processing %d total functions (including %d external)...", len(functions), len(externalFunctions))

		// Build relations (parallelized for large datasets)
		start := time.Now()
		relations = buildRelationsParallel(functions, includeExternal)
		log.Printf("Relation building completed in %v", time.Since(start))
	}

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

	// Use the functions array whether loaded from file or generated

	global.mu.Lock()
	global.functions = functions
	global.relations = relations
	global.index = idx
	global.roots = roots
	global.loadedAt = time.Now()
	global.mu.Unlock()

	// Log statistics about the loaded data
	log.Printf("Data loaded successfully:")
	log.Printf("  - Total functions detected: %d", len(functions))
	log.Printf("  - Total relations built: %d", len(relations))
	log.Printf("  - Total root functions (entry points): %d", len(roots))
	log.Printf("  - Total functions in index: %d", len(idx))
	log.Printf("  - Data loaded at: %s", global.loadedAt.Format("2006-01-02 15:04:05"))

	return nil
}

// buildRelationsParallel builds relations with parallel processing for large datasets
func buildRelationsParallel(functions []analyzer.FunctionInfo, includeExternal bool) []analyzer.OutRelation {
	// For small datasets, use the original sequential method
	if len(functions) < 5000 {
		return analyzer.BuildRelations(functions, includeExternal)
	}

	// For large datasets, use parallel processing
	numWorkers := runtime.NumCPU()
	chunkSize := (len(functions) + numWorkers - 1) / numWorkers

	// Create channels for results
	type relationChunk struct {
		relations []analyzer.OutRelation
		index     int
	}
	resultChan := make(chan relationChunk, numWorkers)
	var wg sync.WaitGroup

	// Launch workers
	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(functions) {
			end = len(functions)
		}
		if start >= len(functions) {
			break
		}

		wg.Add(1)
		go func(start, end, workerIndex int) {
			defer wg.Done()
			chunkFunctions := functions[start:end]
			chunkRelations := analyzer.BuildRelations(chunkFunctions, includeExternal)
			resultChan <- relationChunk{relations: chunkRelations, index: workerIndex}
		}(start, end, i)
	}

	// Close channel when all workers complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allRelations []analyzer.OutRelation
	for chunk := range resultChan {
		allRelations = append(allRelations, chunk.relations...)
	}

	return allRelations
}

// Local interface-detection helper removed; server uses analyzer.EnhanceProjectFunctionsWithTypeInfo.

// limitExternalFunctions reduces the number of external functions to improve performance
// Unused external function limiter removed.

// Legacy call filtering removed; server uses same behavior as CLI.

// handleRelations returns paginated root relations with full dependency closure for each root on the page.
// Query params: page (1-based), pageSize
// Response: { page, pageSize, totalRoots, roots: [...root names...], data: [OutRelation ...] }
func handleRelations(c *gin.Context) {
	global.mu.RLock()
	defer global.mu.RUnlock()
	page := utils.ParseInt(c.Query("page"), 1)
	pageSize := utils.ParseInt(c.Query("pageSize"), 10)
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
	page := utils.ParseInt(c.Query("page"), 1)
	pageSize := utils.ParseInt(c.Query("pageSize"), 10)
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

// Simplified duplicate of CLI findFunctions (cannot import from main package) ----------------------------------------
// Duplicated helper functions removed in favor of shared analyzer helpers.

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
