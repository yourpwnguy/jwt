package cli

import (
	"flag"
	"strings"
)

// args.go lets flags appear before OR after positional tokens.
//
// Go's flag package stops parsing at the first positional, so
// `jwt <token> -o json` silently treated `-o` and `json` as two more
// tokens. That is the worst kind of CLI bug: no error about flags,
// just confusing "invalid token" noise. Since our docs and muscle
// memory put the token first, we pre-split argv ourselves: flags
// (in original order) go to FlagSet.Parse, everything else is a token.

// partitionArgs splits args into flag args and positional args,
// preserving order within each group. Rules:
//   - "-" is always positional (explicit stdin marker).
//   - "--" ends flag parsing; the rest is positional.
//   - "-name"/"--name" with a registered bool flag takes no value;
//     any other registered flag consumes the next arg unless =form.
//   - Unknown dash args stay in the flag group so Parse reports
//     "flag provided but not defined" instead of a token error.
func partitionArgs(fs *flag.FlagSet, args []string) (flags, positional []string) {
	i := 0
	noMoreFlags := false
	for i < len(args) {
		a := args[i]
		if noMoreFlags || a == "-" || len(a) < 2 || a[0] != '-' {
			positional = append(positional, a)
			i++
			continue
		}
		if a == "--" {
			noMoreFlags = true
			i++
			continue
		}
		name := strings.TrimPrefix(strings.TrimPrefix(a, "--"), "-")
		if eq := strings.IndexByte(name, '='); eq >= 0 {
			name = name[:eq]
		}
		hasValue := strings.Contains(a, "=")
		flags = append(flags, a)
		i++
		if hasValue {
			continue
		}
		// Value-taking flags consume the next arg. Bool flags never do.
		// Unknown flags consume nothing here; Parse will reject them.
		if f := fs.Lookup(name); f != nil && !isBoolFlag(f) {
			if i < len(args) {
				flags = append(flags, args[i])
				i++
			}
		}
	}
	return flags, positional
}

// isBoolFlag reports whether a registered flag takes no value.
// It mirrors the stdlib's own check via the IsBoolFlag interface.
func isBoolFlag(f *flag.Flag) bool {
	bf, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && bf.IsBoolFlag()
}
