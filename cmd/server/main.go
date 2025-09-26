package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
		c.JSON(http.StatusOK, global.relations)
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

// load (re)scans repository, rebuilds structures and populates cache.
func load(root string, includeExternal bool, skipPatterns []string) error {
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

		extFuncs, err := scanExternalModules(abs, functions, skipPatterns)
		if err != nil {
			log.Printf("Warning: failed to scan external modules: %v", err)
		} else {
			externalFunctions = extFuncs
			log.Printf("Successfully scanned external modules and found %d external functions", len(externalFunctions))

			// Memory check after scanning
			runtime.ReadMemStats(&m)
			log.Printf("Memory after external scanning: %.2f MB", float64(m.Alloc)/1024/1024)

			// Limit external functions if memory usage is too high
			if len(externalFunctions) > 50000 {
				log.Printf("Warning: Large number of external functions (%d). Consider using more restrictive skip patterns for better performance.", len(externalFunctions))
				// Optionally limit to most relevant functions
				externalFunctions = limitExternalFunctions(externalFunctions, functions, 25000)
				log.Printf("Limited external functions to %d for performance", len(externalFunctions))
			}

			functions = append(functions, externalFunctions...)
		}
	}

	// Add interface implementation detection for better call resolution
	if !includeExternal {
		log.Println("Detecting interface implementations...")
		functions = enhanceProjectFunctionsWithInterfaceDetection(functions, abs)
	}

	// Optimize performance for large datasets with parallel processing
	log.Printf("Processing %d total functions (including %d external)...", len(functions), len(externalFunctions))

	// Filter calls with parallel processing for large datasets
	start := time.Now()
	filterCallsParallel(functions, includeExternal)
	log.Printf("Call filtering completed in %v", time.Since(start))

	// Build relations with parallel processing
	start = time.Now()
	relations := buildRelationsParallel(functions, includeExternal)
	log.Printf("Relation building completed in %v", time.Since(start))
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

	// Log statistics about the loaded data
	log.Printf("Data loaded successfully:")
	log.Printf("  - Total functions detected: %d", len(functions))
	log.Printf("  - Total relations built: %d", len(relations))
	log.Printf("  - Total root functions (entry points): %d", len(roots))
	log.Printf("  - Total functions in index: %d", len(idx))
	log.Printf("  - Data loaded at: %s", global.loadedAt.Format("2006-01-02 15:04:05"))

	return nil
}

// filterCallsParallel replicates minimal call filtering from CreateJsonFile with parallel processing
func filterCallsParallel(functions []analyzer.FunctionInfo, includeExternal bool) {
	if includeExternal {
		// If including external calls, don't filter - keep all calls as-is
		return
	}

	// Build user prefixes map once
	userPrefixes := make(map[string]bool)
	for _, f := range functions {
		if dot := strings.Index(f.Name, "."); dot != -1 {
			userPrefixes[f.Name[:dot]] = true
		}
	}

	// Determine optimal number of workers based on dataset size
	numWorkers := runtime.NumCPU()
	if len(functions) < 1000 {
		// For small datasets, use sequential processing to avoid overhead
		filterCallsSequential(functions, userPrefixes)
		return
	}

	// For large datasets, use parallel processing
	chunkSize := (len(functions) + numWorkers - 1) / numWorkers
	var wg sync.WaitGroup

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
		go func(start, end int) {
			defer wg.Done()
			filterCallsRange(functions[start:end], userPrefixes)
		}(start, end)
	}

	wg.Wait()
}

