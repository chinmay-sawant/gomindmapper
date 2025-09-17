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
	for j := funcLine; j < len(lines); j++ {
		for _, char := range lines[j] {
			if char == '{' {
				braceCount++
				if start == -1 {
					start = j
				}
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					return start, j
				}
			}
		}
	}
	return -1, -1
}

func FindCalls(bodyLines []string) []string {
	var calls []string
	reCalls := regexp.MustCompile(`(\w+(?:\.\w+)*)\(`)
	for _, line := range bodyLines {
		matches := reCalls.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			call := match[1]
			if !contains(calls, call) {
				calls = append(calls, call)
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
