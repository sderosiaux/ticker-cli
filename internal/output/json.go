package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sderosiaux/ticker-cli/internal/model"
)

func writeJSON(w io.Writer, data any, compact bool) error {
	if compact {
		return writeCompact(w, data)
	}

	return writePrettyJSON(w, data)
}

func writePrettyJSON(w io.Writer, data any) error {
	switch data.(type) {
	case []model.Quote, []model.HistoryResult, []model.ChangeResult, []model.AllPeriodsResult:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		err := enc.Encode(data)
		if err != nil {
			return fmt.Errorf("json encode: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("%T: %w for JSON", data, ErrUnsupportedDataType)
	}
}

func writeCompact(w io.Writer, data any) error {
	if quotes, ok := toQuotes(data); ok {
		return compactQuotes(w, quotes)
	}

	if changes, ok := toChanges(data); ok {
		return compactChanges(w, changes)
	}

	if history, ok := toHistory(data); ok {
		return compactHistory(w, history)
	}

	if allPeriods, ok := toAllPeriods(data); ok {
		return compactAllPeriods(w, allPeriods)
	}

	return fmt.Errorf("%T: %w for compact JSON", data, ErrUnsupportedDataType)
}

type compactQuote struct {
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Currency      string  `json:"currency"`
	MarketState   string  `json:"marketState"`
}

func compactQuotes(w io.Writer, quotes []model.Quote) error {
	for _, q := range quotes {
		b, err := json.Marshal(compactQuote{
			Symbol:        q.Symbol,
			Price:         q.Price,
			Change:        q.Change,
			ChangePercent: q.ChangePercent,
			Currency:      q.Currency,
			MarketState:   q.MarketState,
		})
		if err != nil {
			return fmt.Errorf("marshal compact quote: %w", err)
		}

		_, _ = fmt.Fprintf(w, "%s\n", b)
	}

	return nil
}

type compactChange struct {
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Period        string  `json:"period"`
}

func compactChanges(w io.Writer, changes []model.ChangeResult) error {
	for _, c := range changes {
		b, err := json.Marshal(compactChange{
			Symbol:        c.Symbol,
			Price:         c.Price,
			Change:        c.Change,
			ChangePercent: c.ChangePercent,
			Period:        c.Period,
		})
		if err != nil {
			return fmt.Errorf("marshal compact change: %w", err)
		}

		_, _ = fmt.Fprintf(w, "%s\n", b)
	}

	return nil
}

type compactAllPeriodsItem struct {
	Symbol   string              `json:"symbol"`
	Price    float64             `json:"price"`
	Currency string              `json:"currency"`
	Weekly   *model.PeriodChange `json:"weekly,omitempty"`
	YTD      *model.PeriodChange `json:"ytd,omitempty"`
}

func compactAllPeriods(w io.Writer, results []model.AllPeriodsResult) error {
	for _, r := range results {
		b, err := json.Marshal(compactAllPeriodsItem{
			Symbol:   r.Symbol,
			Price:    r.Price,
			Currency: r.Currency,
			Weekly:   r.Weekly,
			YTD:      r.YTD,
		})
		if err != nil {
			return fmt.Errorf("marshal compact all-periods: %w", err)
		}

		_, _ = fmt.Fprintf(w, "%s\n", b)
	}

	return nil
}

type compactHistoryItem struct {
	Symbol string               `json:"symbol"`
	Points []model.HistoryPoint `json:"points"`
}

func compactHistory(w io.Writer, results []model.HistoryResult) error {
	for _, r := range results {
		b, err := json.Marshal(compactHistoryItem{
			Symbol: r.Symbol,
			Points: r.Points,
		})
		if err != nil {
			return fmt.Errorf("marshal compact history: %w", err)
		}

		_, _ = fmt.Fprintf(w, "%s\n", b)
	}

	return nil
}
