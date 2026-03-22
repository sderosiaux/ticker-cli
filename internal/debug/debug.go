package debug

import (
	"fmt"
	"os"
	"time"
)

var Enabled bool

func Log(format string, args ...interface{}) {
	if !Enabled {
		return
	}
	fmt.Fprintf(os.Stderr, "\033[90m[DEBUG] "+format+"\033[0m\n", args...)
}

func Timer(label string) func() {
	if !Enabled {
		return func() {}
	}
	start := time.Now()
	return func() {
		fmt.Fprintf(os.Stderr, "\033[90m[API] %s %dms\033[0m\n", label, time.Since(start).Milliseconds())
	}
}
