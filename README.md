# gosqril

Go client for the SQRIL SaaS payout API: register a sender, decode a QR or look up a typed destination, get a quotation, execute the payout, and verify the webhook.

## Install

```bash
go get github.com/giangtv270304/gosqril
```

Requires Go 1.23+. No dependencies outside the standard library.

## First call

Ask SQRIL for the `client_id` and `client_secret` of a staging account, export them as `SQRIL_CLIENT_ID` and `SQRIL_CLIENT_SECRET`, then run this file:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/giangtv270304/gosqril"
)

func main() {
	client, err := gosqril.NewClient(gosqril.SandboxBaseURL, os.Getenv("SQRIL_CLIENT_ID"), os.Getenv("SQRIL_CLIENT_SECRET"))
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.GetAccountBalances(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(resp.Balances["USD"].Available)
}
```

A longer program is in `example/quickstart`.

## Authentication

Every request carries HTTP Basic credentials (`client_id:client_secret`); the client adds the header for you. The client secret is for servers only: never ship it in a browser or a mobile app.

| Environment | Base URL |
|---|---|
| Sandbox (staging) | `gosqril.SandboxBaseURL` |
| Production | `gosqril.ProductionBaseURL` |

## Flows

Runnable programs, configured from `.env.example`: `example/quickstart`, `example/payout` (decode, quote; `-execute` pays), `example/webhook`.

### Pay a QR or a typed destination

1. `RegisterCustomer` once per sender. Keep `CustomerID`. `CheckCustomer` shows in `Allowed` which corridors the profile satisfies; `UpdateCustomer` fills the gaps.
2. `DecodeQR` for a scanned QR, or `LookupPaymentCode` for a typed destination (PIX key, BREB key, phone, bank account). Both create a `PENDING` transaction and return `TxID`. `MissingFields` lists what is still needed.
3. `GetQuotation` prices the transaction. The quotation is valid for 30 minutes.
4. `ExecutePayout` with the quoted `Amount` and `Currency`. It returns `PROCESSING`.
5. The final status (`SUCCESS`, `FAILED`, later maybe `REFUNDED`) arrives by webhook, or poll `GetTransaction`.

```go
lookup, err := client.LookupPaymentCode(ctx, &gosqril.LookupPaymentCodeRequest{
	CustomerID:             customerID,
	CountryCode:            "CO",
	PaymentMethod:          gosqril.PaymentMethodPaymentKey,
	PaymentDestination:     brebKey,
	PaymentDestinationCode: "BREB",
})
if err != nil {
	return err
}

amount := gosqril.Decimal("5000")

quotation, err := client.GetQuotation(ctx, &gosqril.GetQuotationRequest{
	TxID:       lookup.TxID,
	CustomerID: customerID,
	Amount:     &amount,
})
if err != nil {
	return err
}

payout, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
	IdempotencyKey:  orderID,
	TxID:            lookup.TxID,
	AmountConfirmed: quotation.Amount,
	Currency:        quotation.Currency,
	CustomerID:      customerID,
})
```

`ExecutePayout` moves money and needs an `IdempotencyKey`: one value per logical payout, chosen by you, reused on every retry of that payout. The SDK returns an error without calling the API when it is empty, and never generates one.

When `ExecutePayout` returns an error, the payout was refused only if the error is an `*APIError` with a 4xx status (or the SDK failed before sending: `ErrMissingField`, `ErrInvalidField`, `ErrInvalidDecimal`). Every other error means the outcome is unknown: a network error, a timeout, a 5xx, or `ErrInvalidResponse` (the answer was a 2xx with a body that could not be read, or a redirect). Then call `GetTransaction`, or resend with the same `IdempotencyKey`. Never use a new key for the same payout: that can pay twice.

```go
var apiErr *gosqril.APIError
refused := errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
```

A payout that ended `FAILED` is retried as a new payment: new decode, new quotation, new key.

### Receive a webhook

SQRIL posts a JSON body when a transaction is finalized. Verify the signature over the raw bytes you received, before parsing:

```go
raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
if err != nil {
	http.Error(w, "read error", http.StatusBadRequest)
	return
}

if err := gosqril.VerifyWebhookSignature(webhookSecret, raw, r.Header.Get(gosqril.WebhookSignatureHeader)); err != nil {
	http.Error(w, "invalid signature", http.StatusUnauthorized)
	return
}

event, err := gosqril.ParseWebhookEvent(raw)
if err != nil {
	http.Error(w, "invalid payload", http.StatusBadRequest)
	return
}

switch event.Status {
case gosqril.TransactionStatusSuccess, gosqril.TransactionStatusFailed, gosqril.TransactionStatusRefunded:
	// update the payment identified by event.TxID
}

