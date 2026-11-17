package gosqril_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/giangtv270304/gosqril"
)

func TestAPIError_DocumentedShape(t *testing.T) {
	t.Parallel()

	const body = `{
  "error_code": "CUSTOMER_NOT_ELIGIBLE",
  "error_message": "Customer cus_1 cannot transact to PE until required sender profile fields are completed.",
  "details": {
    "customer_id": "cus_1",
    "country": "PE",
    "missing_sender_fields": [
      {"field": "sender.occupation", "type": "text", "required": true}
    ]
  }
}`

	client, _ := newFakeServer(t, http.StatusBadRequest, body)

	_, err := client.DecodeQR(context.Background(), &gosqril.DecodeQRRequest{QRString: "0002", CustomerID: "cus_1"})

	apiErr := requireAPIError(t, err, http.StatusBadRequest, gosqril.ErrorCodeCustomerNotEligible)

	if !strings.HasPrefix(apiErr.Message, "Customer cus_1 cannot transact to PE") {
		t.Errorf("message = %q", apiErr.Message)
	}

	if got := string(apiErr.Details["country"]); got != `"PE"` {
		t.Errorf("details.country = %s, want \"PE\"", got)
	}

	if _, ok := apiErr.Details["missing_sender_fields"]; !ok {
		t.Error("details.missing_sender_fields is missing")
	}

	if string(apiErr.RawBody) != body {
		t.Errorf("raw body = %s, want the body as sent", apiErr.RawBody)
	}

	if apiErr.RequestID != "" {
		t.Errorf("request id = %q, want empty", apiErr.RequestID)
	}

	wantText := "gosqril: api error: status=400 code=CUSTOMER_NOT_ELIGIBLE message=Customer cus_1"
	if !strings.HasPrefix(apiErr.Error(), wantText) {
		t.Errorf("Error() = %q, want prefix %q", apiErr.Error(), wantText)
	}
}

func TestAPIError_MessageFallsBackToErrorField(t *testing.T) {
	t.Parallel()

	client, _ := newFakeServer(t, http.StatusInternalServerError, `{"error":"upstream exploded"}`)

	_, err := client.GetAccountBalances(context.Background(), nil)

	apiErr := requireAPIError(t, err, http.StatusInternalServerError, "")
	if apiErr.Message != "upstream exploded" {
		t.Errorf("message = %q, want the error field", apiErr.Message)
	}
}

func TestAPIError_UnknownJSONShapeKeepsRawBody(t *testing.T) {
	t.Parallel()

	const body = `{"unexpected":true}`

	client, _ := newFakeServer(t, http.StatusTooManyRequests, body)

	_, err := client.PreviewQuotations(context.Background(), &gosqril.PreviewQuotationsRequest{Amount: "100", Currency: "USD"})

	apiErr := requireAPIError(t, err, http.StatusTooManyRequests, "")
	if apiErr.Message != body || string(apiErr.RawBody) != body {
		t.Errorf("message = %q, raw = %q, want the raw body in both", apiErr.Message, apiErr.RawBody)
	}

	if !gosqril.IsRateLimited(err) {
		t.Error("IsRateLimited = false, want true")
	}
}

func TestAPIError_LongNonJSONBodyIsTruncatedInMessage(t *testing.T) {
	t.Parallel()

	html := "<html>" + strings.Repeat("é", 400) + "</html>"

	client, _ := newFakeServer(t, http.StatusBadGateway, html)

	_, err := client.GetAccountBalances(context.Background(), nil)

	apiErr := requireAPIError(t, err, http.StatusBadGateway, "")

	if string(apiErr.RawBody) != html {
		t.Errorf("raw body has %d bytes, want the full %d", len(apiErr.RawBody), len(html))
	}

	if len(apiErr.Message) > 200 || len(apiErr.Message) < 190 || !strings.HasPrefix(html, apiErr.Message) || !utf8.ValidString(apiErr.Message) {
		t.Errorf("message = %d bytes %q, want a valid UTF-8 prefix of at most 200 bytes", len(apiErr.Message), apiErr.Message)
	}

	if len(apiErr.Error()) > 300 {
		t.Errorf("Error() has %d bytes, want it short", len(apiErr.Error()))
	}
}

func TestAPIError_DetailsThatAreNotAnObject(t *testing.T) {
	t.Parallel()

	for _, details := range []string{`"see support"`, `["a","b"]`, `42`, `null`} {
		body := `{"error_code":"ACCOUNT_PAYOUTS_LOCKED","error_message":"Payouts are locked","details":` + details + `}`

		client, _ := newFakeServer(t, http.StatusPaymentRequired, body)

		_, err := client.GetAccountBalances(context.Background(), nil)

		apiErr := requireAPIError(t, err, http.StatusPaymentRequired, gosqril.ErrorCodeAccountPayoutsLocked)

		if apiErr.Message != "Payouts are locked" || apiErr.Details != nil || string(apiErr.RawBody) != body {
			t.Errorf("details %s: message = %q, details = %v", details, apiErr.Message, apiErr.Details)
		}
	}
}

func TestStatusHelpers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		status int
		helper func(error) bool
	}{
		{"IsUnauthorized", http.StatusUnauthorized, gosqril.IsUnauthorized},
		{"IsNotFound", http.StatusNotFound, gosqril.IsNotFound},
		{"IsConflict", http.StatusConflict, gosqril.IsConflict},
		{"IsRateLimited", http.StatusTooManyRequests, gosqril.IsRateLimited},
		{"IsInsufficientFunds", http.StatusPaymentRequired, gosqril.IsInsufficientFunds},
	}

	for _, testCase := range testCases {
		wrapped := fmt.Errorf("outer: %w", &gosqril.APIError{StatusCode: testCase.status})
		if !testCase.helper(wrapped) {
			t.Errorf("%s = false for a wrapped status %d, want true", testCase.name, testCase.status)
		}

		if testCase.helper(&gosqril.APIError{StatusCode: http.StatusTeapot}) {
			t.Errorf("%s = true for status 418, want false", testCase.name)
		}

		if testCase.helper(errors.New("plain")) || testCase.helper(nil) {
			t.Errorf("%s = true for a non-API error, want false", testCase.name)
		}
	}
}
