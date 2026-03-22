package yahoo

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sderosiaux/ticker-check/internal/model"
)

// round2 rounds a float64 to 2 decimal places.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// Client wraps a Session and provides Yahoo Finance API methods.
type Client struct {
	baseURL string
	session *Session
}

// NewClient creates a Client. Call Init() to initialize the session.
func NewClient(baseURL, crumbURL, consentURL string) *Client {
	return &Client{
		baseURL: baseURL,
		session: &Session{
			client: &http.Client{
				Timeout: 10 * time.Second,
				CheckRedirect: func(req *http.Request, via []*http.Request) error {
					if len(via) >= 1 {
						return http.ErrUseLastResponse
					}
					return nil
				},
			},
			rootURL:    crumbURL,
			crumbURL:   crumbURL,
			consentURL: consentURL,
		},
	}
}

// SetSessionRootURL overrides the URL used to fetch session cookies.
// Default is the crumbURL, but fc.yahoo.com is needed in production.
func (c *Client) SetSessionRootURL(u string) {
	c.session.rootURL = u
}

// Init initializes the underlying session (fetches cookies + crumb).
func (c *Client) Init() error {
	return c.session.Refresh()
}

// quoteFields is the list of fields requested from the v7 API.
var quoteFields = []string{
	"shortName",
	"regularMarketPrice",
	"regularMarketChange",
	"regularMarketChangePercent",
	"currency",
	"marketState",
	"fullExchangeName",
	"regularMarketOpen",
	"regularMarketDayHigh",
	"regularMarketDayLow",
	"regularMarketPreviousClose",
	"regularMarketVolume",
	"marketCap",
	"fiftyTwoWeekHigh",
	"fiftyTwoWeekLow",
}

type floatField struct {
	Raw float64 `json:"raw"`
	Fmt string  `json:"fmt"`
}

type yahooQuote struct {
	Symbol                     string     `json:"symbol"`
	ShortName                  string     `json:"shortName"`
	MarketState                string     `json:"marketState"`
	Currency                   string     `json:"currency"`
	FullExchangeName           string     `json:"fullExchangeName"`
	RegularMarketPrice         floatField `json:"regularMarketPrice"`
	RegularMarketChange        floatField `json:"regularMarketChange"`
	RegularMarketChangePercent floatField `json:"regularMarketChangePercent"`
	RegularMarketOpen          floatField `json:"regularMarketOpen"`
	RegularMarketDayHigh       floatField `json:"regularMarketDayHigh"`
	RegularMarketDayLow        floatField `json:"regularMarketDayLow"`
	RegularMarketPreviousClose floatField `json:"regularMarketPreviousClose"`
	RegularMarketVolume        floatField `json:"regularMarketVolume"`
	MarketCap                  floatField `json:"marketCap"`
	FiftyTwoWeekHigh           floatField `json:"fiftyTwoWeekHigh"`
	FiftyTwoWeekLow            floatField `json:"fiftyTwoWeekLow"`
}

type quoteResponse struct {
	QuoteResponse struct {
		Result []yahooQuote `json:"result"`
		Error  interface{}  `json:"error"`
	} `json:"quoteResponse"`
}

// GetQuotes fetches current quotes for the given symbols.
func (c *Client) GetQuotes(symbols []string) ([]model.Quote, error) {
	if len(symbols) == 0 {
		return []model.Quote{}, nil
	}

	body, err := c.fetchQuotes(symbols)
	if err != nil {
		// On failure, refresh session once and retry.
		if refreshErr := c.session.Refresh(); refreshErr != nil {
			return nil, fmt.Errorf("session refresh: %w", refreshErr)
		}
		body, err = c.fetchQuotes(symbols)
		if err != nil {
			return nil, err
		}
	}

	var resp quoteResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode quote response: %w", err)
	}

	quotes := make([]model.Quote, 0, len(resp.QuoteResponse.Result))
	for _, yq := range resp.QuoteResponse.Result {
		quotes = append(quotes, model.Quote{
			Symbol:        yq.Symbol,
			Name:          yq.ShortName,
			Price:         round2(yq.RegularMarketPrice.Raw),
			Change:        round2(yq.RegularMarketChange.Raw),
			ChangePercent: round2(yq.RegularMarketChangePercent.Raw),
			Currency:      yq.Currency,
			MarketState:   yq.MarketState,
			Exchange:      yq.FullExchangeName,
			Open:          round2(yq.RegularMarketOpen.Raw),
			High:          round2(yq.RegularMarketDayHigh.Raw),
			Low:           round2(yq.RegularMarketDayLow.Raw),
			PrevClose:     round2(yq.RegularMarketPreviousClose.Raw),
			Volume:        yq.RegularMarketVolume.Raw,
			MarketCap:     yq.MarketCap.Raw,
			Week52High:    round2(yq.FiftyTwoWeekHigh.Raw),
			Week52Low:     round2(yq.FiftyTwoWeekLow.Raw),
		})
	}
	return quotes, nil
}

func (c *Client) fetchQuotes(symbols []string) ([]byte, error) {
	params := url.Values{}
	params.Set("symbols", strings.Join(symbols, ","))
	params.Set("fields", strings.Join(quoteFields, ","))
	params.Set("formatted", "true")
	params.Set("lang", "en-US")
	params.Set("region", "US")
	params.Set("corsDomain", "finance.yahoo.com")
	params.Set("crumb", c.session.Crumb())

	endpoint := c.baseURL + "/v7/finance/quote?" + params.Encode()

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	for _, cookie := range c.session.Cookies() {
		req.AddCookie(cookie)
	}

	resp, err := c.session.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("quote API returned %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
