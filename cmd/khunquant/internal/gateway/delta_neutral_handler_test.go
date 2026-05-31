package gateway

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"

	"github.com/cryptoquantumwave/khunquant/pkg/bus"
	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/cron"
	"github.com/cryptoquantumwave/khunquant/pkg/deltaneutral"
	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
	"github.com/cryptoquantumwave/khunquant/pkg/tools"
)

// monitorSpotProvider is a minimal spot provider for monitor handler tests.
// It implements MarketDataProvider, PortfolioProvider, and EarnProvider.
type monitorSpotProvider struct {
	tickerPrice   float64
	tickerErr     error
	balances      []broker.Balance
	balanceErr    error
	earnPositions []broker.EarnPosition
	earnErr       error
}

func (m *monitorSpotProvider) ID() string                     { return "mockspot" }
func (m *monitorSpotProvider) Category() broker.AssetCategory { return broker.CategoryCrypto }
func (m *monitorSpotProvider) GetMarketStatus(_ context.Context, _ string) (broker.MarketStatus, error) {
	return broker.MarketOpen, nil
}

// MarketDataProvider
func (m *monitorSpotProvider) FetchTicker(_ context.Context, _ string) (ccxt.Ticker, error) {
	if m.tickerErr != nil {
		return ccxt.Ticker{}, m.tickerErr
	}
	p := m.tickerPrice
	return ccxt.Ticker{Last: &p}, nil
}
func (m *monitorSpotProvider) FetchTickers(_ context.Context, _ []string) (map[string]ccxt.Ticker, error) {
	return nil, nil
}
func (m *monitorSpotProvider) FetchOHLCV(_ context.Context, _, _ string, _ *int64, _ int) ([]ccxt.OHLCV, error) {
	return nil, nil
}
func (m *monitorSpotProvider) FetchOrderBook(_ context.Context, _ string, _ int) (ccxt.OrderBook, error) {
	return ccxt.OrderBook{}, nil
}
func (m *monitorSpotProvider) LoadMarkets(_ context.Context) (map[string]ccxt.MarketInterface, error) {
	return nil, nil
}

// PortfolioProvider
func (m *monitorSpotProvider) GetBalances(_ context.Context) ([]broker.Balance, error) {
	return m.balances, m.balanceErr
}
func (m *monitorSpotProvider) GetWalletBalances(_ context.Context, _ string) ([]broker.WalletBalance, error) {
	return nil, nil
}
func (m *monitorSpotProvider) FetchPrice(_ context.Context, _, _ string) (float64, error) {
	return m.tickerPrice, m.tickerErr
}
func (m *monitorSpotProvider) SupportedWalletTypes() []string { return nil }

// EarnProvider
func (m *monitorSpotProvider) FetchFlexibleEarnProducts(_ context.Context, _ string) ([]broker.EarnProduct, error) {
	return nil, nil
}
func (m *monitorSpotProvider) FetchFlexibleEarnPositions(_ context.Context) ([]broker.EarnPosition, error) {
	return m.earnPositions, m.earnErr
}
func (m *monitorSpotProvider) SubscribeFlexibleEarn(_ context.Context, _, _ string, _ float64, _ bool) (string, error) {
	return "", nil
}
func (m *monitorSpotProvider) RedeemFlexibleEarn(_ context.Context, _, _ string, _ float64, _ bool) (string, error) {
	return "", nil
}
func (m *monitorSpotProvider) SetFlexibleAutoSubscribe(_ context.Context, _, _ string, _ bool) error {
	return nil
}
func (m *monitorSpotProvider) FetchFlexibleEarnRateHistory(_ context.Context, _, _ string, _ *int64, _ int) ([]broker.EarnRatePoint, error) {
	return nil, nil
}

// seedMonitorPlan creates an active plan for the monitor test and returns the plan ID + job name.
func seedMonitorPlan(t *testing.T, store *deltaneutral.Store, spotProvider string) (int64, string) {
	t.Helper()
	now := time.Now().UTC()
	plan := &deltaneutral.Plan{
		Name:                "monitor-test",
		Asset:               "CHZ",
		Status:              deltaneutral.PlanStatusActive,
		Mode:                deltaneutral.ExecutionModeApproval,
		SpotProvider:        spotProvider,
		SpotSymbol:          "CHZ/USDT",
		SpotSide:            "buy",
		FuturesProvider:     "okx",
		FuturesSymbol:       "CHZ/USDT:USDT",
		FuturesSide:         "short",
		FuturesLeverage:     2,
		FuturesMarginMode:   "cross",
		CapitalUSDT:         100,
		SpotNotionalUSDT:    44.44,
		FuturesNotionalUSDT: 44.44,
		MonitorInterval:     "5m",
		Enabled:             true,
		RiskPolicy:          deltaneutral.DefaultRiskPolicy(),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	id, err := store.SavePlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("seedMonitorPlan: %v", err)
	}
	return id, "dn:" + itoa(id) + ":monitor-test"
}

func itoa(n int64) string {
	return strconv64(n)
}

