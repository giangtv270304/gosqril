package gosqril

import (
	"context"
	"net/http"
)

// DecodeQRUnregisteredRequest is DecodeQRRequest without a customer. QRString is required.
type DecodeQRUnregisteredRequest struct {
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP *int `json:"-"`
	// QRString is the EMV or local QR payload; for KE/TZ a numeric till or paybill is accepted.
	QRString string `json:"qr_string"`
	// PartnerTransactionID can be set once and never changed.
	PartnerTransactionID string `json:"partner_transaction_id,omitempty"`
	// PartnerAppID is 1 to 64 characters of [a-zA-Z0-9_-].
	PartnerAppID string `json:"partner_app_id,omitempty"`
	// PaymentCurrency is the ISO 4217 source currency when the QR does not carry one.
	PaymentCurrency  string `json:"payment_currency,omitempty"`
	WalletIdentifier string `json:"wallet_identifier,omitempty"`
	// CountryCode is an ISO 3166 alpha-2 override, required when the QR is not EMV-parseable.
	CountryCode string `json:"country_code,omitempty"`
	// MerchantID is the till or paybill number for KE/TZ.
	MerchantID string `json:"merchant_id,omitempty"`
	// AccountReference is the paybill account reference, required for Kenya paybills.
	AccountReference string `json:"account_reference,omitempty"`
}

// DecodeQRUnregisteredResponse is the decoded QR and its new transaction.
type DecodeQRUnregisteredResponse = DecodeResult

// DecodeQRUnregistered decodes a QR before the customer is known; the customer is attached at
// ExecutePayout. Not available for Thailand (400 INVALID_REQUEST).
func (c *Client) DecodeQRUnregistered(ctx context.Context, req *DecodeQRUnregisteredRequest) (*DecodeQRUnregisteredResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireField("QRString", req.QRString); err != nil {
		return nil, err
	}

	return call[DecodeQRUnregisteredResponse](ctx, c, request{
		Method:         http.MethodPost,
		Path:           "/decodeQrUnregistered",
		Query:          queryWithDP(req.DP),
		Body:           req,
		IdempotencyKey: "",
	})
}
