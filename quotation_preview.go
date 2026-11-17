package gosqril

import (
	"context"
	"net/http"
)

// PreviewQuotationsRequest asks for indicative prices. Amount and Currency are required.
type PreviewQuotationsRequest struct {
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP *int `json:"-"`
	// Amount is in Currency and above zero.
	Amount Decimal `json:"amount"`
	// Currency is ISO 4217, for example USD.
	Currency string `json:"currency"`
}

// QuotationPreview is the indicative price for one corridor.
type QuotationPreview struct {
	// Amount is in the local currency named by Currency.
	Amount   Decimal `json:"amount"`
	Currency string  `json:"currency"`
	// ExchangeRate is USD per one unit of Currency.
	ExchangeRate  Decimal `json:"exchange_rate"`
	AmountUSD     Decimal `json:"amount_usd"`
	Fee           Decimal `json:"fee"`
	PercentageFee Decimal `json:"percentage_fee"`
	FixedFee      Decimal `json:"fixed_fee"`
	// ExpiresAt is ISO 8601.
	ExpiresAt string `json:"expires_at"`
}

// PreviewQuotationsResponse lists one preview per enabled corridor. A corridor with no rate, or
// whose converted amount is outside its limits, is left out: a short or empty list is normal.
type PreviewQuotationsResponse struct {
	Quotations []QuotationPreview `json:"quotations"`
}

// PreviewQuotations prices an amount for every corridor of the account without creating a
// transaction. Limited to 60 calls per minute.
func (c *Client) PreviewQuotations(ctx context.Context, req *PreviewQuotationsRequest) (*PreviewQuotationsResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	return call[PreviewQuotationsResponse](ctx, c, request{
		Method:         http.MethodPost,
		Path:           "/previewQuotations",
		Query:          queryWithDP(req.DP),
		Body:           req,
		IdempotencyKey: "",
	})
}