func strconv64(n int64) string {
	buf := make([]byte, 0, 20)
	if n == 0 {
		return "0"
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

func newMonitorStore(t *testing.T) *deltaneutral.Store {
	t.Helper()
	s, err := deltaneutral.NewStore(filepath.Join(t.TempDir(), "dn.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

// runMonitor calls handleDeltaNeutralMonitorJob with the given spot provider registered
// under "mockspot". Returns the snapshot saved after the run (if any).
func runMonitor(t *testing.T, store *deltaneutral.Store, planID int64, jobName string, spot *monitorSpotProvider) *deltaneutral.MonitorSnapshot {
	t.Helper()

	// Register mockspot so CreateProviderForAccount resolves it.
	broker.RegisterFactory("mockspot", func(_ *config.Config) (broker.Provider, error) {
		return spot, nil
	})
	t.Cleanup(func() { broker.RegisterFactory("mockspot", nil) })

	cfg := &config.Config{}
	_ = cron.NewCronService(filepath.Join(t.TempDir(), "cron.json"), nil)
	job := &cron.CronJob{Name: jobName}
	msgBus := bus.NewMessageBus()

	_, err := handleDeltaNeutralMonitorJob(context.Background(), job, cfg, store, (*tools.CronTool)(nil), msgBus)
	if err != nil {
		t.Logf("handleDeltaNeutralMonitorJob returned error (may be expected): %v", err)
	}

	// Fetch the snapshot written by the handler.
	snaps, sErr := store.ListSnapshots(context.Background(), planID, 1, 0)
	if sErr != nil || len(snaps) == 0 {
		return nil
	}
	s := snaps[0]
	return &s
}

// TestMonitorSpotBalance_WalletOnly verifies that trading-wallet balance drives spotState.
func TestMonitorSpotBalance_WalletOnly(t *testing.T) {
	store := newMonitorStore(t)
	id, jobName := seedMonitorPlan(t, store, "mockspot")

	spot := &monitorSpotProvider{
		tickerPrice: 0.033,
		balances:    []broker.Balance{{Asset: "CHZ", Free: 1350.0}},
		earnErr:     errFakeEarnUnavailable,
	}

	snap := runMonitor(t, store, id, jobName, spot)
	if snap == nil {
		t.Fatal("expected a snapshot to be saved")
	}
	// spot value = 1350 × 0.033 = 44.55
	if snap.SpotValueUSDT < 44.0 || snap.SpotValueUSDT > 45.0 {
		t.Errorf("expected spot value ~44.55, got %.4f", snap.SpotValueUSDT)
	}
}

// TestMonitorSpotBalance_EarnOnly verifies that earn-wallet balance counts as spot leg.
func TestMonitorSpotBalance_EarnOnly(t *testing.T) {
	store := newMonitorStore(t)
	id, jobName := seedMonitorPlan(t, store, "mockspot")

	spot := &monitorSpotProvider{
		tickerPrice: 0.033,
		balances:    []broker.Balance{{Asset: "CHZ", Free: 0.0000035}}, // effectively zero in trading wallet
		earnPositions: []broker.EarnPosition{
			{Asset: "CHZ", Amount: 1349.5},
		},
	}

	snap := runMonitor(t, store, id, jobName, spot)
	if snap == nil {
		t.Fatal("expected a snapshot to be saved")
	}
	// spot value = (0.0000035 + 1349.5) × 0.033 ≈ 44.53
	if snap.SpotValueUSDT < 44.0 || snap.SpotValueUSDT > 45.0 {
		t.Errorf("expected spot value ~44.53, got %.4f", snap.SpotValueUSDT)
	}
}

// TestMonitorSpotBalance_Split verifies wallet + earn are summed.
func TestMonitorSpotBalance_Split(t *testing.T) {
	store := newMonitorStore(t)
	id, jobName := seedMonitorPlan(t, store, "mockspot")

	spot := &monitorSpotProvider{
		tickerPrice: 0.033,
		balances:    []broker.Balance{{Asset: "CHZ", Free: 500.0}},
		earnPositions: []broker.EarnPosition{
			{Asset: "CHZ", Amount: 850.0},
		},
	}

	snap := runMonitor(t, store, id, jobName, spot)
	if snap == nil {
		t.Fatal("expected a snapshot to be saved")
	}
	// spot value = (500 + 850) × 0.033 = 44.55
	if snap.SpotValueUSDT < 44.0 || snap.SpotValueUSDT > 45.5 {
		t.Errorf("expected spot value ~44.55, got %.4f", snap.SpotValueUSDT)
	}
}

// TestMonitorSpotBalance_FallbackOnAPIFailure verifies plan notional is used when both fail.
func TestMonitorSpotBalance_FallbackOnAPIFailure(t *testing.T) {
	store := newMonitorStore(t)
	id, jobName := seedMonitorPlan(t, store, "mockspot")

	spot := &monitorSpotProvider{
		tickerPrice: 0.033,
		balanceErr:  errFakeEarnUnavailable,
		earnErr:     errFakeEarnUnavailable,
	}

	snap := runMonitor(t, store, id, jobName, spot)
	if snap == nil {
		t.Fatal("expected a snapshot to be saved")
	}
	// fallback: spot value = plan.SpotNotionalUSDT = 44.44
	if snap.SpotValueUSDT < 44.0 || snap.SpotValueUSDT > 45.0 {
		t.Errorf("expected fallback spot value ~44.44, got %.4f", snap.SpotValueUSDT)
	}
}

var errFakeEarnUnavailable = fmt.Errorf("earn API not available in test")