// filterCallsSequential processes calls sequentially for small datasets
func filterCallsSequential(functions []analyzer.FunctionInfo, userPrefixes map[string]bool) {
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

// filterCallsRange processes a range of functions for parallel execution
func filterCallsRange(functions []analyzer.FunctionInfo, userPrefixes map[string]bool) {
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

// enhanceProjectFunctionsWithInterfaceDetection enhances project functions with interface implementation detection
func enhanceProjectFunctionsWithInterfaceDetection(functions []analyzer.FunctionInfo, projectPath string) []analyzer.FunctionInfo {
	// Parse type information for the project
	typeInfo, err := analyzer.ParseTypeInformation(projectPath, make(map[string]analyzer.ExternalModuleInfo))
	if err != nil {
		log.Printf("Warning: failed to parse project type information: %v", err)
		return functions
	}

	// Parse comprehensive file information
	fileInfoMap := make(map[string]analyzer.FileTypeInfo)
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			relPath, _ := filepath.Rel(projectPath, path)
			fileInfo, err := analyzer.ParseGoFileForTypesAndImports(path, projectPath)
			if err != nil {
				return nil // Skip files that can't be parsed
			}
			fileInfoMap[relPath] = fileInfo
		}
		return nil
	})

	if err != nil {
		log.Printf("Warning: failed to parse comprehensive file information: %v", err)
		return functions
	}

	// Find interface implementations
	implementations, err := analyzer.FindInterfaceImplementations(projectPath)
	if err != nil {
		log.Printf("Warning: failed to find interface implementations: %v", err)
		implementations = make(map[string][]analyzer.InterfaceImplementation)
	}

	if len(implementations) > 0 {
		log.Printf("Found %d interface implementations", len(implementations))
	}

	// Process each function to resolve its method calls and add implementation calls
	var enhancedFunctions []analyzer.FunctionInfo
	for _, fn := range functions {
		enhancedCalls := make([]string, 0, len(fn.Calls))

		for _, call := range fn.Calls {
			// Try to resolve the call using comprehensive type information
			resolvedCall := analyzer.ResolveMethodCall(call, fileInfoMap, typeInfo, implementations)
			enhancedCalls = append(enhancedCalls, resolvedCall)

			// If this is an interface method call, add the implementation calls
			implementationFunctions := analyzer.GetImplementationCalls(call, implementations)
			for _, implFunc := range implementationFunctions {
				// Add the implementation function to our function list
				enhancedFunctions = append(enhancedFunctions, implFunc)

				// Also add a call relationship from the current function to the implementation
				enhancedCalls = append(enhancedCalls, implFunc.Name)
			}
		}

		fn.Calls = enhancedCalls
		enhancedFunctions = append(enhancedFunctions, fn)
	}

	return enhancedFunctions
}

// limitExternalFunctions reduces the number of external functions to improve performance
func limitExternalFunctions(externalFunctions []analyzer.FunctionInfo, localFunctions []analyzer.FunctionInfo, maxCount int) []analyzer.FunctionInfo {
	if len(externalFunctions) <= maxCount {
		return externalFunctions
	}

	// Create a map of packages that are directly called by local functions
	calledPackages := make(map[string]int)
	for _, fn := range localFunctions {
		for _, call := range fn.Calls {
			if strings.Contains(call, ".") {
				pkg := strings.Split(call, ".")[0]
				calledPackages[pkg]++
			}
		}
	}

	// Score external functions based on relevance
	type scoredFunction struct {
		function analyzer.FunctionInfo
		score    int
	}

	var scored []scoredFunction
	for _, fn := range externalFunctions {
		score := 0
		if strings.Contains(fn.Name, ".") {
			pkg := strings.Split(fn.Name, ".")[0]
			score = calledPackages[pkg]
		}
		// Prioritize exported functions (capitalized)
		if len(fn.Name) > 0 {
			parts := strings.Split(fn.Name, ".")
			if len(parts) > 1 && len(parts[1]) > 0 && parts[1][0] >= 'A' && parts[1][0] <= 'Z' {
				score += 10
			}
		}
		scored = append(scored, scoredFunction{function: fn, score: score})
	}

	// Sort by score descending
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	// Take top functions
	result := make([]analyzer.FunctionInfo, 0, maxCount)
	for i := 0; i < maxCount && i < len(scored); i++ {
		result = append(result, scored[i].function)
	}

	return result
}

