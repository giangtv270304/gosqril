package gosqril

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Values of APIError.Code documented by SQRIL. The API may send codes that are not listed here.
const (
	ErrorCodeAuthenticationRequired       = "AUTHENTICATION_REQUIRED"
	ErrorCodeInvalidCredentials           = "INVALID_CREDENTIALS"
	ErrorCodeInvalidHMACSignature         = "INVALID_HMAC_SIGNATURE"
	ErrorCodeHMACSignatureRequired        = "HMAC_SIGNATURE_REQUIRED"
	ErrorCodeTransactionNotOwned          = "TRANSACTION_NOT_OWNED"
	ErrorCodeCountryNotSupportedByAccount = "COUNTRY_NOT_SUPPORTED_BY_ACCOUNT"
	ErrorCodeKYCRequired                  = "KYC_REQUIRED"
	ErrorCodeInvalidRequest               = "INVALID_REQUEST"
	ErrorCodeMissingRequiredField         = "MISSING_REQUIRED_FIELD"
	ErrorCodeMissingRequiredFields        = "MISSING_REQUIRED_FIELDS"
	ErrorCodeInvalidQRFormat              = "INVALID_QR_FORMAT"
	ErrorCodeInvalidCountryCode           = "INVALID_COUNTRY_CODE"
	ErrorCodeUnsupportedQRType            = "UNSUPPORTED_QR_TYPE"
	ErrorCodeInvalidAmount                = "INVALID_AMOUNT"
	ErrorCodeInvalidCurrency              = "INVALID_CURRENCY"
	ErrorCodeInvalidStatus                = "INVALID_STATUS"
	ErrorCodeInvalidLimit                 = "INVALID_LIMIT"
	ErrorCodeInvalidStartAfter            = "INVALID_START_AFTER"
	ErrorCodeTransactionNotPending        = "TRANSACTION_NOT_PENDING"
	ErrorCodeQuotationExpired             = "QUOTATION_EXPIRED"
	ErrorCodeDailyLimitExceeded           = "DAILY_LIMIT_EXCEEDED"
	ErrorCodeMonthlyLimitExceeded         = "MONTHLY_LIMIT_EXCEEDED"
	ErrorCodeMethodNotAllowed             = "METHOD_NOT_ALLOWED"
	ErrorCodeIdempotencyKeyRequired       = "IDEMPOTENCY_KEY_REQUIRED"
	ErrorCodeCustomerNotEligible          = "CUSTOMER_NOT_ELIGIBLE"
	ErrorCodeCustomerBlocked              = "CUSTOMER_BLOCKED"
	ErrorCodeTransactionNotFound          = "TRANSACTION_NOT_FOUND"
	ErrorCodeResourceNotFound             = "RESOURCE_NOT_FOUND"
	ErrorCodeRecipientLookupUnavailable   = "RECIPIENT_LOOKUP_UNAVAILABLE"
	ErrorCodeInsufficientFunds            = "INSUFFICIENT_FUNDS"
	ErrorCodeAccountPayoutsLocked         = "ACCOUNT_PAYOUTS_LOCKED"
	ErrorCodeInternalServerError          = "INTERNAL_SERVER_ERROR"
	ErrorCodeProviderError                = "PROVIDER_ERROR"
	ErrorCodeRateLimitExceeded            = "RATE_LIMIT_EXCEEDED"
)

const maxNonJSONMessageBytes = 200

// ErrInvalidResponse wraps the error of a response that could not be read as the documented JSON:
// a 2xx with a broken body, a 1xx or 3xx status, or a body over 10 MiB. The request may have been processed.
var ErrInvalidResponse = errors.New("gosqril: invalid response")

// APIError is returned for every response with status 400 or above. Read it with errors.As.
type APIError struct {
	StatusCode int
	// Code is the API's error_code, empty when the body was not the documented JSON shape.
	Code string
	// Message is error_message, else the "error" field, else the first 200 bytes of the body.
	Message string
	// Details is the free-form "details" object, values left as raw JSON. Nil when absent or
	// when "details" is not an object.
	Details map[string]json.RawMessage
	// RequestID is always empty: SQRIL documents no request id.
	RequestID string
	// RawBody is the response body as received, also when it is not JSON (a proxy HTML page).
	RawBody []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("gosqril: api error: status=%d code=%s message=%s", e.StatusCode, e.Code, e.Message)
}

type errorBody struct {
	ErrorCode    string          `json:"error_code"`
	ErrorMessage string          `json:"error_message"`
	Error        string          `json:"error"`
	Details      json.RawMessage `json:"details"`
}

func newAPIError(statusCode int, raw []byte) *APIError {
	apiErr := &APIError{
		StatusCode: statusCode,
		Code:       "",
		Message:    truncatedText(raw),
		Details:    nil,
		RequestID:  "",
		RawBody:    raw,
	}

	var parsed errorBody
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return apiErr
	}

	apiErr.Code = parsed.ErrorCode

	var details map[string]json.RawMessage
	if err := json.Unmarshal(parsed.Details, &details); err == nil {
		apiErr.Details = details
	}

	switch {
	case parsed.ErrorMessage != "":
		apiErr.Message = parsed.ErrorMessage
	case parsed.Error != "":
		apiErr.Message = parsed.Error
	}

	return apiErr
}

func truncatedText(raw []byte) string {
	if len(raw) <= maxNonJSONMessageBytes {
		return string(raw)
	}

	return strings.ToValidUTF8(string(raw[:maxNonJSONMessageBytes]), "")
}

func hasStatus(err error, statusCode int) bool {
	var apiErr *APIError

	return errors.As(err, &apiErr) && apiErr.StatusCode == statusCode
}

// IsNotFound reports whether err is an *APIError with status 404.
func IsNotFound(err error) bool { return hasStatus(err, http.StatusNotFound) }

// IsUnauthorized reports whether err is an *APIError with status 401.
func IsUnauthorized(err error) bool { return hasStatus(err, http.StatusUnauthorized) }

// IsConflict reports whether err is an *APIError with status 409. SQRIL documents no 409 today.
func IsConflict(err error) bool { return hasStatus(err, http.StatusConflict) }

// IsRateLimited reports whether err is an *APIError with status 429. The SDK does not retry by itself.
func IsRateLimited(err error) bool { return hasStatus(err, http.StatusTooManyRequests) }

// IsInsufficientFunds reports whether err is an *APIError with status 402
// (codes INSUFFICIENT_FUNDS and ACCOUNT_PAYOUTS_LOCKED).
func IsInsufficientFunds(err error) bool { return hasStatus(err, http.StatusPaymentRequired) }
