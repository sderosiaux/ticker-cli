// Package cmd implements the CLI commands for ticker-cli.
package cmd

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/sderosiaux/ticker-cli/internal/debug"
	"github.com/sderosiaux/ticker-cli/internal/model"
	"github.com/sderosiaux/ticker-cli/internal/output"
	"github.com/sderosiaux/ticker-cli/internal/yahoo"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var version = "0.1.0"

var (
	flagFormat       string
	flagCompact      bool
	flagDate         string
	flagRange        string
	flagWeeklyChange bool
	flagYTD          bool
	flagDebug        bool
)

// ExitError signals a specific exit code to main.
type ExitError struct {
	Code int
	Msg  string
}

func (e *ExitError) Error() string { return e.Msg }

// Sentinel errors for flag validation.
var (
	ErrDateFormat   = errors.New("invalid date format")
	ErrDateFuture   = errors.New("date is in the future")
	ErrInvalidRange = errors.New("invalid range")
)

var validRanges = map[string]bool{
	"1d": true, "5d": true, "1mo": true, "3mo": true,
	"6mo": true, "1y": true, "ytd": true,
}

var rootCmd = &cobra.Command{
	Use:     "ticker-cli [symbols...]",
	Short:   "Yahoo Finance price checker for LLM agents",
	Long:    "Fetch current or historical prices from Yahoo Finance. Structured output for piping and LLM tool calls.",
	Version: version,
	Example: `  ticker-cli AAPL SLB BTC-USD GC=F
  ticker-cli --date 2026-03-20 AAPL SLB
  ticker-cli --range 5d AAPL GC=F
  ticker-cli --weekly-change AAPL --format json
  ticker-cli --ytd AAPL --compact`,
	Args:          cobra.MinimumNArgs(1),
	RunE:          run,
	SilenceUsage:  true,
	SilenceErrors: true,
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

// Execute runs the root command.
func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		return fmt.Errorf("execute: %w", err)
	}

	return nil
}

// stderrIsTTY reports whether stderr is a terminal.
func stderrIsTTY() bool {
	return term.IsTerminal(int(os.Stderr.Fd())) //nolint:gosec // fd fits in int
}

// spinner displays a braille animation on stderr (only if TTY).
type spinner struct {
	running atomic.Bool
	done    chan struct{}
}

func startSpinner(msg string) *spinner {
	s := &spinner{done: make(chan struct{})}
	if !stderrIsTTY() {
		close(s.done)

		return s
	}

	s.running.Store(true)

	go func() {
		chars := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}
		i := 0
		for s.running.Load() {
			fmt.Fprintf(os.Stderr, "\r%c %s\033[K", chars[i%10], msg)
			time.Sleep(80 * time.Millisecond)
			i++
		}

		fmt.Fprintf(os.Stderr, "\r\033[K")
		close(s.done)
	}()

	return s
}

func (s *spinner) Stop() {
	if s.running.CompareAndSwap(true, false) {
		<-s.done
	}
}

func computeChange(hist *model.HistoryResult, period string) *model.ChangeResult {
	if len(hist.Points) < 2 {
		return nil
	}

	first := hist.Points[0]
	last := hist.Points[len(hist.Points)-1]
	change := math.Round((last.Close-first.Close)*100) / 100
	changePct := math.Round((change/first.Close)*10000) / 100

	return &model.ChangeResult{
		Symbol:        hist.Symbol,
		Name:          hist.Name,
		Price:         last.Close,
		Currency:      hist.Currency,
		PeriodStart:   first.Close,
		PeriodEnd:     last.Close,
		Change:        change,
		ChangePercent: changePct,
		Period:        period,
	}
}

