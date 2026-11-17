package gosqril

import (
	"context"
	"net/http"
)

// ExecutePayoutRequest confirms a quoted transaction. IdempotencyKey, TxID and CustomerID are
// checked by the SDK; AmountConfirmed and Currency are required by the API.
type ExecutePayoutRequest struct {
	// IdempotencyKey is sent as X-Idempotency-Key. Choose one value per logical payout and reuse
	// it on retries; the SDK never generates it.
	IdempotencyKey string `json:"-"`
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP   *int   `json:"-"`
	TxID string `json:"tx_id"`
	// AmountConfirmed is in the local currency and must equal the transaction's saved amount.
	AmountConfirmed Decimal `json:"amount_confirmed"`
	// Currency must equal the transaction's saved currency.
	Currency string `json:"currency"`
	// CustomerID must match the transaction's customer and must not contain "/".
	CustomerID string `json:"customer_id"`
	// PartnerTransactionID can be set once and never changed.
	PartnerTransactionID string `json:"partner_transaction_id,omitempty"`
	// RecipientName is needed only when the decode listed recipient.name in MissingFields.
	RecipientName string `json:"recipient_name,omitempty"`
}

// ExecutePayoutResponse is the accepted payout. Status is PROCESSING; the final status arrives
// by webhook or GetTransaction.
type ExecutePayoutResponse struct {
	Status     TransactionStatus `json:"status"`
	TxID       string            `json:"tx_id"`
	CustomerID string            `json:"customer_id"`
	Message    string            `json:"message"`
	// ExchangeRate is USD per one unit of Currency.
	ExchangeRate Decimal `json:"exchange_rate"`
	// Fee, PercentageFee, FixedFee and AmountUSD are in USD.
	Fee           Decimal `json:"fee"`
	PercentageFee Decimal `json:"percentage_fee"`
	FixedFee      Decimal `json:"fixed_fee"`
	// Amount is in the local currency named by Currency.
	Amount               Decimal `json:"amount"`
	AmountUSD            Decimal `json:"amount_usd"`
	Currency             string  `json:"currency"`
	Sender               Party   `json:"sender,omitempty"`
	Recipient            Party   `json:"recipient,omitempty"`
	PartnerTransactionID string  `json:"partner_transaction_id,omitempty"`
	// UpdatedAt is ISO 8601.
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ExecutePayout moves money. It fails before calling the API when IdempotencyKey, TxID or
// CustomerID is empty or AmountConfirmed is not a decimal. Any other error that is not an *APIError with a 4xx status (network error,
// timeout, 5xx, ErrInvalidResponse) leaves the outcome unknown: call GetTransaction, or resend
// with the same IdempotencyKey, never with a new one.
func (c *Client) ExecutePayout(ctx context.Context, req *ExecutePayoutRequest) (*ExecutePayoutResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireField("IdempotencyKey", req.IdempotencyKey); err != nil {
		return nil, err
	}

	if err := requireField("TxID", req.TxID); err != nil {
		return nil, err
	}

	if err := requireCustomerID(req.CustomerID); err != nil {
		return nil, err
	}

	return call[ExecutePayoutResponse](ctx, c, request{
		Method:         http.MethodPost,
		Path:           "/executePayout",
		Query:          queryWithDP(req.DP),
		Body:           req,
		IdempotencyKey: req.IdempotencyKey,
	})
}
