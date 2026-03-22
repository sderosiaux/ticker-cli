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
