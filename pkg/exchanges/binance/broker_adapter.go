package binance

// BinanceBrokerAdapter wraps BinanceExchange to implement the broker provider interfaces.
// It satisfies: broker.PortfolioProvider, broker.MarketDataProvider,
// broker.TradingProvider, broker.TransferProvider.

import (
	"context"
	"fmt"

	ccxt "github.com/ccxt/ccxt/go/v4"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
	"github.com/cryptoquantumwave/khunquant/pkg/logger"
	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
)

// catchPanic converts a CCXT panic (which the library uses instead of returning errors) into a Go error.
func catchPanic(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	return fn()
}

// BinanceBrokerAdapter wraps BinanceExchange with the broker.Provider hierarchy.
type BinanceBrokerAdapter struct {
	*BinanceExchange
}

// newBrokerAdapter creates a BinanceBrokerAdapter from resolved credentials.
func newBrokerAdapter(creds config.ExchangeAccount, testnet bool) (*BinanceBrokerAdapter, error) {
	ex, err := NewBinanceExchange(creds, testnet)
	if err != nil {
		return nil, err
	}
	if creds.APIKey.String() != "" {
		logger.RegisterSecret(creds.APIKey.String())
	}
	if creds.Secret.String() != "" {
		logger.RegisterSecret(creds.Secret.String())
	}
	return &BinanceBrokerAdapter{BinanceExchange: ex}, nil
}

// --- broker.Provider ---

func (a *BinanceBrokerAdapter) ID() string { return Name }

func (a *BinanceBrokerAdapter) Category() broker.AssetCategory { return broker.CategoryCrypto }

func (a *BinanceBrokerAdapter) GetMarketStatus(_ context.Context, symbol string) (broker.MarketStatus, error) {
	ticker, err := a.publicSpot.FetchTicker(symbol)
	if err != nil {
		return broker.MarketUnknown, fmt.Errorf("binance: GetMarketStatus: %w", err)
	}
	if ticker.Last != nil {
		return broker.MarketOpen, nil
	}
	return broker.MarketUnknown, nil
}

// --- broker.PortfolioProvider ---

func (a *BinanceBrokerAdapter) GetBalances(ctx context.Context) ([]broker.Balance, error) {
	bals, err := a.BinanceExchange.GetBalances(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]broker.Balance, len(bals))
	for i, b := range bals {
		out[i] = broker.Balance{Asset: b.Asset, Free: b.Free, Locked: b.Locked}
	}
	return out, nil
}

func (a *BinanceBrokerAdapter) GetWalletBalances(ctx context.Context, walletType string) ([]broker.WalletBalance, error) {
	bals, err := a.BinanceExchange.GetWalletBalances(ctx, walletType)
	if err != nil {
		return nil, err
	}
	out := make([]broker.WalletBalance, len(bals))
	for i, b := range bals {
		out[i] = broker.WalletBalance{
			Balance:    broker.Balance{Asset: b.Asset, Free: b.Free, Locked: b.Locked},
			WalletType: b.WalletType,
			Extra:      b.Extra,
		}
	}
	return out, nil
}

func (a *BinanceBrokerAdapter) FetchPrice(ctx context.Context, asset, quote string) (float64, error) {
	return a.BinanceExchange.FetchPrice(ctx, asset, quote)
}

func (a *BinanceBrokerAdapter) SupportedWalletTypes() []string {
	return a.BinanceExchange.SupportedWalletTypes()
}

// --- broker.MarketDataProvider ---

func (a *BinanceBrokerAdapter) FetchTicker(_ context.Context, symbol string) (t ccxt.Ticker, err error) {
	err = catchPanic(func() error { t, err = a.publicSpot.FetchTicker(symbol); return err })
	return
}

func (a *BinanceBrokerAdapter) FetchTickers(_ context.Context, symbols []string) (result map[string]ccxt.Ticker, err error) {
	var tickers ccxt.Tickers
	err = catchPanic(func() error {
		var e error
		if len(symbols) == 0 {
			tickers, e = a.publicSpot.FetchTickers()
		} else {
			tickers, e = a.publicSpot.FetchTickers(ccxt.WithFetchTickersSymbols(symbols))
		}
		return e
	})
	if err != nil {
		return nil, fmt.Errorf("binance: FetchTickers: %w", err)
	}
	return tickers.Tickers, nil
}

func (a *BinanceBrokerAdapter) FetchOHLCV(_ context.Context, symbol, timeframe string, since *int64, limit int) (out []ccxt.OHLCV, err error) {
	opts := []ccxt.FetchOHLCVOptions{ccxt.WithFetchOHLCVTimeframe(timeframe)}
	if since != nil {
		opts = append(opts, ccxt.WithFetchOHLCVSince(*since))
	}
	if limit > 0 {
		opts = append(opts, ccxt.WithFetchOHLCVLimit(int64(limit)))
	}
	err = catchPanic(func() error { out, err = a.publicSpot.FetchOHLCV(symbol, opts...); return err })
	return
}

