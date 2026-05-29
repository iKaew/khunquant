---
name: funding-rate-analysis
description: Analyze perpetual futures funding rate history to identify arbitrage opportunities, sentiment trends, and carry trade setups.
---

# Funding Rate Analysis

Use the `funding_rate_history` tool to fetch public funding rate records and computed statistics.

## Workflow

1. Call `funding_rate_history` with provider, symbol, and an appropriate limit (e.g. 200 for ~14d of 8h intervals).
2. Review the statistics table: 3d, 7d, 14d means, max/min extremes, volatility (std dev), and annualized rate.
3. Interpret the data:
   - **Mean > 0**: Longs paying shorts → bullish market bias (positive carry for shorts).
   - **Mean < 0**: Shorts paying longs → bearish market bias (positive carry for longs).
   - **High volatility (std dev)**: Funding rate is unstable; carry trade is risky.
   - **Annualized rate > 20%**: Significant arbitrage premium exists vs. spot borrowing cost.
4. Compare across providers (binance vs. okx) to spot cross-exchange divergences.
5. Summarize findings: current sentiment, carry opportunity, and risk level.

## Tool Reference

- `funding_rate_history` — fetch history + statistics (public, no credentials needed)
- `futures_get_funding` — current funding rate + next expected rate (also public)
- `get_ohlcv` — price history to correlate funding rate spikes with price moves

## Notes

- Funding interval is typically 8h (3×/day). Some exchanges run 4h or 1h intervals.
- The annualized rate in the output assumes 3 periods/day — adjust mentally for non-standard intervals.
- Extreme funding rates (> 0.1% per period) often revert quickly; useful for mean-reversion signals.
