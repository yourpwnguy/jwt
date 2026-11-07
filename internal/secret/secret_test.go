package secret

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// secret_test.go covers the two things that matter: file loading is
// predictable, and the PEM confusion guard cannot be bypassed with
// raw: or a file.

func TestLoadHMACLiteral(t *testing.T) {
	got, err := LoadHMAC("s3cr3t")
	if err != nil {
		t.Fatalf("LoadHMAC: %v", err)
	}
	if string(got) != "s3cr3t" {
		t.Fatalf("got %q, want s3cr3t", got)
	}
}

func TestLoadHMACRawPrefix(t *testing.T) {
	// Create a file with the same name as the literal so we can prove
	// raw: wins over the file path heuristic.
	dir := t.TempDir()
	name := filepath.Join(dir, "raw-secret")
	if err := os.WriteFile(name, []byte("file-value\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := LoadHMAC("raw:" + name)
	if err != nil {
		t.Fatalf("LoadHMAC: %v", err)
	}
	if string(got) != name {
		t.Fatalf("got %q, want literal path %q", got, name)
	}
}

func TestLoadHMACFileTrimsTextNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.txt")
	if err := os.WriteFile(path, []byte("hunter2\r\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, spec := range []string{"@" + path, path} {
		got, err := LoadHMAC(spec)
		if err != nil {
			t.Fatalf("LoadHMAC(%q): %v", spec, err)
		}
		if string(got) != "hunter2" {
			t.Fatalf("LoadHMAC(%q) = %q, want hunter2", spec, got)
		}
	}
}

func TestLoadHMACBinaryFileKeepsNewlines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.bin")
	want := []byte{0x00, 0x01, 0x0a, 0xff}
	if err := os.WriteFile(path, want, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadHMAC("@" + path)
	if err != nil {
		t.Fatalf("LoadHMAC: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("binary key changed: got %x want %x", got, want)
	}
}

func TestLoadHMACErrors(t *testing.T) {
	dir := t.TempDir()
	empty := filepath.Join(dir, "empty")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		spec string
		want error
	}{
		{"empty", "", ErrEmpty},
		{"raw empty", "raw:", ErrEmpty},
		{"empty file", "@" + empty, ErrEmpty},
		{"pem public key", "-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----", ErrPEMKey},
		{"pem private key", "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----", ErrPEMKey},
		{"certificate", "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----", ErrCertificate},
		{"raw cannot bypass pem guard", "raw:-----BEGIN PUBLIC KEY-----\nMIIB\n-----END PUBLIC KEY-----", ErrPEMKey},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LoadHMAC(tt.spec)
			if !errors.Is(err, tt.want) {
				t.Fatalf("LoadHMAC() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestLoadHMACMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope")
	_, err := LoadHMAC("@" + path)
	if err == nil {
		t.Fatal("missing @file should error")
	}
	if errors.Is(err, ErrEmpty) {
		t.Fatalf("missing file should be a read error, got ErrEmpty")
	}
}

func TestLoadHMACDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadHMAC("@" + dir)
	if err == nil {
		t.Fatal("reading a directory should error")
	}
}
