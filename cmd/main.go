package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/chinmay-sawant/gomindmapper/cmd/analyzer"
)

func main() {
	var path string
	var includeExternal bool
	var skipFolders string
	flag.StringVar(&path, "path", ".", "path to repository")
	flag.BoolVar(&includeExternal, "include-external", false, "include external library calls in output (skip removed_calls.json generation)")
	flag.StringVar(&skipFolders, "skip-folders", "", "comma-separated list of folder patterns to skip when scanning external dependencies (e.g., 'golang.org,google.golang.org')")
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
			funcs, err := analyzer.FindFunctions(path, absPath, module)
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

		// Parse skip patterns
		var skipPatterns []string
		if skipFolders != "" {
			skipPatterns = strings.Split(skipFolders, ",")
			for i, pattern := range skipPatterns {
				skipPatterns[i] = strings.TrimSpace(pattern)
			}
			fmt.Printf("Skipping external dependency folders matching: %v\n", skipPatterns)
		}

		externalFunctions, err := analyzer.ScanExternalModules(absPath, functions, skipPatterns)
		if err != nil {
			fmt.Printf("Warning: failed to scan external modules: %v\n", err)
		} else {
			functions = append(functions, externalFunctions...)
			fmt.Printf("Successfully scanned external modules and found %d external functions\n", len(externalFunctions))
		}
	}

	// Enhance project functions with type resolution before external scanning
	if !includeExternal {
		functions = analyzer.EnhanceProjectFunctionsWithTypeInfo(functions, ".")
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

// legacy buildFunctionMap removed: functionality now in analyzer.BuildRelations
