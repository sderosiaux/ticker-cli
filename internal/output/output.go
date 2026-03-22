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
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Write dispatches data to the appropriate formatter based on format.
// Supported formats: "json", "csv", "table".
// When compact is true and format is "json", NDJSON is emitted.
func Write(w io.Writer, data interface{}, format string, compact bool) error {
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

func toQuotes(data interface{}) ([]model.Quote, bool) {
	q, ok := data.([]model.Quote)
	return q, ok
}

func toHistory(data interface{}) ([]model.HistoryResult, bool) {
	h, ok := data.([]model.HistoryResult)
	return h, ok
}

func toChanges(data interface{}) ([]model.ChangeResult, bool) {
	c, ok := data.([]model.ChangeResult)
	return c, ok
}
