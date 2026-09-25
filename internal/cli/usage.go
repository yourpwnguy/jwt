package cli

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/yourpwnguy/gostyle"
)

// usage.go owns help text and stderr helpers.
// Writers are injected so tests can capture output — never os.Stderr directly.

var style = gostyle.New()

// errPrint writes "[ERR] ..." to w. Colors apply only on terminals so
// piped errors stay greppable. Stdout stays clean for `| jq`.
func errPrint(w io.Writer, format string, args ...any) {
	prefix := "[ERR]"
	if isTerminalWriter(w) {
		prefix = "[" + style.Red("ERR") + "]"
	}
	fmt.Fprintf(w, prefix+" "+format+"\n", args...)
}

// isTerminalWriter reports whether w is a live terminal. Buffers and
// pipes get plain text; humans get color. NO_COLOR always wins.
func isTerminalWriter(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// printUsage writes the top-level help.
func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: jwt [command] [options] [token]

Decode, audit, and verify JWTs offline. Secrets never leave your machine.

Commands:
  decode   Decode tokens (default when no command given)
  audit    Passive security checks (no network, no secrets)
  verify   Check signature + exp/nbf/aud/iss
  version  Print version

Decode examples:
  jwt -t <token>                  decode one token
  jwt <token>                     same, positional
  echo $TOKEN | jwt               pipe
  curl -s api | jq -r .token | jwt
  jwt -tL tokens.txt              batch file (one per line)
  cat app.log | jwt --extract     find tokens in logs
  jwt $TOKEN -o json | jq .payload
  jwt $TOKEN --claim sub          print one claim for scripts
  jwt -tL list -o table           batch overview

Audit examples:
  jwt audit <token>               human triage
  jwt audit -tL list -o json | jq
  jwt audit <token> --fail-on high   exit 2 when high+ breaches

Verify examples:
  jwt verify <token> --secret s3cr3t
  jwt verify <token> --pubkey @key.pem --aud api --iss auth.example.com
  jwt verify <token> --jwks https://auth/.well-known/jwks.json

Flags (decode):
  -t <token>      single token (compat)
  -tL <file>      file with one token per line
  --extract       scan text for embedded JWTs
  -o pretty|json|compact|table
  --H             header only
  --P             payload only
  --claim a.b.c   single claim (dot path)
  -q, --quiet     just data, no Token headers or status line
  --no-color      plain output
  --verify-exp    exit 2 when expired / not-yet-valid
  --leeway 30s    clock skew tolerance

Run "jwt <command> -h" for command flags.
`)
}

// parseLeeway parses durations like "30s", "5m". Empty means zero.
func parseLeeway(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid --leeway %q (try 30s, 5m): %w", s, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("invalid --leeway %q: must be >= 0", s)
	}
	return d, nil
}

// parseFormat validates -o/--format. Unknown values error instead of
// silently falling back to pretty — a misspelled format that ignores
// your scripting choice is worse than an error.
func parseFormat(s string) (string, error) {
	switch s {
	case "pretty", "json", "compact", "table", "":
		if s == "" {
			return "pretty", nil
		}
		return s, nil
	default:
		return "", fmt.Errorf("invalid format %q (want pretty|json|compact|table)", s)
	}
}
