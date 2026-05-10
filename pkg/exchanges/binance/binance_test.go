package binance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/cryptoquantumwave/khunquant/pkg/exchanges"
)

func TestCollectAllWalletBalances_AllWalletsFail(t *testing.T) {
	_, err := collectAllWalletBalances(context.Background(), []string{"spot", "funding"}, func(_ context.Context, wt string) ([]exchanges.WalletBalance, error) {
		return nil, errors.New(wt + ": invalid api key")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{
		"binance: all wallet balance fetches failed",
		"spot: invalid api key",
		"funding: invalid api key",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
}

func TestCompactCCXTError_RemovesStackTrace(t *testing.T) {
	raw := "[ccxtError]::[AuthenticationError]::[binance {\"code\":-2015,\"msg\":\"Invalid API-key\"}]\n" +
		"Stack:\n" +
		"panic:runtime.goexit:panic:github.com/ccxt/ccxt/go/v4.(*Exchange).Fetch2.func1:panic:[ccxtError]\n" +
		"Stack trace:\n" +
		"goroutine 173 [running]:\n" +
		"runtime/debug.Stack()\n"

	got := compactCCXTError(errors.New(raw)).Error()
	if strings.Contains(got, "Stack trace:") || strings.Contains(got, "goroutine 173") {
		t.Fatalf("compacted error still contains stack trace: %q", got)
	}
	for _, want := range []string{
		"[ccxtError]::[AuthenticationError]",
		"Invalid API-key",
		"Stack:",
		"panic:runtime.goexit",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("compacted error %q does not contain %q", got, want)
		}
	}
}

func TestCollectAllWalletBalances_AllWalletsFailCompactsStackTrace(t *testing.T) {
	_, err := collectAllWalletBalances(context.Background(), []string{"spot"}, func(_ context.Context, wt string) ([]exchanges.WalletBalance, error) {
		return nil, fmt.Errorf("%s: %w", wt, errors.New("[ccxtError] auth\nStack:\npanic:auth\nStack trace:\ngoroutine 1"))
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, "Stack trace:") || strings.Contains(msg, "goroutine 1") {
		t.Fatalf("aggregate error still contains stack trace: %q", msg)
	}
	if !strings.Contains(msg, "Stack:\npanic:auth") {
		t.Fatalf("aggregate error should keep compact stack summary: %q", msg)
	}
}

func TestCollectAllWalletBalances_PartialErrorsWithoutBalancesAreNotEmptyPortfolio(t *testing.T) {
	_, err := collectAllWalletBalances(context.Background(), []string{"spot", "funding"}, func(_ context.Context, wt string) ([]exchanges.WalletBalance, error) {
		if wt == "funding" {
			return nil, errors.New("funding: disabled")
		}
		return nil, nil
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, want := range []string{
		"binance: no wallet balances found and some wallet fetches failed",
		"funding: disabled",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error %q does not contain %q", msg, want)
		}
	}
}

func TestCollectAllWalletBalances_AllSuccessfulWalletsEmptyIsNotError(t *testing.T) {
	balances, err := collectAllWalletBalances(context.Background(), []string{"spot", "funding"}, func(_ context.Context, wt string) ([]exchanges.WalletBalance, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 0 {
		t.Fatalf("balances len = %d, want 0", len(balances))
	}
}
