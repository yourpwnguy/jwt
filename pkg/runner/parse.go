package runner

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseFile reads tokens from a file and returns them as a slice
func ParseFile(fn string) ([]string, error) {
	f, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("file provided not found")
	}
	defer f.Close()

	var tokens []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		tokens = append(tokens, line)
	}
	return tokens, nil
}