// Legacy function for backward compatibility
func filterCalls(functions []analyzer.FunctionInfo, includeExternal bool) {
	filterCallsParallel(functions, includeExternal)
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
	// Updated regex to match both regular functions and methods
	// Regular function: func functionName(...)
	// Method: func (receiver Type) methodName(...)
	reFunc := regexp.MustCompile(`^\s*func\s+(\w+)`)
	reMethod := regexp.MustCompile(`^\s*func\s+\([^)]+\)\s+(\w+)`)
	relPath, err := filepath.Rel(absPath, filePath)
	if err != nil {
		return nil, err
	}

	// Find package name
	var packageName string
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			packageName = strings.TrimSpace(strings.TrimPrefix(line, "package "))
			break
		}
	}

	// Collect all function names in this file for reference resolution
	var localFunctions []string
	for _, line := range lines {
		// Check for regular functions
		if matches := reFunc.FindStringSubmatch(line); matches != nil {
			localFunctions = append(localFunctions, matches[1])
		} else if matches := reMethod.FindStringSubmatch(line); matches != nil {
			// Check for methods
			localFunctions = append(localFunctions, matches[1])
		}
	}

	for i, line := range lines {
		var functionName string
		// Check for regular functions first
		if matches := reFunc.FindStringSubmatch(line); matches != nil {
			functionName = matches[1]
		} else if matches := reMethod.FindStringSubmatch(line); matches != nil {
			// Check for methods
			functionName = matches[1]
		}

		if functionName != "" {
			fi := analyzer.FunctionInfo{
				Name:     packageName + "." + functionName,
				Line:     i + 1,
				FilePath: relPath,
			}
			// Find function body
			start, end := analyzer.FindFunctionBody(lines, i)
			if start != -1 && end != -1 && start+1 < end && end < len(lines) {
				calls := analyzer.FindCalls(lines[start+1 : end])

				// Resolve local function references by adding package prefix
				var resolvedCalls []string
				for _, call := range calls {
					if !strings.Contains(call, ".") {
						// Check if it's a local function reference
						for _, localFunc := range localFunctions {
							if call == localFunc {
								resolvedCalls = append(resolvedCalls, packageName+"."+call)
								break
							}
						}
					} else {
						resolvedCalls = append(resolvedCalls, call)
					}
				}
				fi.Calls = resolvedCalls
			}
			funcs = append(funcs, fi)
		}
	}
	return funcs, nil
}

