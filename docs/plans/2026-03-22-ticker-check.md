# ticker-check Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Go CLI that fetches Yahoo Finance prices with structured output for LLM consumption.

**Architecture:** Single binary, Cobra CLI, 2 Yahoo endpoints (v7/quote + v8/chart), 3 output formatters (table/json/csv). Session auth via cookies+crumb, EU consent handling. Errors stderr, data stdout.

**Tech Stack:** Go 1.22+, cobra (CLI), no other external deps. stdlib net/http, encoding/json, encoding/csv.

---

### Task 1: Go Module + Project Skeleton

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`

**Step 1: Initialize Go module**

Run: `cd /Users/sderosiaux/code/personal/ticker-cli && go mod init github.com/sderosiaux/ticker-check`

**Step 2: Install cobra**

Run: `go get github.com/spf13/cobra@latest`

**Step 3: Write main.go**

```go
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
```

**Step 4: Write cmd/root.go with all flags**

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagFormat       string
	flagCompact      bool
	flagDate         string
	flagRange        string
	flagWeeklyChange bool
	flagYTD          bool
	flagDebug        bool
)

var rootCmd = &cobra.Command{
	Use:   "ticker-check [symbols...]",
	Short: "Yahoo Finance price checker for LLM agents",
	Long:  "Fetch current or historical prices from Yahoo Finance. Structured output for piping and LLM tool calls.",
	Example: `  ticker-check AAPL SLB BTC-USD GC=F
  ticker-check --date 2026-03-20 AAPL SLB
  ticker-check --range 5d AAPL GC=F
  ticker-check --weekly-change AAPL --format json
  ticker-check --ytd AAPL --compact`,
	Args: cobra.MinimumNArgs(1),
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVar(&flagFormat, "format", "table", "Output format: table, json, csv")
	rootCmd.Flags().BoolVar(&flagCompact, "compact", false, "Minimal JSON, one line per symbol")
	rootCmd.Flags().StringVar(&flagDate, "date", "", "Close price at YYYY-MM-DD")
	rootCmd.Flags().StringVar(&flagRange, "range", "", "History period: 1d, 5d, 1mo, 3mo, 6mo, 1y, ytd")
	rootCmd.Flags().BoolVar(&flagWeeklyChange, "weekly-change", false, "Show weekly % change")
	rootCmd.Flags().BoolVar(&flagYTD, "ytd", false, "Show year-to-date % change")
	rootCmd.Flags().BoolVar(&flagDebug, "debug", false, "Show API calls and timing")
}

func Execute() error {
	return rootCmd.Execute()
}

func run(cmd *cobra.Command, args []string) error {
	fmt.Fprintln(os.Stderr, "not implemented yet")
	return nil
}
```

**Step 5: Verify it compiles and runs**

Run: `go build -o ticker-check . && ./ticker-check --help`
Expected: Help text with all flags and examples.

**Step 6: Commit**

```bash
git init && git add -A && git commit -m "feat: project skeleton with cobra CLI and all flags"
```

---

### Task 2: Yahoo Session (cookies + crumb)

**Files:**
- Create: `internal/yahoo/session.go`
- Create: `internal/yahoo/session_test.go`

**Step 1: Write session test**

```go
package yahoo

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewSession_GetsCookieAndCrumb(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test-a3"})
		w.WriteHeader(200)
	})
	mux.HandleFunc("/v1/test/getcrumb", func(w http.ResponseWriter, r *http.Request) {
		// Verify cookie is sent
		cookie, err := r.Cookie("A3")
		if err != nil || cookie.Value != "test-a3" {
			t.Error("expected A3 cookie in crumb request")
		}
		w.Write([]byte("test-crumb-123"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	s, err := NewSession(srv.URL, srv.URL, "")
	if err != nil {
		t.Fatalf("NewSession failed: %v", err)
	}
	if s.crumb != "test-crumb-123" {
		t.Errorf("expected crumb test-crumb-123, got %s", s.crumb)
	}
	if len(s.cookies) == 0 {
		t.Error("expected cookies to be set")
	}
}

func TestNewSession_Refresh(t *testing.T) {
	callCount := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "a3-v2"})
		w.WriteHeader(200)
	})
	mux.HandleFunc("/v1/test/getcrumb", func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Write([]byte("crumb-" + string(rune('0'+callCount))))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	s, _ := NewSession(srv.URL, srv.URL, "")
	oldCrumb := s.crumb
	err := s.Refresh()
	if err != nil {
		t.Fatalf("Refresh failed: %v", err)
	}
	if s.crumb == oldCrumb {
		t.Error("expected crumb to change after refresh")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/yahoo/ -run TestNewSession -v`
Expected: FAIL — package/types don't exist yet.

**Step 3: Implement session.go**

