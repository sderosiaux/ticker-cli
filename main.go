package main

import (
	"os"

	"github.com/sderosiaux/ticker-check/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
