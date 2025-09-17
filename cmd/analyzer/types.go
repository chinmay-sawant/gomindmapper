package analyzer

type FunctionInfo struct {
	Name     string
	Line     int
	FilePath string
	Calls    []string
}

type FunctionRelation struct {
	Name     string         `json:"name"`
	Line     int            `json:"line"`
	FilePath string         `json:"filePath"`
	Called   []FunctionInfo `json:"called"`
}
