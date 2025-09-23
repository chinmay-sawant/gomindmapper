package analyzer

// OutCalled is a light-weight representation of a called function used in JSON output.
type OutCalled struct {
	Name     string `json:"name"`
	Line     int    `json:"line"`
	FilePath string `json:"filePath"`
}

// OutRelation represents a function and the functions it directly calls (already filtered to user-defined pkgs).
type OutRelation struct {
	Name     string      `json:"name"`
	Line     int         `json:"line"`
	FilePath string      `json:"filePath"`
	Called   []OutCalled `json:"called,omitempty"`
}

// BuildRelations converts raw FunctionInfo + their Calls into OutRelation list.
// If includeExternal is false, the provided slice must already have Calls filtered to user-defined packages (CreateJsonFile performs this filtering).
// If includeExternal is true, all calls are included in the relations.
// We still defensively exclude relations that have zero called entries to preserve prior semantics unless includeExternal is true.
func BuildRelations(functions []FunctionInfo, includeExternal bool) []OutRelation {
	// index by name for quick lookup
	funcMap := make(map[string]FunctionInfo, len(functions))
	for _, f := range functions {
		funcMap[f.Name] = f
	}

	out := make([]OutRelation, 0, len(functions))
	for _, f := range functions {
		if len(f.Calls) == 0 && !includeExternal {
			continue // skip functions with no user-defined calls (previous behaviour)
		}
		rel := OutRelation{Name: f.Name, Line: f.Line, FilePath: f.FilePath}
		for _, cname := range f.Calls {
			if cf, ok := funcMap[cname]; ok {
				// Function exists in our codebase
				rel.Called = append(rel.Called, OutCalled{Name: cf.Name, Line: cf.Line, FilePath: cf.FilePath})
			} else if includeExternal {
				// External function call - include with placeholder info
				rel.Called = append(rel.Called, OutCalled{Name: cname, Line: 0, FilePath: "external"})
			}
		}
		if len(rel.Called) > 0 || includeExternal {
			// Include the relation if it has calls OR if we're including all functions
			out = append(out, rel)
		}
	}
	return out
}