// scanExternalModules scans external modules when include-external is enabled (optimized version)
// This function now recursively finds all go.mod files in the repository and scans their dependencies
func scanExternalModules(projectPath string, functions []analyzer.FunctionInfo, skipPatterns []string) ([]analyzer.FunctionInfo, error) {
	// Find all go.mod files recursively in the repository
	var goModPaths []string
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && info.Name() == "go.mod" {
			goModPaths = append(goModPaths, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find go.mod files: %v", err)
	}

	log.Printf("Found %d go.mod files in repository", len(goModPaths))

	// Collect all external modules from all go.mod files
	allModules := make(map[string]analyzer.ExternalModuleInfo)
	for _, modPath := range goModPaths {
		modules, err := analyzer.GetExternalModules(modPath)
		if err != nil {
			log.Printf("Warning: failed to get external modules from %s: %v", modPath, err)
			continue
		}
		log.Printf("Found %d modules in %s", len(modules), modPath)

		// Merge modules, avoiding duplicates
		for modName, modInfo := range modules {
			if _, exists := allModules[modName]; !exists {
				allModules[modName] = modInfo
			}
		}
	}

	log.Printf("Total unique external modules found: %d", len(allModules))

	// Filter out modules matching skip patterns
	if len(skipPatterns) > 0 {
		allModules = analyzer.FilterModulesBySkipPatterns(allModules, skipPatterns)
		log.Printf("After filtering skip patterns, scanning %d modules", len(allModules))
	}

	// Add performance optimization: limit the number of modules to scan
	if len(allModules) > 10 {
		log.Printf("Warning: Large number of modules (%d). Consider more restrictive skip patterns for better performance.", len(allModules))
	}

	// We need to collect external calls from the raw function data before filtering
	// Let's re-scan the project to get unfiltered calls
	var allFunctions []analyzer.FunctionInfo
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			funcs, err := findFunctionsWithAllCalls(path, projectPath)
			if err != nil {
				return err
			}
			allFunctions = append(allFunctions, funcs...)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to re-scan project for external calls: %v", err)
	}

	// Filter to only relevant modules (ones that are actually called)
	relevantModules := analyzer.FilterRelevantExternalModules(allFunctions, allModules, skipPatterns)

	var externalFunctions []analyzer.FunctionInfo
	totalModules := len(relevantModules)
	processedModules := 0

	// Process modules with progress reporting
	for modulePath, moduleInfo := range relevantModules {
		processedModules++
		log.Printf("Scanning module %d/%d: %s@%s", processedModules, totalModules, modulePath, moduleInfo.Version)

		// Find module in GOPATH
		localPath, err := analyzer.FindModuleInGoPath(moduleInfo)
		if err != nil {
			log.Printf("Warning: %v", err)
			continue
		}

		// Scan the module with timeout protection
		done := make(chan bool, 1)
		var moduleFunctions []analyzer.FunctionInfo
		var scanErr error

		go func() {
			moduleFunctions, scanErr = analyzer.ScanExternalModule(localPath, moduleInfo)
			done <- true
		}()

		// Wait for completion with timeout
		select {
		case <-done:
			if scanErr != nil {
				log.Printf("Warning: failed to scan module %s: %v", modulePath, scanErr)
				continue
			}
			log.Printf("Found %d functions in module %s", len(moduleFunctions), modulePath)
			externalFunctions = append(externalFunctions, moduleFunctions...)
		case <-time.After(30 * time.Second):
			log.Printf("Warning: timeout scanning module %s, skipping", modulePath)
			continue
		}

		// Memory check for large accumulations
		if len(externalFunctions) > 0 && len(externalFunctions)%10000 == 0 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("Progress: %d external functions collected, Memory: %.2f MB", len(externalFunctions), float64(m.Alloc)/1024/1024)
		}
	}

	return externalFunctions, nil
}

// findFunctionsWithAllCalls is similar to findFunctions but doesn't filter calls (duplicated from CLI)
func findFunctionsWithAllCalls(filePath, absPath string) ([]analyzer.FunctionInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var funcs []analyzer.FunctionInfo
	reFunc := regexp.MustCompile(`^\s*func\s+(\w+)`)
	reMethod := regexp.MustCompile(`^\s*func\s+\([^)]+\)\s+(\w+)`)
	relPath, err := filepath.Rel(absPath, filePath)
	if err != nil {
		return nil, err
	}

	// Find package name
	var packageName string
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			packageName = strings.TrimSpace(strings.TrimPrefix(line, "package "))
			break
		}
	}

	for i, line := range lines {
		var functionName string
		if matches := reFunc.FindStringSubmatch(line); matches != nil {
			functionName = matches[1]
		} else if matches := reMethod.FindStringSubmatch(line); matches != nil {
			functionName = matches[1]
		}

		if functionName != "" {
			funcInfo := analyzer.FunctionInfo{
				Name:     packageName + "." + functionName,
				Line:     i + 1,
				FilePath: relPath,
			}
			// Find function body - get ALL calls without filtering
			start, end := analyzer.FindFunctionBody(lines, i)
			if start != -1 && end != -1 && start+1 < end && end < len(lines) {
				calls := analyzer.FindCalls(lines[start+1 : end])
				funcInfo.Calls = calls // Keep all calls
			}
			funcs = append(funcs, funcInfo)
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