w.Header().Set("Content-Type", "application/json")
_, _ = w.Write([]byte(`{"received": true, "message": "Webhook processed successfully"}`))
```

The payload has no event name and no currency; tell the three events apart by `Status`. Make the handler idempotent on `TxID` + `Status`.

## Methods

| Method | What it does |
|---|---|
| `RegisterCustomer` | Creates or updates a sender profile |
| `CheckCustomer` | Returns one customer and the corridors it may use |
| `UpdateCustomer` | Changes email, phone, occupation, address, gender, nationality |
| `DecodeQR` | Decodes a scanned QR for a registered customer; creates a `PENDING` transaction |
| `DecodeQRUnregistered` | Decodes a QR before the customer is known |
| `LookupPaymentCode` | Resolves a typed destination; creates a `PENDING` transaction |
| `GetQuotation` | Prices a `PENDING` transaction |
| `PreviewQuotations` | Indicative prices for every corridor; creates nothing |
| `ExecutePayout` | Executes the payout; returns `PROCESSING` |
| `ListOccupations` | Occupation codes for `RegisterCustomer` |
| `ListTransactions` | One page of transactions, newest first |
| `GetTransaction` | Returns one transaction |
| `GetAccountBalances` | Balances per currency |
| `ListBanks` | Bank codes of one country |
| `VerifyWebhookSignature`, `ParseWebhookEvent` | Webhook signature check and payload decoding |

### Money

Amounts and rates are `gosqril.Decimal`: a decimal string in Go (`"1000.50"`), in standard currency units, not minor units. SQRIL uses JSON numbers on the wire; the SDK writes the string as a bare number and reads the digits exactly as sent, never through `float64`. Do not parse a `Decimal` into a float; use a decimal library for arithmetic.

- `Amount` and `AmountConfirmed` are in the transaction's local currency. `AmountUSD`, `Fee`, `PercentageFee`, `FixedFee` and `Total` are in USD. `ExchangeRate` is USD per one unit of the local currency.
- Encoding fails with `ErrInvalidDecimal` for anything but digits, an optional leading `-` and an optional fraction.
- The SDK does not check decimal places per currency. VND and IDR take whole numbers only; too many decimals is a 400 `INVALID_AMOUNT`.
- `DP` on a request sets the decimal places of money fields in the response (0 to 10, default 2). The default can round a sub-cent fee to 0.

## Errors

Every response with status 400 or above is a `*gosqril.APIError`.

```go
var apiErr *gosqril.APIError
if errors.As(err, &apiErr) {
	log.Printf("%s (status %d, code %s)", apiErr.Message, apiErr.StatusCode, apiErr.Code)
}

if gosqril.IsNotFound(err) {
	// 404
}
```

`Code` is SQRIL's `error_code` (constants `gosqril.ErrorCode...`), `Details` the free-form `details` object as raw JSON, `RawBody` the body as received. SQRIL sends no request id, so `RequestID` is always empty: give support the `tx_id` and the time of the call. Helpers: `IsUnauthorized` (401), `IsInsufficientFunds` (402), `IsNotFound` (404), `IsConflict` (409), `IsRateLimited` (429). The SDK does not retry; on 429 back off and retry yourself.

A timeout is `errors.Is(err, context.DeadlineExceeded)`. A 2xx response whose body cannot be decoded, or any body over 10 MiB, is `errors.Is(err, gosqril.ErrInvalidResponse)`. An empty required field (`ErrMissingField`, `ErrInvalidField`, `ErrNilRequest`) fails before any request is sent.

## Configuration

```go
client, err := gosqril.NewClient(baseURL, clientID, clientSecret,
	gosqril.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
	gosqril.WithLogger(slog.Default()),
)
```

The default timeout is 30 seconds; `WithTimeout` changes it for the default HTTP client. The default client does not follow redirects; a client passed to `WithHTTPClient` should do the same (`CheckRedirect` returning `http.ErrUseLastResponse`) and must set its own timeout.

The base URL must be `https`. Plain `http` is accepted for `localhost`, `127.0.0.1` and `::1` only, and a URL with userinfo, a query or a fragment is rejected (`ErrInsecureBaseURL`).

Debug logs carry the method, the path and the status, never the query string, headers, keys or bodies. Printing a `*Client` with `%v`, `%+v` or `%#v` shows the base URL and client id, never the secret.

## Limitations

- Accounts with a `signing_secret` are not supported on `ExecutePayout`: SQRIL then requires an HMAC signature the docs do not specify, and the call returns 401 `HMAC_SIGNATURE_REQUIRED`.
- `UpdateCustomerRequest.Nationality` is sent because SQRIL's guides ask for it, but the endpoint schema does not list it; the API may ignore or reject it.
- Accepted `Gender` values are not settled in the docs (`M`/`F` on registration, `MALE`/`FEMALE` for Peru). The SDK sends your value unchanged.
- A `CustomerID` containing `/` is rejected by the SDK on every endpoint.
- `LookupPaymentCode` validates only `CustomerID` and `CountryCode`. The per-corridor fields are yours to set; the docs disagree on PH, UG, KE and TZ.
- `ListTransactions` does not expose the `detail` and `partner_app_id` parameters, which the docs mention but do not declare.
- `ListOccupations` with format `all` returns `Count` as a raw JSON object with undocumented keys.
- `Party` (sender, recipient), `SuggestedValues` and `QRData` are free-form maps: SQRIL documents no fixed key set.
- `Transaction.ErrorMessage` follows the Error Codes page; the field is not in SQRIL's schema.
- Webhooks: no timestamp or replay protection, and no documented retry policy. Verify the signature and deduplicate.

## Tests

```bash
go test ./...
```

Sandbox tests are skipped unless the variables in `.env.example` are set:

```bash
go test -run TestSandbox_ ./...
```

They run against staging only and stop if `SQRIL_BASE_URL` points at production. They leave behind:

- a customer: when `SQRIL_TEST_CUSTOMER_ID` is empty, the main flow and the decode test register one with `name_first` `sdk-sandbox-test`, and the main flow sets its nationality with `UpdateCustomer` (SQRIL has no delete endpoint). When `SQRIL_TEST_CUSTOMER_ID` is set, the tests use that customer, register nothing and do not change it;
- one `PENDING` transaction per run from `LookupPaymentCode` (CO, BREB) with a stored quotation, on whichever customer was used; it becomes `EXPIRED`. One more from `DecodeQR` when `SQRIL_TEST_QR_STRING` is set;
- only with `SQRIL_TEST_EXECUTE_PAYOUT=1`: one `PROCESSING` payout with `partner_transaction_id` `sdk-sandbox-test-<time>`, which locks or deducts sandbox balance.

## Versioning

Semantic versioning. The version is `gosqril.Version`. Breaking changes are listed in the release notes of each major version.
