package tools

import (
	"context"
	"errors"
	"math"
	"strings"
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
)

// --- helpers ---

func makeRateHistory(rates []float64, baseTimeMs int64, intervalMs int64) []ccxt.FundingRateHistory {
	out := make([]ccxt.FundingRateHistory, len(rates))
	for i, r := range rates {
		r := r
		ts := baseTimeMs + int64(i)*intervalMs
		sym := "BTC/USDT:USDT"
		out[i] = ccxt.FundingRateHistory{
			Symbol:      &sym,
			Timestamp:   &ts,
			FundingRate: &r,
		}
	}
	return out
}

// --- tool metadata ---

func TestFundingRateHistoryTool_Metadata(t *testing.T) {
	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	if tool.Name() != NameFundingRateHistory {
		t.Errorf("Name() = %q, want %q", tool.Name(), NameFundingRateHistory)
	}
	if tool.Description() == "" {
		t.Error("Description() must not be empty")
	}
	params := tool.Parameters()
	if params["type"] != "object" {
		t.Errorf("Parameters()[type] = %v, want object", params["type"])
	}
	props, ok := params["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters()[properties] must be map[string]any")
	}
	for _, key := range []string{"provider", "symbol", "account", "limit", "since"} {
		if _, ok := props[key]; !ok {
			t.Errorf("Parameters()[properties] missing key %q", key)
		}
	}
	required, _ := params["required"].([]string)
	if len(required) != 2 {
		t.Errorf("required has %d elements, want 2", len(required))
	}
}

// --- Execute: validation ---

func TestFundingRateHistoryTool_MissingProvider(t *testing.T) {
	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{
		"symbol": "BTC/USDT:USDT",
	})
	if !res.IsError {
		t.Fatal("expected error for missing provider")
	}
	if !strings.Contains(res.ForLLM, "provider and symbol are required") {
		t.Errorf("unexpected error: %s", res.ForLLM)
	}
}

func TestFundingRateHistoryTool_MissingSymbol(t *testing.T) {
	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
	})
	if !res.IsError {
		t.Fatal("expected error for missing symbol")
	}
}

func TestFundingRateHistoryTool_ProviderError(t *testing.T) {
	orig := futuresProviderFn
	futuresProviderFn = func(_ context.Context, _ *config.Config, _, _ string) (broker.FuturesProvider, error) {
		return nil, errors.New("provider not configured")
	}
	defer func() { futuresProviderFn = orig }()

	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "okx",
		"symbol":   "BTC/USDT:USDT",
	})
	if !res.IsError {
		t.Fatal("expected error when provider fails")
	}
}

func TestFundingRateHistoryTool_FetchError(t *testing.T) {
	mp := &mockFuturesProvider{}
	mp.fetchPublicFundingRateHistoryFn = func(_ context.Context, _ string, _ *int64, _ int) ([]ccxt.FundingRateHistory, error) {
		return nil, errors.New("exchange unavailable")
	}
	restore := injectMockFuturesProvider(mp)
	defer restore()

	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT:USDT",
	})
	if !res.IsError {
		t.Fatal("expected error when fetch fails")
	}
	if !strings.Contains(res.ForLLM, "exchange unavailable") {
		t.Errorf("unexpected error message: %s", res.ForLLM)
	}
}

func TestFundingRateHistoryTool_EmptyHistory(t *testing.T) {
	mp := &mockFuturesProvider{}
	mp.fetchPublicFundingRateHistoryFn = func(_ context.Context, _ string, _ *int64, _ int) ([]ccxt.FundingRateHistory, error) {
		return nil, nil
	}
	restore := injectMockFuturesProvider(mp)
	defer restore()

	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT:USDT",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.ForLLM)
	}
	if !strings.Contains(res.ForUser, "No funding rate history") {
		t.Errorf("expected empty-history message, got: %s", res.ForUser)
	}
}

