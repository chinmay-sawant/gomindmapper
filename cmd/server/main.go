package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/chinmay-sawant/gomindmapper/cmd/analyzer"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Server struct {
	functions   []analyzer.FunctionRelation
	functionMap map[string]*analyzer.FunctionRelation
	searchIndex map[string][]*analyzer.FunctionRelation
	mutex       sync.RWMutex
}

type PaginatedResponse struct {
	Data       []analyzer.FunctionRelation `json:"data"`
	Page       int                         `json:"page"`
	PageSize   int                         `json:"pageSize"`
	Total      int                         `json:"total"`
	TotalPages int                         `json:"totalPages"`
}

type SearchResponse struct {
	Functions []analyzer.FunctionRelation `json:"functions"`
	Total     int                         `json:"total"`
}

func NewServer() *Server {
	return &Server{
		functionMap: make(map[string]*analyzer.FunctionRelation),
		searchIndex: make(map[string][]*analyzer.FunctionRelation),
	}
}

func (s *Server) LoadFunctionData(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	module, err := analyzer.GetModule(absPath)
	if err != nil {
		return err
	}

	var functions []analyzer.FunctionInfo
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			funcs, err := s.findFunctions(path, absPath, module)
			if err != nil {
				return err
			}
			functions = append(functions, funcs...)
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Name < functions[j].Name
	})

	s.buildFunctionRelations(functions)
	s.buildSearchIndex()
	return nil
}

func (s *Server) findFunctions(filePath, absPath, module string) ([]analyzer.FunctionInfo, error) {
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

	// Find package name
	var packageName string
	for _, line := range lines {
		if strings.HasPrefix(line, "package ") {
			packageName = strings.TrimSpace(strings.TrimPrefix(line, "package "))
			break
		}
	}

	for i, line := range lines {
		if matches := re.FindStringSubmatch(line); matches != nil {
			funcInfo := analyzer.FunctionInfo{
				Name:     packageName + "." + matches[1],
				Line:     i + 1,
				FilePath: relPath,
			}
			// Find function body
			start, end := analyzer.FindFunctionBody(lines, i)
			if start != -1 && end != -1 && start+1 < end && end < len(lines) {
				calls := analyzer.FindCalls(lines[start+1 : end])
				funcInfo.Calls = calls
			}
			funcs = append(funcs, funcInfo)
		}
	}
	return funcs, nil
}

func (s *Server) buildFunctionRelations(functions []analyzer.FunctionInfo) {
	// Create a map for quick lookup
	funcMap := make(map[string]analyzer.FunctionInfo)
	for _, fn := range functions {
		funcMap[fn.Name] = fn
	}

	// Collect user-defined package prefixes
	userPrefixes := make(map[string]bool)
	for _, fn := range functions {
		if dotIndex := strings.Index(fn.Name, "."); dotIndex != -1 {
			userPrefixes[fn.Name[:dotIndex]] = true
		}
	}

	// Number of CPUs
	numCPU := runtime.NumCPU()
	chunkSize := (len(functions) + numCPU - 1) / numCPU // Ceiling division

	// Channel to collect results
	results := make(chan []analyzer.FunctionRelation, numCPU)
	var wg sync.WaitGroup

	// Process in parallel
	for i := 0; i < numCPU; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(functions) {
			end = len(functions)
		}
		if start >= end {
			break
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			var relations []analyzer.FunctionRelation
			for j := start; j < end; j++ {
				fn := functions[j]
				var called []analyzer.FunctionInfo
				for _, call := range fn.Calls {
					if calledFn, exists := funcMap[call]; exists {
						// Check if it's a user-defined call
						if dotIndex := strings.Index(call, "."); dotIndex != -1 {
							if userPrefixes[call[:dotIndex]] {
								called = append(called, calledFn)
							}
						}
					}
				}
				relation := analyzer.FunctionRelation{
					Name:     fn.Name,
					Line:     fn.Line,
					FilePath: fn.FilePath,
					Called:   called,
				}
				relations = append(relations, relation)
			}
			results <- relations
		}(start, end)
	}

	// Close channel after all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all relations
	var allRelations []analyzer.FunctionRelation
	for relations := range results {
		allRelations = append(allRelations, relations...)
	}

	// Sort by name
	sort.Slice(allRelations, func(i, j int) bool {
		return allRelations[i].Name < allRelations[j].Name
	})

	// Filter out functions with no called entries and cache in memory
	s.mutex.Lock()
	s.functions = make([]analyzer.FunctionRelation, 0, len(allRelations))
	s.functionMap = make(map[string]*analyzer.FunctionRelation)

	for i := range allRelations {
		if len(allRelations[i].Called) > 0 {
			s.functions = append(s.functions, allRelations[i])
			s.functionMap[allRelations[i].Name] = &s.functions[len(s.functions)-1]
		}
	}
	s.mutex.Unlock()
}

