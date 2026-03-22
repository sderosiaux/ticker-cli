package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sderosiaux/ticker-cli/internal/model"
)

func writeJSON(w io.Writer, data interface{}, compact bool) error {
	if compact {
		return writeCompact(w, data)
	}
	return writePrettyJSON(w, data)
}

func writePrettyJSON(w io.Writer, data interface{}) error {
	switch data.(type) {
	case []model.Quote, []model.HistoryResult, []model.ChangeResult:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	default:
		return fmt.Errorf("unsupported data type for JSON: %T", data)
	}
}

func writeCompact(w io.Writer, data interface{}) error {
	if quotes, ok := toQuotes(data); ok {
		return compactQuotes(w, quotes)
	}
	if changes, ok := toChanges(data); ok {
		return compactChanges(w, changes)
	}
	if history, ok := toHistory(data); ok {
		return compactHistory(w, history)
	}
	return fmt.Errorf("unsupported data type for compact JSON: %T", data)
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
			return err
		}
		fmt.Fprintf(w, "%s\n", b)
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
			return err
		}
		fmt.Fprintf(w, "%s\n", b)
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
			return err
		}
		fmt.Fprintf(w, "%s\n", b)
	}
	return nil
}
