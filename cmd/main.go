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
	var includeExternal bool
	flag.StringVar(&path, "path", ".", "path to repository")
	flag.BoolVar(&includeExternal, "include-external", false, "include external library calls in output (skip removed_calls.json generation)")
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

	// If include-external is true, scan external modules
	if includeExternal {
		fmt.Println("Scanning external modules...")
		externalFunctions, err := scanExternalModules(absPath, functions)
		if err != nil {
			fmt.Printf("Warning: failed to scan external modules: %v\n", err)
		} else {
			functions = append(functions, externalFunctions...)
			fmt.Printf("Successfully scanned external modules and found %d external functions\n", len(externalFunctions))
		}
	}

	sort.Slice(functions, func(i, j int) bool {
		return functions[i].Name < functions[j].Name
	})

	// Persist raw filtered functions (existing behaviour)
	analyzer.CreateJsonFile(functions, includeExternal)

	// Build relations and write functionmap.json (replaces buildFunctionMap)
	relations := analyzer.BuildRelations(functions, includeExternal)
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
			funcInfo := analyzer.FunctionInfo{
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
				funcInfo.Calls = resolvedCalls
			}
			funcs = append(funcs, funcInfo)
		}
	}
	return funcs, nil
}

// scanExternalModules scans external modules when include-external is enabled
func scanExternalModules(projectPath string, functions []analyzer.FunctionInfo) ([]analyzer.FunctionInfo, error) {
	// Get external modules from go.mod
	modules, err := analyzer.GetExternalModules(projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get external modules: %v", err)
	}

	fmt.Printf("Found %d modules in go.mod\n", len(modules))

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
	relevantModules := analyzer.FilterRelevantExternalModules(allFunctions, modules)

	var externalFunctions []analyzer.FunctionInfo

	for modulePath, moduleInfo := range relevantModules {
		fmt.Printf("Scanning module: %s@%s\n", modulePath, moduleInfo.Version)

		// Find module in GOPATH
		localPath, err := analyzer.FindModuleInGoPath(moduleInfo)
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
			continue
		}

		// Scan the module
		moduleFunctions, err := analyzer.ScanExternalModule(localPath, moduleInfo)
		if err != nil {
			fmt.Printf("Warning: failed to scan module %s: %v\n", modulePath, err)
			continue
		}

		fmt.Printf("Found %d functions in module %s\n", len(moduleFunctions), modulePath)
		externalFunctions = append(externalFunctions, moduleFunctions...)
	}

	return externalFunctions, nil
}

// findFunctionsWithAllCalls is similar to findFunctions but doesn't filter calls
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

// legacy buildFunctionMap removed: functionality now in analyzer.BuildRelations
