package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// ImportInfo represents information about an import statement
type ImportInfo struct {
	Alias       string // import alias (empty if no alias)
	Path        string // import path
	PackageName string // actual package name
}

// TypeInfo represents information about a type declaration
type TypeInfo struct {
	Name        string
	Package     string
	IsInterface bool
	IsStruct    bool
	Fields      map[string]string // field name -> type
	Methods     []string
	ImportPath  string // for external types
}

// FileTypeInfo represents comprehensive type information for a file
type FileTypeInfo struct {
	Imports    map[string]ImportInfo // alias/package -> ImportInfo
	Types      map[string]TypeInfo   // type name -> TypeInfo
	Structs    map[string]TypeInfo   // struct definitions
	Interfaces map[string]TypeInfo   // interface definitions
}

// InterfaceImplementation represents a struct that implements an interface
type InterfaceImplementation struct {
	InterfaceName string
	StructName    string
	PackageName   string
	FilePath      string
	Methods       map[string]MethodImplementation // method name -> implementation details
}

// MethodImplementation represents details about a method implementation
type MethodImplementation struct {
	Name       string
	StructName string
	FilePath   string
	StartLine  int
	EndLine    int
	Calls      []string // calls made within this method
}

// ParseTypeInformation extracts type information from Go files with enhanced import analysis
func ParseTypeInformation(projectPath string, externalModules map[string]ExternalModuleInfo) (map[string]TypeInfo, error) {
	typeInfo := make(map[string]TypeInfo)
	fileInfoMap := make(map[string]FileTypeInfo)

	// Parse project files for type declarations and imports
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			fileInfo, err := ParseGoFileForTypesAndImports(path, projectPath)
			if err != nil {
				return err
			}
			fileInfoMap[path] = fileInfo
			for k, v := range fileInfo.Types {
				typeInfo[k] = v
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Parse external modules for type declarations
	for _, moduleInfo := range externalModules {
		localPath, err := FindModuleInGoPath(moduleInfo)
		if err != nil {
			continue // Skip modules that can't be found
		}

		externalTypes, err := parseExternalModuleForTypes(localPath, moduleInfo.ModulePath)
		if err != nil {
			continue // Skip modules that can't be parsed
		}

		for k, v := range externalTypes {
			typeInfo[k] = v
		}
	}

	return typeInfo, nil
}

// ParseGoFileForTypesAndImports parses a single Go file and extracts comprehensive type and import information
func ParseGoFileForTypesAndImports(filePath, projectPath string) (FileTypeInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return FileTypeInfo{}, err
	}

	fileInfo := FileTypeInfo{
		Imports:    make(map[string]ImportInfo),
		Types:      make(map[string]TypeInfo),
		Structs:    make(map[string]TypeInfo),
		Interfaces: make(map[string]TypeInfo),
	}

	packageName := node.Name.Name

	// Parse imports
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		alias := ""
		pkgName := filepath.Base(importPath)

		if imp.Name != nil {
			alias = imp.Name.Name
			if alias == "_" || alias == "." {
				continue // Skip blank and dot imports for now
			}
			pkgName = alias
		}

		fileInfo.Imports[pkgName] = ImportInfo{
			Alias:       alias,
			Path:        importPath,
			PackageName: pkgName,
		}
	}

	// Parse type declarations
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			typeInfo := TypeInfo{
				Name:    x.Name.Name,
				Package: packageName,
				Fields:  make(map[string]string),
			}

			// Use package name as key for local types
			key := packageName + "." + x.Name.Name

			switch t := x.Type.(type) {
			case *ast.StructType:
				typeInfo.IsStruct = true
				if t.Fields != nil {
					for _, field := range t.Fields.List {
						if len(field.Names) > 0 && field.Type != nil {
							fieldName := field.Names[0].Name
							fieldType := getTypeStringWithImports(field.Type, fileInfo.Imports)
							typeInfo.Fields[fieldName] = fieldType
						}
					}
				}
				fileInfo.Structs[key] = typeInfo
			case *ast.InterfaceType:
				typeInfo.IsInterface = true
				if t.Methods != nil {
					for _, method := range t.Methods.List {
						if len(method.Names) > 0 {
							typeInfo.Methods = append(typeInfo.Methods, method.Names[0].Name)
						}
					}
				}
				fileInfo.Interfaces[key] = typeInfo
			}

			fileInfo.Types[key] = typeInfo
		}
		return true
	})

	return fileInfo, nil
}