func TestFundingRateHistoryTool_SuccessPath(t *testing.T) {
	// 30 records spaced 8h apart ending ~now
	interval := int64(8 * time.Hour / time.Millisecond)
	baseMs := time.Now().Add(-30*8*time.Hour).UnixMilli()
	rates := make([]float64, 30)
	for i := range rates {
		rates[i] = 0.0001 * float64(i+1)
	}
	history := makeRateHistory(rates, baseMs, interval)

	mp := &mockFuturesProvider{}
	mp.fetchPublicFundingRateHistoryFn = func(_ context.Context, _ string, _ *int64, _ int) ([]ccxt.FundingRateHistory, error) {
		return history, nil
	}
	restore := injectMockFuturesProvider(mp)
	defer restore()

	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT:USDT",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.ForLLM)
	}
	for _, want := range []string{"Statistics", "Recent Records", "Ann. Rate", "BTC/USDT:USDT"} {
		if !strings.Contains(res.ForUser, want) {
			t.Errorf("output missing %q\n---\n%s", want, res.ForUser)
		}
	}
}

func TestFundingRateHistoryTool_WithSince(t *testing.T) {
	interval := int64(8 * time.Hour / time.Millisecond)
	baseMs := time.Now().Add(-14*24*time.Hour).UnixMilli()
	history := makeRateHistory([]float64{0.0001, 0.0002, 0.0003}, baseMs, interval)

	mp := &mockFuturesProvider{}
	mp.fetchPublicFundingRateHistoryFn = func(_ context.Context, _ string, since *int64, _ int) ([]ccxt.FundingRateHistory, error) {
		if since == nil {
			t.Error("since should be set when 'since' arg is provided")
		}
		return history, nil
	}
	restore := injectMockFuturesProvider(mp)
	defer restore()

	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	res := tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT:USDT",
		"since":    "14d",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", res.ForLLM)
	}
}

func TestFundingRateHistoryTool_LimitClamping(t *testing.T) {
	called := false
	mp := &mockFuturesProvider{}
	mp.fetchPublicFundingRateHistoryFn = func(_ context.Context, _ string, _ *int64, limit int) ([]ccxt.FundingRateHistory, error) {
		called = true
		if limit != defaultFundingRateHistoryLimit {
			t.Errorf("limit = %d, want %d (clamped default)", limit, defaultFundingRateHistoryLimit)
		}
		return nil, nil
	}
	restore := injectMockFuturesProvider(mp)
	defer restore()

	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	// limit=0 should clamp to default
	tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT:USDT",
		"limit":    float64(0),
	})
	if !called {
		t.Error("fetch was not called")
	}

	// limit > max should also clamp to default
	called = false
	mp.fetchPublicFundingRateHistoryFn = func(_ context.Context, _ string, _ *int64, limit int) ([]ccxt.FundingRateHistory, error) {
		called = true
		if limit != defaultFundingRateHistoryLimit {
			t.Errorf("over-max limit = %d, want %d (clamped default)", limit, defaultFundingRateHistoryLimit)
		}
		return nil, nil
	}
	tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTC/USDT:USDT",
		"limit":    float64(maxFundingRateHistoryLimit + 1),
	})
	if !called {
		t.Error("fetch was not called for over-max limit")
	}
}

func TestFundingRateHistoryTool_SymbolNormalization(t *testing.T) {
	var capturedSymbol string
	mp := &mockFuturesProvider{}
	mp.fetchPublicFundingRateHistoryFn = func(_ context.Context, symbol string, _ *int64, _ int) ([]ccxt.FundingRateHistory, error) {
		capturedSymbol = symbol
		return nil, nil
	}
	restore := injectMockFuturesProvider(mp)
	defer restore()

	tool := NewFundingRateHistoryTool(config.DefaultConfig())
	tool.Execute(context.Background(), map[string]any{
		"provider": "binance",
		"symbol":   "BTCUSDT",
	})
	if capturedSymbol != "BTC/USDT:USDT" {
		t.Errorf("symbol not normalized: got %q, want BTC/USDT:USDT", capturedSymbol)
	}
}

