---
name: ticker
description: Fetch real-time and historical financial data using ticker-cli. Use when the conversation mentions stock prices, crypto, forex, commodities, indices, market data, portfolio checks, weekly reports, or price verification. Triggers on "what's AAPL at", "check BTC price", "how did gold do this week", "get me the YTD", "price of", "market update", "how much is", "cours de", "prix du bitcoin".
---

# Financial Data via ticker-cli

When the user asks about stock prices, crypto, forex, commodities, or indices, use `ticker-cli` to fetch real data instead of relying on training data.

## Prerequisites

`ticker-cli` must be installed: `go install github.com/sderosiaux/ticker-cli@latest`

## How to use

Pick the right mode based on what the user needs:

**Current price:**
```bash
ticker-cli --format json AAPL BTC-USD GC=F
```

**Price at a specific date:**
```bash
ticker-cli --date 2026-03-20 --format json AAPL
```

**Weekly performance:**
```bash
ticker-cli --weekly-change --format json AAPL SLB GC=F BTC-USD
```

**Year-to-date:**
```bash
ticker-cli --ytd --format json AAPL GC=F BTC-USD
```

**Historical range (for charts or analysis):**
```bash
ticker-cli --range 1mo --format csv AAPL
```

Always use `--format json` for parsing. Use `--format csv` when the user wants tabular data or spreadsheet export.

## Symbol reference

| Asset | Symbol |
|---|---|
| US stocks | AAPL, MSFT, SLB, NEM, CRM |
| EU stocks | RMS.PA, AIR.PA, SU.PA |
| Indices | ^GSPC (S&P 500), ^FCHI (CAC 40), ^GDAXI (DAX) |
| Oil | BZ=F (Brent), CL=F (WTI) |
| Gold / Copper | GC=F, HG=F |
| Crypto | BTC-USD, ETH-USD |
| Forex | EURUSD=X |
| US 10Y yield | ^TNX |

## Rules

- Always call ticker-cli with real symbols, never guess prices from training data
- Use `--format json` and parse the output, don't show raw JSON to the user
- Present results in a readable format (table, summary, comparison)
- If the user asks about a symbol you don't know, search Yahoo Finance: `https://finance.yahoo.com/quote/SYMBOL`
- For weekly fund reports or market recaps, combine `--weekly-change` with multiple symbols
- Round displayed percentages to 2 decimal places