// parseGoFileForTypes parses a single Go file and extracts type information (legacy function)
func parseGoFileForTypes(filePath, projectPath string) (map[string]TypeInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	types := make(map[string]TypeInfo)
	packageName := node.Name.Name

	// Get relative path (not used but kept for potential future use)
	_, _ = filepath.Rel(projectPath, filePath)

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.TypeSpec:
			typeInfo := TypeInfo{
				Name:    x.Name.Name,
				Package: packageName,
				Fields:  make(map[string]string),
			}

			switch t := x.Type.(type) {
			case *ast.StructType:
				typeInfo.IsStruct = true
				if t.Fields != nil {
					for _, field := range t.Fields.List {
						if len(field.Names) > 0 && field.Type != nil {
							fieldName := field.Names[0].Name
							fieldType := getTypeString(field.Type)
							typeInfo.Fields[fieldName] = fieldType
						}
					}
				}
			case *ast.InterfaceType:
				typeInfo.IsInterface = true
				if t.Methods != nil {
					for _, method := range t.Methods.List {
						if len(method.Names) > 0 {
							typeInfo.Methods = append(typeInfo.Methods, method.Names[0].Name)
						}
					}
				}
			}

			// Use package name as key for local types
			key := packageName + "." + x.Name.Name
			types[key] = typeInfo
		}
		return true
	})

	return types, nil
}

// parseExternalModuleForTypes parses external module for type information
func parseExternalModuleForTypes(modulePath, importPath string) (map[string]TypeInfo, error) {
	types := make(map[string]TypeInfo)

	err := filepath.Walk(modulePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		// Skip vendor directories and hidden directories
		if strings.Contains(path, "/vendor/") || strings.Contains(path, "\\.") {
			return nil
		}

		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		if err != nil {
			return nil // Skip files that can't be parsed
		}

		packageName := node.Name.Name
		if packageName == "main" {
			return nil // Skip main packages
		}

		ast.Inspect(node, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.TypeSpec:
				// Only process exported types
				if !ast.IsExported(x.Name.Name) {
					return true
				}

				typeInfo := TypeInfo{
					Name:       x.Name.Name,
					Package:    packageName,
					ImportPath: importPath,
					Fields:     make(map[string]string),
				}

				switch t := x.Type.(type) {
				case *ast.StructType:
					typeInfo.IsStruct = true
					if t.Fields != nil {
						for _, field := range t.Fields.List {
							if len(field.Names) > 0 && field.Type != nil && ast.IsExported(field.Names[0].Name) {
								fieldName := field.Names[0].Name
								fieldType := getTypeString(field.Type)
								typeInfo.Fields[fieldName] = fieldType
							}
						}
					}
				case *ast.InterfaceType:
					typeInfo.IsInterface = true
					if t.Methods != nil {
						for _, method := range t.Methods.List {
							if len(method.Names) > 0 && ast.IsExported(method.Names[0].Name) {
								typeInfo.Methods = append(typeInfo.Methods, method.Names[0].Name)
							}
						}
					}
				}

				// Use full import path as key for external types
				key := importPath + "." + x.Name.Name
				types[key] = typeInfo
			}
			return true
		})

		return nil
	})

	return types, err
}

// getTypeStringWithImports converts an ast.Expr to a string representation with import resolution
func getTypeStringWithImports(expr ast.Expr, imports map[string]ImportInfo) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			// Resolve package name through imports
			if importInfo, exists := imports[pkg.Name]; exists {
				return importInfo.Path + "." + t.Sel.Name
			}
			return pkg.Name + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + getTypeStringWithImports(t.X, imports)
	case *ast.ArrayType:
		return "[]" + getTypeStringWithImports(t.Elt, imports)
	case *ast.MapType:
		return "map[" + getTypeStringWithImports(t.Key, imports) + "]" + getTypeStringWithImports(t.Value, imports)
	}
	return "unknown"
}

// getTypeString converts an ast.Expr to a string representation
func getTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
	case *ast.StarExpr:
		return "*" + getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + getTypeString(t.Elt)
	case *ast.MapType:
		return "map[" + getTypeString(t.Key) + "]" + getTypeString(t.Value)
	}
	return "unknown"
}

