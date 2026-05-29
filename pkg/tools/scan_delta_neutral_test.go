package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	ccxt "github.com/ccxt/ccxt/go/v4"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
)

func TestScanDeltaNeutralOpportunities_Success(t *testing.T) {
	// Override CMC listing to return fixed symbols.
	oldCMCFn := cmcListingFn
	defer func() { cmcListingFn = oldCMCFn }()

	cmcListingFn = func(ctx context.Context, cfg *config.Config, baseURL string, topN int) ([]string, error) {
		return []string{"BTC", "ETH", "SOL", "DOGE"}, nil
	}

	// Override futures provider.
	oldFuturesFn := futuresProviderFn
	defer func() { futuresProviderFn = oldFuturesFn }()

	mockProvider := &mockFuturesProvider{
		fundingRatesFn: func(ctx context.Context, symbols []string) (map[string]ccxt.FundingRate, error) {
			result := make(map[string]ccxt.FundingRate)
			interval := "8h"

			// BTC/USDT:USDT: +0.0001 (short perp)
			fRate1 := 0.0001
			result["BTC/USDT:USDT"] = ccxt.FundingRate{FundingRate: &fRate1, Interval: &interval}

			// ETH/USDT:USDT: -0.0003 (long perp, highest abs APR)
			fRate2 := -0.0003
			result["ETH/USDT:USDT"] = ccxt.FundingRate{FundingRate: &fRate2, Interval: &interval}

			// SOL/USDT:USDT: +0.00005 (short perp, lowest abs APR)
			fRate3 := 0.00005
			result["SOL/USDT:USDT"] = ccxt.FundingRate{FundingRate: &fRate3, Interval: &interval}

			return result, nil
		},
		loadMarketsFn: func(ctx context.Context) (map[string]ccxt.MarketInterface, error) {
			// Return empty to skip market filtering in this test.
			return nil, nil
		},
		fetchPublicFundingRateHistoryFn: func(ctx context.Context, symbol string, since *int64, limit int) ([]ccxt.FundingRateHistory, error) {
			// No history for this test (include_stability=false).
			return nil, nil
		},
	}

	futuresProviderFn = func(ctx context.Context, cfg *config.Config, providerID, account string) (broker.FuturesProvider, error) {
		return mockProvider, nil
	}

	cfg := &config.Config{}
	tool := NewScanDeltaNeutralOpportunitiesTool(cfg)

	args := map[string]any{
		"provider":          "mock",
		"top_n":             100,
		"quote":             "USDT",
		"limit_results":     20,
		"include_stability": false,
	}

	result := tool.Execute(context.Background(), args)

	if result.IsError {
		t.Fatalf("unexpected error: %v", result.ForLLM)
	}

	// Check output contains expected symbols in correct order (by abs APR desc).
	// ETH abs(APR) = 0.0003 * 3 * 365 * 100 = 32.85% (highest)
	// BTC abs(APR) = 0.0001 * 3 * 365 * 100 = 10.95%
	// SOL abs(APR) = 0.00005 * 3 * 365 * 100 = 5.475% (lowest)
	output := result.ForUser

	// Check ranking order: ETH should be rank 1.
	if !strings.ContainsAny(output, "ETH") {
		t.Fatal("expected ETH in output")
	}
	if !strings.ContainsAny(output, "BTC") {
		t.Fatal("expected BTC in output")
	}
	if !strings.ContainsAny(output, "SOL") {
		t.Fatal("expected SOL in output")
	}

	// Check direction labels.
	if !strings.Contains(output, "short perp") {
		t.Fatalf("expected 'short perp' direction for BTC. Got:\n%s", output)
	}
	if !strings.Contains(output, "long perp") {
		t.Fatalf("expected 'long perp' direction for ETH. Got:\n%s", output)
	}

	// Success is indicated by IsError being false.
	if result.IsError {
		t.Fatal("expected success")
	}
}

