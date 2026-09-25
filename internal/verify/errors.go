package verify

import "errors"

// Sentinel errors so the CLI can map failures to exit codes without
// string matching. Validation problems (none, bad sig) become exit 2;
// system problems (no keys, bad files, network) become exit 1.
var (
	// ErrNoKey means no --secret/--pubkey/--jwks was given.
	ErrNoKey = errors.New("no key material")
	// ErrNone means alg=none — never verifiable by policy.
	ErrNone = errors.New("alg=none tokens never verify")
	// ErrUnsupportedAlg means the alg isn't implemented.
	ErrUnsupportedAlg = errors.New("unsupported alg")
)
