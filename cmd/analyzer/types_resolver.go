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
func ResolveMethodCall(call string, fileInfoMap map[string]FileTypeInfo, allTypeInfo map[string]TypeInfo) string {
	parts := strings.Split(call, ".")

	// Handle different call patterns
	switch len(parts) {
	case 2:
		// Direct method call like "FormDatastore.GetFormId"
		return resolveDirectMethodCall(parts[0], parts[1], fileInfoMap, allTypeInfo)
	case 3:
		// Method call on struct field like "svc.FormDatastore.GetFormId"
		return resolveStructFieldMethodCall(parts[0], parts[1], parts[2], fileInfoMap, allTypeInfo)
	default:
		return call // Return original for other patterns
	}
}

// resolveDirectMethodCall resolves calls like "FormDatastore.GetFormId"
func resolveDirectMethodCall(typeName, methodName string, fileInfoMap map[string]FileTypeInfo, allTypeInfo map[string]TypeInfo) string {
	// Look for interface types that match the type name pattern
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
func resolveStructFieldMethodCall(varName, fieldName, methodName string, fileInfoMap map[string]FileTypeInfo, allTypeInfo map[string]TypeInfo) string {
	// Look through all struct definitions to find one with the specified field
	for _, fileInfo := range fileInfoMap {
		for _, structInfo := range fileInfo.Structs {
			if fieldType, exists := structInfo.Fields[fieldName]; exists {
				// Found a struct with this field, now resolve the field type
				resolvedType := resolveFieldType(fieldType, fileInfo.Imports, allTypeInfo)

				// Try direct matching with the resolved type
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
