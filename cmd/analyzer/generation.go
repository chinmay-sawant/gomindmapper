package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FindFunctions scans a Go source file and returns functions/methods with resolved local calls
func FindFunctions(filePath, absPath, module string) ([]FunctionInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var funcs []FunctionInfo
	// Match regular functions and methods
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
		if matches := reFunc.FindStringSubmatch(line); matches != nil {
			localFunctions = append(localFunctions, matches[1])
		} else if matches := reMethod.FindStringSubmatch(line); matches != nil {
			localFunctions = append(localFunctions, matches[1])
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
			fi := FunctionInfo{
				Name:     packageName + "." + functionName,
				Line:     i + 1,
				FilePath: relPath,
			}
			// Find function body
			start, end := FindFunctionBody(lines, i)
			if start != -1 && end != -1 && start+1 < end && end < len(lines) {
				calls := FindCalls(lines[start+1 : end])

				// Resolve local function references by adding package prefix
				var resolvedCalls []string
				for _, call := range calls {
					if !strings.Contains(call, ".") {
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

// FindFunctionsWithAllCalls is similar to FindFunctions but keeps all calls without filtering
func FindFunctionsWithAllCalls(filePath, absPath string) ([]FunctionInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var funcs []FunctionInfo
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
			funcInfo := FunctionInfo{
				Name:     packageName + "." + functionName,
				Line:     i + 1,
				FilePath: relPath,
			}
			start, end := FindFunctionBody(lines, i)
			if start != -1 && end != -1 && start+1 < end && end < len(lines) {
				calls := FindCalls(lines[start+1 : end])
				funcInfo.Calls = calls
			}
			funcs = append(funcs, funcInfo)
		}
	}
	return funcs, nil
}

// ScanExternalModules scans external modules when include-external is enabled
// This function recursively finds all go.mod files in the repository and scans their dependencies
func ScanExternalModules(projectPath string, functions []FunctionInfo, skipPatterns []string) ([]FunctionInfo, error) {
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

	fmt.Printf("Found %d go.mod files in repository\n", len(goModPaths))

	// Collect all external modules from all go.mod files
	allModules := make(map[string]ExternalModuleInfo)
	for _, modPath := range goModPaths {
		modules, err := GetExternalModules(modPath)
		if err != nil {
			fmt.Printf("Warning: failed to get external modules from %s: %v\n", modPath, err)
			continue
		}
		fmt.Printf("Found %d modules in %s\n", len(modules), modPath)

		for modName, modInfo := range modules {
			if _, exists := allModules[modName]; !exists {
				allModules[modName] = modInfo
			}
		}
	}

	fmt.Printf("Total unique external modules found: %d\n", len(allModules))

	// Filter out modules matching skip patterns
	if len(skipPatterns) > 0 {
		allModules = FilterModulesBySkipPatterns(allModules, skipPatterns)
		fmt.Printf("After filtering skip patterns, scanning %d modules\n", len(allModules))
	}

	// Parse type information for better call resolution
	fmt.Println("Analyzing type information...")
	typeInfo, err := ParseTypeInformation(projectPath, allModules)
	if err != nil {
		fmt.Printf("Warning: failed to parse type information: %v\n", err)
		typeInfo = make(map[string]TypeInfo)
	}

	// Collect external calls from the raw function data before filtering
	var allFunctions []FunctionInfo
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			funcs, err := FindFunctionsWithAllCalls(path, projectPath)
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
	relevantModules := FilterRelevantExternalModules(allFunctions, allModules, skipPatterns)

	var externalFunctions []FunctionInfo
	scannedModules := make(map[string]bool)

	for modulePath, moduleInfo := range relevantModules {
		fmt.Printf("Scanning module: %s@%s\n", modulePath, moduleInfo.Version)

		localPath, err := FindModuleInGoPath(moduleInfo)
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
			continue
		}

		moduleFunctions, err := ScanExternalModuleRecursively(localPath, moduleInfo, scannedModules, allModules)
		if err != nil {
			fmt.Printf("Warning: failed to scan module %s: %v\n", modulePath, err)
			continue
		}

		fmt.Printf("Found %d functions (including dependencies) in module %s\n", len(moduleFunctions), modulePath)
		externalFunctions = append(externalFunctions, moduleFunctions...)
	}

	// Post-process external functions with type information
	externalFunctions = enhanceExternalFunctionsWithTypeInfo(externalFunctions, typeInfo)

	return externalFunctions, nil
}

// enhanceExternalFunctionsWithTypeInfo is a placeholder for any external call resolution based on type info
func enhanceExternalFunctionsWithTypeInfo(functions []FunctionInfo, _ map[string]TypeInfo) []FunctionInfo {
	return functions
}

// EnhanceProjectFunctionsWithTypeInfo enhances project functions with type resolution and interface implementation detection
func EnhanceProjectFunctionsWithTypeInfo(functions []FunctionInfo, projectPath string) []FunctionInfo {
	// Parse type information for the project
	typeInfo, err := ParseTypeInformation(projectPath, make(map[string]ExternalModuleInfo))
	if err != nil {
		fmt.Printf("Warning: failed to parse project type information: %v\n", err)
		return functions
	}

	// Parse comprehensive file information
	fileInfoMap := make(map[string]FileTypeInfo)
	err = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			relPath, _ := filepath.Rel(projectPath, path)
			fileInfo, err := ParseGoFileForTypesAndImports(path, projectPath)
			if err != nil {
				return nil // Skip files that can't be parsed
			}
			fileInfoMap[relPath] = fileInfo
		}
		return nil
	})
	if err != nil {
		fmt.Printf("Warning: failed to parse comprehensive file information: %v\n", err)
		return functions
	}

	// Find interface implementations
	implementations, err := FindInterfaceImplementations(projectPath)
	if err != nil {
		fmt.Printf("Warning: failed to find interface implementations: %v\n", err)
		implementations = make(map[string][]InterfaceImplementation)
	}

	// Process each function to resolve its method calls and add implementation calls
	var enhancedFunctions []FunctionInfo
	for _, fn := range functions {
		enhancedCalls := make([]string, 0, len(fn.Calls))

		for _, call := range fn.Calls {
			// Try to resolve the call using comprehensive type information
			resolvedCall := ResolveMethodCall(call, fileInfoMap, typeInfo, implementations)
			enhancedCalls = append(enhancedCalls, resolvedCall)

			// If this is an interface method call, add the implementation calls
			implementationFunctions := GetImplementationCalls(call, implementations)
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
