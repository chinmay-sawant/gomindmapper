package analyzer

type FunctionInfo struct {
	Name     string
	Line     int
	FilePath string
	Calls    []string
}