// --- computeFundingStats ---

func TestComputeFundingStats_Empty(t *testing.T) {
	s := computeFundingStats("test", nil)
	if s.count != 0 || s.mean != 0 || s.stddev != 0 {
		t.Errorf("empty records should give zero stats: %+v", s)
	}
}

func TestComputeFundingStats_AllNilRates(t *testing.T) {
	records := []ccxt.FundingRateHistory{{FundingRate: nil}, {FundingRate: nil}}
	s := computeFundingStats("test", records)
	if s.count != 2 {
		t.Errorf("count = %d, want 2", s.count)
	}
	if s.mean != 0 {
		t.Errorf("mean = %f, want 0 for nil rates", s.mean)
	}
}

func TestComputeFundingStats_SingleRecord(t *testing.T) {
	rate := 0.0005
	records := []ccxt.FundingRateHistory{{FundingRate: &rate}}
	s := computeFundingStats("test", records)
	if s.count != 1 {
		t.Errorf("count = %d, want 1", s.count)
	}
	if s.mean != rate {
		t.Errorf("mean = %f, want %f", s.mean, rate)
	}
	if s.max != rate || s.min != rate {
		t.Errorf("max/min mismatch: max=%f min=%f want %f", s.max, s.min, rate)
	}
	if s.stddev != 0 {
		t.Errorf("stddev = %f, want 0 for single record", s.stddev)
	}
}

func TestComputeFundingStats_MultipleRecords(t *testing.T) {
	rates := []float64{0.0001, 0.0002, 0.0003, 0.0004, 0.0005}
	records := make([]ccxt.FundingRateHistory, len(rates))
	for i, r := range rates {
		r := r
		records[i] = ccxt.FundingRateHistory{FundingRate: &r}
	}
	s := computeFundingStats("test", records)

	wantMean := 0.0003
	if math.Abs(s.mean-wantMean) > 1e-10 {
		t.Errorf("mean = %f, want %f", s.mean, wantMean)
	}
	if s.max != 0.0005 {
		t.Errorf("max = %f, want 0.0005", s.max)
	}
	if s.min != 0.0001 {
		t.Errorf("min = %f, want 0.0001", s.min)
	}
	if s.stddev <= 0 {
		t.Errorf("stddev = %f, want > 0", s.stddev)
	}
	if s.count != 5 {
		t.Errorf("count = %d, want 5", s.count)
	}
}

func TestComputeFundingStats_NegativeRates(t *testing.T) {
	rates := []float64{-0.0003, -0.0001, 0.0002}
	records := make([]ccxt.FundingRateHistory, len(rates))
	for i, r := range rates {
		r := r
		records[i] = ccxt.FundingRateHistory{FundingRate: &r}
	}
	s := computeFundingStats("test", records)
	if s.max != 0.0002 {
		t.Errorf("max = %f, want 0.0002", s.max)
	}
	if s.min != -0.0003 {
		t.Errorf("min = %f, want -0.0003", s.min)
	}
	if s.mean >= 0 {
		t.Errorf("mean = %f, want negative", s.mean)
	}
}

// --- formatFundingRateStats ---

func TestFormatFundingRateStats_NilTimestamp(t *testing.T) {
	rate := 0.0001
	records := []ccxt.FundingRateHistory{{FundingRate: &rate}} // no Timestamp
	out := formatFundingRateStats("binance", "BTC/USDT:USDT", records)
	if !strings.Contains(out, "Statistics") {
		t.Errorf("output missing Statistics section: %s", out)
	}
	if !strings.Contains(out, "Recent Records") {
		t.Errorf("output missing Recent Records section: %s", out)
	}
	if !strings.Contains(out, "-") {
		t.Errorf("output should show '-' for nil timestamp: %s", out)
	}
}

