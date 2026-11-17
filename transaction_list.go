package gosqril

import (
	"context"
	"net/http"
	"strconv"
)

// DateField selects the date that ListTransactions filters and orders by.
type DateField string

// Date fields accepted by ListTransactions. CompletedAt and RefundedAt cannot be combined
// with Status or CustomerID.
const (
	DateFieldCreatedAt   DateField = "created_at"
	DateFieldCompletedAt DateField = "completed_at"
	DateFieldRefundedAt  DateField = "refunded_at"
)

// ListTransactionsRequest: zero values are left out and the API applies its defaults.
type ListTransactionsRequest struct {
	Status TransactionStatus
	// CustomerID is one id, or up to 10 ids separated by commas, without "/". Cannot be combined with Status.
	CustomerID string
	// Limit is 1 to 100; the API default is 50.
	Limit int
	// StartAfter is the NextCursor of the previous page.
	StartAfter string
	// StartDate and EndDate are ISO 8601 date-times, inclusive.
	StartDate string
	EndDate   string
	DateField DateField
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP *int
}

// ListTransactionsResponse is one page, newest first.
type ListTransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
	// Total is the number of transactions in this page.
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	HasMore bool `json:"has_more"`
	// NextCursor is set when HasMore is true; pass it as StartAfter.
	NextCursor *string `json:"next_cursor"`
}

// ListTransactions returns one page of transactions. Pass nil for the first page with defaults.
// It fails before calling the API when CustomerID contains "/".
func (c *Client) ListTransactions(ctx context.Context, req *ListTransactionsRequest) (*ListTransactionsResponse, error) {
	if req == nil {
		req = new(ListTransactionsRequest)
	}

	if err := validateOptionalCustomerID(req.CustomerID); err != nil {
		return nil, err
	}

	query := queryWithDP(req.DP)

	if req.Limit > 0 {
		query.Set("limit", strconv.Itoa(req.Limit))
	}

	for name, value := range map[string]string{
		"status":      string(req.Status),
		"customer_id": req.CustomerID,
		"start_after": req.StartAfter,
		"start_date":  req.StartDate,
		"end_date":    req.EndDate,
		"date_field":  string(req.DateField),
	} {
		if value != "" {
			query.Set(name, value)
		}
	}

	return call[ListTransactionsResponse](ctx, c, request{
		Method:         http.MethodGet,
		Path:           "/listTransactions",
		Query:          query,
		Body:           nil,
		IdempotencyKey: "",
	})
}