// ResolveMethodCall attempts to resolve a method call to its actual interface/struct method
func ResolveMethodCall(call string, fileInfoMap map[string]FileTypeInfo, allTypeInfo map[string]TypeInfo, implementations map[string][]InterfaceImplementation) string {
	parts := strings.Split(call, ".")

	// Handle different call patterns
	switch len(parts) {
	case 2:
		// Direct method call like "FormDatastore.GetFormId"
		return resolveDirectMethodCall(parts[0], parts[1], fileInfoMap, allTypeInfo, implementations)
	case 3:
		// Method call on struct field like "svc.FormDatastore.GetFormId"
		return resolveStructFieldMethodCall(parts[0], parts[1], parts[2], fileInfoMap, allTypeInfo, implementations)
	default:
		return call // Return original for other patterns
	}
}

// resolveDirectMethodCall resolves calls like "FormDatastore.GetFormId"
func resolveDirectMethodCall(typeName, methodName string, fileInfoMap map[string]FileTypeInfo, allTypeInfo map[string]TypeInfo, implementations map[string][]InterfaceImplementation) string {
	// First, try to find the interface implementation
	for interfaceName, impls := range implementations {
		for _, impl := range impls {
			if strings.Contains(interfaceName, typeName) {
				if _, exists := impl.Methods[methodName]; exists {
					// Found the actual implementation! Return the struct method
					return impl.StructName + "." + methodName
				}
			}
		}
	}

	// Fallback to original logic
	for _, info := range allTypeInfo {
		if info.IsInterface && (strings.HasSuffix(info.Name, typeName) || strings.Contains(info.Name, typeName)) {
			// Check if this interface has the method
			for _, method := range info.Methods {
				if method == methodName {
					// Found the method in the interface
					if info.ImportPath != "" {
						return info.ImportPath + "." + info.Name + "." + methodName
					}
					return info.Package + "." + info.Name + "." + methodName
				}
			}
		}
	}
	return typeName + "." + methodName
}

// resolveStructFieldMethodCall resolves calls like "svc.FormDatastore.GetFormId"
func resolveStructFieldMethodCall(varName, fieldName, methodName string, fileInfoMap map[string]FileTypeInfo, allTypeInfo map[string]TypeInfo, implementations map[string][]InterfaceImplementation) string {
	// Look through all struct definitions to find one with the specified field
	for _, fileInfo := range fileInfoMap {
		for _, structInfo := range fileInfo.Structs {
			if fieldType, exists := structInfo.Fields[fieldName]; exists {
				// Found a struct with this field, now try to find interface implementation
				resolvedType := resolveFieldType(fieldType, fileInfo.Imports, allTypeInfo)

				// Look for interface implementations that match this field type
				for interfaceName, impls := range implementations {
					// Check if the interface name matches the field type pattern
					if strings.Contains(fieldType, strings.Split(interfaceName, ".")[1]) ||
						strings.Contains(resolvedType, interfaceName) {
						for _, impl := range impls {
							if _, methodExists := impl.Methods[methodName]; methodExists {
								// Found the actual implementation!
								return impl.StructName + "." + methodName
							}
						}
					}
				}

				// Fallback to original logic
				if resolvedType != "" {
					// Check if the resolved type has the method
					for typeName, typeInfo := range allTypeInfo {
						if matchesResolvedType(typeInfo, resolvedType) && typeInfo.IsInterface {
							for _, method := range typeInfo.Methods {
								if method == methodName {
									// Found the method! Return using typeName directly
									return typeName + "." + methodName
								}
							}
						}
					}
				}

				// Fallback: try matching the original field type directly
				for typeName, typeInfo := range allTypeInfo {
					if typeInfo.IsInterface && strings.Contains(fieldType, typeInfo.Name) {
						for _, method := range typeInfo.Methods {
							if method == methodName {
								return typeName + "." + methodName
							}
						}
					}
				}
			}
		}
	}

	return varName + "." + fieldName + "." + methodName // Return original if can't resolve
}

// resolveFieldType resolves a field type string using import information
func resolveFieldType(fieldType string, imports map[string]ImportInfo, allTypeInfo map[string]TypeInfo) string {
	// Remove pointer notation
	fieldType = strings.TrimPrefix(fieldType, "*")

	// If it contains a dot, it's already qualified
	if strings.Contains(fieldType, ".") {
		return fieldType
	}

	// Try to resolve through imports
	for _, importInfo := range imports {
		qualifiedType := importInfo.Path + "." + fieldType
		if _, exists := allTypeInfo[qualifiedType]; exists {
			return qualifiedType
		}
	}

	return fieldType
}

