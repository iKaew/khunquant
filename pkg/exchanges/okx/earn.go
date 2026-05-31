package okx

// OKX Simple Earn (Flexible savings) support for the broker.EarnProvider interface.
// CCXT has no unified Earn methods, so these call CCXT implicit (raw) endpoints on
// the OKX client. OKX savings are keyed by currency (ccy), not a product id.
// Responses follow OKX's {"code":"0","data":[...]} envelope.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"

	"github.com/cryptoquantumwave/khunquant/pkg/providers/broker"
)

const okxBaseURL = "https://www.okx.com"

// okxSignedGet makes an authenticated GET request to an OKX endpoint that is not
// available in the CCXT Go binding (e.g. the newer /api/v5/financial-product/ paths).
// Credentials are read from the embedded CCXT client.
func (a *OKXBrokerAdapter) okxSignedGet(path string, params map[string]string) ([]map[string]interface{}, error) {
	apiKey := fmt.Sprint(a.client.Core.ApiKey)
	secret := fmt.Sprint(a.client.Core.Secret)
	passphrase := fmt.Sprint(a.client.Core.Password)

	fullPath := path
	if len(params) > 0 {
		q := url.Values{}
		for k, v := range params {
			q.Set(k, v)
		}
		fullPath = path + "?" + q.Encode()
	}

	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	prehash := timestamp + "GET" + fullPath
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(prehash))
	sign := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequest("GET", okxBaseURL+fullPath, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("OK-ACCESS-KEY", apiKey)
	req.Header.Set("OK-ACCESS-SIGN", sign)
	req.Header.Set("OK-ACCESS-TIMESTAMP", timestamp)
	req.Header.Set("OK-ACCESS-PASSPHRASE", passphrase)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var envelope struct {
		Code string                   `json:"code"`
		Msg  string                   `json:"msg"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse: %w — body: %s", err, body)
	}
	if envelope.Code != "0" {
		return nil, fmt.Errorf("OKX API error code=%s msg=%s", envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

// Compile-time guarantee that the adapter satisfies broker.EarnProvider.
var _ broker.EarnProvider = (*OKXBrokerAdapter)(nil)

// okxFloat coerces a CCXT raw value (float64 or numeric string) to float64.
func okxFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	}
	return 0
}

// okxString coerces a CCXT raw value to string.
func okxString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprint(t)
	}
}

// okxData extracts the "data" array from an OKX raw response envelope.
func okxData(res interface{}) []map[string]interface{} {
	m, ok := res.(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := m["data"].([]interface{})
	if !ok {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(raw))
	for _, r := range raw {
		if rm, ok := r.(map[string]interface{}); ok {
			out = append(out, rm)
		}
	}
	return out
}

// --- broker.EarnProvider ---

// FetchFlexibleEarnProducts returns OKX flexible savings products (one per ccy).
// The APY endpoint is public, so this works without credentials. asset == ""
// returns all currencies. APY is normalized to a fraction (estRate is a fraction).
func (a *OKXBrokerAdapter) FetchFlexibleEarnProducts(_ context.Context, asset string) ([]broker.EarnProduct, error) {
	var products []broker.EarnProduct
	err := catchPanic(func() error {
		params := map[string]interface{}{}
		if asset != "" {
			params["ccy"] = asset
		}
		res := <-a.publicClient.Core.PublicGetFinanceSavingsLendingRateSummary(params)
		if ccxt.IsError(res) {
			return ccxt.CreateReturnError(res)
		}
		for _, row := range okxData(res) {
			products = append(products, broker.EarnProduct{
				Exchange:     Name,
				Asset:        okxString(row["ccy"]),
				ProductID:    okxString(row["ccy"]), // OKX savings are keyed by currency
				APY:          okxFloat(row["estRate"]),
				CanSubscribe: true,
			})
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("okx earn: list products: %w", err)
	}
	return products, nil
}

// FetchFlexibleEarnPositions returns currently held OKX flexible savings / earn balances.
//
// OKX has multiple earn products with different APIs:
//  1. Old Savings (lending pool): /api/v5/finance/savings/balance → amt field
//  2. Newer "Simple Earn" may show as frozenBal in /api/v5/account/balance (trading UTA)
//  3. Funding account frozen balances: /api/v5/asset/balances → frozenBal field
//
// All three are queried and merged; duplicates (same asset already found) are skipped.
func (a *OKXBrokerAdapter) FetchFlexibleEarnPositions(_ context.Context) ([]broker.EarnPosition, error) {
	if err := a.requireAuth(); err != nil {
		return nil, err
	}
	var positions []broker.EarnPosition

	// Helper: check if an asset is already in positions.
	has := func(asset string) bool {
		for _, p := range positions {
			if p.Asset == asset {
				return true
			}
		}
		return false
	}

	// ── Source 1: old Savings / Simple Earn lending pool ─────────────────
	_ = catchPanic(func() error {
		res := <-a.client.Core.PrivateGetFinanceSavingsBalance(map[string]interface{}{})
		if ccxt.IsError(res) {
			return ccxt.CreateReturnError(res)
		}
		for _, row := range okxData(res) {
			asset := okxString(row["ccy"])
			if !has(asset) {
				positions = append(positions, broker.EarnPosition{
					Exchange:  Name,
					Asset:     asset,
					ProductID: asset,
					Amount:    okxFloat(row["amt"]),
					APY:       okxFloat(row["earningRate"]),
				})
			}
		}
		return nil
	})

	// ── Source 2: trading account frozenBal (UTA — Simple Earn locks assets here) ──
	// OKX Unified Trade Account shows earn-locked assets as frozenBal in account/balance.
	// cashBal = freely tradable; frozenBal = locked in earn/orders.
	// We add frozenBal only when the asset isn't already counted from savings.
	_ = catchPanic(func() error {
		res := <-a.client.Core.PrivateGetAccountBalance(map[string]interface{}{})
		if ccxt.IsError(res) {
			return nil // supplemental: ignore error
		}
		m, ok := res.(map[string]interface{})
		if !ok {
			return nil
		}
		dataArr, _ := m["data"].([]interface{})
		if len(dataArr) == 0 {
			return nil
		}
		acct, _ := dataArr[0].(map[string]interface{})
		details, _ := acct["details"].([]interface{})
		for _, d := range details {
			dm, _ := d.(map[string]interface{})
			asset := okxString(dm["ccy"])
			frozen := okxFloat(dm["frozenBal"])
			if frozen > 0 && !has(asset) {
				positions = append(positions, broker.EarnPosition{
					Exchange:  Name,
					Asset:     asset,
					ProductID: asset + ":trading:frozen",
					Amount:    frozen,
				})
			}
		}
		return nil
	})

	// ── Source 3: funding account frozenBal ───────────────────────────────
	// OKX Simple Earn Flexible draws from the funding account; the subscribed
	// amount appears as frozenBal in /api/v5/asset/balances.
	_ = catchPanic(func() error {
		res := <-a.client.Core.PrivateGetAssetBalances(map[string]interface{}{})
		if ccxt.IsError(res) {
			return nil // supplemental: ignore error
		}
		for _, row := range okxData(res) {
			asset := okxString(row["ccy"])
			frozen := okxFloat(row["frozenBal"])
			if frozen > 0 && !has(asset) {
				positions = append(positions, broker.EarnPosition{
					Exchange:  Name,
					Asset:     asset,
					ProductID: asset + ":funding:frozen",
					Amount:    frozen,
				})
			}
		}
		return nil
	})

	// ── Source 4: OKX Simple Earn Flexible (new /financial-product/ API) ─
	// OKX's newer "Simple Earn Flexible" uses /api/v5/financial-product/simple-earn-flexible/
	// which is not in CCXT v4.5.56. We call it directly via HMAC-signed HTTP.
	// NOTE (2026-05-31): OKX's web servers currently return an HTML page for this
	// path (www.okx.com routes it to their SPA instead of the API backend), so
	// okxSignedGet returns a parse error and this block is skipped. Once OKX
	// deploys the endpoint correctly this source will activate automatically.
	if rows, err := a.okxSignedGet("/api/v5/financial-product/simple-earn-flexible/saving-balance", nil); err == nil {
		for _, row := range rows {
			asset := okxString(row["ccy"])
			// OKX may use "amt", "investAmt", or "bal" for the subscribed amount.
			amt := okxFloat(row["amt"])
			if amt == 0 {
				amt = okxFloat(row["investAmt"])
			}
			if amt == 0 {
				amt = okxFloat(row["bal"])
			}
			apy := okxFloat(row["rate"])
			if amt > 0 && !has(asset) {
				positions = append(positions, broker.EarnPosition{
					Exchange:  Name,
					Asset:     asset,
					ProductID: asset + ":simple-earn-flexible",
					Amount:    amt,
					APY:       apy,
				})
			}
		}
	}

	// ── Source 5: OKX On-chain Earn active orders ─────────────────────────
	// Same situation as Source 4 — endpoint is documented but not yet live at www.okx.com.
	if rows, err := a.okxSignedGet("/api/v5/financial-product/on-chain-earn/active-orders", nil); err == nil {
		for _, row := range rows {
			asset := okxString(row["ccy"])
			amt := okxFloat(row["investAmt"])
			if amt == 0 {
				amt = okxFloat(row["amt"])
			}
			if amt == 0 {
				amt = okxFloat(row["bal"])
			}
			apy := okxFloat(row["rate"])
			ordID := okxString(row["ordId"])
			if amt > 0 && !has(asset) {
				positions = append(positions, broker.EarnPosition{
					Exchange:  Name,
					Asset:     asset,
					ProductID: ordID + ":on-chain-earn",
					Amount:    amt,
					APY:       apy,
				})
			}
		}
	}

	return positions, nil
}

// purchaseRedempt issues an OKX savings purchase or redemption for a currency.
// side is "purchase" or "redempt".
func (a *OKXBrokerAdapter) purchaseRedempt(asset string, amount float64, side string) (string, error) {
	var txID string
	err := catchPanic(func() error {
		params := map[string]interface{}{
			"ccy":  asset,
			"amt":  strconv.FormatFloat(amount, 'f', -1, 64),
			"side": side,
		}
		res := <-a.client.Core.PrivatePostFinanceSavingsPurchaseRedempt(params)
		if ccxt.IsError(res) {
			return ccxt.CreateReturnError(res)
		}
		for _, row := range okxData(res) {
			txID = okxString(row["ccy"]) + ":" + okxString(row["side"])
		}
		return nil
	})
	return txID, err
}

// SubscribeFlexibleEarn purchases amount of asset into OKX flexible savings.
// OKX savings draw from the Funding account, so this first transfers from the
// trading account if needed (best-effort; ignored if already in funding).
func (a *OKXBrokerAdapter) SubscribeFlexibleEarn(_ context.Context, _ /*productID*/ string, asset string, amount float64, _ bool) (string, error) {
	if err := a.requireAuth(); err != nil {
		return "", err
	}
	// Best-effort move from trading -> funding; OKX savings subscribe from funding.
	_ = catchPanic(func() error {
		_, e := a.client.Transfer(asset, amount, "trading", "funding")
		return e
	})
	txID, err := a.purchaseRedempt(asset, amount, "purchase")
	if err != nil {
		return "", fmt.Errorf("okx earn: subscribe: %w", err)
	}
	return txID, nil
}

// RedeemFlexibleEarn redeems amount of asset from OKX flexible savings. OKX has no
// "redeem all" flag, so redeemAll requires the caller to pass the full amount.
func (a *OKXBrokerAdapter) RedeemFlexibleEarn(_ context.Context, _ /*productID*/ string, asset string, amount float64, redeemAll bool) (string, error) {
	if err := a.requireAuth(); err != nil {
		return "", err
	}
	if redeemAll && amount <= 0 {
		return "", fmt.Errorf("okx earn: redeem: OKX requires an explicit amount (no redeem-all flag); pass the held amount")
	}
	txID, err := a.purchaseRedempt(asset, amount, "redempt")
	if err != nil {
		return "", fmt.Errorf("okx earn: redeem: %w", err)
	}
	return txID, nil
}

// SetFlexibleAutoSubscribe is not exposed by the OKX API. OKX "Default Subscribe"
// (auto-earn) is configured in the OKX app, not via a per-currency API toggle.
func (a *OKXBrokerAdapter) SetFlexibleAutoSubscribe(_ context.Context, _ /*productID*/ string, _ string, _ bool) error {
	return fmt.Errorf("okx earn: auto-subscribe is not configurable via API — enable 'Default Subscribe' in the OKX app")
}

// FetchFlexibleEarnRateHistory returns historical rate data for a flexible savings currency.
// Calls /api/v5/finance/savings/lending-rate-history (PUBLIC endpoint).
// The response rate is already a fraction (0.05 == 5% APY). productID is ignored for OKX
// (savings are keyed by currency). asset is required.
func (a *OKXBrokerAdapter) FetchFlexibleEarnRateHistory(ctx context.Context, productID, asset string, since *int64, limit int) ([]broker.EarnRatePoint, error) {
	var points []broker.EarnRatePoint
	err := catchPanic(func() error {
		params := map[string]interface{}{
			"ccy":   asset,
			"limit": "100",
		}
		res := <-a.publicClient.Core.PublicGetFinanceSavingsLendingRateHistory(params)
		if ccxt.IsError(res) {
			return ccxt.CreateReturnError(res)
		}
		for _, row := range okxData(res) {
			rate := okxFloat(row["rate"])
			timestamp := int64(okxFloat(row["ts"]))
			points = append(points, broker.EarnRatePoint{
				Rate:      rate,
				Timestamp: timestamp,
			})
		}
		// Cap by limit if specified (caller may pass 0 for no limit).
		if limit > 0 && len(points) > limit {
			points = points[:limit]
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("okx earn: fetch rate history: %w", err)
	}
	return points, nil
}
