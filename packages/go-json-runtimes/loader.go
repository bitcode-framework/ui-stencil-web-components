package runtimes

import (
	"fmt"
	"os"
)

func LoadScript(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read script file: %w", err)
	}
	return string(data), nil
}