func (s *Server) buildSearchIndex() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.searchIndex = make(map[string][]*analyzer.FunctionRelation)

	for i := range s.functions {
		funcRef := &s.functions[i]
		name := strings.ToLower(funcRef.Name)

		// Index full name
		s.searchIndex[name] = append(s.searchIndex[name], funcRef)

		// Index individual words in the function name
		words := strings.FieldsFunc(name, func(c rune) bool {
			return c == '.' || c == '_' || c == '-'
		})

		for _, word := range words {
			if len(word) > 0 {
				s.searchIndex[word] = append(s.searchIndex[word], funcRef)
			}
		}
	}
}

func (s *Server) handleGetFunctions(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Parse pagination parameters
	page := 1
	pageSize := 50

	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := r.URL.Query().Get("pageSize"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 1000 {
			pageSize = parsed
		}
	}

	total := len(s.functions)
	totalPages := (total + pageSize - 1) / pageSize

	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		start = total
		end = total
	} else if end > total {
		end = total
	}

	response := PaginatedResponse{
		Data:       s.functions[start:end],
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleSearchFunctions(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	query := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("q")))
	if query == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	// Find matching functions
	matchedFunctions := make(map[*analyzer.FunctionRelation]bool)

	// Exact name match
	if functions, exists := s.searchIndex[query]; exists {
		for _, fn := range functions {
			matchedFunctions[fn] = true
		}
	}

	// Partial matches
	for indexKey, functions := range s.searchIndex {
		if strings.Contains(indexKey, query) {
			for _, fn := range functions {
				matchedFunctions[fn] = true
			}
		}
	}

	// Convert to slice
	var results []analyzer.FunctionRelation
	for fn := range matchedFunctions {
		results = append(results, *fn)
	}

	// Sort results
	sort.Slice(results, func(i, j int) bool {
		return results[i].Name < results[j].Name
	})

	response := SearchResponse{
		Functions: results,
		Total:     len(results),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleGetFunction(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	vars := mux.Vars(r)
	functionName := vars["name"]

	if functionName == "" {
		http.Error(w, "Function name is required", http.StatusBadRequest)
		return
	}

	function, exists := s.functionMap[functionName]
	if !exists {
		http.Error(w, "Function not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(function)
}

// Add endpoint to get function with its dependencies (for expand functionality)
func (s *Server) handleGetFunctionWithDependencies(w http.ResponseWriter, r *http.Request) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	vars := mux.Vars(r)
	functionName := vars["name"]

	if functionName == "" {
		http.Error(w, "Function name is required", http.StatusBadRequest)
		return
	}

	rootFunction, exists := s.functionMap[functionName]
	if !exists {
		http.Error(w, "Function not found", http.StatusNotFound)
		return
	}

	// Collect the function and all its dependencies
	visited := make(map[string]bool)
	var result []analyzer.FunctionRelation

	var collectDependencies func(*analyzer.FunctionRelation)
	collectDependencies = func(fn *analyzer.FunctionRelation) {
		if visited[fn.Name] {
			return
		}
		visited[fn.Name] = true
		result = append(result, *fn)

		// Collect called functions
		for _, called := range fn.Called {
			if depFunc, exists := s.functionMap[called.Name]; exists {
				collectDependencies(depFunc)
			}
		}
	}

	collectDependencies(rootFunction)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
	var path string
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else {
		path = "."
	}

	server := NewServer()

	fmt.Println("Loading function data...")
	if err := server.LoadFunctionData(path); err != nil {
		log.Fatal("Failed to load function data:", err)
	}
	fmt.Printf("Loaded %d functions into memory\n", len(server.functions))

	router := mux.NewRouter()

	// API endpoints
	api := router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/functions", server.handleGetFunctions).Methods("GET")
	api.HandleFunc("/functions/search", server.handleSearchFunctions).Methods("GET")
	api.HandleFunc("/functions/{name:.+}/dependencies", server.handleGetFunctionWithDependencies).Methods("GET")
	api.HandleFunc("/functions/{name:.+}", server.handleGetFunction).Methods("GET")

	// Setup CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(router)

	fmt.Println("Server starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
