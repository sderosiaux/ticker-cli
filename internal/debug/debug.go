// Package debug provides conditional debug logging for API calls.
package debug

import (
	"fmt"
	"os"
	"time"
)

var (
	// Enabled controls whether debug output is emitted.
	Enabled bool
	// ColorEnabled controls whether debug output uses ANSI colors.
	ColorEnabled bool
)

// Logf writes a debug message to stderr when Enabled is true.
func Logf(format string, args ...any) {
	if !Enabled {
		return
	}

	if ColorEnabled {
		fmt.Fprintf(os.Stderr, "\033[90m[DEBUG] "+format+"\033[0m\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Timer returns a function that, when called, logs the elapsed time.
func Timer(label string) func() {
	if !Enabled {
		return func() {}
	}

	start := time.Now()

	return func() {
		ms := time.Since(start).Milliseconds()
		if ColorEnabled {
			fmt.Fprintf(os.Stderr, "\033[90m[API] %s %dms\033[0m\n", label, ms)
		} else {
			fmt.Fprintf(os.Stderr, "[API] %s %dms\n", label, ms)
		}
	}
}
