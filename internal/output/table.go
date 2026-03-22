package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/sderosiaux/ticker-cli/internal/model"
)

const (
	colorReset = "\033[0m"
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	maxNameLen = 20
)

func colorize(val float64, s string, useColor bool) string {
	if !useColor {
		return s
	}

	if val > 0 {
		return colorGreen + s + colorReset
	}

	if val < 0 {
		return colorRed + s + colorReset
	}

	return s
}

func signPrefix(v float64) string {
	if v > 0 {
		return "+"
	}

	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen-1] + "."
}

func writeTable(w io.Writer, data any) error {
	useColor := IsTTY()

	return writeTableWithColor(w, data, useColor)
}

func writeTableWithColor(w io.Writer, data any, useColor bool) error {
	if quotes, ok := toQuotes(data); ok {
		return tableQuotes(w, quotes, useColor)
	}

	if history, ok := toHistory(data); ok {
		return tableHistory(w, history)
	}

	if changes, ok := toChanges(data); ok {
		return tableChanges(w, changes, useColor)
	}

	return fmt.Errorf("%T: %w for table", data, ErrUnsupportedDataType)
}

func tableQuotes(w io.Writer, quotes []model.Quote, useColor bool) error {
	for _, q := range quotes {
		chg := fmt.Sprintf("%s%.2f", signPrefix(q.Change), q.Change)
		pct := fmt.Sprintf("%s%.2f%%", signPrefix(q.ChangePercent), q.ChangePercent)
		chg = colorize(q.Change, chg, useColor)
		pct = colorize(q.ChangePercent, pct, useColor)

		name := truncate(q.Name, maxNameLen)
		_, _ = fmt.Fprintf(w, "%-10s %-*s %12s  %8s  %8s  %s\n",
			q.Symbol,
			maxNameLen, name,
			formatPrice(q.Price, q.Currency),
			chg,
			pct,
			q.MarketState,
		)
	}

	return nil
}

func tableHistory(w io.Writer, results []model.HistoryResult) error {
	for _, r := range results {
		_, _ = fmt.Fprintf(w, "%s (%s)\n", r.Symbol, r.Name)
		_, _ = fmt.Fprintf(w, "%-12s %10s %10s %10s %10s %12s\n",
			"Date", "Open", "High", "Low", "Close", "Volume")
		_, _ = fmt.Fprintln(w, strings.Repeat("-", 68))
		for _, p := range r.Points {
			_, _ = fmt.Fprintf(w, "%-12s %10.2f %10.2f %10.2f %10.2f %12d\n",
				p.Date, p.Open, p.High, p.Low, p.Close, p.Volume)
		}
		_, _ = fmt.Fprintln(w)
	}

	return nil
}

func tableChanges(w io.Writer, changes []model.ChangeResult, useColor bool) error {
	for _, c := range changes {
		chg := fmt.Sprintf("%s%.2f", signPrefix(c.Change), c.Change)
		pct := fmt.Sprintf("%s%.2f%%", signPrefix(c.ChangePercent), c.ChangePercent)
		chg = colorize(c.Change, chg, useColor)
		pct = colorize(c.ChangePercent, pct, useColor)

		name := truncate(c.Name, maxNameLen)
		_, _ = fmt.Fprintf(w, "%-10s %-*s %12s  %8s  %8s  %s\n",
			c.Symbol,
			maxNameLen, name,
			formatPrice(c.Price, c.Currency),
			chg,
			pct,
			c.Period,
		)
	}

	return nil
}

func formatPrice(price float64, currency string) string {
	sym := "$"

	switch currency {
	case "EUR":
		sym = "\u20ac"
	case "GBP":
		sym = "\u00a3"
	case "JPY":
		sym = "\u00a5"
	}

	raw := fmt.Sprintf("%.2f", price)
	parts := strings.SplitN(raw, ".", 2)
	intPart := parts[0]
	// insert commas
	if len(intPart) > 3 {
		var b strings.Builder

		offset := len(intPart) % 3
		if offset > 0 {
			b.WriteString(intPart[:offset])
		}

		for i := offset; i < len(intPart); i += 3 {
			if b.Len() > 0 {
				b.WriteByte(',')
			}

			b.WriteString(intPart[i : i+3])
		}

		intPart = b.String()
	}

	return fmt.Sprintf("%s%s.%s", sym, intPart, parts[1])
}
