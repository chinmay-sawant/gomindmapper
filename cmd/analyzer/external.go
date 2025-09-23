package analyzer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ExternalModuleInfo represents information about an external module
type ExternalModuleInfo struct {
	ModulePath string
	Version    string
	LocalPath  string
}

// GetExternalModules parses go.mod and go.sum to extract external module information
func GetExternalModules(projectPath string) (map[string]ExternalModuleInfo, error) {
	modules := make(map[string]ExternalModuleInfo)

	// Read go.mod file
	goModPath := filepath.Join(projectPath, "go.mod")
	file, err := os.Open(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open go.mod: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inRequireBlock := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if strings.HasPrefix(line, "//") || line == "" {
			continue
		}

		// Handle require block
		if strings.HasPrefix(line, "require") {
			if strings.Contains(line, "(") {
				inRequireBlock = true
				// Check if there's a requirement on the same line
				if strings.Contains(line, ")") {
					inRequireBlock = false
				}
				continue
			} else {
				// Single line require
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					modulePath := parts[1]
					version := parts[2]
					modules[modulePath] = ExternalModuleInfo{
						ModulePath: modulePath,
						Version:    version,
					}
				}
				continue
			}
		}

		// Handle lines inside require block
		if inRequireBlock {
			if strings.Contains(line, ")") {
				inRequireBlock = false
				// Check if there's a requirement before the closing bracket
				parts := strings.Fields(strings.Replace(line, ")", "", 1))
				if len(parts) >= 2 {
					modulePath := parts[0]
					version := parts[1]
					modules[modulePath] = ExternalModuleInfo{
						ModulePath: modulePath,
						Version:    version,
					}
				}
				continue
			}

			parts := strings.Fields(line)
			if len(parts) >= 2 {
				modulePath := parts[0]
				version := parts[1]
				modules[modulePath] = ExternalModuleInfo{
					ModulePath: modulePath,
					Version:    version,
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading go.mod: %v", err)
	}

	return modules, nil
}

// FindModuleInGoPath searches for the module in GOPATH/pkg/mod
func FindModuleInGoPath(moduleInfo ExternalModuleInfo) (string, error) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		// Try default GOPATH location
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("GOPATH not set and cannot determine home directory")
		}
		gopath = filepath.Join(homeDir, "go")
	}

	modCachePath := filepath.Join(gopath, "pkg", "mod")

	// Convert module path to filesystem path
	// Example: github.com/chinmay-sawant/gochromedp@v1.0.2
	// becomes: github.com/chinmay-sawant/gochromedp@v1.0.2
	version := strings.TrimPrefix(moduleInfo.Version, "v")
	moduleDir := fmt.Sprintf("%s@v%s", moduleInfo.ModulePath, version)

	// Handle different possible path formats
	possiblePaths := []string{
		filepath.Join(modCachePath, moduleDir),
		filepath.Join(modCachePath, strings.ToLower(moduleDir)),
	}

	// Also try with exclamation marks for uppercase letters (Go module cache encoding)
	encodedPath := encodeModulePath(moduleDir)
	possiblePaths = append(possiblePaths, filepath.Join(modCachePath, encodedPath))

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("module %s not found in module cache", moduleInfo.ModulePath)
}

