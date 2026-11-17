package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestPreviewQuotations_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK,
		`{"quotations":[{"amount":2540000,"currency":"VND","exchange_rate":0.0000393701,"amount_usd":100,"fee":1.1,"percentage_fee":1,"fixed_fee":0.1,"expires_at":"2024-01-01T00:30:00.000Z"},{"amount":5600.75,"currency":"PHP","exchange_rate":0.0178547516,"amount_usd":100.00,"fee":1.60,"percentage_fee":1.50,"fixed_fee":0.10,"expires_at":"2024-01-01T00:30:00.000Z"}]}`)

	resp, err := client.PreviewQuotations(context.Background(), &gosqril.PreviewQuotationsRequest{
		DP:       ptr(2),
		Amount:   "100.00",
		Currency: "USD",
	})
	if err != nil {
		t.Fatalf("PreviewQuotations: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/previewQuotations", "dp=2", `{"amount":100.00,"currency":"USD"}`)

	if len(resp.Quotations) != 2 {
		t.Fatalf("quotations = %d, want 2", len(resp.Quotations))
	}

	want := gosqril.QuotationPreview{
		Amount:        "5600.75",
		Currency:      "PHP",
		ExchangeRate:  "0.0178547516",
		AmountUSD:     "100.00",
		Fee:           "1.60",
		PercentageFee: "1.50",
		FixedFee:      "0.10",
		ExpiresAt:     "2024-01-01T00:30:00.000Z",
	}
	if resp.Quotations[1] != want {
		t.Errorf("quotation = %+v, want %+v", resp.Quotations[1], want)
	}

	if resp.Quotations[0].Amount != "2540000" || resp.Quotations[0].ExchangeRate != "0.0000393701" {
		t.Errorf("first quotation = %+v", resp.Quotations[0])
	}
}

func TestPreviewQuotations_EmptyList(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusOK, `{"quotations":[]}`)

	resp, err := client.PreviewQuotations(context.Background(), &gosqril.PreviewQuotationsRequest{Amount: "1", Currency: "USD"})
	if err != nil || len(resp.Quotations) != 0 {
		t.Errorf("resp = %+v, err = %v, want an empty list and no error", resp, err)
	}
}

func TestPreviewQuotations_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusBadRequest,
		`{"error_code":"INVALID_AMOUNT","error_message":"amount must be greater than 0"}`)

	_, err := client.PreviewQuotations(context.Background(), &gosqril.PreviewQuotationsRequest{Amount: "0", Currency: "USD"})

	assertAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeInvalidAmount)
}
