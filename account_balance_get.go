package gosqril

import (
	"context"
	"net/http"
)

// Balance is the account balance in one currency, in standard units of that currency.
type Balance struct {
	// Available can be negative; payouts are then locked (402 ACCOUNT_PAYOUTS_LOCKED).
	Available Decimal `json:"available"`
	// Locked is held by payouts in progress.
	Locked Decimal `json:"locked"`
	Total  Decimal `json:"total"`
	// Currency is an ISO 4217 code, the same as the map key.
	Currency string `json:"currency"`
}

// GetAccountBalancesRequest has no fields.
type GetAccountBalancesRequest struct{}

// GetAccountBalancesResponse holds one Balance per currency code (USD and the account's currencies).
type GetAccountBalancesResponse struct {
	Balances map[string]Balance `json:"balances"`
}

// GetAccountBalances returns the balances of the account. req may be nil.
func (c *Client) GetAccountBalances(ctx context.Context, _ *GetAccountBalancesRequest) (*GetAccountBalancesResponse, error) {
	return call[GetAccountBalancesResponse](ctx, c, request{
		Method:         http.MethodGet,
		Path:           "/getAccountBalances",
		Query:          nil,
		Body:           nil,
		IdempotencyKey: "",
	})
}
