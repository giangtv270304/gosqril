package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestListTransactions_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"transactions":[`+transactionBody+`],"total":1,"limit":1,"has_more":true,"next_cursor":"tx_1234567890_abc123"}`)

	resp, err := client.ListTransactions(context.Background(), &gosqril.ListTransactionsRequest{
		Status:     gosqril.TransactionStatusSuccess,
		CustomerID: "",
		Limit:      1,
		StartAfter: "tx_0",
		StartDate:  "2024-01-01T00:00:00Z",
		EndDate:    "2024-01-31T23:59:59Z",
		DateField:  gosqril.DateFieldCreatedAt,
		DP:         ptr(2),
	})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/listTransactions",
		"date_field=created_at&dp=2&end_date=2024-01-31T23%3A59%3A59Z&limit=1&start_after=tx_0&start_date=2024-01-01T00%3A00%3A00Z&status=SUCCESS", "")

	if len(resp.Transactions) != 1 || resp.Total != 1 || resp.Limit != 1 || !resp.HasMore ||
		resp.NextCursor == nil || *resp.NextCursor != "tx_1234567890_abc123" {
		t.Fatalf("page = %+v", resp)
	}

	assertTransaction(t, &resp.Transactions[0])
}

func TestListTransactions_NilRequestAndLastPage(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"transactions":[],"total":0,"limit":50,"has_more":false,"next_cursor":null}`)

	resp, err := client.ListTransactions(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/listTransactions", "", "")

	if len(resp.Transactions) != 0 || resp.Limit != 50 || resp.HasMore || resp.NextCursor != nil {
		t.Errorf("page = %+v", resp)
	}
}

func TestListTransactions_CustomerFilter(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, `{"transactions":[],"total":0,"limit":50,"has_more":false}`)

	_, err := client.ListTransactions(context.Background(), &gosqril.ListTransactionsRequest{CustomerID: "cus_1,cus_2"})
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/listTransactions", "customer_id=cus_1%2Ccus_2", "")
}

func TestListTransactions_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusBadRequest,
		`{"error_code":"INVALID_LIMIT","error_message":"limit must be between 1 and 100"}`)

	_, err := client.ListTransactions(context.Background(), &gosqril.ListTransactionsRequest{Limit: 1000})

	assertAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeInvalidLimit)
}
