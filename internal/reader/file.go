// Package reader owns file I/O: opening, size-guarding, and line scanning.
//
// It stays deliberately dumb — no token knowledge, no Bearer stripping.
// Callers in internal/input decide what lines mean. That keeps this
// package reusable and its failure modes obvious (open vs. read vs. too big).
package reader

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Guards: a token list is small text. 5MB already holds ~1500 max-size
// tokens; beyond that the user should split the file. The per-line cap
// (256KB) covers --extract log lines while still bounding Scanner memory.
const (
	MaxFileBytes = 5 << 20
	MaxLineBytes = 256 * 1024
)

// ReadLines returns non-empty trimmed lines from path.
// Empty lines (or whitespace-only) are skipped so hand-edited token
// files with blank separators just work.
func ReadLines(path string) ([]string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	if fi.IsDir() {
		return nil, fmt.Errorf("failed to open file %q: is a directory", path)
	}
	if fi.Size() > MaxFileBytes {
		return nil, fmt.Errorf("file %q too large (%d bytes > %d)", path, fi.Size(), MaxFileBytes)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %q: %w", path, err)
	}
	defer file.Close()

	return scanLines(file)
}

// scanLines buffers with an enlarged Scanner window so long --extract
// lines don't hit the default 64KB "token too long" error.
func scanLines(file *os.File) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), MaxLineBytes)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return lines, nil
}
