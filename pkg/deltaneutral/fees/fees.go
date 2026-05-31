package fees

import (
	"context"
	"time"
)

// FetchFeesRequest specifies the query for accumulated position fees.
type FetchFeesRequest struct {
	FuturesSymbol string    // CCXT notation, e.g. "CHZ/USDT:USDT"
	Since         time.Time // position open time
	Until         time.Time // time.Now() for open positions
}

// PositionFees holds accumulated trading and funding fees for a futures position.
// Negative values mean the user paid; positive means received (rebate or funding).
type PositionFees struct {
	TradingFeeUSDT float64
	FundingFeeUSDT float64
	PeriodStart    time.Time
	PeriodEnd      time.Time
	FetchedAt      time.Time
}

// FeesFetcher fetches accumulated fees for a futures position over a time window.
type FeesFetcher interface {
	FetchFuturesPositionFees(ctx context.Context, req FetchFeesRequest) (*PositionFees, error)
}
