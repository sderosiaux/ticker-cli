package yahoo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newChartTestServer(chartHandler http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-cookie"})
			w.WriteHeader(http.StatusOK)

			return
		}

		http.NotFound(w, r)
	})
	mux.HandleFunc("/v1/test/getcrumb", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("test-crumb"))
	})
	mux.HandleFunc("/v8/finance/chart/", chartHandler)

	return httptest.NewServer(mux)
}

func chartJSON(t *testing.T, symbol, shortName, currency string, timestamps []int64, opens, highs, lows, closes []float64, volumes []int64) []byte {
	t.Helper()

	resp := chartJSONPayload{
		Chart: chartJSONChart{
			Result: []chartJSONResult{
				{
					Meta: chartJSONMeta{
						Symbol:    symbol,
						ShortName: shortName,
						Currency:  currency,
					},
					Timestamp: timestamps,
					Indicators: chartJSONIndicators{
						Quote: []chartJSONQuote{
							{
								Open:   opens,
								High:   highs,
								Low:    lows,
								Close:  closes,
								Volume: volumes,
							},
						},
					},
				},
			},
		},
	}

	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}

	return b
}

type chartJSONPayload struct {
	Chart chartJSONChart `json:"chart"`
}

type chartJSONChart struct {
	Result []chartJSONResult `json:"result"`
}

type chartJSONResult struct {
	Meta       chartJSONMeta       `json:"meta"`
	Timestamp  []int64             `json:"timestamp"`
	Indicators chartJSONIndicators `json:"indicators"`
}

type chartJSONMeta struct {
	Symbol    string `json:"symbol"`
	ShortName string `json:"shortName"`
	Currency  string `json:"currency"`
}

type chartJSONIndicators struct {
	Quote []chartJSONQuote `json:"quote"`
}

type chartJSONQuote struct {
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Close  []float64 `json:"close"`
	Volume []int64   `json:"volume"`
}

func TestGetChart_Range(t *testing.T) {
	srv := newChartTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("range") != "5d" {
			t.Errorf("expected range=5d, got %q", q.Get("range"))
		}

		if q.Get("interval") != "1d" {
			t.Errorf("expected interval=1d, got %q", q.Get("interval"))
		}

		if q.Get("crumb") != "test-crumb" {
			t.Errorf("expected crumb=test-crumb, got %q", q.Get("crumb"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(chartJSON(t,
			"AAPL", "Apple Inc.", "USD",
			[]int64{1711065600, 1711152000, 1711238400, 1711324800, 1711411200},
			[]float64{176.5, 177.0, 177.5, 178.0, 178.5},
			[]float64{178.0, 178.5, 179.0, 179.5, 180.0},
			[]float64{176.0, 176.5, 177.0, 177.5, 178.0},
			[]float64{177.0, 177.5, 178.0, 178.5, 179.0},
			[]int64{50000000, 51000000, 52000000, 53000000, 54000000},
		))
	})
	defer srv.Close()

	c := NewClient(srv.URL, srv.URL, "")

	err := c.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	result, err := c.GetChart("AAPL", "5d", "", "")
	if err != nil {
		t.Fatalf("GetChart failed: %v", err)
	}

	if result.Symbol != "AAPL" {
		t.Errorf("symbol: got %q, want AAPL", result.Symbol)
	}

	if result.Name != "Apple Inc." {
		t.Errorf("name: got %q, want Apple Inc.", result.Name)
	}

	if result.Currency != "USD" {
		t.Errorf("currency: got %q, want USD", result.Currency)
	}

	if len(result.Points) != 5 {
		t.Fatalf("expected 5 points, got %d", len(result.Points))
	}

	// Verify close prices
	expectedCloses := []float64{177.0, 177.5, 178.0, 178.5, 179.0}
	for i, pt := range result.Points {
		if pt.Close != expectedCloses[i] {
			t.Errorf("point[%d] close: got %f, want %f", i, pt.Close, expectedCloses[i])
		}
	}

	// Verify date formatting (UTC)
	if result.Points[0].Date != "2024-03-22" {
		t.Errorf("point[0] date: got %q, want 2024-03-22", result.Points[0].Date)
	}
}

func TestGetChart_DateRange(t *testing.T) {
	var gotPeriod1, gotPeriod2 string

	srv := newChartTestServer(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotPeriod1 = q.Get("period1")
		gotPeriod2 = q.Get("period2")

		if q.Get("range") != "" {
			t.Errorf("expected no range param, got %q", q.Get("range"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(chartJSON(t,
			"AAPL", "Apple Inc.", "USD",
			[]int64{1774224000},
			[]float64{180.0},
			[]float64{182.0},
			[]float64{179.0},
			[]float64{181.0},
			[]int64{55000000},
		))
	})
	defer srv.Close()

	c := NewClient(srv.URL, srv.URL, "")

	err := c.Init()
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	result, err := c.GetChart("AAPL", "", "2026-03-20", "2026-03-20")
	if err != nil {
		t.Fatalf("GetChart failed: %v", err)
	}

	if gotPeriod1 == "" {
		t.Error("expected period1 query param to be set")
	}

	if gotPeriod2 == "" {
		t.Error("expected period2 query param to be set")
	}

	if len(result.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(result.Points))
	}

	if result.Points[0].Close != 181.0 {
		t.Errorf("close: got %f, want 181.0", result.Points[0].Close)
	}
}
