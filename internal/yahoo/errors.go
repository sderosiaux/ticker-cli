package yahoo

import "errors"

// Sentinel errors for Yahoo Finance API operations.
var (
	ErrNoChartData   = errors.New("no chart data")
	ErrNoIndicators  = errors.New("no quote indicators")
	ErrAPIStatus     = errors.New("unexpected API status")
	ErrMissingCookie = errors.New("A3 cookie not found")
	ErrEmptyCrumb    = errors.New("empty crumb response")
)
