package cmd

import (
	"fmt"
	"os"
	"sync/atomic"
	"time"

	"github.com/sderosiaux/ticker-check/internal/debug"
	"github.com/sderosiaux/ticker-check/internal/model"
	"github.com/sderosiaux/ticker-check/internal/output"
	"github.com/sderosiaux/ticker-check/internal/yahoo"
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

// spinner displays a braille animation on stderr.
type spinner struct {
	running atomic.Bool
	done    chan struct{}
}

func startSpinner(msg string) *spinner {
	s := &spinner{done: make(chan struct{})}
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
	s.running.Store(false)
	<-s.done
}

func computeChange(hist *model.HistoryResult, period string) *model.ChangeResult {
	if len(hist.Points) < 2 {
		return nil
	}
	first := hist.Points[0]
	last := hist.Points[len(hist.Points)-1]
	change := last.Close - first.Close
	changePct := (change / first.Close) * 100
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

func run(cmd *cobra.Command, args []string) error {
	debug.Enabled = flagDebug
	symbols := args

	sp := startSpinner("Fetching prices...")

	client := yahoo.NewClient(
		"https://query1.finance.yahoo.com",
		"https://query2.finance.yahoo.com",
		"https://consent.yahoo.com",
	)

	debug.Log("initializing session")
	done := debug.Timer("session.Init")
	if err := client.Init(); err != nil {
		sp.Stop()
		return fmt.Errorf("session init: %w", err)
	}
	done()

	var (
		data     interface{}
		errCount int
	)

	switch {
	case flagDate != "":
		debug.Log("mode: date=%s", flagDate)
		results := make([]model.HistoryResult, 0, len(symbols))
		for _, sym := range symbols {
			d := debug.Timer("GetChart " + sym)
			hist, err := client.GetChart(sym, "", flagDate, flagDate)
			d()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", sym, err)
				errCount++
				continue
			}
			results = append(results, *hist)
		}
		data = results

	case flagRange != "":
		debug.Log("mode: range=%s", flagRange)
		results := make([]model.HistoryResult, 0, len(symbols))
		for _, sym := range symbols {
			d := debug.Timer("GetChart " + sym)
			hist, err := client.GetChart(sym, flagRange, "", "")
			d()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", sym, err)
				errCount++
				continue
			}
			results = append(results, *hist)
		}
		data = results

	case flagWeeklyChange:
		debug.Log("mode: weekly-change")
		results := make([]model.ChangeResult, 0, len(symbols))
		for _, sym := range symbols {
			d := debug.Timer("GetChart " + sym)
			hist, err := client.GetChart(sym, "5d", "", "")
			d()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", sym, err)
				errCount++
				continue
			}
			cr := computeChange(hist, "5d")
			if cr == nil {
				fmt.Fprintf(os.Stderr, "error: %s: insufficient data for change\n", sym)
				errCount++
				continue
			}
			results = append(results, *cr)
		}
		data = results

	case flagYTD:
		debug.Log("mode: ytd")
		results := make([]model.ChangeResult, 0, len(symbols))
		for _, sym := range symbols {
			d := debug.Timer("GetChart " + sym)
			hist, err := client.GetChart(sym, "ytd", "", "")
			d()
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %s: %v\n", sym, err)
				errCount++
				continue
			}
			cr := computeChange(hist, "ytd")
			if cr == nil {
				fmt.Fprintf(os.Stderr, "error: %s: insufficient data for change\n", sym)
				errCount++
				continue
			}
			results = append(results, *cr)
		}
		data = results

	default:
		debug.Log("mode: quotes")
		d := debug.Timer("GetQuotes")
		quotes, err := client.GetQuotes(symbols)
		d()
		if err != nil {
			sp.Stop()
			return err
		}
		data = quotes
	}

	sp.Stop()

	if errCount == len(symbols) {
		os.Exit(2)
	}

	if err := output.Write(os.Stdout, data, flagFormat, flagCompact); err != nil {
		return err
	}

	if errCount > 0 {
		os.Exit(1)
	}

	return nil
}
