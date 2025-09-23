package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func CreateJsonFile(functions []FunctionInfo, includeExternal bool) {
	// Compute user-defined package prefixes (e.g., "pdf", "handlers", "analyzer")
	userPrefixes := make(map[string]bool)
	for _, f := range functions {
		if dot := strings.Index(f.Name, "."); dot != -1 {
			userPrefixes[f.Name[:dot]] = true
		}
	}

	// Track removed calls for reporting
	removedPerFunc := make(map[string][]string)
	removedSet := make(map[string]bool)

	// Filter calls based on includeExternal parameter
	if !includeExternal {
		// Filter calls to only include user-defined package prefixed calls
		for i := range functions {
			if len(functions[i].Calls) == 0 {
				continue
			}
			var filtered []string
			var removed []string
			for _, c := range functions[i].Calls {
				// we only consider dotted calls (pkg.Func)
				if !strings.Contains(c, ".") {
					removed = append(removed, c)
					removedSet[c] = true
					continue
				}
				parts := strings.Split(c, ".")
				if len(parts) == 0 {
					removed = append(removed, c)
					removedSet[c] = true
					continue
				}
				if userPrefixes[parts[0]] {
					// keep the call
					filtered = append(filtered, c)
				} else {
					removed = append(removed, c)
					removedSet[c] = true
				}
			}
			if len(filtered) == 0 {
				functions[i].Calls = nil
			} else {
				functions[i].Calls = filtered
			}
			if len(removed) > 0 {
				removedPerFunc[functions[i].Name] = removed
			}
		}
	}
	// If includeExternal is true, keep all calls as-is (no filtering)

	data, err := json.MarshalIndent(functions, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile("functions.json", data, 0644)
	if err != nil {
		fmt.Println(err)
	}

	// Write a report of removed calls only if we filtered calls
	if !includeExternal {
		var removedList []string
		for c := range removedSet {
			removedList = append(removedList, c)
		}
		report := map[string]interface{}{
			"removedPerFunction": removedPerFunc,
			"uniqueRemovedCalls": removedList,
		}
		rdata, rerr := json.MarshalIndent(report, "", "  ")
		if rerr == nil {
			_ = os.WriteFile("removed_calls.json", rdata, 0644)
		}
	}
}
