package version

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/yourpwnguy/gostyle"
)

// Number holds the current version of the application.
//
// This variable is set at build time via ldflags. When building with
// the Makefile, it automatically extracts the version from git tags:
//
//	make build  # Uses: git describe --tags --dirty --always
//
// Possible values:
//   - "dev"                    - Default when not set
//   - "v1.0.0"                 - Clean tagged release
//   - "v1.0.0-dirty"           - Tagged with uncommitted changes
//   - "v1.0.0-5-g2a3b4c5"      - 5 commits after v1.0.0 tag
//   - "v1.0.0-5-g2a3b4c5-dirty" - Above with uncommitted changes
//
// Number holds the current version of the application.
// Set at build time via ldflags, or detected from build info.
var Number = ""

// style provides colored terminal output.
var style = gostyle.New()

// init determines the version from available sources.
func init() {
	// If already set via ldflags, use that
	if Number != "" {
		return
	}

	// Try to get version from go install build info
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Number = info.Main.Version
		return
	}

	// Fallback to dev
	Number = "dev"
}

// Print outputs the current version information to stderr.
//
// The output format is:
//
//	[INFO] jwt version: v1.0.0
//
// Output goes to stderr to keep stdout clean for piping/redirection.
func Print() {
	prefix := "[" + style.Blue("INFO") + "]"
	ver := style.BrGreen(Number)
	fmt.Fprintf(os.Stderr, "%s jwt version: %s", prefix, ver)
}
