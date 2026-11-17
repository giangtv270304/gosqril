package gosqril

import (
	"context"
	"net/http"
)

// PaymentType is the kind of KE/TZ merchant number. Unknown values from the API are kept as sent.
type PaymentType string

// Payment types returned for KE and TZ.
const (
	PaymentTypeTill    PaymentType = "till"
	PaymentTypePaybill PaymentType = "paybill"
)

// MerchantInfo describes a KE/TZ merchant. Every field is optional.
type MerchantInfo struct {
	MerchantID       string      `json:"merchant_id,omitempty"`
	DisplayName      string      `json:"display_name,omitempty"`
	CountryCode      string      `json:"country_code,omitempty"`
	PaymentType      PaymentType `json:"payment_type,omitempty"`
	LipaNumber       string      `json:"lipa_number,omitempty"`
	AccountReference string      `json:"account_reference,omitempty"`
}

// DecodeResult is the PENDING transaction created by DecodeQR, DecodeQRUnregistered and
// LookupPaymentCode. It never carries an exchange rate; call GetQuotation for that.
type DecodeResult struct {
	IsDynamic  bool `json:"is_dynamic"`
	IsBusiness bool `json:"is_business"`
	// Amount is in the local currency; "0" when the caller still has to supply it.
	Amount Decimal `json:"amount"`
	// Merchant may be empty. For LATAM unregistered decodes it is the QR's own label.
	Merchant     string        `json:"merchant"`
	MerchantID   string        `json:"merchant_id,omitempty"`
	MerchantName string        `json:"merchant_name,omitempty"`
	PaymentType  PaymentType   `json:"payment_type,omitempty"`
	MerchantInfo *MerchantInfo `json:"merchant_info,omitempty"`
	// Country is an ISO 3166 alpha-2 code; Currency is ISO 4217.
	Country              string `json:"country"`
	TxID                 string `json:"tx_id"`
	Currency             string `json:"currency"`
	PartnerTransactionID string `json:"partner_transaction_id"`
	// AmountUSD, Fee, PercentageFee and FixedFee are in USD; nil unless Amount is above zero
	// and pricing succeeded.
	AmountUSD     *Decimal `json:"amount_usd,omitempty"`
	Fee           *Decimal `json:"fee,omitempty"`
	PercentageFee *Decimal `json:"percentage_fee,omitempty"`
	FixedFee      *Decimal `json:"fixed_fee,omitempty"`
	// MissingFields lists data still required before ExecutePayout.
	MissingFields []MissingField `json:"missing_fields,omitempty"`
	// SuggestedValues holds hints keyed by field path.
	SuggestedValues map[string]any `json:"suggested_values,omitempty"`
	Sender          Party          `json:"sender,omitempty"`
	Recipient       Party          `json:"recipient,omitempty"`
	// QRType is set for the Philippines: qrph_p2m or qrph_p2p.
	QRType string `json:"qr_type,omitempty"`
}

// DecodeQRRequest carries a scanned QR payload. QRString and CustomerID are required.
type DecodeQRRequest struct {
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP *int `json:"-"`
	// QRString is the EMV or local QR payload; for KE/TZ a numeric till or paybill is accepted.
	QRString string `json:"qr_string"`
	// CustomerID is a registered customer and must not contain "/".
	CustomerID string `json:"customer_id"`
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

// DecodeQRResponse is the decoded QR and its new transaction.
type DecodeQRResponse = DecodeResult

// DecodeQR decodes a scanned QR for a registered customer and creates a PENDING transaction.
// Typed destinations (PIX key, phone, bank account) must use LookupPaymentCode instead.
func (c *Client) DecodeQR(ctx context.Context, req *DecodeQRRequest) (*DecodeQRResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireField("QRString", req.QRString); err != nil {
		return nil, err
	}

	if err := requireCustomerID(req.CustomerID); err != nil {
		return nil, err
	}

	return call[DecodeQRResponse](ctx, c, request{
		Method:         http.MethodPost,
		Path:           "/decodeQr",
		Query:          queryWithDP(req.DP),
		Body:           req,
		IdempotencyKey: "",
	})
}
