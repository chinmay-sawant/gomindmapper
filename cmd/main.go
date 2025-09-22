package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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

	// Persist raw filtered functions (existing behaviour)
	analyzer.CreateJsonFile(functions)

	// Build relations and write functionmap.json (replaces buildFunctionMap)
	relations := analyzer.BuildRelations(functions)
	data, err := json.MarshalIndent(relations, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling relations:", err)
		return
	}
	if err := os.WriteFile("functionmap.json", data, 0644); err != nil {
		fmt.Println("Error writing functionmap.json:", err)
		return
	}
	fmt.Println("functionmap.json created successfully")
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

	// Collect all function names in this file for reference resolution
	var localFunctions []string
	for _, line := range lines {
		if matches := re.FindStringSubmatch(line); matches != nil {
			localFunctions = append(localFunctions, matches[1])
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
				funcInfo.Calls = resolvedCalls
			}
			funcs = append(funcs, funcInfo)
		}
	}
	return funcs, nil
}

// legacy buildFunctionMap removed: functionality now in analyzer.BuildRelations
