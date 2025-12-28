package reader

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ReadLines reads all non-empty lines from a file and returns them as a slice.
//
// The function handles:
//   - Opening and properly closing the file
//   - Trimming leading/trailing whitespace from each line
//   - Filtering out empty lines (after trimming)
//   - Detecting and reporting read errors
//
// Parameters:
//   - path: The filesystem path to the file to read
//
// Returns:
//   - A slice of non-empty, trimmed lines from the file
//   - An error if the file cannot be opened or read
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer file.Close()

	return scanLines(file)
}

// scanLines reads lines from an opened file using a buffered scanner.
//
// This is an internal function separated from ReadLines for better
// testability and separation of concerns. It handles:
//   - Buffered reading
//   - Line-by-line processing
//   - Whitespace trimming
//   - Empty line filtering
//   - Scanner error detection
//
// Parameters:
//   - file: An opened *os.File to read from
//
// Returns:
//   - A slice of processed lines
//   - An error if the scanner encounters a read error
func scanLines(file *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines (including lines that were only whitespace)
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	// Check for scanner errors (e.g., line too long, I/O error)
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return lines, nil
}
