package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func GetModule(absPath string) (string, error) {
	goModPath := filepath.Join(absPath, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("module not found in go.mod")
}

func FindFunctionBody(lines []string, funcLine int) (int, int) {
	braceCount := 0
	start := -1

	// Skip the function declaration line and find the opening brace
	for j := funcLine; j < len(lines); j++ {
		line := lines[j]
		for _, char := range line {
			switch char {
			case '{':
				braceCount++
				if start == -1 {
					start = j
				}
			case '}':
				braceCount--
				if braceCount == 0 && start != -1 {
					return start, j
				}
			}
		}
		// If we found the opening brace, we can start looking for the closing one
		if start != -1 && braceCount == 0 {
			break
		}
	}
	return -1, -1
}

func FindCalls(bodyLines []string) []string {
	var calls []string
	reCalls := regexp.MustCompile(`(\w+(?:\.\w+)*)\(`)
	// Enhanced regex to capture function references passed as arguments
	// This captures function names that appear as arguments (not followed by parentheses)
	reFuncRefs := regexp.MustCompile(`(?:\s|,|\()(handle[A-Za-z]\w+)(?:\s*[,\)\s]|$)`)
	// Enhanced regex to capture method calls on struct fields (e.g., svc.FormDatastore.GetFormId)
	reMethodCalls := regexp.MustCompile(`(\w+\.\w+\.\w+)\(`)

	// Standard library packages to ignore
	standardPackages := map[string]bool{
		"fmt":      true,
		"os":       true,
		"strings":  true,
		"regexp":   true,
		"encoding": true,
		"bytes":    true,
		"strconv":  true,
		"time":     true,
		"context":  true,
		"sync":     true,
		"runtime":  true,
		"sort":     true,
		"json":     true,
		"xml":      true,
		"filepath": true,
		"bufio":    true,
		"io":       true,
		"ioutil":   true,
		"log":      true,
		"errors":   true,
		"flag":     true,
		"math":     true,
		"unicode":  true,
		"reflect":  true,
		"syscall":  true,
		"unsafe":   true,
		"archive":  true,
		"compress": true,
		"crypto":   true,
		"database": true,
		"debug":    true,
		"expvar":   true,
		"hash":     true,
		"html":     true,
		"image":    true,
		"index":    true,
		"internal": true,
		"mime":     true,
		"net":      true,
		"path":     true,
		"plugin":   true,
		"testing":  true,
		"text":     true,
	}

	// Regex function patterns to exclude
	regexFunctions := map[string]bool{
		"FindAllSubmatch":       true,
		"FindSubmatch":          true,
		"FindAllStringSubmatch": true,
		"FindStringSubmatch":    true,
		"FindAllIndex":          true,
		"FindIndex":             true,
		"FindAllString":         true,
		"FindString":            true,
		"FindAll":               true,
		"Find":                  true,
		"Match":                 true,
		"MatchString":           true,
		"ReplaceAll":            true,
		"ReplaceAllString":      true,
		"ReplaceAllFunc":        true,
		"Split":                 true,
		"FindSubmatchIndex":     true,
		"FindAllSubmatchIndex":  true,
	}

	// Error handling patterns to exclude
	errorFunctions := map[string]bool{
		"Error":  true,
		"Errorf": true,
		"Errors": true,
		"err":    true,
	}

	// Wait group functions to exclude
	waitGroupFunctions := map[string]bool{
		"Add":  true,
		"Done": true,
		"Wait": true,
	}

	// Keywords and common variable names to exclude
	excludeKeywords := map[string]bool{
		"true":      true,
		"false":     true,
		"nil":       true,
		"err":       true,
		"error":     true,
		"string":    true,
		"int":       true,
		"float":     true,
		"bool":      true,
		"byte":      true,
		"rune":      true,
		"if":        true,
		"for":       true,
		"switch":    true,
		"case":      true,
		"default":   true,
		"return":    true,
		"break":     true,
		"continue":  true,
		"goto":      true,
		"var":       true,
		"const":     true,
		"type":      true,
		"func":      true,
		"package":   true,
		"import":    true,
		"range":     true,
		"select":    true,
		"go":        true,
		"defer":     true,
		"chan":      true,
		"map":       true,
		"struct":    true,
		"interface": true,
	}

	for _, line := range bodyLines {
		// First, find method calls on struct fields (e.g., svc.FormDatastore.GetFormId())
		methodMatches := reMethodCalls.FindAllStringSubmatch(line, -1)
		for _, match := range methodMatches {
			call := match[1]
			parts := strings.Split(call, ".")
			if len(parts) >= 3 {
				// For calls like svc.FormDatastore.GetFormId, we want to capture FormDatastore.GetFormId
				// This helps with type resolution later
				methodCall := strings.Join(parts[1:], ".")
				if !contains(calls, methodCall) {
					calls = append(calls, methodCall)
				}
				// Also capture the full call for complete analysis
				if !contains(calls, call) {
					calls = append(calls, call)
				}
			}
		}

		// Find traditional function calls (package.function())
		matches := reCalls.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			call := match[1]
			// Skip if it's a builtin function (no dot)
			if !strings.Contains(call, ".") {
				continue
			}
			// Skip if it's a standard library call
			parts := strings.Split(call, ".")
			if len(parts) > 0 && standardPackages[parts[0]] {
				continue
			}

			// Skip regex functions
			if len(parts) > 1 {
				functionName := parts[len(parts)-1]
				if regexFunctions[functionName] {
					continue
				}
				// Skip error functions
				if errorFunctions[functionName] {
					continue
				}
				// Skip wait group functions (wg.Add, wg.Done, wg.Wait)
				if len(parts) >= 2 && strings.HasPrefix(parts[0], "wg") && waitGroupFunctions[functionName] {
					continue
				}
			}

			if !contains(calls, call) {
				calls = append(calls, call)
			}
		}

		// Find function references passed as arguments
		refMatches := reFuncRefs.FindAllStringSubmatch(line, -1)
		for _, match := range refMatches {
			if len(match) > 1 {
				funcRef := strings.TrimSpace(match[1])

				// Skip if it's a keyword or common variable name
				if excludeKeywords[funcRef] {
					continue
				}

				// Skip if it's too short to be meaningful
				if len(funcRef) < 3 {
					continue
				}

				// Skip if it contains dots (already handled by reCalls)
				if strings.Contains(funcRef, ".") {
					continue
				}

				// Add the current package prefix to make it consistent with other calls
				if !contains(calls, funcRef) {
					calls = append(calls, funcRef)
				}
			}
		}
	}
	return calls
}

