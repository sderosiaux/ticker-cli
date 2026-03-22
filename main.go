package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/sderosiaux/ticker-check/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			// ExitError messages are already printed to stderr by run()
			os.Exit(exitErr.Code)
		}
		// Other errors (validation, cobra) — print to stderr
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
