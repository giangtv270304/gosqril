package gosqril

import (
	"context"
	"net/http"
)

// GetQuotationRequest prices a PENDING transaction. TxID and CustomerID are required.
type GetQuotationRequest struct {
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP   *int   `json:"-"`
	TxID string `json:"tx_id"`
	// CustomerID must match the transaction's customer and must not contain "/".
	CustomerID string `json:"customer_id"`
	// Amount is in the local currency and above zero. Required when the transaction has no
	// saved amount; ignored when it has one.
	Amount *Decimal `json:"amount,omitempty"`
	// PartnerTransactionID can be set once; a different value later is a 400.
	PartnerTransactionID string `json:"partner_transaction_id,omitempty"`
}

// GetQuotationResponse is the stored quotation, valid for 30 minutes.
type GetQuotationResponse struct {
	TxID string `json:"tx_id"`
	// Amount is in the local currency named by Currency.
	Amount   Decimal `json:"amount"`
	Currency string  `json:"currency"`
	// ExchangeRate is USD per one unit of Currency.
	ExchangeRate Decimal `json:"exchange_rate"`
	AmountUSD    Decimal `json:"amount_usd"`
	// Fee is the total fee in USD: PercentageFee plus FixedFee, within rounding.
	Fee           Decimal `json:"fee"`
	PercentageFee Decimal `json:"percentage_fee"`
	FixedFee      Decimal `json:"fixed_fee"`
	// ExpiresAt is ISO 8601.
	ExpiresAt string `json:"expires_at"`
}

// GetQuotation prices a PENDING transaction and stores the quotation. Confirm the returned
// Amount and Currency in ExecutePayout before ExpiresAt.
func (c *Client) GetQuotation(ctx context.Context, req *GetQuotationRequest) (*GetQuotationResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireField("TxID", req.TxID); err != nil {
		return nil, err
	}

	if err := requireCustomerID(req.CustomerID); err != nil {
		return nil, err
	}

	return call[GetQuotationResponse](ctx, c, request{
		Method:         http.MethodPost,
		Path:           "/getQuotation",
		Query:          queryWithDP(req.DP),
		Body:           req,
		IdempotencyKey: "",
	})
}