// CallInfo represents a function call with its line number
type CallInfo struct {
	Name string
	Line int
}

// FindCallsWithLines finds function calls in the body lines and returns them with line numbers
func FindCallsWithLines(bodyLines []string, startLineOffset int) []CallInfo {
	var calls []CallInfo
	reCalls := regexp.MustCompile(`(\w+(?:\.\w+)*)\(`)
	// Enhanced regex to capture function references passed as arguments
	reFuncRefs := regexp.MustCompile(`(?:\s|,|\()(handle[A-Za-z]\w+)(?:\s*[,\)\s]|$)`)
	// Enhanced regex to capture method calls on struct fields (e.g., svc.FormDatastore.GetFormId)
	reMethodCalls := regexp.MustCompile(`(\w+\.\w+\.\w+)\(`)

	// Standard library packages to ignore
	standardPackages := map[string]bool{
		"fmt":      true,
		"os":       true,
		"strings":  true,
		"regexp":   true,
		"encoding": true,
		"bytes":    true,
		"strconv":  true,
		"time":     true,
		"context":  true,
		"sync":     true,
		"runtime":  true,
		"sort":     true,
		"json":     true,
		"xml":      true,
		"filepath": true,
		"bufio":    true,
		"io":       true,
		"ioutil":   true,
		"log":      true,
		"errors":   true,
		"flag":     true,
		"math":     true,
		"unicode":  true,
		"reflect":  true,
		"syscall":  true,
		"unsafe":   true,
		"archive":  true,
		"compress": true,
		"crypto":   true,
		"database": true,
		"debug":    true,
		"expvar":   true,
		"hash":     true,
		"html":     true,
		"image":    true,
		"index":    true,
		"internal": true,
		"mime":     true,
		"net":      true,
		"path":     true,
		"plugin":   true,
		"testing":  true,
		"text":     true,
	}

	// Regex function patterns to exclude
	regexFunctions := map[string]bool{
		"FindAllSubmatch":       true,
		"FindSubmatch":          true,
		"FindAllStringSubmatch": true,
		"FindStringSubmatch":    true,
		"FindAllIndex":          true,
		"FindIndex":             true,
		"FindAllString":         true,
		"FindString":            true,
		"FindAll":               true,
		"Find":                  true,
		"Match":                 true,
		"MatchString":           true,
		"ReplaceAll":            true,
		"ReplaceAllString":      true,
		"ReplaceAllFunc":        true,
		"Split":                 true,
		"FindSubmatchIndex":     true,
		"FindAllSubmatchIndex":  true,
	}

	// Error handling patterns to exclude
	errorFunctions := map[string]bool{
		"Error":  true,
		"Errorf": true,
		"Errors": true,
		"err":    true,
	}

	// Wait group functions to exclude
	waitGroupFunctions := map[string]bool{
		"Add":  true,
		"Done": true,
		"Wait": true,
	}

	// Keywords and common variable names to exclude
	excludeKeywords := map[string]bool{
		"true":      true,
		"false":     true,
		"nil":       true,
		"err":       true,
		"error":     true,
		"string":    true,
		"int":       true,
		"float":     true,
		"bool":      true,
		"byte":      true,
		"rune":      true,
		"if":        true,
		"for":       true,
		"switch":    true,
		"case":      true,
		"default":   true,
		"return":    true,
		"break":     true,
		"continue":  true,
		"goto":      true,
		"var":       true,
		"const":     true,
		"type":      true,
		"func":      true,
		"package":   true,
		"import":    true,
		"range":     true,
		"select":    true,
		"go":        true,
		"defer":     true,
		"chan":      true,
		"map":       true,
		"struct":    true,
		"interface": true,
	}

	// Helper function to check if call already exists
	callExists := func(callName string) bool {
		for _, call := range calls {
			if call.Name == callName {
				return true
			}
		}
		return false
	}

	for i, line := range bodyLines {
		currentLineNum := startLineOffset + i + 1

		// First, find method calls on struct fields (e.g., svc.FormDatastore.GetFormId())
		methodMatches := reMethodCalls.FindAllStringSubmatch(line, -1)
		for _, match := range methodMatches {
			call := match[1]
			parts := strings.Split(call, ".")
			if len(parts) >= 3 {
				// For calls like svc.FormDatastore.GetFormId, we want to capture FormDatastore.GetFormId
				methodCall := strings.Join(parts[1:], ".")
				if !callExists(methodCall) {
					calls = append(calls, CallInfo{Name: methodCall, Line: currentLineNum})
				}
				// Also capture the full call for complete analysis
				if !callExists(call) {
					calls = append(calls, CallInfo{Name: call, Line: currentLineNum})
				}
			}
		}

		// Find traditional function calls (package.function())
		matches := reCalls.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			call := match[1]
			// Skip if it's a builtin function (no dot)
			if !strings.Contains(call, ".") {
				continue
			}
			// Skip if it's a standard library call
			parts := strings.Split(call, ".")
			if len(parts) > 0 && standardPackages[parts[0]] {
				continue
			}

			// Skip regex functions
			if len(parts) > 1 {
				functionName := parts[len(parts)-1]
				if regexFunctions[functionName] {
					continue
				}
				// Skip error functions
				if errorFunctions[functionName] {
					continue
				}
				// Skip wait group functions (wg.Add, wg.Done, wg.Wait)
				if len(parts) >= 2 && strings.HasPrefix(parts[0], "wg") && waitGroupFunctions[functionName] {
					continue
				}
			}

			if !callExists(call) {
				calls = append(calls, CallInfo{Name: call, Line: currentLineNum})
			}
		}

		// Find function references passed as arguments
		refMatches := reFuncRefs.FindAllStringSubmatch(line, -1)
		for _, match := range refMatches {
			if len(match) > 1 {
				funcRef := strings.TrimSpace(match[1])

				// Skip if it's a keyword or common variable name
				if excludeKeywords[funcRef] {
					continue
				}

				// Skip if it's too short to be meaningful
				if len(funcRef) < 3 {
					continue
				}

				// Skip if it contains dots (already handled by reCalls)
				if strings.Contains(funcRef, ".") {
					continue
				}

				if !callExists(funcRef) {
					calls = append(calls, CallInfo{Name: funcRef, Line: currentLineNum})
				}
			}
		}
	}
	return calls
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
