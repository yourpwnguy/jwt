package runner

import (
	"fmt"
	"os"
)

// Function for checking the version info
func CheckVersion() {
	fmt.Fprintln(os.Stderr, Succfix, "Current jwt version:", g.BrGreen("v1.0"))
}