func (a *BinanceBrokerAdapter) FetchOrderBook(_ context.Context, symbol string, depth int) (ob ccxt.OrderBook, err error) {
	err = catchPanic(func() error {
		if depth > 0 {
			ob, err = a.publicSpot.FetchOrderBook(symbol, ccxt.WithFetchOrderBookLimit(int64(depth)))
		} else {
			ob, err = a.publicSpot.FetchOrderBook(symbol)
		}
		return err
	})
	return
}

func (a *BinanceBrokerAdapter) LoadMarkets(_ context.Context) (m map[string]ccxt.MarketInterface, err error) {
	err = catchPanic(func() error { m, err = a.publicSpot.LoadMarkets(); return err })
	return
}

// --- broker.TradingProvider ---

func (a *BinanceBrokerAdapter) CreateOrder(_ context.Context, symbol, orderType, side string, amount float64, price *float64, params map[string]interface{}) (o ccxt.Order, err error) {
	opts := []ccxt.CreateOrderOptions{}
	if price != nil {
		opts = append(opts, ccxt.WithCreateOrderPrice(*price))
	}
	if len(params) > 0 {
		opts = append(opts, ccxt.WithCreateOrderParams(params))
	}
	err = catchPanic(func() error { o, err = a.spot.CreateOrder(symbol, orderType, side, amount, opts...); return err })
	return
}

func (a *BinanceBrokerAdapter) CancelOrder(_ context.Context, id, symbol string) (o ccxt.Order, err error) {
	err = catchPanic(func() error { o, err = a.spot.CancelOrder(id, ccxt.WithCancelOrderSymbol(symbol)); return err })
	return
}

func (a *BinanceBrokerAdapter) FetchOrder(_ context.Context, id, symbol string) (o ccxt.Order, err error) {
	err = catchPanic(func() error { o, err = a.spot.FetchOrder(id, ccxt.WithFetchOrderSymbol(symbol)); return err })
	return
}

func (a *BinanceBrokerAdapter) FetchOpenOrders(_ context.Context, symbol string) (orders []ccxt.Order, err error) {
	err = catchPanic(func() error {
		if symbol != "" {
			orders, err = a.spot.FetchOpenOrders(ccxt.WithFetchOpenOrdersSymbol(symbol))
		} else {
			orders, err = a.spot.FetchOpenOrders()
		}
		return err
	})
	return
}

func (a *BinanceBrokerAdapter) FetchClosedOrders(_ context.Context, symbol string, since *int64, limit int) (orders []ccxt.Order, err error) {
	opts := []ccxt.FetchClosedOrdersOptions{}
	if symbol != "" {
		opts = append(opts, ccxt.WithFetchClosedOrdersSymbol(symbol))
	}
	if since != nil {
		opts = append(opts, ccxt.WithFetchClosedOrdersSince(*since))
	}
	if limit > 0 {
		opts = append(opts, ccxt.WithFetchClosedOrdersLimit(int64(limit)))
	}
	err = catchPanic(func() error { orders, err = a.spot.FetchClosedOrders(opts...); return err })
	return
}

func (a *BinanceBrokerAdapter) FetchMyTrades(_ context.Context, symbol string, since *int64, limit int) (trades []ccxt.Trade, err error) {
	opts := []ccxt.FetchMyTradesOptions{}
	if symbol != "" {
		opts = append(opts, ccxt.WithFetchMyTradesSymbol(symbol))
	}
	if since != nil {
		opts = append(opts, ccxt.WithFetchMyTradesSince(*since))
	}
	if limit > 0 {
		opts = append(opts, ccxt.WithFetchMyTradesLimit(int64(limit)))
	}
	err = catchPanic(func() error { trades, err = a.spot.FetchMyTrades(opts...); return err })
	return
}

// --- broker.TransferProvider ---

func (a *BinanceBrokerAdapter) Transfer(_ context.Context, asset string, amount float64, fromAccount, toAccount string) (ccxt.TransferEntry, error) {
	return a.spot.Transfer(asset, amount, fromAccount, toAccount)
}

func init() {
	broker.RegisterFactory(Name, func(cfg *config.Config) (broker.Provider, error) {
		acc, ok := cfg.Exchanges.Binance.ResolveAccount("")
		if !ok {
			// No credentials configured — create a public-only instance for market data.
			return newBrokerAdapter(config.ExchangeAccount{}, cfg.Exchanges.Binance.Testnet)
		}
		return newBrokerAdapter(acc, cfg.Exchanges.Binance.Testnet)
	})
	broker.RegisterAccountFactory(Name, func(cfg *config.Config, accountName string) (broker.Provider, error) {
		acc, ok := cfg.Exchanges.Binance.ResolveAccount(accountName)
		if !ok {
			var names []string
			for i, a := range cfg.Exchanges.Binance.Accounts {
				n := a.Name
				if n == "" {
					n = fmt.Sprintf("%d", i+1)
				}
				names = append(names, n)
			}
			return nil, fmt.Errorf("%s: account %q not found (available: %v)", Name, accountName, names)
		}
		return newBrokerAdapter(acc, cfg.Exchanges.Binance.Testnet)
	})
}
