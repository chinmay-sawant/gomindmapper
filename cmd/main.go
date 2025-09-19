package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/chinmay-sawant/gomindmapper/cmd/analyzer"
)

func main() {
	var path string
	flag.StringVar(&path, "path", ".", "path to repository")
	flag.Parse()

	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	module, err := analyzer.GetModule(absPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	var functions []analyzer.FunctionInfo
	err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			funcs, err := findFunctions(path, absPath, module)
			if err != nil {
				return err
			}
			functions = append(functions, funcs...)
		}
		return nil
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Name < functions[j].Name
	})

	analyzer.CreateJsonFile(functions)
	buildFunctionMap(functions)
}

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

func buildFunctionMap(functions []analyzer.FunctionInfo) {
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

	// Prepare simplified output structure that omits empty "called" and excludes the "Calls" field
	type outCalled struct {
		Name     string `json:"name"`
		Line     int    `json:"line"`
		FilePath string `json:"filePath"`
	}
	type outRel struct {
		Name     string      `json:"name"`
		Line     int         `json:"line"`
		FilePath string      `json:"filePath"`
		Called   []outCalled `json:"called,omitempty"`
	}

	out := make([]outRel, 0, len(allRelations))
	for _, r := range allRelations {
		if len(r.Called) == 0 {
			// skip functions with no called entries
			continue
		}
		o := outRel{
			Name:     r.Name,
			Line:     r.Line,
			FilePath: r.FilePath,
		}
		for _, c := range r.Called {
			o.Called = append(o.Called, outCalled{Name: c.Name, Line: c.Line, FilePath: c.FilePath})
		}
		out = append(out, o)
	}

	// Write to JSON
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	err = os.WriteFile("functionmap.json", data, 0644)
	if err != nil {
		fmt.Println("Error writing file:", err)
		return
	}

	fmt.Println("functionmap.json created successfully")
}
