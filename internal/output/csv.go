package output

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

func writeCSV(w io.Writer, data any) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	if quotes, ok := toQuotes(data); ok {
		_ = cw.Write([]string{"symbol", "name", "price", "change", "change_pct", "currency", "market_state"})
		for _, q := range quotes {
			_ = cw.Write([]string{
				q.Symbol,
				q.Name,
				fmt.Sprintf("%.2f", q.Price),
				fmt.Sprintf("%.2f", q.Change),
				fmt.Sprintf("%.2f", q.ChangePercent),
				q.Currency,
				q.MarketState,
			})
		}

		err := cw.Error()
		if err != nil {
			return fmt.Errorf("csv write: %w", err)
		}

		return nil
	}

	if history, ok := toHistory(data); ok {
		_ = cw.Write([]string{"symbol", "date", "open", "high", "low", "close", "volume"})
		for _, h := range history {
			for _, p := range h.Points {
				_ = cw.Write([]string{
					h.Symbol,
					p.Date,
					fmt.Sprintf("%.2f", p.Open),
					fmt.Sprintf("%.2f", p.High),
					fmt.Sprintf("%.2f", p.Low),
					fmt.Sprintf("%.2f", p.Close),
					strconv.FormatInt(p.Volume, 10),
				})
			}
		}

		err := cw.Error()
		if err != nil {
			return fmt.Errorf("csv write: %w", err)
		}

		return nil
	}

	if changes, ok := toChanges(data); ok {
		_ = cw.Write([]string{"symbol", "name", "price", "period_start", "period_end", "change", "change_pct", "period"})
		for _, c := range changes {
			_ = cw.Write([]string{
				c.Symbol,
				c.Name,
				fmt.Sprintf("%.2f", c.Price),
				fmt.Sprintf("%.2f", c.PeriodStart),
				fmt.Sprintf("%.2f", c.PeriodEnd),
				fmt.Sprintf("%.2f", c.Change),
				fmt.Sprintf("%.2f", c.ChangePercent),
				c.Period,
			})
		}

		err := cw.Error()
		if err != nil {
			return fmt.Errorf("csv write: %w", err)
		}

		return nil
	}

	return fmt.Errorf("%T: %w for CSV", data, ErrUnsupportedDataType)
}
