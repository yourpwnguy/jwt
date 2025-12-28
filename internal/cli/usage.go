package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/yourpwnguy/gostyle"
)

// style is the shared styling instance for consistent colored output.
var style = gostyle.New()

// PrintError prints a formatted error message to stderr with a red [ERR] prefix.
//
// The function follows Printf semantics, accepting a format string and
// variadic arguments. A newline is automatically appended.
//
// Example:
//
//	cli.PrintError("failed to parse token: %v", err)
//	// Output: [ERR] failed to parse token: invalid format
func PrintError(format string, args ...any) {
	prefix := "[" + style.Red("ERR") + "]"
	fmt.Fprintf(os.Stderr, prefix+" "+format+"\n", args...)
}

// PrintInfo prints a formatted informational message to stderr with
// a blue [INFO] prefix.
//
// The function follows Printf semantics, accepting a format string and
// variadic arguments. A newline is automatically appended.
//
// Example:
//
//	cli.PrintInfo("processing %d tokens", count)
//	// Output: [INFO] processing 5 tokens
func PrintInfo(format string, args ...any) {
	prefix := "[" + style.Blue("INFO") + "]"
	fmt.Fprintf(os.Stderr, prefix+" "+format+"\n", args...)
}

// printUsage displays the help message for the jwt command.
//
// This function is assigned to flag.Usage to provide a custom,
// formatted help message when -h/--help is used or when invalid
// arguments are provided.
//
// The output is written to flag.CommandLine.Output() to respect
// any custom output configuration.
func printUsage() {
	usage := `
Usage: jwt [options]

Options: [flag] [argument] [Description]

INPUT:
  -t  <token>    JWT token string to decode
  -tL <file>     File containing list of JWT tokens

DEBUG:
  -v             Display current version
`
	fmt.Fprint(flag.CommandLine.Output(), usage)
}
