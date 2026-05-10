package binance

import (
	"context"
	"errors"
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

func TestCollectAllWalletBalances_EmptySuccessfulWalletIsNotError(t *testing.T) {
	balances, err := collectAllWalletBalances(context.Background(), []string{"spot", "funding"}, func(_ context.Context, wt string) ([]exchanges.WalletBalance, error) {
		if wt == "funding" {
			return nil, errors.New("funding: disabled")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(balances) != 0 {
		t.Fatalf("balances len = %d, want 0", len(balances))
	}
}