// encodeModulePath encodes uppercase letters in module paths as Go module cache does
func encodeModulePath(path string) string {
	var result strings.Builder
	for _, r := range path {
		if r >= 'A' && r <= 'Z' {
			result.WriteRune('!')
			result.WriteRune(r + 32) // Convert to lowercase
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ScanExternalModule scans an external module for Go functions
func ScanExternalModule(modulePath string, moduleInfo ExternalModuleInfo) ([]FunctionInfo, error) {
	var functions []FunctionInfo

	err := filepath.Walk(modulePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, test files, and non-Go files
		if info.IsDir() || strings.HasSuffix(path, "_test.go") || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip vendor directories and hidden directories
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "\\.") {
			return nil
		}

		funcs, err := scanExternalGoFile(path, modulePath, moduleInfo.ModulePath)
		if err != nil {
			// Log error but continue scanning other files
			fmt.Printf("Warning: failed to scan %s: %v\n", path, err)
			return nil
		}

		functions = append(functions, funcs...)
		return nil
	})

	return functions, err
}

// scanExternalGoFile scans a single Go file in an external module
func scanExternalGoFile(filePath, modulePath, moduleImportPath string) ([]FunctionInfo, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	var funcs []FunctionInfo

	// Regex patterns for functions and methods
	reFunc := regexp.MustCompile(`^\s*func\s+([A-Z]\w*)`)               // Only exported functions
	reMethod := regexp.MustCompile(`^\s*func\s+\([^)]+\)\s+([A-Z]\w*)`) // Only exported methods

	// Get relative path from module root
	relPath, err := filepath.Rel(modulePath, filePath)
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

	// If package is main, skip it as it's likely an example or cmd
	if packageName == "main" {
		return funcs, nil
	}

	// Note: We use the moduleImportPath directly for external functions
	// to maintain consistency with Go import paths

	// Collect all exported function names in this file
	var localFunctions []string
	for _, line := range lines {
		if matches := reFunc.FindStringSubmatch(line); matches != nil {
			localFunctions = append(localFunctions, matches[1])
		} else if matches := reMethod.FindStringSubmatch(line); matches != nil {
			localFunctions = append(localFunctions, matches[1])
		}
	}

	// Process each function/method
	for i, line := range lines {
		var functionName string

		// Check for regular exported functions
		if matches := reFunc.FindStringSubmatch(line); matches != nil {
			functionName = matches[1]
		} else if matches := reMethod.FindStringSubmatch(line); matches != nil {
			// Check for exported methods
			functionName = matches[1]
		}

		if functionName != "" {
			// Create function info with external module path
			funcInfo := FunctionInfo{
				Name:     moduleImportPath + "." + functionName,
				Line:     i + 1,
				FilePath: "external:" + relPath, // Mark as external with relative path
			}

			// Find function body and calls
			start, end := FindFunctionBody(lines, i)
			if start != -1 && end != -1 && start+1 < end && end < len(lines) {
				calls := FindCalls(lines[start+1 : end])

				// Resolve local function references
				var resolvedCalls []string
				for _, call := range calls {
					if !strings.Contains(call, ".") {
						// Check if it's a local function reference
						for _, localFunc := range localFunctions {
							if call == localFunc {
								resolvedCalls = append(resolvedCalls, moduleImportPath+"."+call)
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

// FilterRelevantExternalModules filters external modules to only include those that are actually called
func FilterRelevantExternalModules(functions []FunctionInfo, modules map[string]ExternalModuleInfo) map[string]ExternalModuleInfo {
	relevantModules := make(map[string]ExternalModuleInfo)

	// Collect all external calls
	externalCalls := make(map[string]bool)
	for _, fn := range functions {
		for _, call := range fn.Calls {
			if strings.Contains(call, ".") {
				parts := strings.Split(call, ".")
				if len(parts) >= 2 {
					// Extract potential module path
					packagePath := strings.Join(parts[:len(parts)-1], ".")
					externalCalls[packagePath] = true
				}
			}
		}
	}

	// Find matching modules
	for modulePath, moduleInfo := range modules {
		// Check if any external call matches this module
		for callPath := range externalCalls {
			// Check if the call matches the module path
			if strings.HasPrefix(callPath, modulePath) || strings.HasSuffix(modulePath, callPath) {
				relevantModules[modulePath] = moduleInfo
				break
			}

			// Also check if the call matches a known package name pattern
			// e.g., "gochromedp.ConvertHTMLToPDF" should match "github.com/chinmay-sawant/gochromedp"
			moduleBase := filepath.Base(modulePath)
			if callPath == moduleBase {
				relevantModules[modulePath] = moduleInfo
				break
			}
		}
	}
	return relevantModules
}
