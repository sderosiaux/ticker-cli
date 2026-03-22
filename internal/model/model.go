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
	Open          float64 `json:"open,omitempty"`
	High          float64 `json:"high,omitempty"`
	Low           float64 `json:"low,omitempty"`
	PrevClose     float64 `json:"prevClose,omitempty"`
	Volume        float64 `json:"volume,omitempty"`
	MarketCap     float64 `json:"marketCap,omitempty"`
	Week52High    float64 `json:"week52High,omitempty"`
	Week52Low     float64 `json:"week52Low,omitempty"`
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
