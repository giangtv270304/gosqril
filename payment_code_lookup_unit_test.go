package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestLookupPaymentCode_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, decodeBody)

	resp, err := client.LookupPaymentCode(context.Background(), &gosqril.LookupPaymentCodeRequest{
		DP:                     ptr(2),
		CustomerID:             "cus_abc123",
		CountryCode:            "BO",
		PaymentMethod:          gosqril.PaymentMethodBankTransfer,
		PaymentDestination:     "1234567890",
		PaymentDestinationCode: "1001",
		AccountType:            gosqril.AccountTypeSavings,
		Recipient:              &gosqril.LookupRecipient{NameFirst: "Jane", NameLast: "Smith", ICNumber: "7654321"},
		MerchantID:             "",
		AccountReference:       "",
		Amount:                 ptr(gosqril.Decimal("150.50")),
		Currency:               "BOB",
		PartnerTransactionID:   "partner_tx_123",
		PartnerAppID:           "",
		WalletIdentifier:       "",
	})
	if err != nil {
		t.Fatalf("LookupPaymentCode: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/lookupPaymentCode", "dp=2",
		`{"customer_id":"cus_abc123","country_code":"BO","payment_method":"bank_transfer","payment_destination":"1234567890","payment_destination_code":"1001","account_type":"SAVINGS","recipient":{"name_first":"Jane","name_last":"Smith","ic_number":"7654321"},"amount":150.50,"currency":"BOB","partner_transaction_id":"partner_tx_123"}`)
	assertDecodeResult(t, resp)
}

func TestLookupPaymentCode_MinimalBody(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, decodeBody)

	_, err := client.LookupPaymentCode(context.Background(), &gosqril.LookupPaymentCodeRequest{
		CustomerID:  "cus_abc123",
		CountryCode: "KE",
		MerchantID:  "123456",
	})
	if err != nil {
		t.Fatalf("LookupPaymentCode: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/lookupPaymentCode", "",
		`{"customer_id":"cus_abc123","country_code":"KE","merchant_id":"123456"}`)
}

func TestLookupPaymentCode_RecipientName(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, decodeBody)

	_, err := client.LookupPaymentCode(context.Background(), &gosqril.LookupPaymentCodeRequest{
		CustomerID:             "cus_abc123",
		CountryCode:            "PH",
		PaymentMethod:          gosqril.PaymentMethodBankTransfer,
		PaymentDestination:     "1234567890",
		PaymentDestinationCode: "BDO",
		Recipient:              &gosqril.LookupRecipient{Name: "Jane Smith"},
	})
	if err != nil {
		t.Fatalf("LookupPaymentCode: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/lookupPaymentCode", "",
		`{"customer_id":"cus_abc123","country_code":"PH","payment_method":"bank_transfer","payment_destination":"1234567890","payment_destination_code":"BDO","recipient":{"name":"Jane Smith"}}`)
}

func TestLookupPaymentCode_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusServiceUnavailable,
		`{"error_code":"RECIPIENT_LOOKUP_UNAVAILABLE","error_message":"Recipient account name could not be resolved"}`)

	_, err := client.LookupPaymentCode(context.Background(), &gosqril.LookupPaymentCodeRequest{
		CustomerID: "cus_abc123", CountryCode: "CO",
	})

	assertAPIError(t, err, http.StatusServiceUnavailable, gosqril.ErrorCodeRecipientLookupUnavailable)
}
