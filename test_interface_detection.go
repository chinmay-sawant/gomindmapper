package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/chinmay-sawant/gomindmapper/cmd/analyzer"
)

func main() {
	projectPath := "."

	fmt.Println("Testing interface implementation detection...")

	// Find interface implementations
	implementations, err := analyzer.FindInterfaceImplementations(projectPath)
	if err != nil {
		log.Fatalf("Error finding interface implementations: %v", err)
	}

	fmt.Printf("Found %d interface implementations:\n", len(implementations))

	for interfaceName, impls := range implementations {
		fmt.Printf("\nInterface: %s\n", interfaceName)
		for _, impl := range impls {
			fmt.Printf("  Implemented by: %s\n", impl.StructName)
			fmt.Printf("  Package: %s\n", impl.PackageName)
			fmt.Printf("  File: %s\n", impl.FilePath)
			fmt.Printf("  Methods:\n")
			for methodName, methodImpl := range impl.Methods {
				fmt.Printf("    %s (lines %d-%d)\n", methodName, methodImpl.StartLine, methodImpl.EndLine)
				if len(methodImpl.Calls) > 0 {
					fmt.Printf("      Calls: %v\n", methodImpl.Calls)
				}
			}
		}
	}

	// Test GetImplementationCalls function
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("Testing GetImplementationCalls...")

	testCalls := []string{
		"svc.Filler.Fill",
		"FormDatastore.GetFormId",
		"service.DataStore.Save",
	}

	for _, call := range testCalls {
		fmt.Printf("\nTesting call: %s\n", call)
		implFuncs := analyzer.GetImplementationCalls(call, implementations)
		if len(implFuncs) > 0 {
			fmt.Printf("  Found %d implementation functions:\n", len(implFuncs))
			for _, fn := range implFuncs {
				fmt.Printf("    %s at %s:%d\n", fn.Name, fn.FilePath, fn.Line)
				if len(fn.Calls) > 0 {
					fmt.Printf("      Internal calls: %v\n", fn.Calls)
				}
			}
		} else {
			fmt.Printf("  No implementation functions found\n")
		}
	}
}
