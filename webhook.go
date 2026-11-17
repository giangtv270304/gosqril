package gosqril

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// WebhookSignatureHeader is the request header that carries the webhook signature.
const WebhookSignatureHeader = "X-SQRIL-Signature"

// Errors returned by the webhook functions. Compare with errors.Is.
var (
	ErrInvalidWebhookSignature = errors.New("gosqril: invalid webhook signature")
	ErrEmptyWebhookSecret      = errors.New("gosqril: webhook secret is required")
	ErrInvalidWebhookPayload   = errors.New("gosqril: invalid webhook payload")
)

// WebhookEvent is the body SQRIL posts when a transaction is finalized. There is no event name:
// tell success, failed and refunded apart by Status. Deliveries may repeat; key on TxID and Status.
type WebhookEvent struct {
	TxID string `json:"tx_id"`
	// Status is SUCCESS, FAILED or REFUNDED. An unknown value is kept as sent.
	Status TransactionStatus `json:"status"`
	// Amount is in the transaction's local currency; the payload carries no currency.
	Amount *Decimal `json:"amount,omitempty"`
	// Fee, PercentageFee and FixedFee are in USD.
	Fee           *Decimal `json:"fee,omitempty"`
	PercentageFee *Decimal `json:"percentage_fee,omitempty"`
	FixedFee      *Decimal `json:"fixed_fee,omitempty"`
	// ErrorMessage is the payout failure reason, on FAILED only.
	ErrorMessage string `json:"error_message,omitempty"`
	// RefundReason is set on REFUNDED only, and only when SQRIL operations gave a reason.
	RefundReason string `json:"refund_reason,omitempty"`
	Sender       Party  `json:"sender,omitempty"`
	Recipient    Party  `json:"recipient,omitempty"`
}

// VerifyWebhookSignature checks signature (the X-SQRIL-Signature header) against
// base64(HMAC-SHA256(secret, rawBody)) in constant time. Pass the exact bytes received, never a
// re-encoded JSON. It returns ErrInvalidWebhookSignature when the signature is empty or wrong.
func VerifyWebhookSignature(secret string, rawBody []byte, signature string) error {
	if secret == "" {
		return ErrEmptyWebhookSecret
	}

	received, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(received) == 0 {
		return ErrInvalidWebhookSignature
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)

	if !hmac.Equal(received, mac.Sum(nil)) {
		return ErrInvalidWebhookSignature
	}

	return nil
}

// ParseWebhookEvent decodes a webhook body. It fails with ErrInvalidWebhookPayload on invalid JSON
// or when tx_id or status is missing. Verify the signature first; this function does not.
func ParseWebhookEvent(rawBody []byte) (*WebhookEvent, error) {
	event := new(WebhookEvent)
	if err := json.Unmarshal(rawBody, event); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidWebhookPayload, err)
	}

	if event.TxID == "" {
		return nil, fmt.Errorf("%w: tx_id is missing", ErrInvalidWebhookPayload)
	}

	if event.Status == "" {
		return nil, fmt.Errorf("%w: status is missing", ErrInvalidWebhookPayload)
	}

	return event, nil
}
