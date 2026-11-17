package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

const decodeBody = `{"is_dynamic":true,"is_business":true,"amount":100000,"merchant":"MERCHANT NAME","country":"VN","tx_id":"tx_1234567890_abc123","currency":"VND","partner_transaction_id":"partner_tx_123","amount_usd":6.67,"fee":0.07,"percentage_fee":0.05,"fixed_fee":0.02,"missing_fields":[{"field":"recipient.name","type":"text","label":"Recipient name"}],"suggested_values":{"recipient.name":"MERCHANT NAME"},"sender":{"name_first":"John","ic_type":"PP"},"recipient":{"account_no":"1234567890","bank_no":"014"},"qr_type":"qrph_p2m","future_field":[1,2]}`

func assertDecodeResult(t *testing.T, result *gosqril.DecodeResult) {
	t.Helper()

	if !result.IsDynamic || !result.IsBusiness || result.Merchant != "MERCHANT NAME" || result.Country != "VN" {
		t.Errorf("result = %+v", result)
	}

	if result.TxID != "tx_1234567890_abc123" || result.Currency != "VND" || result.PartnerTransactionID != "partner_tx_123" {
		t.Errorf("ids = %s %s %s", result.TxID, result.Currency, result.PartnerTransactionID)
	}

	if result.Amount != "100000" {
		t.Errorf("amount = %s, want 100000", result.Amount)
	}

	assertDecimal(t, "amount_usd", result.AmountUSD, "6.67")
	assertDecimal(t, "fee", result.Fee, "0.07")
	assertDecimal(t, "percentage_fee", result.PercentageFee, "0.05")
	assertDecimal(t, "fixed_fee", result.FixedFee, "0.02")

	if len(result.MissingFields) != 1 || result.MissingFields[0].Field != "recipient.name" ||
		result.MissingFields[0].Type != "text" || result.MissingFields[0].Label != "Recipient name" ||
		result.MissingFields[0].Required != nil {
		t.Errorf("missing fields = %+v", result.MissingFields)
	}

	if result.SuggestedValues["recipient.name"] != "MERCHANT NAME" || result.Sender["ic_type"] != "PP" ||
		result.Recipient["bank_no"] != "014" || result.QRType != "qrph_p2m" {
		t.Errorf("free-form fields = %v %v %v %s", result.SuggestedValues, result.Sender, result.Recipient, result.QRType)
	}
}

func TestDecodeQR_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, decodeBody)

	resp, err := client.DecodeQR(context.Background(), &gosqril.DecodeQRRequest{
		DP:                   ptr(4),
		QRString:             "00020101021238570010A000000727",
		CustomerID:           "cus_abc123",
		PartnerTransactionID: "partner_tx_123",
		PartnerAppID:         "",
		PaymentCurrency:      "",
		WalletIdentifier:     "",
		CountryCode:          "VN",
		MerchantID:           "",
		AccountReference:     "",
	})
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/decodeQr", "dp=4",
		`{"qr_string":"00020101021238570010A000000727","customer_id":"cus_abc123","partner_transaction_id":"partner_tx_123","country_code":"VN"}`)
	assertDecodeResult(t, resp)
}

func TestDecodeQR_OpenAmountAndMerchantInfo(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusOK,
		`{"is_dynamic":false,"is_business":false,"amount":0,"merchant":"","merchant_id":"123456","merchant_name":"SHOP","payment_type":"paybill","merchant_info":{"merchant_id":"123456","display_name":"SHOP","country_code":"KE","payment_type":"voucher","lipa_number":"123456","account_reference":"INV1"},"country":"KE","tx_id":"tx_2","currency":"KES","partner_transaction_id":"","missing_fields":[{"field":"amount","type":"numeric","required":true}]}`)

	resp, err := client.DecodeQR(context.Background(), &gosqril.DecodeQRRequest{QRString: "123456", CustomerID: "cus_abc123"})
	if err != nil {
		t.Fatalf("DecodeQR: %v", err)
	}

	if resp.Amount != "0" || resp.AmountUSD != nil || resp.Fee != nil || resp.PercentageFee != nil || resp.FixedFee != nil {
		t.Errorf("money = %s %v %v, want 0 and nil prices", resp.Amount, resp.AmountUSD, resp.Fee)
	}

	if resp.PaymentType != gosqril.PaymentTypePaybill || resp.MerchantInfo == nil ||
		resp.MerchantInfo.PaymentType != "voucher" || resp.MerchantInfo.AccountReference != "INV1" {
		t.Errorf("merchant = %s %+v", resp.PaymentType, resp.MerchantInfo)
	}

	if len(resp.MissingFields) != 1 || resp.MissingFields[0].Required == nil || !*resp.MissingFields[0].Required {
		t.Errorf("missing fields = %+v", resp.MissingFields)
	}
}

func TestDecodeQR_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusBadRequest,
		`{"error_code":"INVALID_COUNTRY_CODE","error_message":"Invalid or unsupported country code for this account."}`)

	_, err := client.DecodeQR(context.Background(), &gosqril.DecodeQRRequest{QRString: "0002", CustomerID: "cus_abc123"})

	apiErr := requireAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeInvalidCountryCode)
	if apiErr.Message != "Invalid or unsupported country code for this account." {
		t.Errorf("message = %q", apiErr.Message)
	}
}
