// Package output provides formatters for writing data as JSON, CSV, and table.
package output

import (
	"fmt"
	"io"
	"os"

	"github.com/sderosiaux/ticker-cli/internal/model"
	"golang.org/x/term"
)

// IsTTY reports whether stdout is a terminal.
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd())) //nolint:gosec // fd fits in int
}

// Write dispatches data to the appropriate formatter based on format.
// Supported formats: "json", "csv", "table".
// When compact is true and format is "json", NDJSON is emitted.
func Write(w io.Writer, data any, format string, compact bool) error {
	if compact {
		format = "json"
	}

	switch format {
	case "json":
		return writeJSON(w, data, compact)
	case "csv":
		return writeCSV(w, data)
	case "table":
		return writeTable(w, data)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

func toQuotes(data any) ([]model.Quote, bool) {
	q, ok := data.([]model.Quote)

	return q, ok
}

func toHistory(data any) ([]model.HistoryResult, bool) {
	h, ok := data.([]model.HistoryResult)

	return h, ok
}

func toChanges(data any) ([]model.ChangeResult, bool) {
	c, ok := data.([]model.ChangeResult)

	return c, ok
}
