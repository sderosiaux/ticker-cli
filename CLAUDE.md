# ticker-cli

Yahoo Finance price checker. One binary, zero config, structured output.

## Install

```bash
go install github.com/sderosiaux/ticker-cli@latest
```

## Commands

```bash
# Current prices
ticker-cli AAPL SLB BTC-USD GC=F CL=F EURUSD=X

# Historical date
ticker-cli --date 2026-03-20 AAPL SLB GC=F

# Range history (1d, 5d, 1mo, 3mo, 6mo, 1y, ytd)
ticker-cli --range 5d AAPL GC=F

# Weekly change
ticker-cli --weekly-change AAPL SLB

# Year-to-date change
ticker-cli --ytd AAPL GC=F BTC-USD
```

## Global Flags

| Flag | Purpose |
|---|---|
| `--format json\|csv\|table` | Output format (default: table) |
| `--compact` | NDJSON one line per symbol (implies json) |
| `--date YYYY-MM-DD` | Close price at specific date |
| `--range 1d\|5d\|1mo\|ytd` | Historical OHLCV over period |
| `--weekly-change` | % change over last trading week |
| `--ytd` | Year-to-date % change |
| `--debug` | Show API calls and timing on stderr |

## LLM Usage Patterns

```bash
# Parse single price
ticker-cli --compact AAPL | jq .price

# Get all as structured JSON
ticker-cli --format json AAPL BTC-USD GC=F EURUSD=X

# Weekly report data
ticker-cli --weekly-change --format csv AAPL SLB NEM GC=F BZ=F CL=F BTC-USD EURUSD=X

# Verify historical price
ticker-cli --date 2026-03-20 --format json AAPL
```

## Yahoo Finance Symbols

| Asset | Symbol |
|---|---|
| US stocks | AAPL, MSFT, SLB, NEM |
| EU stocks | RMS.PA, AIR.PA, SU.PA |
| Indices | ^GSPC, ^FCHI, ^GDAXI |
| Brent/WTI | BZ=F, CL=F |
| Gold/Copper | GC=F, HG=F |
| Bitcoin | BTC-USD |
| EUR/USD | EURUSD=X |
| US 10Y | ^TNX |

## Output

- **stdout**: data only (JSON/CSV/table)
- **stderr**: spinner, errors, debug info
- **exit 0**: all symbols OK
- **exit 1**: some symbols failed
- **exit 2**: all failed or fatal error

## Development

```bash
go build -o ticker-cli .
go test ./... -count=1
```
