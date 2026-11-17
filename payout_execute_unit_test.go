package gosqril_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestExecutePayout_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusAccepted,
		`{"status":"PROCESSING","tx_id":"tx_1234567890_abc123","customer_id":"cus_abc123","message":"Payout initiated successfully","exchange_rate":0.0000666667,"fee":0.07,"percentage_fee":0.05,"fixed_fee":0.02,"amount":100000,"amount_usd":6.67,"currency":"VND","sender":{"name_first":"John","name_last":"Doe","ic_number":"E1234567","ic_type":"PP","ic_country":"SG","phone":"+6591234567","email":"john.doe@example.com"},"recipient":{"name":"MERCHANT NAME","account_no":"1234567890","bank_no":"014"},"partner_transaction_id":"partner_tx_123","updated_at":"2024-01-01T00:00:00.000Z"}`)

	resp, err := client.ExecutePayout(context.Background(), &gosqril.ExecutePayoutRequest{
		IdempotencyKey:       "payout-2024-0001",
		DP:                   nil,
		TxID:                 "tx_1234567890_abc123",
		AmountConfirmed:      "100000",
		Currency:             "VND",
		CustomerID:           "cus_abc123",
		PartnerTransactionID: "partner_tx_123",
		RecipientName:        "MERCHANT NAME",
	})
	if err != nil {
		t.Fatalf("ExecutePayout: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/executePayout", "",
		`{"tx_id":"tx_1234567890_abc123","amount_confirmed":100000,"currency":"VND","customer_id":"cus_abc123","partner_transaction_id":"partner_tx_123","recipient_name":"MERCHANT NAME"}`)

	if key := recorded.snapshot().header.Get("X-Idempotency-Key"); key != "payout-2024-0001" {
		t.Errorf("X-Idempotency-Key = %q, want payout-2024-0001", key)
	}

	if resp.Status != gosqril.TransactionStatusProcessing || resp.TxID != "tx_1234567890_abc123" ||
		resp.CustomerID != "cus_abc123" || resp.Message != "Payout initiated successfully" {
		t.Errorf("payout = %+v", resp)
	}

	got := []gosqril.Decimal{resp.ExchangeRate, resp.Fee, resp.PercentageFee, resp.FixedFee, resp.Amount, resp.AmountUSD}
	want := []gosqril.Decimal{"0.0000666667", "0.07", "0.05", "0.02", "100000", "6.67"}

	for index := range want {
		if got[index] != want[index] {
			t.Errorf("money field %d = %s, want %s", index, got[index], want[index])
		}
	}

	if resp.Currency != "VND" || resp.Sender["ic_country"] != "SG" || resp.Recipient["bank_no"] != "014" ||
		resp.PartnerTransactionID != "partner_tx_123" || resp.UpdatedAt != "2024-01-01T00:00:00.000Z" {
		t.Errorf("payout = %+v", resp)
	}
}

func TestExecutePayout_DecimalAmountAndDP(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusAccepted, `{"status":"PROCESSING","tx_id":"tx_1"}`)

	_, err := client.ExecutePayout(context.Background(), &gosqril.ExecutePayoutRequest{
		IdempotencyKey:  "key-1",
		DP:              ptr(0),
		TxID:            "tx_1",
		AmountConfirmed: "1000.50",
		Currency:        "PHP",
		CustomerID:      "cus_abc123",
	})
	if err != nil {
		t.Fatalf("ExecutePayout: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/executePayout", "dp=0",
		`{"tx_id":"tx_1","amount_confirmed":1000.50,"currency":"PHP","customer_id":"cus_abc123"}`)
}

func TestExecutePayout_BrokenAcceptedBodyIsUnknownOutcome(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"status":"PROCESSING","tx_id":`, ``, `<html>accepted</html>`} {
		client, recorded := newFakeServer(t, http.StatusAccepted, body)

		resp, err := client.ExecutePayout(context.Background(), &gosqril.ExecutePayoutRequest{
			IdempotencyKey: "key-1", TxID: "tx_1", AmountConfirmed: "100000", Currency: "VND", CustomerID: "cus_abc123",
		})
		if !errors.Is(err, gosqril.ErrInvalidResponse) || resp != nil {
			t.Errorf("body %q: resp = %v, error = %v, want ErrInvalidResponse", body, resp, err)
		}

		var apiErr *gosqril.APIError
		if errors.As(err, &apiErr) {
			t.Errorf("body %q: error = %v, want no *APIError", body, err)
		}

		if calls := recorded.snapshot().calls; calls != 1 {
			t.Errorf("body %q: calls = %d, want 1 (the SDK must not retry)", body, calls)
		}
	}
}

func TestExecutePayout_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusPaymentRequired,
		`{"error_code":"INSUFFICIENT_FUNDS","error_message":"Account balance insufficient"}`)

	_, err := client.ExecutePayout(context.Background(), &gosqril.ExecutePayoutRequest{
		IdempotencyKey: "key-1", TxID: "tx_1", AmountConfirmed: "100000", Currency: "VND", CustomerID: "cus_abc123",
	})

	assertAPIError(t, err, http.StatusPaymentRequired, gosqril.ErrorCodeInsufficientFunds)

	if !gosqril.IsInsufficientFunds(err) {
		t.Error("IsInsufficientFunds = false, want true")
	}
}

func TestExecutePayout_SigningSecretAccountGets401(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusUnauthorized,
		`{"error_code":"HMAC_SIGNATURE_REQUIRED","error_message":"HMAC signature required"}`)

	_, err := client.ExecutePayout(context.Background(), &gosqril.ExecutePayoutRequest{
		IdempotencyKey: "key-1", TxID: "tx_1", AmountConfirmed: "100000", Currency: "VND", CustomerID: "cus_abc123",
	})

	assertAPIError(t, err, http.StatusUnauthorized, gosqril.ErrorCodeHMACSignatureRequired)

	if !gosqril.IsUnauthorized(err) {
		t.Error("IsUnauthorized = false, want true")
	}
}
