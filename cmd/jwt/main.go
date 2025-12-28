package main

import (
	"fmt"
	"os"

	"github.com/yourpwnguy/jwt/internal/cli"
	"github.com/yourpwnguy/jwt/internal/encoding"
	"github.com/yourpwnguy/jwt/internal/jwt"
	"github.com/yourpwnguy/jwt/internal/reader"
)

// main is the entry point of the application.
// It delegates to run() and handles any errors by printing to stderr
// and exiting with a non-zero status code.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run orchestrates the main application flow:
//  1. Parse command-line options
//  2. Collect tokens from all input sources (CLI + FILE)
//  3. Process and display each token
//
// Returns an error if option parsing or token collection fails.
// Token processing errors are handled individually and do not
// cause the function to return an error.
func run() error {
	opts, err := cli.Parse()
	if err != nil {
		return err
	}

	tokens, err := collectTokens(opts)
	if err != nil {
		return err
	}

	processTokens(tokens)
	return nil
}

// collectTokens aggregates JWT tokens from all configured input sources.
//
// It collects tokens in the following order:
//  1. Direct token string from -t flag (if provided)
//  2. Tokens from file specified by -tL flag (if provided)
//
// The function returns a combined slice of all tokens. If the file
// cannot be read, an error is returned and no tokens are processed.
func collectTokens(opts cli.Options) ([]string, error) {
	var tokens []string

	if opts.Token != "" {
		tokens = append(tokens, opts.Token)
	}

	if opts.File != "" {
		fileTokens, err := reader.ReadLines(opts.File)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, fileTokens...)
	}

	return tokens, nil
}

// processTokens iterates through all collected tokens and processes
// each one individually.
//
// A blank line is printed before output for better visual separation (maybe we don't need it)
// from any preceding command-line output. Each token is processed
// independently, so a failure in one token does not affect others.
// TODO: We can make it concurrent
func processTokens(tokens []string) {
	fmt.Println()

	var useIndex bool
	if len(tokens) < 2 {
		useIndex = false
	}

	for i, token := range tokens {
		processToken(i+1, token, useIndex)
	}
}

// processToken handles the parsing and display of a single JWT token.
//
// The function performs the following steps:
//  1. Attempts to decode the token from base64 (in case it's double-encoded)
//  2. Parses the JWT structure (header, payload, signature)
//  3. Outputs the formatted result or an error message
//
// Parameters:
//   - index: The 1-based position of the token (for display purposes)
//   - token: The raw JWT string to process
//   - useIndex: This specifies if we should use "Token idx:" format
//
// Errors during parsing are printed to stderr but do not halt execution.
func processToken(index int, token string, useIndex bool) {
	// Attempt base64 decoding if token is encoded
	if decoded, err := encoding.DecodeBase64(token); err == nil {
		token = decoded
	}

	output, err := jwt.Parse(token)
	if err != nil {
		cli.PrintError("Token %d: %v", index, err)
		return
	}

	if useIndex {
		fmt.Printf("Token %d:\n%s\n\n", index, output)
	}
	fmt.Printf("%s\n", output)
}
