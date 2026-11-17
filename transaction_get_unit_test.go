package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

const transactionBody = `{"id":"tx_1234567890_abc123","exchange_app_id":"app_1","customer_id":"cus_abc123","status":"SUCCESS","amount":100000,"amount_usd":6.67,"fee":0.10,"percentage_fee":0.08,"fixed_fee":0.02,"total":6.77,"exchange_rate":0.0000666667,"currency":"VND","country_code":"VN","partner_transaction_id":"partner_tx_123","recipient_bank_no":"014","recipient_account_no":"1234567890","recipient_account_name":"MERCHANT NAME","sender":{"name_first":"John"},"recipient":{"name":"MERCHANT NAME"},"qr_data":{"raw":"0002"},"missing_fields":[],"quotation_expires_at":"2024-01-01T00:30:00.000Z","created_at":"2024-01-01T00:00:00.000Z","updated_at":"2024-01-01T00:01:00.000Z","completed_at":"2024-01-01T00:01:00.000Z","future_field":"x"}`

func assertTransaction(t *testing.T, transaction *gosqril.Transaction) {
	t.Helper()

	if transaction.ID != "tx_1234567890_abc123" || transaction.ExchangeAppID != "app_1" || transaction.CustomerID != "cus_abc123" ||
		transaction.Status != gosqril.TransactionStatusSuccess || transaction.Currency != "VND" || transaction.CountryCode != "VN" {
		t.Errorf("transaction = %+v", transaction)
	}

	if transaction.Amount != "100000" {
		t.Errorf("amount = %s, want 100000", transaction.Amount)
	}

	assertDecimal(t, "amount_usd", transaction.AmountUSD, "6.67")
	assertDecimal(t, "fee", transaction.Fee, "0.10")
	assertDecimal(t, "percentage_fee", transaction.PercentageFee, "0.08")
	assertDecimal(t, "fixed_fee", transaction.FixedFee, "0.02")
	assertDecimal(t, "total", transaction.Total, "6.77")
	assertDecimal(t, "exchange_rate", transaction.ExchangeRate, "0.0000666667")

	if transaction.RecipientBankNo != "014" || transaction.RecipientAccountNo != "1234567890" ||
		transaction.RecipientAccountName != "MERCHANT NAME" || transaction.PartnerTransactionID != "partner_tx_123" {
		t.Errorf("recipient columns = %+v", transaction)
	}

	if transaction.Sender["name_first"] != "John" || transaction.Recipient["name"] != "MERCHANT NAME" || transaction.QRData["raw"] != "0002" {
		t.Errorf("free-form fields = %v %v %v", transaction.Sender, transaction.Recipient, transaction.QRData)
	}

	if transaction.QuotationExpiresAt != "2024-01-01T00:30:00.000Z" || transaction.CreatedAt != "2024-01-01T00:00:00.000Z" ||
		transaction.CompletedAt != "2024-01-01T00:01:00.000Z" || transaction.RefundedAt != "" || transaction.ErrorMessage != "" {
		t.Errorf("timestamps = %+v", transaction)
	}
}

func TestGetTransaction_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, `{"transaction":`+transactionBody+`}`)

	resp, err := client.GetTransaction(context.Background(), &gosqril.GetTransactionRequest{
		TransactionID: "tx_1234567890_abc123",
		DP:            ptr(2),
	})
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}

	assertRequest(t, recorded, http.MethodGet, "/getTransaction", "dp=2&transaction_id=tx_1234567890_abc123", "")
	assertTransaction(t, &resp.Transaction)
}

func TestGetTransaction_PendingAndUnknownStatus(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusOK,
		`{"transaction":{"id":"tx_2","status":"ON_HOLD","amount":0,"currency":"COP","amount_usd":null,"error_message":"Payout failed"}}`)

	resp, err := client.GetTransaction(context.Background(), &gosqril.GetTransactionRequest{TransactionID: "tx_2"})
	if err != nil {
		t.Fatalf("GetTransaction: %v", err)
	}

	transaction := resp.Transaction
	if transaction.Status != "ON_HOLD" || transaction.Amount != "0" || transaction.ErrorMessage != "Payout failed" {
		t.Errorf("transaction = %+v", transaction)
	}

	if transaction.AmountUSD != nil || transaction.Fee != nil || transaction.Total != nil || transaction.ExchangeRate != nil {
		t.Errorf("optional money = %v %v %v %v, want all nil", transaction.AmountUSD, transaction.Fee, transaction.Total, transaction.ExchangeRate)
	}
}

func TestGetTransaction_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusNotFound,
		`{"error_code":"TRANSACTION_NOT_FOUND","error_message":"Transaction not found"}`)

	_, err := client.GetTransaction(context.Background(), &gosqril.GetTransactionRequest{TransactionID: "tx_missing"})

	assertAPIError(t, err, http.StatusNotFound, gosqril.ErrorCodeTransactionNotFound)

	if !gosqril.IsNotFound(err) {
		t.Error("IsNotFound = false, want true")
	}
}
