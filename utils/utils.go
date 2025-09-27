package utils

import "fmt"

func ParseInt(s string, def int) int {
	if s == "" {
		return def
	}
	var v int
	_, err := fmt.Sscanf(s, "%d", &v)
	if err != nil {
		return def
	}
	return v
}