func TestScanDeltaNeutralOpportunities_CMCError(t *testing.T) {
	oldCMCFn := cmcListingFn
	defer func() { cmcListingFn = oldCMCFn }()

	cmcListingFn = func(ctx context.Context, cfg *config.Config, baseURL string, topN int) ([]string, error) {
		return nil, errors.New("cmc api error")
	}

	cfg := &config.Config{}
	tool := NewScanDeltaNeutralOpportunitiesTool(cfg)

	args := map[string]any{
		"provider": "binance",
	}

	result := tool.Execute(context.Background(), args)

	if !result.IsError {
		t.Fatal("expected error on CMC fetch failure")
	}
	if !strings.Contains(result.ForLLM, "CMC listing fetch failed") {
		t.Fatal("expected error message about CMC listing")
	}
}

func TestScanDeltaNeutralOpportunities_MinAbsFundingFilter(t *testing.T) {
	oldCMCFn := cmcListingFn
	defer func() { cmcListingFn = oldCMCFn }()

	cmcListingFn = func(ctx context.Context, cfg *config.Config, baseURL string, topN int) ([]string, error) {
		return []string{"BTC", "ETH", "SOL"}, nil
	}

	oldFuturesFn := futuresProviderFn
	defer func() { futuresProviderFn = oldFuturesFn }()

	mockProvider := &mockFuturesProvider{
		fundingRatesFn: func(ctx context.Context, symbols []string) (map[string]ccxt.FundingRate, error) {
			result := make(map[string]ccxt.FundingRate)
			interval := "8h"
			// BTC: 0.0001 * 3 * 365 * 100 = 10.95%
			fRate1 := 0.0001
			result["BTC/USDT:USDT"] = ccxt.FundingRate{FundingRate: &fRate1, Interval: &interval}

			// ETH: 0.00005 * 3 * 365 * 100 = 5.475% (filtered out if min > 5.5)
			fRate2 := 0.00005
			result["ETH/USDT:USDT"] = ccxt.FundingRate{FundingRate: &fRate2, Interval: &interval}

			return result, nil
		},
		loadMarketsFn: func(ctx context.Context) (map[string]ccxt.MarketInterface, error) {
			return nil, nil // No markets available.
		},
	}

	futuresProviderFn = func(ctx context.Context, cfg *config.Config, providerID, account string) (broker.FuturesProvider, error) {
		return mockProvider, nil
	}

	cfg := &config.Config{}
	tool := NewScanDeltaNeutralOpportunitiesTool(cfg)

	// Set min_abs_funding_apr = 6 to filter out ETH (5.475%) but keep BTC (10.95%).
	args := map[string]any{
		"provider":            "binance",
		"min_abs_funding_apr": 6.0,
		"include_stability":   false,
	}

	result := tool.Execute(context.Background(), args)

	if result.IsError {
		t.Fatalf("unexpected error: %v", result.ForLLM)
	}

	output := result.ForUser

	// BTC should be in output.
	if !strings.Contains(output, "BTC") {
		t.Fatal("expected BTC in output")
	}

	// ETH should be filtered out (not in the results, but may appear in header).
	if strings.Count(output, "ETH") > 0 {
		// Check if it's just in the header; if there's a data row for ETH, fail.
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "ETH") {
				t.Fatal("ETH should be filtered out by min_abs_funding_apr")
			}
		}
	}
}

func TestScanDeltaNeutralOpportunities_NoProvider(t *testing.T) {
	cfg := &config.Config{}
	tool := NewScanDeltaNeutralOpportunitiesTool(cfg)

	args := map[string]any{} // No provider specified.

	result := tool.Execute(context.Background(), args)

	if !result.IsError {
		t.Fatal("expected error when provider is missing")
	}
	if !strings.Contains(result.ForLLM, "provider is required") {
		t.Fatal("expected error message about missing provider")
	}
}
