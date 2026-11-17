package gosqril_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/giangtv270304/gosqril"
)

func TestDecodeQRUnregistered_Success(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, decodeBody)

	resp, err := client.DecodeQRUnregistered(context.Background(), &gosqril.DecodeQRUnregisteredRequest{
		DP:                   nil,
		QRString:             "00020101021238570010A000000727",
		PartnerTransactionID: "",
		PartnerAppID:         "app-1",
		PaymentCurrency:      "USD",
		WalletIdentifier:     "wallet-9",
		CountryCode:          "",
		MerchantID:           "123456",
		AccountReference:     "INV1",
	})
	if err != nil {
		t.Fatalf("DecodeQRUnregistered: %v", err)
	}

	assertRequest(t, recorded, http.MethodPost, "/decodeQrUnregistered", "",
		`{"qr_string":"00020101021238570010A000000727","partner_app_id":"app-1","payment_currency":"USD","wallet_identifier":"wallet-9","merchant_id":"123456","account_reference":"INV1"}`)
	assertDecodeResult(t, resp)
}

func TestDecodeQRUnregistered_APIError(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusBadRequest,
		`{"error_code":"INVALID_REQUEST","error_message":"customer_id is required to decode this QR"}`)

	_, err := client.DecodeQRUnregistered(context.Background(), &gosqril.DecodeQRUnregisteredRequest{QRString: "0002"})

	apiErr := requireAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeInvalidRequest)
	if apiErr.Message != "customer_id is required to decode this QR" {
		t.Errorf("message = %q", apiErr.Message)
	}
}
