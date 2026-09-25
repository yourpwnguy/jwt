// Package output is the UI layer: turning decoded tokens into text.
//
// Rule: core logic never prints. jwt.Parse, audit.Run, and verify.Verify
// return data; only this package (and cli for errors) touches stdout.
// That keeps formatting decisions — colors, tables, quiet mode — from
// leaking into domain code and keeps benchmarks honest.
package output

import "time"

// Format selects machine vs. human rendering.
//
//   - FormatPretty: indented JSON for eyes (default). May include a
//     human time line; NOT guaranteed jq-parseable.
//   - FormatJSON: indented JSON, no colors, no extras. Pipe to jq.
//   - FormatCompact: single-line JSON per token. Best for logs.
//   - FormatTable: one row per token (batch-friendly).
type Format string

const (
	FormatPretty  Format = "pretty"
	FormatJSON    Format = "json"
	FormatCompact Format = "compact"
	FormatTable   Format = "table"
)

// Options controls rendering. Zero value is sensible: pretty, colors
// when TTY, "Token N:" headers when batch.
type Options struct {
	// Format selects pretty/json/compact/table.
	Format Format
	// HeaderOnly prints just the header object.
	HeaderOnly bool
	// PayloadOnly prints just the payload object.
	PayloadOnly bool
	// Claim prints a single dot-path claim (e.g. "sub", "user.role").
	// It overrides HeaderOnly/PayloadOnly/Format.
	Claim string
	// Quiet suppresses "Token N:" headers in batch output.
	Quiet bool
	// NoColor disables styling even on a TTY.
	NoColor bool
	// Now anchors time summaries; zero means time.Now().
	Now time.Time
	// FailOn names the audit gate threshold (e.g. "high") for the
	// human header. Empty omits it. Machine JSON is unaffected.
	FailOn string
}

// Normalize fills defaults so callers don't have to.
// Unknown formats fall back to pretty rather than erroring —
// a decoder should still show you the token.
func (o *Options) Normalize() {
	switch o.Format {
	case FormatJSON, FormatCompact, FormatTable:
	default:
		o.Format = FormatPretty
	}
	if o.Now.IsZero() {
		o.Now = time.Now()
	}
}