```go
package yahoo

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"
	defaultAccept    = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
	defaultAcceptLang = "en-US,en;q=0.9"
)

type Session struct {
	client     *http.Client
	rootURL    string
	crumbURL   string
	consentURL string
	cookies    []*http.Cookie
	crumb      string
}

func NewSession(rootURL, crumbURL, consentURL string) (*Session, error) {
	s := &Session{
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 1 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
		rootURL:    rootURL,
		crumbURL:   crumbURL,
		consentURL: consentURL,
	}
	if err := s.Refresh(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Session) Refresh() error {
	cookies, err := s.getCookie()
	if err != nil {
		return err
	}
	s.cookies = cookies

	crumb, err := s.getCrumb()
	if err != nil {
		return err
	}
	s.crumb = crumb
	return nil
}

func (s *Session) getCookie() ([]*http.Cookie, error) {
	req, err := http.NewRequest(http.MethodGet, s.rootURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cookie request: %w", err)
	}
	req.Header.Set("Accept", defaultAccept)
	req.Header.Set("Accept-Language", defaultAcceptLang)
	req.Header.Set("User-Agent", defaultUserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cookie fetch: %w", err)
	}
	defer resp.Body.Close()

	if isEUConsentRedirect(resp) {
		return s.getCookieEU(resp)
	}

	cookies := resp.Cookies()
	if !hasA3Cookie(cookies) {
		return nil, errors.New("A3 session cookie missing")
	}
	return cookies, nil
}

func (s *Session) getCookieEU(initialResp *http.Response) ([]*http.Cookie, error) {
	// Follow redirect to consent page
	client3 := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	req1, _ := http.NewRequest(http.MethodGet, s.rootURL, nil)
	req1.Header.Set("Accept", defaultAccept)
	req1.Header.Set("Accept-Language", defaultAcceptLang)
	req1.Header.Set("User-Agent", defaultUserAgent)

	resp1, err := client3.Do(req1)
	if err != nil {
		return nil, fmt.Errorf("EU consent redirect: %w", err)
	}
	defer resp1.Body.Close()

	sessionID, csrfToken, err := extractSessionAndCSRF(resp1)
	if err != nil {
		return nil, err
	}

	// Collect GUCS cookies from redirect chain
	var gucsCookies []*http.Cookie
	if resp1.Request != nil && resp1.Request.Response != nil &&
		resp1.Request.Response.Request != nil && resp1.Request.Response.Request.Response != nil {
		gucsCookies = parseSetCookieHeaders(resp1.Request.Response.Request.Response.Header)
	}

	// Submit consent
	formData := url.Values{
		"csrfToken": {csrfToken},
		"sessionId": {sessionID},
		"namespace": {"yahoo"},
		"agree":     {"agree"},
	}

	consentPath := fmt.Sprintf("/v2/collectConsent?sessionId=%s", sessionID)
	req2, err := http.NewRequest(http.MethodPost, s.consentURL+consentPath, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("consent request: %w", err)
	}

	req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req2.Header.Set("Accept", defaultAccept)
	req2.Header.Set("User-Agent", defaultUserAgent)
	req2.Header.Set("Content-Length", strconv.Itoa(len(formData.Encode())))
	for _, c := range gucsCookies {
		req2.AddCookie(c)
	}

	client2 := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 2 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	resp2, err := client2.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("consent submit: %w", err)
	}
	defer resp2.Body.Close()

	cookies := parseSetCookieHeaders(resp2.Header)
	if !hasA3Cookie(cookies) {
		return nil, errors.New("A3 cookie missing after EU consent")
	}
	return cookies, nil
}

func (s *Session) getCrumb() (string, error) {
	req, err := http.NewRequest(http.MethodGet, s.crumbURL+"/v1/test/getcrumb", nil)
	if err != nil {
		return "", fmt.Errorf("crumb request: %w", err)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", defaultUserAgent)
	for _, c := range s.cookies {
		req.AddCookie(c)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("crumb fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("crumb response: %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	return string(body), nil
}

// Cookies returns session cookies for use in API requests.
func (s *Session) Cookies() []*http.Cookie { return s.cookies }

// Crumb returns the session crumb token.
func (s *Session) Crumb() string { return s.crumb }

func isEUConsentRedirect(resp *http.Response) bool {
	return resp.StatusCode >= 300 && resp.StatusCode < 400 &&
		strings.Contains(resp.Header.Get("Location"), "/consent")
}

func hasA3Cookie(cookies []*http.Cookie) bool {
	for _, c := range cookies {
		if c.Name == "A3" {
			return true
		}
	}
	return false
}

func extractSessionAndCSRF(resp *http.Response) (string, string, error) {
	sidRe := regexp.MustCompile(`sessionId=([A-Za-z0-9_-]+)`)
	csrfRe := regexp.MustCompile(`gcrumb=([A-Za-z0-9_]+)`)

	var sidMatch, csrfMatch []string
	if resp.Request != nil && resp.Request.URL != nil {
		sidMatch = sidRe.FindStringSubmatch(resp.Request.URL.String())
	}
	if len(sidMatch) < 2 {
		return "", "", errors.New("cannot extract sessionId from consent redirect")
	}
	if resp.Request.Response != nil && resp.Request.Response.Request != nil && resp.Request.Response.Request.URL != nil {
		csrfMatch = csrfRe.FindStringSubmatch(resp.Request.Response.Request.URL.String())
	}
	if len(csrfMatch) < 2 {
		return "", "", errors.New("cannot extract CSRF token from consent redirect")
	}
	return sidMatch[1], csrfMatch[1], nil
}

func parseSetCookieHeaders(headers http.Header) []*http.Cookie {
	var cookies []*http.Cookie
	for _, raw := range headers.Values("Set-Cookie") {
		parts := strings.Split(raw, ";")
		if len(parts) == 0 {
			continue
		}
		nv := strings.SplitN(parts[0], "=", 2)
		if len(nv) != 2 {
			continue
		}
		cookies = append(cookies, &http.Cookie{
			Name:  strings.TrimSpace(nv[0]),
			Value: strings.TrimSpace(nv[1]),
		})
	}
	return cookies
}
```

