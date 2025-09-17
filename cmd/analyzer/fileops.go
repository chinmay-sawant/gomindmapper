package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
)

func CreateJsonFile(functions []FunctionInfo) {
	data, err := json.MarshalIndent(functions, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	err = os.WriteFile("functions.json", data, 0644)
	if err != nil {
		fmt.Println(err)
	}
}
