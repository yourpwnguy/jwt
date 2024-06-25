package runner

import (
	"flag"
	"fmt"
	"os"

	"github.com/iaakanshff/gostyle"
)

// tokenoptions struct holds the options used in the program
type Options struct {
	Token   string // JWT token string
	File    string // File containing JWT tokens
	Version bool   // Representing current version
}

// Some colored messages ( mainly prefix )
var (
	g       = gostyle.New()
	Succfix = "[" + g.Blue("INFO") + "]"
	Errfix  = "[" + g.Red("ERR") + "]"
)

func ParseOptions() (Options, error) {
	var options Options

	// Define options
	flag.StringVar(&options.Token, "t", "", "")
	flag.StringVar(&options.File, "tL", "", "")
	flag.BoolVar(&options.Version, "v", false, "")

	// Customize usage message
	flag.Usage = func() {
		h := "\nUsage: jwt [options]\n\n"
		h += "Options: [flag] [argument] [Description]\n\n"
		h += "INPUT:\n"
		h += "  -t string\tJWT token string to decode\n"
		h += "  -tL FILE\tFile containing list of JWT tokens\n\n"
		h += "DEBUG:\n"
		h += "  -v none\tCheck current version\n"
		fmt.Fprint(flag.CommandLine.Output(), h)
	}

	flag.Parse()

	// Checking if no options are provided
	if flag.NFlag() == 0 && flag.NArg() == 0 {
		flag.Usage()
		return options, fmt.Errorf("")
	}

	// Printing the current version if set
	if options.Version {
		CheckVersion()
		os.Exit(0)
	}
	// Checking if neither token nor file is provided
	if options.Token == "" && options.File == "" {
		fmt.Println(Errfix, "Neither token nor token file is provided")
		return options, fmt.Errorf("")
	}

	return options, nil
}
