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

// BuildRelations converts raw FunctionInfo + their filtered Calls into OutRelation list identical to previous buildFunctionMap output.
// The provided slice must already have Calls filtered to user-defined packages (CreateJsonFile performs this filtering). We
// still defensively exclude relations that have zero called entries to preserve prior semantics.
func BuildRelations(functions []FunctionInfo) []OutRelation {
	// index by name for quick lookup
	funcMap := make(map[string]FunctionInfo, len(functions))
	for _, f := range functions {
		funcMap[f.Name] = f
	}

	out := make([]OutRelation, 0, len(functions))
	for _, f := range functions {
		if len(f.Calls) == 0 {
			continue // skip functions with no user-defined calls (previous behaviour)
		}
		rel := OutRelation{Name: f.Name, Line: f.Line, FilePath: f.FilePath}
		for _, cname := range f.Calls {
			if cf, ok := funcMap[cname]; ok { // only include if present
				rel.Called = append(rel.Called, OutCalled{Name: cf.Name, Line: cf.Line, FilePath: cf.FilePath})
			}
		}
		if len(rel.Called) > 0 { // only append if at least one resolved call
			out = append(out, rel)
		}
	}
	return out
}
