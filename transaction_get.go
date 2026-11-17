package gosqril

import (
	"context"
	"net/http"
)

// TransactionStatus is the state of a transaction. Unknown values from the API are kept as sent.
type TransactionStatus string

// Transaction states. PENDING becomes PROCESSING at ExecutePayout, then SUCCESS or FAILED;
// PENDING becomes EXPIRED when the quotation lapses; SUCCESS can become REFUNDED.
const (
	TransactionStatusPending    TransactionStatus = "PENDING"
	TransactionStatusProcessing TransactionStatus = "PROCESSING"
	TransactionStatusSuccess    TransactionStatus = "SUCCESS"
	TransactionStatusFailed     TransactionStatus = "FAILED"
	TransactionStatusExpired    TransactionStatus = "EXPIRED"
	TransactionStatusRefunded   TransactionStatus = "REFUNDED"
)

// Transaction is returned by GetTransaction and ListTransactions. Every field is optional in
// the API; an absent field is the zero value or nil.
type Transaction struct {
	// ID is the tx_id.
	ID            string            `json:"id"`
	ExchangeAppID string            `json:"exchange_app_id"`
	CustomerID    string            `json:"customer_id"`
	Status        TransactionStatus `json:"status"`
	// Amount is in the local currency named by Currency.
	Amount Decimal `json:"amount"`
	// AmountUSD, Fee, PercentageFee, FixedFee and Total are in USD; nil before a quotation.
	AmountUSD     *Decimal `json:"amount_usd,omitempty"`
	Fee           *Decimal `json:"fee,omitempty"`
	PercentageFee *Decimal `json:"percentage_fee,omitempty"`
	FixedFee      *Decimal `json:"fixed_fee,omitempty"`
	Total         *Decimal `json:"total,omitempty"`
	// ExchangeRate is USD per one unit of Currency.
	ExchangeRate         *Decimal `json:"exchange_rate,omitempty"`
	Currency             string   `json:"currency"`
	CountryCode          string   `json:"country_code"`
	PartnerTransactionID string   `json:"partner_transaction_id,omitempty"`
	RecipientBankNo      string   `json:"recipient_bank_no,omitempty"`
	RecipientAccountNo   string   `json:"recipient_account_no,omitempty"`
	RecipientAccountName string   `json:"recipient_account_name,omitempty"`
	Sender               Party    `json:"sender,omitempty"`
	Recipient            Party    `json:"recipient,omitempty"`
	// QRData is a free-form object describing the decoded QR.
	QRData        map[string]any `json:"qr_data,omitempty"`
	MissingFields []MissingField `json:"missing_fields,omitempty"`
	// Timestamps are ISO 8601. CompletedAt is kept when the transaction is refunded later.
	QuotationExpiresAt string `json:"quotation_expires_at,omitempty"`
	CreatedAt          string `json:"created_at"`
	UpdatedAt          string `json:"updated_at"`
	CompletedAt        string `json:"completed_at,omitempty"`
	RefundedAt         string `json:"refunded_at,omitempty"`
	// ErrorMessage is the failure reason of a FAILED payout.
	ErrorMessage string `json:"error_message,omitempty"`
}

// GetTransactionRequest selects the transaction to read.
type GetTransactionRequest struct {
	// TransactionID is the tx_id returned by a decode or lookup call.
	TransactionID string
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP *int
}

// GetTransactionResponse wraps the transaction.
type GetTransactionResponse struct {
	Transaction Transaction `json:"transaction"`
}

// GetTransaction returns one transaction. An unknown id, or one of another account, is an
// *APIError; check it with IsNotFound.
func (c *Client) GetTransaction(ctx context.Context, req *GetTransactionRequest) (*GetTransactionResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireField("TransactionID", req.TransactionID); err != nil {
		return nil, err
	}

	query := queryWithDP(req.DP)
	query.Set("transaction_id", req.TransactionID)

	return call[GetTransactionResponse](ctx, c, request{
		Method:         http.MethodGet,
		Path:           "/getTransaction",
		Query:          query,
		Body:           nil,
		IdempotencyKey: "",
	})
}
