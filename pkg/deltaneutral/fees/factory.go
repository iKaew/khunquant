package fees

import (
	"fmt"
	"strings"

	"github.com/cryptoquantumwave/khunquant/pkg/config"
)

// NewFeesFetcher returns a FeesFetcher for the given provider and account name.
// Returns an error if the provider is not supported or the account is not found.
// To add a new exchange: implement FeesFetcher and add a case here.
func NewFeesFetcher(provider, account string, cfg *config.Config) (FeesFetcher, error) {
	switch strings.ToLower(provider) {
	case "okx":
		creds, ok := cfg.Exchanges.OKX.ResolveAccount(account)
		if !ok {
			return nil, fmt.Errorf("fees: okx account %q not found in config", account)
		}
		return newOKXFeesFetcher(creds)
	case "binance", "binanceusdm":
		creds, ok := cfg.Exchanges.Binance.ResolveAccount(account)
		if !ok {
			return nil, fmt.Errorf("fees: binance account %q not found in config", account)
		}
		return newBinanceFeesFetcher(creds)
	default:
		return nil, fmt.Errorf("fees: provider %q is not supported", provider)
	}
}
