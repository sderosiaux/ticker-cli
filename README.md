# ticker-cli

[![CI](https://github.com/sderosiaux/ticker-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/sderosiaux/ticker-cli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/sderosiaux/ticker-cli)](https://goreportcard.com/report/github.com/sderosiaux/ticker-cli)
[![Release](https://img.shields.io/github/v/release/sderosiaux/ticker-cli)](https://github.com/sderosiaux/ticker-cli/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

Get financial prices from your terminal. Stocks, crypto, forex, commodities, indices -- all from Yahoo Finance, in one command.

Works for humans and LLM agents. Ask your AI assistant something like:

> "How did my watchlist do this week? AAPL, GC=F, BTC-USD, EURUSD=X"

The agent runs `ticker-cli --weekly-change --format json AAPL GC=F BTC-USD EURUSD=X`, parses the JSON, and gives you a summary:

> AAPL dropped 1.9% to $248. Gold held up at +0.4%. Bitcoin took the biggest hit at -5.2%. EUR/USD flat.

Or ask it to verify a price from a report:

> "Was AAPL really at $248 on March 20th?"

It runs `ticker-cli --date 2026-03-20 --format json AAPL` and confirms.

---

```
$ ticker-cli AAPL BTC-USD GC=F EURUSD=X ^GSPC
AAPL       Apple Inc.                $247.99     -0.97    -0.39%  CLOSED
BTC-USD    Bitcoin USD            $84,231.00  -520.00    -0.61%  REGULAR
GC=F       Gold Apr 26             $3,043.20   +12.40    +0.41%  REGULAR
EURUSD=X   EUR/USD                     $1.08     +0.00    +0.12%  REGULAR
^GSPC      S&P 500                 $5,667.56   -23.14    -0.41%  CLOSED
```

## Install

```bash
go install github.com/sderosiaux/ticker-cli@latest
```

Or build from source:

```bash
git clone https://github.com/sderosiaux/ticker-cli.git
cd ticker-cli
make install
```

Requires Go 1.22+. Single binary, no config files, no API keys.

## Usage

### Current prices

```bash
ticker-cli AAPL SLB BTC-USD GC=F CL=F EURUSD=X
```

### Price at a specific date

Useful to verify a past report or reconcile a trade.

```bash
ticker-cli --date 2026-03-20 AAPL SLB GC=F
```

### Historical range

Get daily OHLCV data over a period. Works with `1d`, `5d`, `1mo`, `3mo`, `6mo`, `1y`, `ytd`.

```bash
ticker-cli --range 5d AAPL --format csv
```
```
symbol,date,open,high,low,close,volume
AAPL,2026-03-16,252.11,253.89,249.88,252.82,32074200
AAPL,2026-03-17,252.96,255.13,252.18,254.23,32361600
AAPL,2026-03-18,252.63,254.94,249.00,249.94,35757900
AAPL,2026-03-19,249.40,251.83,247.30,248.96,34864100
AAPL,2026-03-20,247.98,249.20,246.00,247.99,88268000
```

### Weekly change

How much did it move this week?

```bash
ticker-cli --weekly-change AAPL SLB NEM GC=F BTC-USD
```

### Year-to-date

```bash
ticker-cli --ytd AAPL GC=F BTC-USD
```
```
AAPL       Apple Inc.                $247.99    -23.02    -8.49%  ytd
GC=F       Gold Apr 26             $4,523.10   +208.70    +4.84%  ytd
BTC-USD    Bitcoin USD            $68,268.55  -20463.43   -23.06%  ytd
```

## Output formats

Table is the default. For piping or programmatic use:

```bash
# JSON array
ticker-cli --format json AAPL BTC-USD GC=F

# CSV with headers
ticker-cli --format csv AAPL BTC-USD

# NDJSON -- one JSON object per line, minimal fields
ticker-cli --compact AAPL BTC-USD GC=F
```

Compact output is designed for piping into `jq`:

```bash
$ ticker-cli --compact AAPL | jq .price
247.99
```

## LLM agent usage

Data goes to stdout, everything else (spinner, errors, debug) goes to stderr. JSON output is always arrays for lists, objects for details. No surprises.

An agent can call:

```bash
# Weekly report data
ticker-cli --weekly-change --format csv AAPL SLB NEM GC=F BZ=F CL=F BTC-USD EURUSD=X

# Verify a historical price
ticker-cli --date 2026-03-20 --format json AAPL

# Quick price check
ticker-cli --compact AAPL | jq .price
```

No config files, no auth tokens, no state. Every call is self-contained.

## Flags

| Flag | What it does |
|---|---|
| `--format json\|csv\|table` | Output format (default: table) |
| `--compact` | NDJSON, one line per symbol, essential fields only |
| `--date YYYY-MM-DD` | Close price at a specific date |
| `--range 1d\|5d\|1mo\|3mo\|6mo\|1y\|ytd` | Daily OHLCV over a period |
| `--weekly-change` | % change over the last trading week |
| `--ytd` | Year-to-date % change |
| `--debug` | Show API calls and timing on stderr |

## Yahoo Finance symbols

| Asset type | Examples |
|---|---|
| US stocks | `AAPL`, `MSFT`, `SLB`, `NEM`, `CRM` |
| European stocks | `RMS.PA` (Hermes), `AIR.PA` (Air Liquide), `SU.PA` (Schneider) |
| Indices | `^GSPC` (S&P 500), `^FCHI` (CAC 40), `^GDAXI` (DAX) |
| Oil | `BZ=F` (Brent), `CL=F` (WTI) |
| Metals | `GC=F` (Gold), `HG=F` (Copper) |
| Crypto | `BTC-USD`, `ETH-USD` |
| Forex | `EURUSD=X`, `GBPUSD=X` |
| Bonds | `^TNX` (US 10Y yield) |

Find any symbol at [finance.yahoo.com](https://finance.yahoo.com).

## Claude Code skill

If you use [Claude Code](https://docs.anthropic.com/en/docs/claude-code), install the ticker skill so Claude automatically uses `ticker-cli` when you ask about prices:

```bash
npx skills add sderosiaux/ticker-cli
```

After that, Claude calls `ticker-cli` whenever you mention stocks, crypto, gold, forex, etc. No need to tell it how -- it picks the right flags from context.

## How it works

Two Yahoo Finance endpoints, no API key required:

- `/v7/finance/quote` for current prices (batch, all symbols in one call)
- `/v8/finance/chart/{symbol}` for historical data (per symbol)

Authentication uses session cookies (`fc.yahoo.com`) and a crumb token, same approach as the [ticker](https://github.com/achannarasappa/ticker) project this was inspired by. EU GDPR consent redirects are handled automatically.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | All symbols returned data |
| 1 | Some symbols failed (partial results on stdout) |
| 2 | All symbols failed or fatal error |

## License

MIT
