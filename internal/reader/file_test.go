package reader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "toks.txt")
	content := "  token1  \n\n   \ntoken2\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	lines, err := ReadLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 || lines[0] != "token1" || lines[1] != "token2" {
		t.Errorf("lines=%v", lines)
	}
	if _, err := ReadLines(filepath.Join(dir, "missing")); err == nil {
		t.Error("expected error for missing file")
	}
	if _, err := ReadLines(dir); err == nil {
		t.Error("expected error for directory")
	}
}
