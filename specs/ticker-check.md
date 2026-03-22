# Feature: ticker-check CLI
## Status: Phase 2

## Problem
Who: LLM agents and humans analyzing financial data
Pain: No lightweight CLI to get structured Yahoo Finance prices; existing tools are TUIs or heavy libs
Trigger: Need to verify prices, check historical data, compute weekly/YTD changes from terminal or LLM tool calls
Impact: Manual web lookups, unreliable scraping, wasted tokens on noisy output
Why Now: LLM tool-use needs clean, structured, pipeable financial data

## Codebase Findings
### Related Code
Greenfield project. Inspired by github.com/achannarasappa/ticker (Go TUI).

### Integration Points
| Component | Connection | Risk |
|---|---|---|
| Yahoo Finance v7 `/quote` | Current prices, market state | Session/crumb auth can break |
| Yahoo Finance v8 `/chart` | Historical OHLCV | Rate limiting, data gaps on weekends/holidays |

### Red Flags
| Location | Issue | Impact |
|---|---|---|
| Yahoo session | Cookie + crumb auth, EU consent redirect | Must handle or prices fail |
| Weekend/holiday dates | No trading data | Must return nearest trading day or error clearly |

### Questions from Code
- (resolved) Coinbase needed? No — Yahoo covers crypto via BTC-USD etc.

## Certainty Map

### Known-Knowns
| Fact | Source | Confidence |
|---|---|---|
| Yahoo v7/quote returns realtime prices | Ticker fork code | High |
| Yahoo v8/chart returns historical OHLCV | Yahoo Finance docs | High |
| Session needs cookies + crumb token | Ticker fork session code | High |
| Yahoo symbols: BTC-USD, GC=F, EURUSD=X, ^GSPC | Yahoo Finance | High |
| Go + Cobra is proven for CLI tools | Ticker fork | High |

### Known-Unknowns
| Question | Impact if Wrong | Resolution |
|---|---|---|
| Yahoo rate limits for burst queries? | Could get 429s | Test with 10+ symbols, add retry |
| v7/v8 endpoints still stable in 2026? | Build breaks | Test at build time, version pin nothing |
| EU consent flow still required? | Auth fails for EU users | Implement like ticker fork |

### Unknown-Unknowns
| Risk Area | Why Risky | Mitigation |
|---|---|---|
| Yahoo API changes without notice | Undocumented API | Clean abstraction layer, easy to swap |
| Symbol format edge cases | Weird suffixes (.PA, =X, =F, ^prefix) | Pass through as-is, Yahoo handles it |

### Assumptions
| Assumption | If Wrong | Validation |
|---|---|---|
| Single HTTP request per symbol batch | Need multiple calls | Test with 20+ symbols |
| v8/chart supports date range queries | Need different endpoint | Test with period1/period2 params |
| No API key needed (cookie auth only) | Need registration | Verify with fresh session |

## Scope

### In (v1)
- Current price query: `ticker-check AAPL SLB BTC-USD`
- Historical date: `--date 2026-03-20`
- Range history: `--range 1d|5d|1mo|3mo|6mo|1y|ytd`
- Weekly change: `--weekly-change`
- YTD change: `--ytd`
- Output formats: table (default), `--format json|csv`
- Compact mode: `--compact` (one JSON line per symbol, for piping)
- Clean error messages per symbol (not crash on one bad symbol)
- Exit code 0 on success, 1 on partial failure, 2 on total failure

### Out (Not v1)
- TUI / interactive mode — not the goal
- Portfolio tracking / lots / cost basis — different tool
- Websocket streaming — one-shot only
- Coinbase integration — Yahoo covers crypto
- Config file — CLI args only, stateless
- Currency conversion — show native currency
- Sorting options — output order = input order

### Future (v2+)
- `--watch N` for polling every N seconds
- `--compare AAPL,MSFT` side-by-side
- `--alerts` price threshold notifications
- Config file for default watchlists

