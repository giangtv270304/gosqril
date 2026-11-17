package gosqril

import (
	"context"
	"net/http"
)

// PaymentMethod is the kind of typed destination. Which one applies depends on the corridor.
type PaymentMethod string

// Payment methods accepted by LookupPaymentCode.
const (
	PaymentMethodPaymentKey   PaymentMethod = "payment_key"
	PaymentMethodEWallet      PaymentMethod = "ewallet"
	PaymentMethodBankTransfer PaymentMethod = "bank_transfer"
)

// AccountType is the recipient bank account type, used for Bolivia.
type AccountType string

// Account types accepted by LookupPaymentCode.
const (
	AccountTypeSavings  AccountType = "SAVINGS"
	AccountTypeChecking AccountType = "CHECKING"
)

// LookupRecipient is the recipient identity. Bolivia bank transfers need NameFirst, NameLast and
// ICNumber; set only the fields the corridor asks for.
type LookupRecipient struct {
	Name      string `json:"name,omitempty"`
	NameFirst string `json:"name_first,omitempty"`
	NameLast  string `json:"name_last,omitempty"`
	ICNumber  string `json:"ic_number,omitempty"`
}

// LookupPaymentCodeRequest describes a typed destination. CustomerID and CountryCode are required;
// the other fields depend on the corridor and are validated by the API, not by the SDK.
type LookupPaymentCodeRequest struct {
	// DP is the number of decimal places (0 to 10) of money fields in the response; nil for the API default of 2.
	DP *int `json:"-"`
	// CustomerID is a registered customer and must not contain "/".
	CustomerID string `json:"customer_id"`
	// CountryCode is the ISO 3166 alpha-2 destination country.
	CountryCode string `json:"country_code"`
	// PaymentMethod is payment_key (AR, BR, CO), ewallet (ID, PE) or bank_transfer
	// (ID, BO, NG, GH, ZM). Leave empty for KE, TZ and VN, which use MerchantID.
	PaymentMethod PaymentMethod `json:"payment_method,omitempty"`
	// PaymentDestination is the PIX key, BREB key, phone, wallet phone or bank account number.
	PaymentDestination string `json:"payment_destination,omitempty"`
	// PaymentDestinationCode is the scheme, wallet or bank code: PIX, BREB, DANA, YAPE, or a
	// code from ListBanks.
	PaymentDestinationCode string           `json:"payment_destination_code,omitempty"`
	AccountType            AccountType      `json:"account_type,omitempty"`
	Recipient              *LookupRecipient `json:"recipient,omitempty"`
	// MerchantID is the till or paybill number (KE/TZ) or the bank account number (VN).
	MerchantID string `json:"merchant_id,omitempty"`
	// AccountReference is the paybill reference (KE/TZ) or the bank BIN (VN).
	AccountReference string `json:"account_reference,omitempty"`
	// Amount is in the local currency. Required for Peru; nil elsewhere.
	Amount *Decimal `json:"amount,omitempty"`
	// Currency is ISO 4217 and defaults to the corridor currency.
	Currency string `json:"currency,omitempty"`
	// PartnerTransactionID can be set once and never changed.
	PartnerTransactionID string `json:"partner_transaction_id,omitempty"`
	// PartnerAppID is 1 to 64 characters of [a-zA-Z0-9_-].
	PartnerAppID     string `json:"partner_app_id,omitempty"`
	WalletIdentifier string `json:"wallet_identifier,omitempty"`
}

// LookupPaymentCodeResponse is the resolved destination and its new transaction.
type LookupPaymentCodeResponse = DecodeResult

// LookupPaymentCode resolves a typed destination (not a QR) for a registered customer and
// creates a PENDING transaction.
func (c *Client) LookupPaymentCode(ctx context.Context, req *LookupPaymentCodeRequest) (*LookupPaymentCodeResponse, error) {
	if req == nil {
		return nil, ErrNilRequest
	}

	if err := requireCustomerID(req.CustomerID); err != nil {
		return nil, err
	}

	if err := requireField("CountryCode", req.CountryCode); err != nil {
		return nil, err
	}

	return call[LookupPaymentCodeResponse](ctx, c, request{
		Method:         http.MethodPost,
		Path:           "/lookupPaymentCode",
		Query:          queryWithDP(req.DP),
		Body:           req,
		IdempotencyKey: "",
	})
}
