package cli

import (
	"flag"
	"reflect"
	"testing"
)

// Flags work before, after, and around positionals: `jwt <token> -o json`
// must behave like `jwt -o json <token>`. The stdlib stops parsing at the
// first positional, so we pre-split argv ourselves (see args.go).

func testFlagSet() *flag.FlagSet {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.String("t", "", "")
	fs.String("o", "", "")
	fs.Bool("q", false, "")
	return fs
}

func TestPartitionArgs(t *testing.T) {
	for _, tc := range []struct {
		name         string
		args         []string
		wantFlags    []string
		wantPosition []string
	}{
		{"flags first", []string{"-o", "json", "tok"}, []string{"-o", "json"}, []string{"tok"}},
		{"flags last", []string{"tok", "-o", "json"}, []string{"-o", "json"}, []string{"tok"}},
		{"mixed", []string{"a", "-q", "b", "-o", "json", "c"}, []string{"-q", "-o", "json"}, []string{"a", "b", "c"}},
		{"eq form", []string{"tok", "-o=json"}, []string{"-o=json"}, []string{"tok"}},
		{"double dash ends flags", []string{"tok", "--", "-o"}, nil, []string{"tok", "-o"}},
		{"lone dash is positional", []string{"-", "-o", "json"}, []string{"-o", "json"}, []string{"-"}},
		{"unknown flag kept for Parse to reject", []string{"tok", "-bogus"}, []string{"-bogus"}, []string{"tok"}},
		{"missing value left for Parse", []string{"-o"}, []string{"-o"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gotFlags, gotPos := partitionArgs(testFlagSet(), tc.args)
			if !reflect.DeepEqual(gotFlags, tc.wantFlags) || !reflect.DeepEqual(gotPos, tc.wantPosition) {
				t.Errorf("got (%v, %v) want (%v, %v)", gotFlags, gotPos, tc.wantFlags, tc.wantPosition)
			}
		})
	}
}
