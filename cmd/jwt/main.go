package main

import (
	"fmt"

	"github.com/yourpwnguy/jwt/pkg/runner"
)

func main() {

	// Parsing the command line arguments
	options, err := runner.ParseOptions()
	if err != nil {
		return
	}

	// Storing the tokens provided in a slice
	var tokens []string
	if options.Token != "" {
		tokens = append(tokens, options.Token)
	}

	// If file is given fetch tokens from it
	if options.File != "" {
		fTok, err := runner.ParseFile(options.File)
		if err != nil {
			fmt.Println(runner.Errfix, err)
			return
		}
		tokens = append(tokens, fTok...)
	}

	// Just for a better output
	fmt.Println()

	// Parsing and printing token info
	for i, tokn := range tokens {

		// If the jwt token is b64 encoded, we decode it 
		b64Tokn, err := runner.DecodeB64Tokn(tokn)
		if err == nil {

			// If successfully decoded set the original token to the b64 decoded one
			tokn = b64Tokn
		}

		// Obtain the json string info of the token
		tokenInfoJSON, err := runner.ExtractTokenInfo(tokn)
		if err != nil {
			fmt.Printf("%v Token %d: %v\n", runner.Errfix, i+1, err)
			continue
		}
		fmt.Printf("Token %d:\n%s\n", i+1, tokenInfoJSON)
	}
}