// matchesResolvedType checks if a TypeInfo matches the resolved type string
func matchesResolvedType(typeInfo TypeInfo, resolvedType string) bool {
	if typeInfo.ImportPath != "" {
		fullName := typeInfo.ImportPath + "." + typeInfo.Name
		return fullName == resolvedType
	}
	fullName := typeInfo.Package + "." + typeInfo.Name
	return fullName == resolvedType
}

// FindInterfaceImplementations scans the project to find struct implementations of interfaces
func FindInterfaceImplementations(projectPath string) (map[string][]InterfaceImplementation, error) {
	implementations := make(map[string][]InterfaceImplementation)
	interfaceMap := make(map[string]TypeInfo)
	structMethods := make(map[string]map[string]MethodImplementation)

	// First pass: collect all interfaces and struct methods
	err := filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			fset := token.NewFileSet()
			node, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if parseErr != nil {
				return nil // Skip files with parse errors
			}

			packageName := node.Name.Name

			// Collect interfaces
			ast.Inspect(node, func(n ast.Node) bool {
				if typeSpec, ok := n.(*ast.TypeSpec); ok {
					if interfaceType, isInterface := typeSpec.Type.(*ast.InterfaceType); isInterface {
						typeInfo := TypeInfo{
							Name:        typeSpec.Name.Name,
							Package:     packageName,
							IsInterface: true,
							Methods:     []string{},
						}

						if interfaceType.Methods != nil {
							for _, method := range interfaceType.Methods.List {
								if len(method.Names) > 0 {
									typeInfo.Methods = append(typeInfo.Methods, method.Names[0].Name)
								}
							}
						}
						interfaceMap[packageName+"."+typeSpec.Name.Name] = typeInfo
					}
				}
				return true
			})

			// Collect struct methods
			ast.Inspect(node, func(n ast.Node) bool {
				if funcDecl, ok := n.(*ast.FuncDecl); ok && funcDecl.Recv != nil {
					if len(funcDecl.Recv.List) > 0 {
						recvType := ""
						switch t := funcDecl.Recv.List[0].Type.(type) {
						case *ast.Ident:
							recvType = t.Name
						case *ast.StarExpr:
							if ident, ok := t.X.(*ast.Ident); ok {
								recvType = ident.Name
							}
						}

						if recvType != "" {
							structKey := packageName + "." + recvType
							if structMethods[structKey] == nil {
								structMethods[structKey] = make(map[string]MethodImplementation)
							}

							// Get method body calls
							calls := []string{}
							if funcDecl.Body != nil {
								for _, stmt := range funcDecl.Body.List {
									calls = append(calls, extractCallsFromStatement(stmt)...)
								}
							}

							structMethods[structKey][funcDecl.Name.Name] = MethodImplementation{
								Name:       funcDecl.Name.Name,
								StructName: recvType,
								FilePath:   path,
								StartLine:  fset.Position(funcDecl.Pos()).Line,
								EndLine:    fset.Position(funcDecl.End()).Line,
								Calls:      calls,
							}
						}
					}
				}
				return true
			})
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Second pass: match struct methods to interface methods
	for interfaceName, interfaceInfo := range interfaceMap {
		for structName, methods := range structMethods {
			// Check if this struct implements the interface
			if implementsInterface(methods, interfaceInfo.Methods) {
				impl := InterfaceImplementation{
					InterfaceName: interfaceName,
					StructName:    structName,
					PackageName:   interfaceInfo.Package,
					Methods:       make(map[string]MethodImplementation),
				}

				// Map interface methods to struct methods
				for _, methodName := range interfaceInfo.Methods {
					if methodImpl, exists := methods[methodName]; exists {
						impl.Methods[methodName] = methodImpl
					}
				}

				implementations[interfaceName] = append(implementations[interfaceName], impl)
			}
		}
	}

	return implementations, nil
}

// implementsInterface checks if a struct's methods satisfy an interface
func implementsInterface(structMethods map[string]MethodImplementation, interfaceMethods []string) bool {
	for _, methodName := range interfaceMethods {
		if _, exists := structMethods[methodName]; !exists {
			return false
		}
	}
	return true
}

