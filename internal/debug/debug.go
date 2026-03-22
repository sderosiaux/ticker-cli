package debug

import (
	"fmt"
	"os"
	"time"
)

var (
	Enabled      bool
	ColorEnabled bool
)

func Log(format string, args ...interface{}) {
	if !Enabled {
		return
	}
	if ColorEnabled {
		fmt.Fprintf(os.Stderr, "\033[90m[DEBUG] "+format+"\033[0m\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

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
