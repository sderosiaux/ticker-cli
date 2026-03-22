package yahoo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// newTestServer creates a single httptest server that handles
// session endpoints (/ for cookie, /v1/test/getcrumb) and
// the quote endpoint (/v7/finance/quote).
func newTestServer(quoteHandler http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/v1/test/getcrumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test-crumb"))
	})
	mux.HandleFunc("/v7/finance/quote", quoteHandler)
	return httptest.NewServer(mux)
}

func quoteJSON(quotes ...map[string]interface{}) []byte {
	resp := map[string]interface{}{
		"quoteResponse": map[string]interface{}{
			"result": quotes,
			"error":  nil,
		},
	}
	b, _ := json.Marshal(resp)
	return b
}

func ff(val float64, fmt string) map[string]interface{} {
	return map[string]interface{}{"raw": val, "fmt": fmt}
}

func TestGetQuotes(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(quoteJSON(map[string]interface{}{
			"symbol":                     "AAPL",
			"shortName":                  "Apple Inc.",
			"regularMarketPrice":         ff(178.52, "178.52"),
			"regularMarketChange":        ff(2.31, "2.31"),
			"regularMarketChangePercent": ff(1.31, "1.31%"),
			"currency":                   "USD",
			"marketState":                "REGULAR",
			"fullExchangeName":           "NasdaqGS",
			"regularMarketOpen":          ff(176.0, "176.00"),
			"regularMarketDayHigh":       ff(179.0, "179.00"),
			"regularMarketDayLow":        ff(175.5, "175.50"),
			"regularMarketPreviousClose": ff(176.21, "176.21"),
			"regularMarketVolume":        ff(52000000, "52,000,000"),
			"marketCap":                  ff(2.8e12, "2.8T"),
			"fiftyTwoWeekHigh":           ff(199.62, "199.62"),
			"fiftyTwoWeekLow":            ff(124.17, "124.17"),
		}))
	})
	defer srv.Close()

	c := NewClient(srv.URL, srv.URL, "")
	if err := c.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	quotes, err := c.GetQuotes([]string{"AAPL"})
	if err != nil {
		t.Fatalf("GetQuotes failed: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}

	q := quotes[0]
	if q.Symbol != "AAPL" {
		t.Errorf("symbol: got %q, want AAPL", q.Symbol)
	}
	if q.Price != 178.52 {
		t.Errorf("price: got %f, want 178.52", q.Price)
	}
	if q.Change != 2.31 {
		t.Errorf("change: got %f, want 2.31", q.Change)
	}
	if q.ChangePercent != 1.31 {
		t.Errorf("changePercent: got %f, want 1.31", q.ChangePercent)
	}
	if q.MarketState != "REGULAR" {
		t.Errorf("marketState: got %q, want REGULAR", q.MarketState)
	}
	if q.Currency != "USD" {
		t.Errorf("currency: got %q, want USD", q.Currency)
	}
	if q.Exchange != "NasdaqGS" {
		t.Errorf("exchange: got %q, want NasdaqGS", q.Exchange)
	}
	if q.Open != 176.0 {
		t.Errorf("open: got %f, want 176.0", q.Open)
	}
	if q.Week52High != 199.62 {
		t.Errorf("week52High: got %f, want 199.62", q.Week52High)
	}
}

func TestGetQuotes_MultipleSymbols(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(quoteJSON(
			map[string]interface{}{
				"symbol":                     "AAPL",
				"shortName":                  "Apple Inc.",
				"regularMarketPrice":         ff(178.52, "178.52"),
				"regularMarketChange":        ff(2.31, "2.31"),
				"regularMarketChangePercent": ff(1.31, "1.31%"),
				"currency":                   "USD",
				"marketState":                "REGULAR",
				"fullExchangeName":           "NasdaqGS",
			},
			map[string]interface{}{
				"symbol":                     "BTC-USD",
				"shortName":                  "Bitcoin USD",
				"regularMarketPrice":         ff(65432.10, "65,432.10"),
				"regularMarketChange":        ff(-512.30, "-512.30"),
				"regularMarketChangePercent": ff(-0.78, "-0.78%"),
				"currency":                   "USD",
				"marketState":                "REGULAR",
				"fullExchangeName":           "CCC",
			},
		))
	})
	defer srv.Close()

	c := NewClient(srv.URL, srv.URL, "")
	if err := c.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	quotes, err := c.GetQuotes([]string{"AAPL", "BTC-USD"})
	if err != nil {
		t.Fatalf("GetQuotes failed: %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("expected 2 quotes, got %d", len(quotes))
	}
	if quotes[0].Symbol != "AAPL" {
		t.Errorf("first symbol: got %q, want AAPL", quotes[0].Symbol)
	}
	if quotes[1].Symbol != "BTC-USD" {
		t.Errorf("second symbol: got %q, want BTC-USD", quotes[1].Symbol)
	}
	if quotes[1].Price != 65432.10 {
		t.Errorf("BTC price: got %f, want 65432.10", quotes[1].Price)
	}
}

func TestGetQuotes_EmptySymbols(t *testing.T) {
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		t.Error("quote endpoint should not be called for empty symbols")
	})
	defer srv.Close()

	c := NewClient(srv.URL, srv.URL, "")
	if err := c.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	quotes, err := c.GetQuotes([]string{})
	if err != nil {
		t.Fatalf("GetQuotes failed: %v", err)
	}
	if len(quotes) != 0 {
		t.Errorf("expected 0 quotes, got %d", len(quotes))
	}
}
