package output_test

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sderosiaux/ticker-cli/internal/model"
	"github.com/sderosiaux/ticker-cli/internal/output"
)

var testQuotes = []model.Quote{
	{
		Symbol:        "AAPL",
		Name:          "Apple Inc.",
		Price:         178.52,
		Change:        1.23,
		ChangePercent: 0.69,
		Currency:      "USD",
		MarketState:   "REGULAR",
	},
	{
		Symbol:        "BTC-USD",
		Name:          "Bitcoin USD",
		Price:         84231.00,
		Change:        -520.00,
		ChangePercent: -0.61,
		Currency:      "USD",
		MarketState:   "REGULAR",
	},
}

var testHistory = []model.HistoryResult{
	{
		Symbol:   "AAPL",
		Name:     "Apple Inc.",
		Currency: "USD",
		Points: []model.HistoryPoint{
			{Date: "2026-03-18", Open: 176.50, High: 178.00, Low: 176.00, Close: 177.00, Volume: 50000000},
			{Date: "2026-03-19", Open: 177.00, High: 179.00, Low: 176.50, Close: 178.52, Volume: 48000000},
		},
	},
}

var testChanges = []model.ChangeResult{
	{
		Symbol:        "AAPL",
		Name:          "Apple Inc.",
		Price:         178.52,
		Currency:      "USD",
		PeriodStart:   170.00,
		PeriodEnd:     178.52,
		Change:        8.52,
		ChangePercent: 5.01,
		Period:        "1mo",
	},
}

func TestJSON_Quotes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testQuotes, "json", false)
	if err != nil {
		t.Fatal(err)
	}

	var got []model.Quote

	err = json.Unmarshal(buf.Bytes(), &got)
	if err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if len(got) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(got))
	}

	if got[0].Symbol != "AAPL" {
		t.Errorf("expected AAPL, got %s", got[0].Symbol)
	}
}

func TestJSON_History(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testHistory, "json", false)
	if err != nil {
		t.Fatal(err)
	}

	var got []model.HistoryResult

	err = json.Unmarshal(buf.Bytes(), &got)
	if err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if len(got[0].Points) != 2 {
		t.Fatalf("expected 2 points, got %d", len(got[0].Points))
	}
}

func TestJSON_Changes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testChanges, "json", false)
	if err != nil {
		t.Fatal(err)
	}

	var got []model.ChangeResult

	err = json.Unmarshal(buf.Bytes(), &got)
	if err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if got[0].Period != "1mo" {
		t.Errorf("expected 1mo, got %s", got[0].Period)
	}
}

func TestCompact_Quotes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testQuotes, "json", true)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d: %s", len(lines), buf.String())
	}

	for i, line := range lines {
		var obj map[string]any

		err = json.Unmarshal([]byte(line), &obj)
		if err != nil {
			t.Fatalf("line %d invalid JSON: %v\nline: %s", i, err, line)
		}
		// compact should have only essential fields
		if _, ok := obj["symbol"]; !ok {
			t.Errorf("line %d missing symbol", i)
		}

		if _, ok := obj["exchange"]; ok {
			t.Errorf("line %d should not have exchange in compact mode", i)
		}
	}
}

func TestCompact_Changes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testChanges, "json", true)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(lines))
	}

	var obj map[string]any

	err = json.Unmarshal([]byte(lines[0]), &obj)
	if err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := obj["period"]; !ok {
		t.Error("missing period field")
	}
}

func TestCSV_Quotes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testQuotes, "csv", false)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// header + 2 data rows
	if len(records) != 3 {
		t.Fatalf("expected 3 rows (header+2), got %d", len(records))
	}

	if records[0][0] != "symbol" {
		t.Errorf("expected header 'symbol', got %s", records[0][0])
	}

	if records[1][0] != "AAPL" {
		t.Errorf("expected AAPL, got %s", records[1][0])
	}
}

func TestCSV_History(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testHistory, "csv", false)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// header + 2 points
	if len(records) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(records))
	}

	if records[0][0] != "symbol" {
		t.Errorf("expected header 'symbol', got %s", records[0][0])
	}
}

func TestCSV_Changes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testChanges, "csv", false)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}
}

func TestTable_Quotes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testQuotes, "table", false)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "AAPL") {
		t.Error("table output missing AAPL")
	}

	if !strings.Contains(out, "BTC-USD") {
		t.Error("table output missing BTC-USD")
	}

	if !strings.Contains(out, "178.52") {
		t.Error("table output missing price")
	}
}

func TestTable_History(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testHistory, "table", false)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "AAPL") {
		t.Error("table output missing AAPL")
	}

	if !strings.Contains(out, "2026-03-18") {
		t.Error("table output missing date")
	}
}

func TestTable_Changes(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testChanges, "table", false)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "AAPL") {
		t.Error("table output missing AAPL")
	}

	if !strings.Contains(out, "1mo") {
		t.Error("table output missing period")
	}
}

var testAllPeriods = []model.AllPeriodsResult{
	{
		Symbol:   "AAPL",
		Name:     "Apple Inc.",
		Price:    178.52,
		Currency: "USD",
		Weekly:   &model.PeriodChange{Change: 1.50, ChangePercent: 0.85},
		YTD:      &model.PeriodChange{Change: -12.30, ChangePercent: -6.44},
	},
}

func TestJSON_AllPeriods(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testAllPeriods, "json", false)
	if err != nil {
		t.Fatal(err)
	}

	var got []model.AllPeriodsResult

	err = json.Unmarshal(buf.Bytes(), &got)
	if err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}

	if got[0].Weekly == nil || got[0].YTD == nil {
		t.Error("expected Weekly and YTD to be set")
	}
}

func TestNDJSON_AllPeriods(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testAllPeriods, "ndjson", false)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 NDJSON line, got %d", len(lines))
	}

	var obj map[string]any

	err = json.Unmarshal([]byte(lines[0]), &obj)
	if err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if _, ok := obj["weekly"]; !ok {
		t.Error("missing weekly field")
	}

	if _, ok := obj["ytd"]; !ok {
		t.Error("missing ytd field")
	}
}

func TestNDJSON_Format(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testQuotes, "ndjson", false)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d", len(lines))
	}
}

func TestCSV_AllPeriods(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testAllPeriods, "csv", false)
	if err != nil {
		t.Fatal(err)
	}

	r := csv.NewReader(strings.NewReader(buf.String()))

	records, err := r.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	// header + 1 data row
	if len(records) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(records))
	}

	if records[0][4] != "weekly_change" {
		t.Errorf("expected weekly_change header, got %s", records[0][4])
	}
}

func TestTable_AllPeriods(t *testing.T) {
	var buf bytes.Buffer

	err := output.Write(&buf, testAllPeriods, "table", false)
	if err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "AAPL") {
		t.Error("table output missing AAPL")
	}
}

func TestUnsupportedFormat(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, testQuotes, "xml", false)

	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestUnsupportedData(t *testing.T) {
	var buf bytes.Buffer
	err := output.Write(&buf, "not a slice", "json", false)

	if err == nil {
		t.Error("expected error for unsupported data type")
	}
}
