package main

import (
	"flag"
	"fmt"
	"os"
	"github.com/iaakanshff/jwt/cmd"
)

// tokenFlags struct holds the flags used in the program
type tokenFlags struct {
	Token     string // JWT token string
	File      string // File containing JWT tokens
	Version   bool // Representing current version
}

func main() {
	const version string = "v1.0"
	var flags tokenFlags

	// Define flags
	flag.StringVar(&flags.Token, "t", "", "JWT token string")
	flag.StringVar(&flags.File, "f", "", "File containing JWT tokens")
	flag.BoolVar(&flags.Version, "v", false, "Check current version")

	// Customize usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nUsage: jwt [options]\n\n")
		fmt.Fprintf(os.Stderr, "Options: [flag] [argument] [Description]\n\n")
		fmt.Fprintf(os.Stderr, "  -t string\tJWT token string to decode\n")
		fmt.Fprintf(os.Stderr, "  -f FILE\tFile containing JWT tokens\n")
		fmt.Fprintf(os.Stderr, "  -v none\tCheck current version\n")
	}

	flag.Parse()

	// Checking if no flags are provided
	if flag.NFlag() == 0 && flag.NArg() == 0 {
		flag.Usage()
		fmt.Println()
		return
	}

	// Printing the current version if set
	if flags.Version {
		fmt.Printf("[\033[34m+\033[0m] Current Version: %v\n\n", version)
		return
	}
	// Checking if neither token nor file is provided
	if flags.Token == "" && flags.File == "" {
		fmt.Print("[-] No token provided\n")
		return
	}

	// Storing the tokens provided in a slice
	var tokens []string
	if flags.Token != "" {
		tokens = append(tokens, flags.Token)
	}

	if flags.File != "" {
		fTok, err := cmd.ParseFile(flags.File)
		if err != nil {
			fmt.Println(err)
			return
		}
		tokens = append(tokens, fTok...)
	}

	// Just for a better output
	fmt.Println()

	// Parsing and printing token info
	for i, tn := range tokens {
		tokenInfoJSON, err := cmd.ParseTokenInfo(tn)
		if err != nil {
			fmt.Printf("[-] Token %d: %v\n\n", i+1, err)
			continue
		}
		fmt.Printf("Token %d:\n%s\n\n", i+1, tokenInfoJSON)
	}
}