func TestFormatFundingRateStats_NilFundingRate(t *testing.T) {
	ts := time.Now().UnixMilli()
	records := []ccxt.FundingRateHistory{{Timestamp: &ts}} // no FundingRate
	out := formatFundingRateStats("okx", "ETH/USDT:USDT", records)
	// Should not panic, should show '-' for rate
	if !strings.Contains(out, "-") {
		t.Errorf("output should show '-' for nil rate: %s", out)
	}
}

func TestFormatFundingRateStats_WindowsNoData(t *testing.T) {
	// Records from 60 days ago — no window (3d/7d/14d) should match
	rate := 0.0001
	ts := time.Now().Add(-60 * 24 * time.Hour).UnixMilli()
	records := []ccxt.FundingRateHistory{{FundingRate: &rate, Timestamp: &ts}}
	out := formatFundingRateStats("binance", "BTC/USDT:USDT", records)
	if !strings.Contains(out, "(no data)") {
		t.Errorf("expected '(no data)' for empty windows, got: %s", out)
	}
}

func TestFormatFundingRateStats_MoreThan10Records(t *testing.T) {
	// 20 recent records — recent section should show exactly 10
	interval := int64(8 * time.Hour / time.Millisecond)
	baseMs := time.Now().Add(-20 * 8 * time.Hour).UnixMilli()
	history := makeRateHistory(make([]float64, 20), baseMs, interval)
	out := formatFundingRateStats("binance", "BTC/USDT:USDT", history)
	if !strings.Contains(out, "20 records") {
		t.Errorf("header should show total count: %s", out)
	}
	// Check we have at most 10 data lines in the recent section (lines starting with a year digit).
	recentIdx := strings.Index(out, "=== Recent Records")
	if recentIdx < 0 {
		t.Fatal("missing Recent Records section")
	}
	recentSection := out[recentIdx:]
	dataLines := 0
	for _, l := range strings.Split(recentSection, "\n") {
		if len(l) > 0 && l[0] == '2' { // timestamp lines start with year '2...'
			dataLines++
		}
	}
	if dataLines > 10 {
		t.Errorf("recent section has %d data lines, want ≤10", dataLines)
	}
}

func TestFormatFundingRateStats_AnnualizedRate(t *testing.T) {
	rate := 0.0001
	ts := time.Now().Add(-1 * time.Hour).UnixMilli()
	records := []ccxt.FundingRateHistory{{FundingRate: &rate, Timestamp: &ts}}
	out := formatFundingRateStats("binance", "BTC/USDT:USDT", records)
	// annualized = 0.0001 * 3 * 365 * 100% = 10.95%
	if !strings.Contains(out, "10.9500%") && !strings.Contains(out, "Ann. Rate") {
		t.Errorf("output should contain annualized rate: %s", out)
	}
}

func TestFormatFundingRateStats_SortsByTimestamp(t *testing.T) {
	// Feed records out-of-order: most recent first
	now := time.Now()
	r1, r2, r3 := 0.0001, 0.0002, 0.0003
	t1 := now.Add(-24 * time.Hour).UnixMilli()
	t2 := now.Add(-16 * time.Hour).UnixMilli()
	t3 := now.Add(-8 * time.Hour).UnixMilli()
	records := []ccxt.FundingRateHistory{
		{FundingRate: &r3, Timestamp: &t3},
		{FundingRate: &r1, Timestamp: &t1},
		{FundingRate: &r2, Timestamp: &t2},
	}
	out := formatFundingRateStats("binance", "BTC/USDT:USDT", records)
	// Recent records section should show t3 first (newest → oldest display)
	recentIdx := strings.Index(out, "=== Recent Records")
	if recentIdx < 0 {
		t.Fatal("missing Recent Records section")
	}
	// The function shows newest first in the recent block
	time1Str := time.UnixMilli(t3).UTC().Format("2006-01-02 15:04:05")
	if !strings.Contains(out[recentIdx:], time1Str) {
		t.Errorf("recent section should contain t3 timestamp %s: %s", time1Str, out)
	}
}