// extractCallsFromStatement extracts function calls from an AST statement
func extractCallsFromStatement(stmt ast.Stmt) []string {
	var calls []string

	ast.Inspect(stmt, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			callName := getCallName(callExpr.Fun)
			if callName != "" {
				calls = append(calls, callName)
			}
		}
		return true
	})

	return calls
}

// getCallName extracts the call name from a function call expression
func getCallName(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		if x, ok := e.X.(*ast.Ident); ok {
			return x.Name + "." + e.Sel.Name
		} else if sel, ok := e.X.(*ast.SelectorExpr); ok {
			if base, ok := sel.X.(*ast.Ident); ok {
				return base.Name + "." + sel.Sel.Name + "." + e.Sel.Name
			}
		}
	}
	return ""
}

// GetImplementationCalls returns all the calls made within interface method implementations
func GetImplementationCalls(interfaceCall string, implementations map[string][]InterfaceImplementation) []FunctionInfo {
	var implementationFuncs []FunctionInfo

	// Parse the interface call (e.g., "svc.Filler.Fill")
	parts := strings.Split(interfaceCall, ".")
	if len(parts) < 2 {
		return implementationFuncs
	}

	methodName := parts[len(parts)-1] // "Fill"

	// Find matching interface implementations
	for _, impls := range implementations {
		for _, impl := range impls {
			if methodImpl, exists := impl.Methods[methodName]; exists {
				// Create a FunctionInfo for the implementation method
				implFunc := FunctionInfo{
					Name:     impl.StructName + "." + methodName,
					FilePath: methodImpl.FilePath,
					Line:     methodImpl.StartLine,
					Calls:    []string{},
				}

				// Add all calls made within this implementation, resolving method calls on the same struct
				for _, call := range methodImpl.Calls {
					resolvedCall := call

					// Resolve method calls on the same struct (e.g., "t.validateConnection" -> "TestFormDatastore.validateConnection")
					if strings.HasPrefix(call, "t.") {
						methodCallName := strings.TrimPrefix(call, "t.")
						structName := strings.Split(impl.StructName, ".")[len(strings.Split(impl.StructName, "."))-1]
						resolvedCall = structName + "." + methodCallName
					}

					// Filter out standard library calls and gin/ent calls should be preserved
					if shouldIncludeCall(resolvedCall) {
						implFunc.Calls = append(implFunc.Calls, resolvedCall)

						// Also create separate FunctionInfo entries for the internal methods if they exist
						if strings.HasPrefix(call, "t.") {
							internalMethodName := strings.TrimPrefix(call, "t.")
							if internalImpl, internalExists := impl.Methods[internalMethodName]; internalExists {
								internalFunc := FunctionInfo{
									Name:     impl.StructName + "." + internalMethodName,
									FilePath: internalImpl.FilePath,
									Line:     internalImpl.StartLine,
									Calls:    []string{},
								}

								// Recursively process calls within the internal method
								for _, internalCall := range internalImpl.Calls {
									internalResolvedCall := internalCall
									if strings.HasPrefix(internalCall, "t.") {
										internalMethodCallName := strings.TrimPrefix(internalCall, "t.")
										structName := strings.Split(impl.StructName, ".")[len(strings.Split(impl.StructName, "."))-1]
										internalResolvedCall = structName + "." + internalMethodCallName
									}

									if shouldIncludeCall(internalResolvedCall) {
										internalFunc.Calls = append(internalFunc.Calls, internalResolvedCall)
									}
								}

								implementationFuncs = append(implementationFuncs, internalFunc)
							}
						}
					}
				}

				implementationFuncs = append(implementationFuncs, implFunc)
			}
		}
	}

	return implementationFuncs
}

// shouldIncludeCall determines if a call should be included in the analysis
func shouldIncludeCall(call string) bool {
	// Skip standard library calls
	standardPackages := []string{
		"fmt.", "os.", "strings.", "regexp.", "encoding.", "bytes.", "strconv.",
		"time.", "context.", "sync.", "runtime.", "sort.", "json.", "log.",
		"errors.", "filepath.", "bufio.", "io.", "math.", "unicode.", "reflect.",
	}

	for _, pkg := range standardPackages {
		if strings.HasPrefix(call, pkg) {
			return false
		}
	}

	// Keep gin and ent calls (as per user requirement)
	if strings.Contains(call, "gin") || strings.Contains(call, "ent") {
		return true
	}

	// Include other calls
	return true
}
