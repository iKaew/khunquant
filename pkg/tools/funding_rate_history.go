package tools

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

const (
	defaultFundingRateHistoryLimit = 200
	maxFundingRateHistoryLimit     = 500
	fundingPerDay                  = 3.0 // typical: every 8 hours
)

type FundingRateHistoryTool struct{ cfg *config.Config }

func NewFundingRateHistoryTool(cfg *config.Config) *FundingRateHistoryTool {
	return &FundingRateHistoryTool{cfg: cfg}
}

func (t *FundingRateHistoryTool) Name() string { return NameFundingRateHistory }

func (t *FundingRateHistoryTool) Description() string {
	return DescFundingRateHistory
}

func (t *FundingRateHistoryTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"provider": map[string]any{
				"type":        "string",
				"description": "Exchange provider name: 'binance' or 'okx'.",
			},
			"account": map[string]any{
				"type":        "string",
				"description": "Account name within the provider. Leave empty to use the default account.",
			},
			"symbol": map[string]any{
				"type":        "string",
				"description": "Futures symbol in CCXT format, e.g. 'BTC/USDT:USDT'.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("Number of funding rate records to fetch (max %d, default %d).", maxFundingRateHistoryLimit, defaultFundingRateHistoryLimit),
			},
			"since": map[string]any{
				"type":        "string",
				"description": "Start time for history, e.g. '30d', '7d', or '2026-01-01'.",
			},
		},
		"required": []string{"provider", "symbol"},
	}
}

func (t *FundingRateHistoryTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	providerID := stringArg(args, "provider")
	account := stringArg(args, "account")
	symbol := normalizeFuturesSymbol(stringArg(args, "symbol"))
	limit := int(numberArg(args, "limit"))
	if limit <= 0 || limit > maxFundingRateHistoryLimit {
		limit = defaultFundingRateHistoryLimit
	}

	var since *int64
	if s := stringArg(args, "since"); s != "" {
		if ts := parseTimeParam(s); ts != nil {
			ms := ts.UnixMilli()
			since = &ms
		}
	}

	if providerID == "" || symbol == "" {
		return ErrorResult("provider and symbol are required")
	}

	fp, err := futuresProvider(ctx, t.cfg, providerID, account)
	if err != nil {
		return ErrorResult(err.Error()).WithError(err)
	}

	history, err := fp.FetchPublicFundingRateHistory(ctx, symbol, since, limit)
	if err != nil {
		return ErrorResult(fmt.Sprintf("funding_rate_history: %v", err)).WithError(err)
	}
	if len(history) == 0 {
		return UserResult(fmt.Sprintf("No funding rate history found for %s on %s.", symbol, providerID))
	}

	out := formatFundingRateStats(providerID, symbol, history)
	return UserResult(out)
}

// fundingStatWindow holds statistics for a time window.
type fundingStatWindow struct {
	label  string
	mean   float64
	stddev float64
	max    float64
	min    float64
	count  int
}

func formatFundingRateStats(providerID, symbol string, history []ccxt.FundingRateHistory) string {
	// Sort ascending by timestamp.
	sorted := make([]ccxt.FundingRateHistory, len(history))
	copy(sorted, history)
	sort.Slice(sorted, func(i, j int) bool {
		ti := int64(0)
		if sorted[i].Timestamp != nil {
			ti = *sorted[i].Timestamp
		}
		tj := int64(0)
		if sorted[j].Timestamp != nil {
			tj = *sorted[j].Timestamp
		}
		return ti < tj
	})

	now := time.Now().UTC()
	windows := []struct {
		label string
		dur   time.Duration
	}{
		{"3d", 3 * 24 * time.Hour},
		{"7d", 7 * 24 * time.Hour},
		{"14d", 14 * 24 * time.Hour},
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Funding Rate History: %s on %s (%d records)\n\n", symbol, providerID, len(history)))

	// All-time stats from the fetched window.
	allStats := computeFundingStats("all (fetched)", sorted)
	sb.WriteString("=== Statistics ===\n")
	sb.WriteString(fmt.Sprintf("%-14s %8s %10s %10s %10s %8s %12s\n",
		"Window", "Count", "Mean", "Max", "Min", "Std Dev", "Ann. Rate"))
	sb.WriteString(strings.Repeat("-", 78) + "\n")

	printStat := func(stat fundingStatWindow) {
		annualized := stat.mean * fundingPerDay * 365
		sb.WriteString(fmt.Sprintf("%-14s %8d %10.6f%% %10.6f%% %10.6f%% %8.6f%% %11.4f%%\n",
			stat.label,
			stat.count,
			stat.mean*100,
			stat.max*100,
			stat.min*100,
			stat.stddev*100,
			annualized*100,
		))
	}

	// Rolling windows.
	for _, w := range windows {
		cutoff := now.Add(-w.dur)
		cutoffMs := cutoff.UnixMilli()
		var windowed []ccxt.FundingRateHistory
		for _, r := range sorted {
			if r.Timestamp != nil && *r.Timestamp >= cutoffMs {
				windowed = append(windowed, r)
			}
		}
		if len(windowed) > 0 {
			printStat(computeFundingStats(w.label, windowed))
		} else {
			sb.WriteString(fmt.Sprintf("%-14s (no data)\n", w.label))
		}
	}
	printStat(allStats)

	// Recent records (latest 10).
	recent := sorted
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	sb.WriteString("\n=== Recent Records (latest 10) ===\n")
	sb.WriteString(fmt.Sprintf("%-25s %12s\n", "Time (UTC)", "Rate"))
	sb.WriteString(strings.Repeat("-", 38) + "\n")
	for i := len(recent) - 1; i >= 0; i-- {
		r := recent[i]
		ts := "-"
		if r.Timestamp != nil {
			ts = time.UnixMilli(*r.Timestamp).UTC().Format("2006-01-02 15:04:05")
		}
		rate := "-"
		if r.FundingRate != nil {
			rate = fmt.Sprintf("%+.6f%%", *r.FundingRate*100)
		}
		sb.WriteString(fmt.Sprintf("%-25s %12s\n", ts, rate))
	}

	sb.WriteString("\nNote: Annualized rate assumes 3 funding periods/day (every 8h).\n")
	return sb.String()
}

func computeFundingStats(label string, records []ccxt.FundingRateHistory) fundingStatWindow {
	if len(records) == 0 {
		return fundingStatWindow{label: label}
	}
	rates := make([]float64, 0, len(records))
	for _, r := range records {
		if r.FundingRate != nil {
			rates = append(rates, *r.FundingRate)
		}
	}
	if len(rates) == 0 {
		return fundingStatWindow{label: label, count: len(records)}
	}

	sum := 0.0
	maxR := rates[0]
	minR := rates[0]
	for _, r := range rates {
		sum += r
		if r > maxR {
			maxR = r
		}
		if r < minR {
			minR = r
		}
	}
	mean := sum / float64(len(rates))

	variance := 0.0
	for _, r := range rates {
		d := r - mean
		variance += d * d
	}
	variance /= float64(len(rates))

	return fundingStatWindow{
		label:  label,
		mean:   mean,
		stddev: math.Sqrt(variance),
		max:    maxR,
		min:    minR,
		count:  len(rates),
	}
}