**Step 4: Run tests**

Run: `go test ./internal/yahoo/ -run TestNewSession -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/yahoo/ && git commit -m "feat: yahoo session auth with cookie+crumb and EU consent"
```

---

### Task 3: v7 Quote API (current prices)

**Files:**
- Create: `internal/yahoo/quote.go`
- Create: `internal/yahoo/quote_test.go`
- Create: `internal/model/model.go`

**Step 1: Write model.go**

```go
package model

type Quote struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Currency      string  `json:"currency"`
	MarketState   string  `json:"marketState"`
	Exchange      string  `json:"exchange"`
	// Extended fields (non-compact)
	Open        float64 `json:"open,omitempty"`
	High        float64 `json:"high,omitempty"`
	Low         float64 `json:"low,omitempty"`
	PrevClose   float64 `json:"prevClose,omitempty"`
	Volume      float64 `json:"volume,omitempty"`
	MarketCap   float64 `json:"marketCap,omitempty"`
	Week52High  float64 `json:"week52High,omitempty"`
	Week52Low   float64 `json:"week52Low,omitempty"`
}

type HistoryPoint struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

type HistoryResult struct {
	Symbol   string         `json:"symbol"`
	Name     string         `json:"name"`
	Currency string         `json:"currency"`
	Points   []HistoryPoint `json:"points"`
}

type ChangeResult struct {
	Symbol        string  `json:"symbol"`
	Name          string  `json:"name"`
	Price         float64 `json:"price"`
	Currency      string  `json:"currency"`
	PeriodStart   float64 `json:"periodStart"`
	PeriodEnd     float64 `json:"periodEnd"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Period        string  `json:"period"`
}
```

**Step 2: Write quote_test.go**

```go
package yahoo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetQuotes(t *testing.T) {
	mux := http.NewServeMux()
	// Session endpoints
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test"})
	})
	mux.HandleFunc("/v1/test/getcrumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("crumb123"))
	})
	// Quote endpoint
	mux.HandleFunc("/v7/finance/quote", func(w http.ResponseWriter, r *http.Request) {
		symbols := r.URL.Query().Get("symbols")
		if symbols != "AAPL" {
			t.Errorf("expected symbols=AAPL, got %s", symbols)
		}
		crumb := r.URL.Query().Get("crumb")
		if crumb != "crumb123" {
			t.Errorf("expected crumb=crumb123, got %s", crumb)
		}
		resp := map[string]interface{}{
			"quoteResponse": map[string]interface{}{
				"result": []map[string]interface{}{
					{
						"symbol":    "AAPL",
						"shortName": "Apple Inc.",
						"regularMarketPrice":         map[string]interface{}{"raw": 178.52, "fmt": "178.52"},
						"regularMarketChange":        map[string]interface{}{"raw": 1.23, "fmt": "1.23"},
						"regularMarketChangePercent": map[string]interface{}{"raw": 0.69, "fmt": "0.69%"},
						"regularMarketPreviousClose": map[string]interface{}{"raw": 177.29, "fmt": "177.29"},
						"regularMarketOpen":          map[string]interface{}{"raw": 177.50, "fmt": "177.50"},
						"regularMarketDayHigh":       map[string]interface{}{"raw": 179.00, "fmt": "179.00"},
						"regularMarketDayLow":        map[string]interface{}{"raw": 177.00, "fmt": "177.00"},
						"regularMarketVolume":        map[string]interface{}{"raw": 54230100.0, "fmt": "54.23M"},
						"marketState":                "REGULAR",
						"currency":                   "USD",
						"fullExchangeName":           "NasdaqGS",
						"quoteType":                  "EQUITY",
						"fiftyTwoWeekHigh":           map[string]interface{}{"raw": 199.62, "fmt": "199.62"},
						"fiftyTwoWeekLow":            map[string]interface{}{"raw": 140.00, "fmt": "140.00"},
						"marketCap":                  map[string]interface{}{"raw": 2.8e12, "fmt": "2.8T"},
					},
				},
				"error": nil,
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, "")
	quotes, err := client.GetQuotes([]string{"AAPL"})
	if err != nil {
		t.Fatalf("GetQuotes failed: %v", err)
	}
	if len(quotes) != 1 {
		t.Fatalf("expected 1 quote, got %d", len(quotes))
	}
	q := quotes[0]
	if q.Symbol != "AAPL" {
		t.Errorf("expected AAPL, got %s", q.Symbol)
	}
	if q.Price != 178.52 {
		t.Errorf("expected 178.52, got %f", q.Price)
	}
	if q.Change != 1.23 {
		t.Errorf("expected 1.23, got %f", q.Change)
	}
	if q.MarketState != "REGULAR" {
		t.Errorf("expected REGULAR, got %s", q.MarketState)
	}
}
```

**Step 3: Implement quote.go**

```go
package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/sderosiaux/ticker-check/internal/model"
)

type Client struct {
	session *Session
	baseURL string
}

func NewClient(baseURL, crumbURL, consentURL string) *Client {
	return &Client{baseURL: baseURL}
}

// Init initializes the session. Must be called before API calls.
func (c *Client) Init() error {
	s, err := NewSession(
		c.baseURL,
		c.baseURL, // crumb URL same base in prod: query2.finance.yahoo.com
		"",
	)
	if err != nil {
		return err
	}
	c.session = s
	return nil
}

// initForTest allows tests to set custom URLs.
func (c *Client) initForTest(rootURL, crumbURL, consentURL string) error {
	s, err := NewSession(rootURL, crumbURL, consentURL)
	if err != nil {
		return err
	}
	c.session = s
	return nil
}

type v7Response struct {
	QuoteResponse struct {
		Quotes []v7Quote   `json:"result"`
		Error  interface{} `json:"error"`
	} `json:"quoteResponse"`
}

type v7Quote struct {
	Symbol                     string         `json:"symbol"`
	ShortName                  string         `json:"shortName"`
	MarketState                string         `json:"marketState"`
	Currency                   string         `json:"currency"`
	ExchangeName               string         `json:"fullExchangeName"`
	QuoteType                  string         `json:"quoteType"`
	RegularMarketPrice         floatField     `json:"regularMarketPrice"`
	RegularMarketChange        floatField     `json:"regularMarketChange"`
	RegularMarketChangePercent floatField     `json:"regularMarketChangePercent"`
	RegularMarketPreviousClose floatField     `json:"regularMarketPreviousClose"`
	RegularMarketOpen          floatField     `json:"regularMarketOpen"`
	RegularMarketDayHigh       floatField     `json:"regularMarketDayHigh"`
	RegularMarketDayLow        floatField     `json:"regularMarketDayLow"`
	RegularMarketVolume        floatField     `json:"regularMarketVolume"`
	PostMarketPrice            floatField     `json:"postMarketPrice"`
	PostMarketChange           floatField     `json:"postMarketChange"`
	PostMarketChangePercent    floatField     `json:"postMarketChangePercent"`
	PreMarketPrice             floatField     `json:"preMarketPrice"`
	PreMarketChange            floatField     `json:"preMarketChange"`
	PreMarketChangePercent     floatField     `json:"preMarketChangePercent"`
	FiftyTwoWeekHigh           floatField     `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekLow            floatField     `json:"fiftyTwoWeekLow"`
	MarketCap                  floatField     `json:"marketCap"`
}

type floatField struct {
	Raw float64 `json:"raw"`
	Fmt string  `json:"fmt"`
}

var quoteFields = []string{
	"shortName", "regularMarketPrice", "regularMarketChange",
	"regularMarketChangePercent", "regularMarketPreviousClose",
	"regularMarketOpen", "regularMarketDayHigh", "regularMarketDayLow",
	"regularMarketVolume", "postMarketPrice", "postMarketChange",
	"postMarketChangePercent", "preMarketPrice", "preMarketChange",
	"preMarketChangePercent", "fiftyTwoWeekHigh", "fiftyTwoWeekLow",
	"marketCap",
}

func (c *Client) GetQuotes(symbols []string) ([]model.Quote, error) {
	reqURL, _ := url.Parse(c.baseURL + "/v7/finance/quote")
	q := reqURL.Query()
	q.Set("symbols", strings.Join(symbols, ","))
	q.Set("fields", strings.Join(quoteFields, ","))
	q.Set("formatted", "true")
	q.Set("lang", "en-US")
	q.Set("region", "US")
	q.Set("corsDomain", "finance.yahoo.com")
	if c.session != nil && c.session.Crumb() != "" {
		q.Set("crumb", c.session.Crumb())
	}
	reqURL.RawQuery = q.Encode()

	req, _ := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", defaultUserAgent)
	if c.session != nil {
		for _, cookie := range c.session.Cookies() {
			req.AddCookie(cookie)
		}
	}

	httpClient := &http.Client{}
	if c.session != nil && c.session.client != nil {
		httpClient = c.session.client
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("quote request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Retry with session refresh
		if c.session != nil {
			if err := c.session.Refresh(); err != nil {
				return nil, fmt.Errorf("session refresh: %w", err)
			}
			return c.GetQuotes(symbols) // one retry
		}
		return nil, fmt.Errorf("quote response: %d", resp.StatusCode)
	}

	var v7 v7Response
	if err := json.NewDecoder(resp.Body).Decode(&v7); err != nil {
		return nil, fmt.Errorf("decode quotes: %w", err)
	}

	quotes := make([]model.Quote, 0, len(v7.QuoteResponse.Quotes))
	for _, yq := range v7.QuoteResponse.Quotes {
		quotes = append(quotes, model.Quote{
			Symbol:        yq.Symbol,
			Name:          yq.ShortName,
			Price:         yq.RegularMarketPrice.Raw,
			Change:        yq.RegularMarketChange.Raw,
			ChangePercent: yq.RegularMarketChangePercent.Raw,
			Currency:      yq.Currency,
			MarketState:   yq.MarketState,
			Exchange:      yq.ExchangeName,
			Open:          yq.RegularMarketOpen.Raw,
			High:          yq.RegularMarketDayHigh.Raw,
			Low:           yq.RegularMarketDayLow.Raw,
			PrevClose:     yq.RegularMarketPreviousClose.Raw,
			Volume:        yq.RegularMarketVolume.Raw,
			MarketCap:     yq.MarketCap.Raw,
			Week52High:    yq.FiftyTwoWeekHigh.Raw,
			Week52Low:     yq.FiftyTwoWeekLow.Raw,
		})
	}
	return quotes, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/yahoo/ -run TestGetQuotes -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ && git commit -m "feat: v7 quote API with model types"
```

---

### Task 4: v8 Chart API (historical data)

**Files:**
- Create: `internal/yahoo/chart.go`
- Create: `internal/yahoo/chart_test.go`

**Step 1: Write chart_test.go**

```go
package yahoo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetChart_Range(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test"})
	})
	mux.HandleFunc("/v1/test/getcrumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("crumb"))
	})
	mux.HandleFunc("/v8/finance/chart/AAPL", func(w http.ResponseWriter, r *http.Request) {
		rng := r.URL.Query().Get("range")
		if rng != "5d" {
			t.Errorf("expected range=5d, got %s", rng)
		}
		resp := v8Response{
			Chart: v8Chart{
				Result: []v8Result{{
					Meta: v8Meta{Symbol: "AAPL", ShortName: "Apple Inc.", Currency: "USD"},
					Timestamps: []int64{1711065600, 1711152000, 1711238400, 1711324800, 1711411200},
					Indicators: v8Indicators{
						Quote: []v8IndicatorQuote{{
							Open:   []float64{176.5, 177.0, 177.5, 178.0, 178.5},
							High:   []float64{178.0, 178.5, 179.0, 179.5, 180.0},
							Low:    []float64{176.0, 176.5, 177.0, 177.5, 178.0},
							Close:  []float64{177.0, 177.5, 178.0, 178.5, 179.0},
							Volume: []int64{50000000, 51000000, 52000000, 53000000, 54000000},
						}},
					},
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, "")
	result, err := client.GetChart("AAPL", "5d", "", "")
	if err != nil {
		t.Fatalf("GetChart failed: %v", err)
	}
	if result.Symbol != "AAPL" {
		t.Errorf("expected AAPL, got %s", result.Symbol)
	}
	if len(result.Points) != 5 {
		t.Fatalf("expected 5 points, got %d", len(result.Points))
	}
	if result.Points[0].Close != 177.0 {
		t.Errorf("expected close 177.0, got %f", result.Points[0].Close)
	}
}

func TestGetChart_DateRange(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "A3", Value: "test"})
	})
	mux.HandleFunc("/v1/test/getcrumb", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("crumb"))
	})
	mux.HandleFunc("/v8/finance/chart/AAPL", func(w http.ResponseWriter, r *http.Request) {
		p1 := r.URL.Query().Get("period1")
		p2 := r.URL.Query().Get("period2")
		if p1 == "" || p2 == "" {
			t.Error("expected period1 and period2 params")
		}
		resp := v8Response{
			Chart: v8Chart{
				Result: []v8Result{{
					Meta: v8Meta{Symbol: "AAPL", ShortName: "Apple Inc.", Currency: "USD"},
					Timestamps: []int64{1711065600},
					Indicators: v8Indicators{
						Quote: []v8IndicatorQuote{{
							Open:   []float64{176.5},
							High:   []float64{178.0},
							Low:    []float64{176.0},
							Close:  []float64{177.29},
							Volume: []int64{54230100},
						}},
					},
				}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := NewClient(srv.URL, srv.URL, "")
	result, err := client.GetChart("AAPL", "", "2026-03-20", "2026-03-20")
	if err != nil {
		t.Fatalf("GetChart failed: %v", err)
	}
	if len(result.Points) != 1 {
		t.Fatalf("expected 1 point, got %d", len(result.Points))
	}
	if result.Points[0].Close != 177.29 {
		t.Errorf("expected 177.29, got %f", result.Points[0].Close)
	}
}
```

**Step 2: Run test to verify fail**

Run: `go test ./internal/yahoo/ -run TestGetChart -v`

**Step 3: Implement chart.go**

```go
package yahoo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sderosiaux/ticker-check/internal/model"
)

type v8Response struct {
	Chart v8Chart `json:"chart"`
}

type v8Chart struct {
	Result []v8Result  `json:"result"`
	Error  interface{} `json:"error"`
}

type v8Result struct {
	Meta       v8Meta       `json:"meta"`
	Timestamps []int64      `json:"timestamp"`
	Indicators v8Indicators `json:"indicators"`
}

type v8Meta struct {
	Symbol    string `json:"symbol"`
	ShortName string `json:"shortName"`
	Currency  string `json:"currency"`
}

type v8Indicators struct {
	Quote []v8IndicatorQuote `json:"quote"`
}

type v8IndicatorQuote struct {
	Open   []float64 `json:"open"`
	High   []float64 `json:"high"`
	Low    []float64 `json:"low"`
	Close  []float64 `json:"close"`
	Volume []int64   `json:"volume"`
}

// GetChart fetches historical data. Use rangeStr for period-based ("5d","1mo") or
// dateFrom/dateTo for date-based ("2026-03-20"). One of them must be set.
func (c *Client) GetChart(symbol, rangeStr, dateFrom, dateTo string) (*model.HistoryResult, error) {
	reqURL, _ := url.Parse(fmt.Sprintf("%s/v8/finance/chart/%s", c.baseURL, symbol))
	q := reqURL.Query()
	q.Set("interval", "1d")
	q.Set("lang", "en-US")
	q.Set("region", "US")

	if rangeStr != "" {
		q.Set("range", rangeStr)
	} else if dateFrom != "" {
		t1, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			return nil, fmt.Errorf("invalid dateFrom: %w", err)
		}
		q.Set("period1", strconv.FormatInt(t1.Unix(), 10))
		if dateTo != "" {
			t2, err := time.Parse("2006-01-02", dateTo)
			if err != nil {
				return nil, fmt.Errorf("invalid dateTo: %w", err)
			}
			// Add 1 day to include the end date
			q.Set("period2", strconv.FormatInt(t2.Add(24*time.Hour).Unix(), 10))
		} else {
			q.Set("period2", strconv.FormatInt(t1.Add(24*time.Hour).Unix(), 10))
		}
	}

	if c.session != nil && c.session.Crumb() != "" {
		q.Set("crumb", c.session.Crumb())
	}
	reqURL.RawQuery = q.Encode()

	req, _ := http.NewRequest(http.MethodGet, reqURL.String(), nil)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", defaultUserAgent)
	if c.session != nil {
		for _, cookie := range c.session.Cookies() {
			req.AddCookie(cookie)
		}
	}

	httpClient := &http.Client{}
	if c.session != nil && c.session.client != nil {
		httpClient = c.session.client
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chart request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("chart response: %d for %s", resp.StatusCode, symbol)
	}

	var v8 v8Response
	if err := json.NewDecoder(resp.Body).Decode(&v8); err != nil {
		return nil, fmt.Errorf("decode chart: %w", err)
	}

	if len(v8.Chart.Result) == 0 {
		return nil, fmt.Errorf("%s: no chart data returned", symbol)
	}

	r := v8.Chart.Result[0]
	points := make([]model.HistoryPoint, 0, len(r.Timestamps))

	if len(r.Indicators.Quote) > 0 {
		iq := r.Indicators.Quote[0]
		for i, ts := range r.Timestamps {
			pt := model.HistoryPoint{
				Date: time.Unix(ts, 0).UTC().Format("2006-01-02"),
			}
			if i < len(iq.Open) { pt.Open = iq.Open[i] }
			if i < len(iq.High) { pt.High = iq.High[i] }
			if i < len(iq.Low) { pt.Low = iq.Low[i] }
			if i < len(iq.Close) { pt.Close = iq.Close[i] }
			if i < len(iq.Volume) { pt.Volume = iq.Volume[i] }
			points = append(points, pt)
		}
	}

	return &model.HistoryResult{
		Symbol:   r.Meta.Symbol,
		Name:     r.Meta.ShortName,
		Currency: r.Meta.Currency,
		Points:   points,
	}, nil
}
```

**Step 4: Run tests**

Run: `go test ./internal/yahoo/ -run TestGetChart -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ && git commit -m "feat: v8 chart API for historical data"
```

---

### Task 5: Output Formatters (table, json, csv)

**Files:**
- Create: `internal/output/table.go`
- Create: `internal/output/json.go`
- Create: `internal/output/csv.go`
- Create: `internal/output/output.go`
- Create: `internal/output/output_test.go`

**Step 1: Write output_test.go**

```go
package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sderosiaux/ticker-check/internal/model"
)

var testQuotes = []model.Quote{
	{Symbol: "AAPL", Name: "Apple Inc.", Price: 178.52, Change: 1.23, ChangePercent: 0.69, Currency: "USD", MarketState: "REGULAR"},
	{Symbol: "BTC-USD", Name: "Bitcoin USD", Price: 84231.00, Change: -520.00, ChangePercent: -0.61, Currency: "USD", MarketState: "REGULAR"},
}

func TestJSON_Quotes(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, testQuotes, false); err != nil {
		t.Fatal(err)
	}
	var parsed []model.Quote
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(parsed) != 2 {
		t.Errorf("expected 2 quotes, got %d", len(parsed))
	}
}

func TestCompact_Quotes(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteJSON(&buf, testQuotes, true); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d", len(lines))
	}
	// Verify each line is valid JSON
	for _, line := range lines {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("invalid JSON line: %s", line)
		}
	}
}

func TestCSV_Quotes(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteCSV(&buf, testQuotes); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 { // header + 2 rows
		t.Errorf("expected 3 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "symbol,") {
		t.Errorf("expected CSV header, got: %s", lines[0])
	}
}

func TestTable_Quotes(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteTable(&buf, testQuotes); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "AAPL") {
		t.Error("expected AAPL in table output")
	}
	if !strings.Contains(out, "BTC-USD") {
		t.Error("expected BTC-USD in table output")
	}
}
```

**Step 2: Run to verify fail**

Run: `go test ./internal/output/ -v`

**Step 3: Implement output files**

`output.go` — shared types and dispatcher:
```go
package output

import (
	"fmt"
	"io"
)

func Write(w io.Writer, data interface{}, format string, compact bool) error {
	switch format {
	case "json":
		return WriteJSON(w, data, compact)
	case "csv":
		return WriteCSVAny(w, data)
	case "table":
		return WriteTableAny(w, data)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}
```

`json.go`:
```go
package output

import (
	"encoding/json"
	"io"

	"github.com/sderosiaux/ticker-check/internal/model"
)

type compactQuote struct {
	Symbol        string  `json:"symbol"`
	Price         float64 `json:"price"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Currency      string  `json:"currency"`
	MarketState   string  `json:"marketState"`
}

func WriteJSON(w io.Writer, data interface{}, compact bool) error {
	if compact {
		if quotes, ok := data.([]model.Quote); ok {
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			for _, q := range quotes {
				if err := enc.Encode(compactQuote{
					Symbol: q.Symbol, Price: q.Price,
					Change: q.Change, ChangePercent: q.ChangePercent,
					Currency: q.Currency, MarketState: q.MarketState,
				}); err != nil {
					return err
				}
			}
			return nil
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(data)
}
```

`csv.go`:
```go
package output

import (
	"encoding/csv"
	"fmt"
	"io"

	"github.com/sderosiaux/ticker-check/internal/model"
)

func WriteCSV(w io.Writer, quotes []model.Quote) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()
	cw.Write([]string{"symbol", "name", "price", "change", "change_pct", "currency", "market_state"})
	for _, q := range quotes {
		cw.Write([]string{
			q.Symbol, q.Name,
			fmt.Sprintf("%.2f", q.Price),
			fmt.Sprintf("%.2f", q.Change),
			fmt.Sprintf("%.2f", q.ChangePercent),
			q.Currency, q.MarketState,
		})
	}
	return cw.Error()
}

func WriteCSVAny(w io.Writer, data interface{}) error {
	if quotes, ok := data.([]model.Quote); ok {
		return WriteCSV(w, quotes)
	}
	// Handle other types (HistoryResult, ChangeResult) similarly
	return fmt.Errorf("CSV not supported for this data type")
}
```

`table.go`:
```go
package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sderosiaux/ticker-check/internal/model"
	"golang.org/x/term"
)

func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func color(code, text string) string {
	if !isTTY() {
		return text
	}
	return fmt.Sprintf("\033[%sm%s\033[0m", code, text)
}

func changeColor(val float64) string {
	if val > 0 {
		return "32" // green
	} else if val < 0 {
		return "31" // red
	}
	return "0"
}

func WriteTable(w io.Writer, quotes []model.Quote) error {
	for _, q := range quotes {
		sign := ""
		if q.Change > 0 { sign = "+" }
		cc := changeColor(q.Change)
		fmt.Fprintf(w, "%-10s %-20s %10s  %s  %s  %s\n",
			q.Symbol,
			truncate(q.Name, 20),
			formatPrice(q.Price, q.Currency),
			color(cc, fmt.Sprintf("%s%.2f", sign, q.Change)),
			color(cc, fmt.Sprintf("%s%.2f%%", sign, q.ChangePercent)),
			q.MarketState,
		)
	}
	return nil
}

func WriteTableAny(w io.Writer, data interface{}) error {
	if quotes, ok := data.([]model.Quote); ok {
		return WriteTable(w, quotes)
	}
	return fmt.Errorf("table not supported for this data type")
}

func formatPrice(price float64, currency string) string {
	sym := "$"
	switch currency {
	case "EUR": sym = "E"
	case "GBP": sym = "£"
	}
	if price >= 10000 {
		return fmt.Sprintf("%s%,.0f", sym, price)
	}
	return fmt.Sprintf("%s%.2f", sym, price)
}

func truncate(s string, max int) string {
	if len(s) <= max { return s }
	return s[:max-1] + "."
}
```

Note: `golang.org/x/term` needed for TTY detection. Add: `go get golang.org/x/term`

**Step 4: Run tests**

Run: `go get golang.org/x/term && go test ./internal/output/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/output/ && git commit -m "feat: output formatters (table, json, csv, compact)"
```

---

### Task 6: Wire Everything in cmd/root.go

**Files:**
- Modify: `cmd/root.go`
- Create: `internal/debug/debug.go`

**Step 1: Write debug.go**

```go
package debug

import (
	"fmt"
	"os"
	"time"
)

var Enabled bool

func Log(format string, args ...interface{}) {
	if !Enabled { return }
	fmt.Fprintf(os.Stderr, "\033[90m[DEBUG] "+format+"\033[0m\n", args...)
}

func Timer(label string) func() {
	if !Enabled { return func() {} }
	start := time.Now()
	return func() {
		fmt.Fprintf(os.Stderr, "\033[90m[API] %s %dms\033[0m\n", label, time.Since(start).Milliseconds())
	}
}
```

**Step 2: Rewrite cmd/root.go run function**

Wire: parse flags → init session (with spinner) → call appropriate API → format output.

```go
func run(cmd *cobra.Command, args []string) error {
	debug.Enabled = flagDebug
	symbols := args

	// Spinner on stderr
	spinner := startSpinner("Fetching prices")

	// Init Yahoo client
	client := yahoo.NewClient(
		"https://query1.finance.yahoo.com",
		"https://query2.finance.yahoo.com",
		"https://consent.yahoo.com",
	)
	if err := client.Init(); err != nil {
		spinner.Stop()
		fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m Session error: %v\n", err)
		fmt.Fprintf(os.Stderr, "  Try: ticker-check --debug %s\n", strings.Join(symbols, " "))
		return err
	}

	var data interface{}
	var err error

	switch {
	case flagDate != "":
		data, err = fetchDate(client, symbols, flagDate)
	case flagRange != "":
		data, err = fetchRange(client, symbols, flagRange)
	case flagWeeklyChange:
		data, err = fetchChange(client, symbols, "5d", "weekly")
	case flagYTD:
		data, err = fetchChange(client, symbols, "ytd", "ytd")
	default:
		data, err = client.GetQuotes(symbols)
	}

	spinner.Stop()

	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m %v\n", err)
		return err
	}

	return output.Write(os.Stdout, data, flagFormat, flagCompact)
}
```

**Step 3: Implement fetchDate, fetchRange, fetchChange helpers**

These call `client.GetChart()` per symbol and aggregate results.

**Step 4: Implement spinner (from cli-best-practices skill)**

Braille spinner on stderr, stopped before writing stdout.

**Step 5: Build and test end-to-end**

Run: `go build -o ticker-check . && ./ticker-check --debug AAPL`
Expected: Real price data from Yahoo Finance.

**Step 6: Commit**

```bash
git add -A && git commit -m "feat: wire CLI with all modes and spinner"
```

---

### Task 7: CSV/Table for History and Change types

**Files:**
- Modify: `internal/output/csv.go`
- Modify: `internal/output/table.go`
- Modify: `internal/output/json.go`
- Create: `internal/output/output_history_test.go`

**Step 1: Add tests for HistoryResult and ChangeResult output**

**Step 2: Extend WriteCSVAny, WriteTableAny, WriteJSON to handle all model types**

**Step 3: Run tests, verify pass**

**Step 4: Commit**

```bash
git add internal/output/ && git commit -m "feat: output formatters for history and change results"
```

---

### Task 8: CLAUDE.md + Final Polish

**Files:**
- Create: `CLAUDE.md`
- Create: `Makefile`

**Step 1: Write CLAUDE.md**

LLM-friendly docs: auth (none needed), all commands, flags table, usage patterns.

**Step 2: Write Makefile**

```makefile
build:
	go build -o ticker-check .

test:
	go test ./... -count=1

install:
	go install .
```

**Step 3: Full test run**

Run: `go test ./... -count=1 -v`
Expected: All tests pass.

**Step 4: Build and smoke test**

```bash
make build
./ticker-check AAPL BTC-USD GC=F EURUSD=X
./ticker-check --format json AAPL
./ticker-check --compact AAPL
./ticker-check --format csv AAPL
./ticker-check --range 5d AAPL
./ticker-check --weekly-change AAPL SLB
./ticker-check --ytd AAPL
./ticker-check --date 2026-03-20 AAPL
```

**Step 5: Commit**

```bash
git add -A && git commit -m "feat: CLAUDE.md, Makefile, ready for use"
```