### Anti-Goals
- Not a TUI
- Not a portfolio tracker
- Not a trading tool
- No persistent state

## Edge Cases
| Scenario | Expected | Severity |
|---|---|---|
| Invalid symbol (ZZZZZZ) | Error line for that symbol, others still shown | P1 |
| Weekend/holiday --date | Return last trading day's close, note it | P1 |
| Market closed (current price) | Show last close + market state "CLOSED" | P2 |
| No network | Clear error message, exit 2 | P1 |
| Yahoo session fails | Retry once, then error | P1 |
| --date in the future | Error: date in the future | P2 |
| --date before symbol existed | Error or empty for that symbol | P3 |
| 50+ symbols in one call | Batch if needed (Yahoo limit ~50?) | P2 |
| Crypto on --date (trades 24/7) | Return exact date close | P3 |

## Failure Modes
| What Fails | User Sees | Recovery |
|---|---|---|
| Yahoo 429 rate limit | "Rate limited, retry in Ns" | Exponential backoff, 1 retry |
| Yahoo auth/crumb fails | "Session error, retrying..." | Re-init session, 1 retry |
| Network timeout | "Network error: timeout" | Exit 2 |
| Invalid symbol | "ZZZZZZ: symbol not found" on stderr | Continue with valid symbols |
| Yahoo API format change | Unexpected JSON structure | Error with raw response hint |

## Success Criteria

### Functional (Must)
- `ticker-check AAPL` returns current price, change, change%, market state
- `ticker-check --date 2026-03-20 AAPL` returns close price at that date
- `ticker-check --range 5d AAPL` returns daily OHLCV for 5 days
- `ticker-check --weekly-change AAPL` returns last week's % change
- `ticker-check --ytd AAPL` returns YTD % change
- `--format json` outputs valid JSON to stdout
- `--format csv` outputs valid CSV to stdout
- `--compact` outputs one JSON object per line
- Errors go to stderr, data to stdout
- Works with stocks, crypto, forex, commodities, indices

### Quality (Should)
- Response time < 2s for 10 symbols
- Binary size < 15MB
- Zero runtime dependencies
- Clean `--help` output

### User Outcome
- LLM agent can call `ticker-check --format json AAPL BTC-USD GC=F` and parse structured output
- Human can pipe `ticker-check --compact AAPL | jq .price`

### Demo Script
```
# Current prices
$ ticker-check AAPL BTC-USD GC=F
AAPL    Apple Inc          $178.52  +1.23  +0.69%  REGULAR
BTC-USD Bitcoin USD      $84,231.00  -520.00  -0.61%  REGULAR
GC=F    Gold Futures      $3,043.20  +12.40  +0.41%  REGULAR

# Historical
$ ticker-check --date 2026-03-20 AAPL --format json
[{"symbol":"AAPL","date":"2026-03-20","close":177.29,"open":176.50,"high":178.10,"low":176.00,"volume":54230100}]

# Weekly change
$ ticker-check --weekly-change AAPL SLB --format csv
symbol,name,price,weekly_change,weekly_change_pct
AAPL,Apple Inc,178.52,2.30,1.31
SLB,Schlumberger,42.10,-0.85,-1.98

# Compact for piping
$ ticker-check --compact AAPL | jq .changePercent
0.69
```

## Open Questions
None blocking.

## Decisions
| Decision | Rationale | Reversible? |
|---|---|---|
| Go + Cobra | Single binary, proven pattern from ticker fork | Yes |
| Yahoo only, no Coinbase | Yahoo covers all asset classes needed | Yes |
| v7 for current + v8 for historical | Minimal endpoints, max coverage | Yes |
| No config file | Stateless CLI, LLM-friendly | Yes |
| Errors to stderr | Clean stdout for piping/parsing | No (convention) |
| Input order = output order | Predictable for LLM parsing | Yes |

## Changelog
- 2026-03-22: Created, Phase 0-5 (greenfield, fast track)
