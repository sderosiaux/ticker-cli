// Package yahoo provides a client for the Yahoo Finance API.
package yahoo

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/sderosiaux/ticker-cli/internal/model"
)

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol    string `json:"symbol"`
				ShortName string `json:"shortName"`
				Currency  string `json:"currency"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Open   []float64 `json:"open"`
					High   []float64 `json:"high"`
					Low    []float64 `json:"low"`
					Close  []float64 `json:"close"`
					Volume []int64   `json:"volume"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

// GetChart fetches historical OHLCV data for a symbol.
// Use rangeStr (e.g. "5d", "1mo", "ytd") or dateFrom/dateTo ("YYYY-MM-DD").
func (c *Client) GetChart(symbol, rangeStr, dateFrom, dateTo string) (*model.HistoryResult, error) {
	params := url.Values{}
	params.Set("interval", "1d")
	params.Set("lang", "en-US")
	params.Set("region", "US")

	if rangeStr != "" {
		params.Set("range", rangeStr)
	}

	if dateFrom != "" {
		t, err := time.Parse("2006-01-02", dateFrom)
		if err != nil {
			return nil, fmt.Errorf("parse dateFrom: %w", err)
		}

		params.Set("period1", strconv.FormatInt(t.UTC().Unix(), 10))

		endDate := t
		if dateTo != "" {
			endDate, err = time.Parse("2006-01-02", dateTo)
			if err != nil {
				return nil, fmt.Errorf("parse dateTo: %w", err)
			}
		}
		// Add 1 day to make the range inclusive
		params.Set("period2", strconv.FormatInt(endDate.UTC().Add(24*time.Hour).Unix(), 10))
	}

	if crumb := c.session.Crumb(); crumb != "" {
		params.Set("crumb", crumb)
	}

	endpoint := c.baseURL + "/v8/finance/chart/" + url.PathEscape(symbol) + "?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", userAgent)

	for _, cookie := range c.session.Cookies() {
		req.AddCookie(cookie)
	}

	body, err := c.doChartRequest(req)
	if err != nil {
		// Retry once with session refresh
		refreshErr := c.session.Refresh()
		if refreshErr != nil {
			return nil, fmt.Errorf("session refresh: %w", refreshErr)
		}

		req2, _ := http.NewRequest(http.MethodGet, endpoint, nil)
		req2.Header.Set("User-Agent", userAgent)

		for _, cookie := range c.session.Cookies() {
			req2.AddCookie(cookie)
		}

		body, err = c.doChartRequest(req2)
		if err != nil {
			return nil, err
		}
	}

	var cr chartResponse

	err = json.Unmarshal(body, &cr)
	if err != nil {
		return nil, fmt.Errorf("decode chart response: %w", err)
	}

	if len(cr.Chart.Result) == 0 {
		return nil, fmt.Errorf("no chart data for %s", symbol)
	}

	r := cr.Chart.Result[0]
	points := make([]model.HistoryPoint, 0, len(r.Timestamp))

	q := r.Indicators.Quote
	if len(q) == 0 {
		return nil, fmt.Errorf("no quote indicators for %s", symbol)
	}

	r2 := func(v float64) float64 { return math.Round(v*100) / 100 }

	for i, ts := range r.Timestamp {
		pt := model.HistoryPoint{
			Date: time.Unix(ts, 0).UTC().Format("2006-01-02"),
		}

		if i < len(q[0].Open) {
			pt.Open = r2(q[0].Open[i])
		}

		if i < len(q[0].High) {
			pt.High = r2(q[0].High[i])
		}

		if i < len(q[0].Low) {
			pt.Low = r2(q[0].Low[i])
		}

		if i < len(q[0].Close) {
			pt.Close = r2(q[0].Close[i])
		}

		if i < len(q[0].Volume) {
			pt.Volume = q[0].Volume[i]
		}

		points = append(points, pt)
	}

	return &model.HistoryResult{
		Symbol:   r.Meta.Symbol,
		Name:     r.Meta.ShortName,
		Currency: r.Meta.Currency,
		Points:   points,
	}, nil
}

func (c *Client) doChartRequest(req *http.Request) ([]byte, error) {
	resp, err := c.session.client.Do(req) //nolint:gosec // URL is constructed from trusted baseURL
	if err != nil {
		return nil, err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("chart API returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
