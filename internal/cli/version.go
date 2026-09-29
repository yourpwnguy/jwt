package cli

import (
	"fmt"
	"io"

	"github.com/yourpwnguy/jwt/v2/internal/version"
)

// version.go prints the build version to the given writer (stderr).
// Keeping it here instead of calling version.Print directly makes
// output capturable in tests and keeps all CLI printing injectable.

// printVersion writes "[INFO] jwt version: X" to w.
func printVersion(w io.Writer) {
	// version.Number is set via ldflags; "dev" fallback otherwise.
	fmt.Fprintf(w, "[INFO] jwt version: %s\n", version.Number)
}
