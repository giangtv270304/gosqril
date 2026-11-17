package gosqril_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/giangtv270304/gosqril"
)

const (
	webhookSecret      = "whsec_unit_test_secret"
	webhookSuccessBody = `{"tx_id":"tx_1234567890_abc123","status":"SUCCESS","amount":100000,"fee":0.35,"percentage_fee":0.25,"fixed_fee":0.1,"sender":{"name_first":"John","name_last":"Doe","country":"MY","ic_number":"123456789012","ic_type":"NIC","phone":"+60123456789","email":"john.doe@example.com"},"recipient":{"name_first":"Jane","name_last":"Smith","country":"VN","account_no":"1234567890","bank_code":"970436","bank_name":"Vietcombank"}}`
	// base64(HMAC-SHA256(webhookSecret, webhookSuccessBody)), computed once outside the SDK.
	webhookSuccessSignature = "EiI8Lzxbq7cCl6vXpsmksN3mvLNfnnwL0zZjDRrb734="
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func TestVerifyWebhook(t *testing.T) {
	t.Parallel()

	body := []byte(webhookSuccessBody)

	if err := gosqril.VerifyWebhookSignature(webhookSecret, body, webhookSuccessSignature); err != nil {
		t.Fatalf("fixed vector: %v, want nil", err)
	}

	if err := gosqril.VerifyWebhookSignature(webhookSecret, body, " "+webhookSuccessSignature+"\n"); err != nil {
		t.Errorf("signature with surrounding spaces: %v, want nil", err)
	}

	for index := range body {
		changed := append([]byte(nil), body...)
		changed[index] ^= 0x01

		if err := gosqril.VerifyWebhookSignature(webhookSecret, changed, webhookSuccessSignature); !errors.Is(err, gosqril.ErrInvalidWebhookSignature) {
			t.Fatalf("body changed at byte %d: error = %v, want ErrInvalidWebhookSignature", index, err)
		}
	}

	rawMAC, _ := base64.StdEncoding.DecodeString(webhookSuccessSignature)

	rejected := map[string]string{
		"empty signature":          "",
		"not base64":               "%%%not-base64%%%",
		"hex encoding":             hex.EncodeToString(rawMAC),
		"truncated":                base64.StdEncoding.EncodeToString(rawMAC[:16]),
		"signed with other secret": sign("other-secret", body),
		"signature of other body":  sign(webhookSecret, []byte(`{}`)),
	}
	for name, signature := range rejected {
		if err := gosqril.VerifyWebhookSignature(webhookSecret, body, signature); !errors.Is(err, gosqril.ErrInvalidWebhookSignature) {
			t.Errorf("%s: error = %v, want ErrInvalidWebhookSignature", name, err)
		}
	}

	if err := gosqril.VerifyWebhookSignature("", body, sign("", body)); !errors.Is(err, gosqril.ErrEmptyWebhookSecret) {
		t.Errorf("empty secret: error = %v, want ErrEmptyWebhookSecret", err)
	}
}

func TestParseWebhookEvent_Success(t *testing.T) {
	t.Parallel()

	event, err := gosqril.ParseWebhookEvent([]byte(webhookSuccessBody))
	if err != nil {
		t.Fatalf("ParseWebhookEvent: %v", err)
	}

	if event.TxID != "tx_1234567890_abc123" || event.Status != gosqril.TransactionStatusSuccess {
		t.Errorf("event = %s %s", event.TxID, event.Status)
	}

	assertDecimal(t, "amount", event.Amount, "100000")
	assertDecimal(t, "fee", event.Fee, "0.35")
	assertDecimal(t, "percentage_fee", event.PercentageFee, "0.25")
	assertDecimal(t, "fixed_fee", event.FixedFee, "0.1")

	if event.Sender["ic_type"] != "NIC" || event.Recipient["bank_name"] != "Vietcombank" {
		t.Errorf("sender = %v, recipient = %v", event.Sender, event.Recipient)
	}
}

func TestParseWebhookEvent_FailedAndRefunded(t *testing.T) {
	t.Parallel()

	failed, err := gosqril.ParseWebhookEvent([]byte(
		`{"tx_id":"tx_2","status":"FAILED","error_message":"Transaction failed","sender":{"name_first":"John"},"recipient":{"name_first":"Jane"}}`))
	if err != nil {
		t.Fatalf("failed event: %v", err)
	}

	if failed.Status != gosqril.TransactionStatusFailed || failed.ErrorMessage != "Transaction failed" {
		t.Errorf("failed event = %+v", failed)
	}

	if failed.Amount != nil || failed.Fee != nil || failed.PercentageFee != nil || failed.FixedFee != nil {
		t.Errorf("failed event money = %v %v %v %v, want all nil", failed.Amount, failed.Fee, failed.PercentageFee, failed.FixedFee)
	}

	refunded, err := gosqril.ParseWebhookEvent([]byte(
		`{"tx_id":"tx_3","status":"REFUNDED","amount":100000,"fee":0.35,"percentage_fee":0.25,"fixed_fee":0.1,"refund_reason":"Payment reversal","future_field":{"a":1}}`))
	if err != nil {
		t.Fatalf("refunded event: %v", err)
	}

	if refunded.Status != gosqril.TransactionStatusRefunded || refunded.RefundReason != "Payment reversal" {
		t.Errorf("refunded event = %+v", refunded)
	}

	assertDecimal(t, "amount", refunded.Amount, "100000")
}

func TestParseWebhookEvent_UnknownStatusIsKept(t *testing.T) {
	t.Parallel()

	event, err := gosqril.ParseWebhookEvent([]byte(`{"tx_id":"tx_4","status":"REVERSED"}`))
	if err != nil {
		t.Fatalf("ParseWebhookEvent: %v", err)
	}

	if event.Status != "REVERSED" {
		t.Errorf("status = %q, want REVERSED", event.Status)
	}
}

func TestParseWebhookEvent_Invalid(t *testing.T) {
	t.Parallel()

	for _, body := range []string{``, `not json`, `{"status":"SUCCESS"}`, `{"tx_id":"tx_5"}`, `{"tx_id":"tx_5","status":"SUCCESS","amount":"abc"}`} {
		event, err := gosqril.ParseWebhookEvent([]byte(body))
		if !errors.Is(err, gosqril.ErrInvalidWebhookPayload) || event != nil {
			t.Errorf("body %q: event = %v, error = %v, want ErrInvalidWebhookPayload", body, event, err)
		}
	}
}
