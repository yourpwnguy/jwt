package cli

import (
	"errors"
	"flag"

	"github.com/yourpwnguy/jwt/internal/version"
)

// Options holds the parsed command-line configuration.
//
// This struct is populated by the Parse function and contains all
// user-provided settings needed to run the application.
type Options struct {
	// Token is a single JWT string provided via the -t flag.
	// Empty if not provided.
	Token string

	// File is the path to a file containing JWT tokens (one per line),
	// provided via the -tL flag. Empty if not provided.
	File string

	// Version indicates whether the user requested version information
	// via the -v flag.
	Version bool
}

// Sentinel errors for CLI operations.
//
// These errors are used to signal specific conditions during option
// parsing and validation. They can be checked using errors.Is().
var (
	// ErrNoInput indicates that neither a token nor a file was provided.
	ErrNoInput = errors.New("neither token nor token file provided")

	// ErrShowUsage is a silent error indicating that usage information
	// was displayed and the program should exit without an error message.
	ErrShowUsage = errors.New("")
)

// Parse processes command-line arguments and returns the parsed Options.
func Parse() (Options, error) {
	var opts Options

	flag.StringVar(&opts.Token, "t", "", "")
	flag.StringVar(&opts.File, "tL", "", "")
	flag.BoolVar(&opts.Version, "v", false, "")
	flag.Usage = printUsage
	flag.Parse()

	return validate(opts)
}

// validate checks the parsed options for correctness and handles
// special cases like version display.
//
// Validation rules:
//  1. If no flags or arguments provided → show usage
//  2. If version requested → show version and exit
//  3. If neither token nor file provided → return error
//
// Returns the validated options and any error encountered.
func validate(opts Options) (Options, error) {
	// No arguments provided - show usage and exit cleanly
	if flag.NFlag() == 0 && flag.NArg() == 0 {
		printUsage()
		return opts, ErrShowUsage
	}

	// Version requested - display and exit cleanly
	if opts.Version {
		version.Print()
		return opts, ErrShowUsage
	}

	// Validate that at least one input source is provided
	if opts.Token == "" && opts.File == "" {
		PrintError("%v", ErrNoInput)
		return opts, ErrNoInput
	}

	return opts, nil
}
