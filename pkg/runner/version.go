package runner

import (
	"fmt"
	"os"
)

func CheckVersion() {
	fmt.Fprintln(os.Stderr, "Current jwt version:", g.BrGreen("v1.0"))
}