func errorf(format string, args ...any) {
	if stderrIsTTY() {
		fmt.Fprintf(os.Stderr, "\033[31m✗\033[0m "+format+"\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	}
}

func fetchDate(client *yahoo.Client, symbols []string) (any, int, error) {
	debug.Logf("mode: date=%s", flagDate)

	errCount := 0
	results := make([]model.HistoryResult, 0, len(symbols))

	for _, sym := range symbols {
		d := debug.Timer("GetChart " + sym)
		hist, err := client.GetChart(sym, "", flagDate, flagDate)
		d()

		if err != nil {
			errorf("%s: %v. Check symbol at https://finance.yahoo.com/quote/%s", sym, err, sym)
			errCount++

			continue
		}

		results = append(results, *hist)
	}

	return results, errCount, nil
}

func fetchRange(client *yahoo.Client, symbols []string) (any, int, error) {
	debug.Logf("mode: range=%s", flagRange)

	errCount := 0
	results := make([]model.HistoryResult, 0, len(symbols))

	for _, sym := range symbols {
		d := debug.Timer("GetChart " + sym)
		hist, err := client.GetChart(sym, flagRange, "", "")
		d()

		if err != nil {
			errorf("%s: %v. Check symbol at https://finance.yahoo.com/quote/%s", sym, err, sym)
			errCount++

			continue
		}

		results = append(results, *hist)
	}

	return results, errCount, nil
}

func fetchWeeklyChange(client *yahoo.Client, symbols []string) (any, int, error) {
	debug.Logf("mode: weekly-change")

	errCount := 0
	results := make([]model.ChangeResult, 0, len(symbols))

	for _, sym := range symbols {
		d := debug.Timer("GetChart " + sym)
		hist, err := client.GetChart(sym, "5d", "", "")
		d()

		if err != nil {
			errorf("%s: %v. Check symbol at https://finance.yahoo.com/quote/%s", sym, err, sym)
			errCount++

			continue
		}

		cr := computeChange(hist, "5d")
		if cr == nil {
			errorf("%s: insufficient data for weekly change", sym)
			errCount++

			continue
		}

		results = append(results, *cr)
	}

	return results, errCount, nil
}

func fetchYTD(client *yahoo.Client, symbols []string) (any, int, error) {
	debug.Logf("mode: ytd")

	errCount := 0
	results := make([]model.ChangeResult, 0, len(symbols))

	for _, sym := range symbols {
		d := debug.Timer("GetChart " + sym)
		hist, err := client.GetChart(sym, "ytd", "", "")
		d()

		if err != nil {
			errorf("%s: %v. Check symbol at https://finance.yahoo.com/quote/%s", sym, err, sym)
			errCount++

			continue
		}

		cr := computeChange(hist, "ytd")
		if cr == nil {
			errorf("%s: insufficient data for YTD change", sym)
			errCount++

			continue
		}

		results = append(results, *cr)
	}

	return results, errCount, nil
}

func dispatch(client *yahoo.Client, symbols []string) (any, int, error) {
	switch {
	case flagDate != "":
		return fetchDate(client, symbols)
	case flagRange != "":
		return fetchRange(client, symbols)
	case flagWeeklyChange:
		return fetchWeeklyChange(client, symbols)
	case flagYTD:
		return fetchYTD(client, symbols)
	default:
		return fetchQuotes(client, symbols)
	}
}

func fetchQuotes(client *yahoo.Client, symbols []string) (any, int, error) {
	debug.Logf("mode: quotes")

	d := debug.Timer("GetQuotes")
	quotes, err := client.GetQuotes(symbols)
	d()

	if err != nil {
		return nil, 0, fmt.Errorf("fetch quotes: %w", err)
	}

	return quotes, 0, nil
}

func run(_ *cobra.Command, args []string) error {
	debug.Enabled = flagDebug
	debug.ColorEnabled = stderrIsTTY()
	symbols := args

	// Validate flags
	if flagDate != "" {
		t, err := time.Parse("2006-01-02", flagDate)
		if err != nil {
			return fmt.Errorf("--date %s: %w", flagDate, ErrDateFormat)
		}

		if t.After(time.Now()) {
			return fmt.Errorf("--date %s: %w", flagDate, ErrDateFuture)
		}
	}

	if flagRange != "" && !validRanges[flagRange] {
		return fmt.Errorf("--range %s: %w", flagRange, ErrInvalidRange)
	}

	sp := startSpinner("Fetching prices...")

	client := yahoo.NewClient(
		"https://query1.finance.yahoo.com",
		"https://query2.finance.yahoo.com",
		"https://consent.yahoo.com",
	)
	client.SetSessionRootURL("https://fc.yahoo.com")

	debug.Logf("initializing session")

	done := debug.Timer("session.Init")

	err := client.Init()
	if err != nil {
		sp.Stop()
		errorf("Session failed: %v", err)
		fmt.Fprintf(os.Stderr, "  Try: ticker-cli --debug %s\n", strings.Join(symbols, " "))

		return &ExitError{Code: 2, Msg: "session init failed"}
	}

	done()

	var (
		data     any
		errCount int
	)

	data, errCount, err = dispatch(client, symbols)
	if err != nil {
		sp.Stop()

		return err
	}

	sp.Stop()

	if errCount == len(symbols) {
		return &ExitError{Code: 2, Msg: "all symbols failed"}
	}

	err = output.Write(os.Stdout, data, flagFormat, flagCompact)
	if err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	if errCount > 0 {
		return &ExitError{Code: 1, Msg: fmt.Sprintf("%d/%d symbols failed", errCount, len(symbols))}
	}

	return nil
}
