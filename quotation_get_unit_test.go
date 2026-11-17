package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestGetQuotation_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"tx_id":"tx_1234567890_abc123","amount":100000,"currency":"VND","exchange_rate":0.0000639967,"amount_usd":6.4,"fee":0.45,"percentage_fee":0.35,"fixed_fee":0.1,"expires_at":"2024-01-01T00:30:00.000Z"}`)

	resp, err := client.GetQuotation(context.Background(), &gosqril.GetQuotationRequest{
		DP:                   ptr(10),
		TxID:                 "tx_1234567890_abc123",
		CustomerID:           "cus_abc123",
		Amount:               ptr(gosqril.Decimal("100000")),
		PartnerTransactionID: "",
	})
	if err != nil {
		t.Fatalf("GetQuotation: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/getQuotation", "dp=10",
		`{"tx_id":"tx_1234567890_abc123","customer_id":"cus_abc123","amount":100000}`)

	want := gosqril.GetQuotationResponse{
		TxID:          "tx_1234567890_abc123",
		Amount:        "100000",
		Currency:      "VND",
		ExchangeRate:  "0.0000639967",
		AmountUSD:     "6.4",
		Fee:           "0.45",
		PercentageFee: "0.35",
		FixedFee:      "0.1",
		ExpiresAt:     "2024-01-01T00:30:00.000Z",
	}
	if *resp != want {
		t.Errorf("quotation = %+v, want %+v", *resp, want)
	}
}

func TestGetQuotation_MoneyKeepsExactDigits(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"tx_id":"tx_1","amount":1234567890123456789012345.10,"currency":"PHP","exchange_rate":6.4e-8,"amount_usd":"1000.50","fee":0.0000000001,"percentage_fee":0.10,"fixed_fee":0,"expires_at":"2024-01-01T00:30:00.000Z"}`)

	resp, err := client.GetQuotation(context.Background(), &gosqril.GetQuotationRequest{
		TxID:       "tx_1",
		CustomerID: "cus_abc123",
		Amount:     ptr(gosqril.Decimal("1234567890123456789012345.10")),
	})
	if err != nil {
		t.Fatalf("GetQuotation: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/getQuotation", "",
		`{"tx_id":"tx_1","customer_id":"cus_abc123","amount":1234567890123456789012345.10}`)

	got := []gosqril.Decimal{resp.Amount, resp.ExchangeRate, resp.AmountUSD, resp.Fee, resp.PercentageFee, resp.FixedFee}
	want := []gosqril.Decimal{"1234567890123456789012345.10", "0.000000064", "1000.50", "0.0000000001", "0.10", "0"}

	for index := range want {
		if got[index] != want[index] {
			t.Errorf("money field %d = %s, want %s", index, got[index], want[index])
		}
	}
}

func TestGetQuotation_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusBadRequest,
		`{"error_code":"DAILY_LIMIT_EXCEEDED","error_message":"Daily limit exceeded","details":{"limit_usd":6500,"used_usd":6400.25,"remaining_usd":99.75,"resets_at":"2026-04-23T00:00:00.000Z"}}`)

	_, err := client.GetQuotation(context.Background(), &gosqril.GetQuotationRequest{TxID: "tx_1", CustomerID: "cus_abc123"})

	apiErr := requireAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeDailyLimitExceeded)
	if got := string(apiErr.Details["used_usd"]); got != "6400.25" {
		t.Errorf("details.used_usd = %s, want the raw JSON 6400.25", got)
	}
}
