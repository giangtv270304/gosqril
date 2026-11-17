# API spec for the SDK: gosqril (SQRIL SaaS payout API)

Source of truth: vendor docs `openapi.json` (OpenAPI 3.1, title "SQRIL SaaS API", version 1.0.0) plus the doc pages (`documentation/*`, one page per endpoint and per webhook). The endpoint pages are generated from the same OpenAPI and add JSON examples only. Anything the docs do not state is marked `UNCONFIRMED:` and repeated in "Open questions".

## 1. Identity

| Item | Value |
|---|---|
| SDK name | `gosqril` |
| Go package | `package gosqril` |
| Go module | `github.com/giangtv270304/gosqril` |
| Sandbox base URL | `https://stg-api.sqril.io` (the only server in `openapi.json`; constant `SandboxBaseURL`) |
| Production base URL | `https://api.sqril.io` (given by the user, not in the docs; constant `ProductionBaseURL`) |
| Path style | Flat RPC paths at the root, no version prefix: `/registerCustomer`, `/getQuotation`, ... |
| Content type | `application/json` for every request body and every response |
| Keys | `client_id` and `client_secret` per account ("exchange app"). UNCONFIRMED: the docs do not say where a developer obtains them (they mention a "Balance page" but give no portal URL) |
| Webhook secret | Per-account `webhook_secret`. UNCONFIRMED: how the webhook URL, event filter and secret are configured |

## 2. Authentication

| Header | Value | Applies to |
|---|---|---|
| `Authorization` | `Basic base64(client_id:client_secret)` | every endpoint |
| `Content-Type` | `application/json` | requests with a body |
| `X-Idempotency-Key` | caller-chosen string, unique per logical payout | `POST /executePayout` only, required (missing: 400 `IDEMPOTENCY_KEY_REQUIRED`) |
| `User-Agent` | `gosqril/<Version>` | every endpoint (SDK convention, not asked for by SQRIL) |

- Credentials in the JSON body (`client_id` / `client_secret`) are deprecated and not supported. `exchange_app_id` in a body or query is ignored; the account always comes from the Basic credentials.
- No request signing for the SDK: no `signer/` package.
- Exception documented for `executePayout` only: "If your account has `signing_secret`, send HMAC-SHA256 over JSON body (excluding `signature`) via `X-Signature` or body `signature`". Related error codes: `HMAC_SIGNATURE_REQUIRED` (401), `INVALID_HMAC_SIGNATURE` (401). UNCONFIRMED: signature encoding (hex or base64), the exact bytes signed, and whether any account we use has `signing_secret`. The SDK does not implement it in this version; an account with `signing_secret` will get 401 on `ExecutePayout`.
- Replayed idempotency key with a stored 202: the stored response is returned again (after ownership and customer checks). A failed `tx_id` or key replays the same failure.

## 3. Response shapes

No envelope. The success body is the resource object itself; the SDK returns it as is.

### Success

| HTTP | Used by |
|---|---|
| 200 | every endpoint except the two below; also `registerCustomer` when it updates an existing customer |
| 201 | `registerCustomer` when it creates a customer |
| 202 | `executePayout` (always; body `status` is `PROCESSING`) |

`registerCustomer`: the description says "Creates (201) or updates (200)"; the OpenAPI lists only 201. The SDK treats every 2xx as success.

### Error (same shape on every endpoint, status >= 400)

| Field | Type | Notes |
|---|---|---|
| `error_code` | string | machine code, see table below. Maps to `APIError.Code` |
| `error_message` | string | human text. Maps to `APIError.Message` |
| `error` | string | declared in the OpenAPI, never shown in an example. UNCONFIRMED: when it is sent. Fallback for `Message` when `error_message` is empty |
| `details` | object, free-form | optional. Keep as `map[string]json.RawMessage` on `APIError.Details` |

- Request id / trace id: none. No field and no response header is documented. UNCONFIRMED. `APIError.RequestID` stays empty; error text omits it.
- `previewQuotations` 400/401/429 bodies are typed "Any type" in the OpenAPI. UNCONFIRMED: assumed to be the same shape; the SDK must keep the raw body either way.
- A non-JSON error body (proxy HTML) is kept in `APIError.RawBody`.

Real examples from the docs (decodeQr page):

```json
{"error_code":"MISSING_REQUIRED_FIELD","error_message":"qr_string is required"}
```

```json
{"error_code":"INVALID_COUNTRY_CODE","error_message":"Invalid or unsupported country code for this account."}
```

```json
{"error_code":"INVALID_REQUEST","error_message":"Typed/manual payment destinations (PIX key, phone, bank account, BREB key) must be initiated via /lookupPaymentCode, not /decodeQr."}
```

```json
{
  "error_code": "CUSTOMER_NOT_ELIGIBLE",
  "error_message": "Customer {{customer_id}} cannot transact to PE until required sender profile fields are completed. Use the updateCustomer API to add or update the missing sender data, then retry decode.",
  "details": {
    "customer_id": "{{customer_id}}",
    "country": "PE",
    "missing_sender_fields": [
      {"field": "sender.occupation", "type": "text", "required": true},
      {"field": "sender.email", "type": "email", "required": true},
      {"field": "sender.phone", "type": "phone", "required": true},
      {"field": "sender.gender", "type": "select", "required": true}
    ]
  }
}
```

Documented extra data on errors (UNCONFIRMED: whether each sits inside `details` or at the top level, except where an example shows it):

| Code | Extra data |
|---|---|
| `CUSTOMER_NOT_ELIGIBLE` | `details.customer_id`, `details.country`, `details.missing_sender_fields[]` (`field`, `type`, `required`) |
| `TRANSACTION_NOT_PENDING` | `current_status` |
| `MISSING_REQUIRED_FIELDS` | the field list ("often includes `missing_fields`") |
| `DAILY_LIMIT_EXCEEDED`, `MONTHLY_LIMIT_EXCEEDED` | "details carry `limit_usd`, `used_usd`, `remaining_usd` and `resets_at`" |
| `ACCOUNT_PAYOUTS_LOCKED` | "includes `available`" |

### HTTP statuses used

200, 201, 202, 400, 401, 402, 403, 404, 405, 429, 500, 503.

SDK helpers by status: `IsUnauthorized` 401, `IsNotFound` 404, `IsRateLimited` 429, `IsConflict` 409 (409 is never documented; keep the helper for the skill's contract). Suggested extra: `IsInsufficientFunds` 402.

### Error codes (full table, from the Error Codes page)

| `error_code` | HTTP | Meaning |
|---|---|---|
| `AUTHENTICATION_REQUIRED` | 401 | Authentication is missing |
| `INVALID_CREDENTIALS` | 401 / 403 | Credentials invalid, or caller lacks access |
| `INVALID_HMAC_SIGNATURE` | 401 | HMAC signature is invalid |
| `HMAC_SIGNATURE_REQUIRED` | 401 | HMAC signature required but not provided |
| `TRANSACTION_NOT_OWNED` | 403 | Transaction does not belong to the account, or `customer_id` does not match the transaction |
| `COUNTRY_NOT_SUPPORTED_BY_ACCOUNT` | 403 | Account does not support the country/corridor (for example `listBanks?country_code=ID` without IDR) |
| `KYC_REQUIRED` | 403 | KYC verification required first |
| `INVALID_REQUEST` | 400 | Malformed request or failed validation |
| `MISSING_REQUIRED_FIELD` | 400 | One or more required fields missing |
| `MISSING_REQUIRED_FIELDS` | 400 | Multiple required fields missing (often includes `missing_fields`) |
| `INVALID_QR_FORMAT` | 400 | QR payload missing or invalid |
| `INVALID_COUNTRY_CODE` | 400 | Country code invalid or unsupported for this operation |
| `UNSUPPORTED_QR_TYPE` | 400 | QR recognised but not payable (retired VNPAY scheme, or Indonesian QRIS with tip/convenience fee) |
| `INVALID_AMOUNT` | 400 | Amount missing, <= 0, out of range, or too many decimals for the currency |
| `INVALID_CURRENCY` | 400 | Currency invalid, unsupported, or no exchange rate |
| `INVALID_STATUS` | 400 | Status filter/value invalid |
| `INVALID_LIMIT` | 400 | Pagination limit invalid |
| `INVALID_START_AFTER` | 400 | Pagination cursor (`start_after`) invalid |
| `TRANSACTION_NOT_PENDING` | 400 | Transaction is not `PENDING` (often includes `current_status`) |
| `QUOTATION_EXPIRED` | 400 | Quotation expired; `executePayout` on an `EXPIRED` transaction (start again from decodeQr) |
| `DAILY_LIMIT_EXCEEDED` | 400 | Per-sender daily limit (UTC day) would be exceeded |
| `MONTHLY_LIMIT_EXCEEDED` | 400 | Per-sender monthly limit (UTC month) would be exceeded |
| `METHOD_NOT_ALLOWED` | 405 | HTTP method not allowed |
| `IDEMPOTENCY_KEY_REQUIRED` | 400 | `X-Idempotency-Key` header required |
| `CUSTOMER_NOT_ELIGIBLE` | 400 | Sender profile fields missing, or identity document not accepted for the customer's country (LATAM) |
| `CUSTOMER_BLOCKED` | 403 | Registered customer is blocked; cleared by SQRIL support only |
| `TRANSACTION_NOT_FOUND` | 404 | Transaction not found |
| `RESOURCE_NOT_FOUND` | 404 | Resource not found (customer, account) |
| `RECIPIENT_LOOKUP_UNAVAILABLE` | 503 | Recipient account name could not be resolved; nothing locked; retry from decodeQr or supply `recipient_name` |
| `INSUFFICIENT_FUNDS` | 402 | Account balance insufficient |
| `ACCOUNT_PAYOUTS_LOCKED` | 402 | Payouts locked because the account balance is negative; includes `available` |
| `INTERNAL_SERVER_ERROR` | 500 | Unexpected server error |
| `PROVIDER_ERROR` | 400 / 500 | Upstream payment-provider failure |
| `WEBHOOK_SECRET_NOT_CONFIGURED` | 500 | Webhook signing secret not configured (SQRIL's inbound handlers) |
| `INVALID_WEBHOOK_SIGNATURE` | 401 / 400 | Inbound webhook signature verification failed (SQRIL's inbound handlers) |
| `RATE_LIMIT_EXCEEDED` | 429 | Too many requests; retry with exponential backoff (1s, 2s, 4s, up to about 30s) and jitter |

Rate limits (per minute, per account and calling IP, fixed one-minute windows): `executePayout` 60, `previewQuotations` 60. Other endpoints are not limited today, but a 429 can come from any endpoint. The SDK does not retry by itself.

### Payout failure reasons (`error_message` of a `FAILED` transaction; not HTTP errors)

Found in getTransaction, listTransactions and the failed webhook. Fixed strings:

`Rejected by the payout network` · `Recipient account was not accepted by the payout network` · `Sender details were not accepted for this corridor` · `Amount is below the payout network minimum` · `Payout network temporarily unavailable, please retry later` · `Payout could not be submitted, please retry` · `Insufficient account balance` · `Payment was reversed before settlement` · `Payout failed`

A retry is a new payment: new decode, new quotation, new idempotency key.

## 4. Endpoints

Legend: `?` = optional. `Decimal` = the money type of section 6. All timestamps are strings (ISO 8601, example `2026-04-22T10:00:00.000Z`); dates are `YYYY-MM-DD` strings. The OpenAPI marks no response field as required except where "required" is written; Go response fields therefore use `omitempty`-tolerant decoding and zero values.

Query parameter `dp` (integer 0 to 10, default 2): decimal places for monetary fields in the response (USD amounts, fees, exchange rates); the default rounds sub-cent fees to 0; response formatting only, stored values and deductions stay at full precision. Go: `DP *int` on the request struct, sent only when set. Accepted by the endpoints marked "dp" below.

SDK-side checks before any call (error returned without calling the API): empty `customer_id`, `tx_id`, `transaction_id`, `country_code`, `qr_string`, idempotency key, where the endpoint requires them; `customer_id` containing `/`.

| # | Method | Path | SDK method | Go file | Success | dp |
|---|---|---|---|---|---|---|
| 1 | POST | `/registerCustomer` | `RegisterCustomer` | `customer_register.go` | 201 or 200, `Customer` | no |
| 2 | GET | `/checkCustomer` | `CheckCustomer` | `customer_check.go` | 200, `Customer` | no |
| 3 | PUT | `/updateCustomer` | `UpdateCustomer` | `customer_update.go` | 200, `Customer` | no |
| 4 | POST | `/decodeQr` | `DecodeQR` | `qr_decode.go` | 200, `DecodeResult` | yes |
| 5 | POST | `/decodeQrUnregistered` | `DecodeQRUnregistered` | `qr_decode_unregistered.go` | 200, `DecodeResult` | yes |
| 6 | POST | `/lookupPaymentCode` | `LookupPaymentCode` | `payment_code_lookup.go` | 200, `DecodeResult` | yes |
| 7 | POST | `/getQuotation` | `GetQuotation` | `quotation_get.go` | 200, quotation | yes |
| 8 | POST | `/previewQuotations` | `PreviewQuotations` | `quotation_preview.go` | 200, `quotations[]` | yes |
| 9 | POST | `/executePayout` | `ExecutePayout` | `payout_execute.go` | 202, payout | yes |
| 10 | GET | `/listOccupations` | `ListOccupations` | `occupation_list.go` | 200, occupations | no |
| 11 | GET | `/listTransactions` | `ListTransactions` | `transaction_list.go` | 200, page of `Transaction` | yes |
| 12 | GET | `/getTransaction` | `GetTransaction` | `transaction_get.go` | 200, `{transaction}` | yes |
| 13 | GET | `/getAccountBalances` | `GetAccountBalances` | `account_balance_get.go` | 200, `{balances}` | no |
| 14 | GET | `/listBanks` | `ListBanks` | `bank_list.go` | 200, banks | no |

Each file holds `XRequest`, `XResponse` and the method (`func (c *Client) X(ctx context.Context, req *XRequest) (*XResponse, error)`).

### 4.1 `POST /registerCustomer` — `RegisterCustomer`

Registers or upserts a sender profile under the account. Flat JSON at the root (not nested under `sender`). One-time call per customer; the returned `customer_id` is required by decodeQr, lookupPaymentCode, getQuotation, executePayout. UNCONFIRMED: which field identifies an existing customer for the upsert.

| Body field | Type | Required | Notes |
|---|---|---|---|
| `name_first` | string | yes | immutable afterwards |
| `name_last` | string | yes | immutable afterwards |
| `ic_number` | string | yes | ID number of the `ic_type` |
| `ic_type` | string enum | yes | `NIC`, `PP`, `WEP`, `DL` |
| `ic_country` | string | yes | ISO 3166 alpha-2, issuing country |
| `occupation` | string | yes | `OCC1` .. `OCC12`; listOccupations says ISCO-08 codes are preferred for new integrations and legacy codes stay accepted |
| `country_of_residence` | string | yes | ISO 3166 alpha-2 |
| `dob` | string date | yes | `YYYY-MM-DD` |
| `address` | string | yes | complete address |
| `ic_expiry_date` | string date | yes | `YYYY-MM-DD`; `2099-01-01` for non-expiring documents |
| `gender` | string | no | "M or F". Peru notes say "MALE or FEMALE". UNCONFIRMED which values are accepted |
| `phone` | string | no | E.164. Needed for PE, KE, TZ, UG |
| `email` | string | no | Needed for TH, PE |
| `nationality` | string | no | ISO 3166 alpha-2. Required for VN (checked at the payer's first VN payout) |

Response: `Customer` (section 5). Errors listed: 400. Example response has no `allowed`.

### 4.2 `GET /checkCustomer` — `CheckCustomer`

| Query | Type | Required |
|---|---|---|
| `customer_id` | string | yes |

Response: `Customer` including `allowed` (map corridor code to bool; `false` means updateCustomer is needed first; `allowed.VN` does not reflect `nationality` yet). Errors listed: 401, 404.

### 4.3 `PUT /updateCustomer` — `UpdateCustomer`

| Where | Field | Type | Required | Notes |
|---|---|---|---|---|
| query | `customer_id` | string | yes | |
| body | `email` | string | no | |
| body | `phone` | string | no | |
| body | `occupation` | string | no | |
| body | `address` | string | no | |
| body | `gender` | string | no | "MALE or FEMALE for Peru (PE)" |

- "Provide one or more updatable sender fields." `name_first` and `name_last` cannot be updated. Go: pointer fields with `omitempty`.
- UNCONFIRMED: `nationality` is not in the body schema, but three doc pages say "backfill existing payers with updateCustomer". The SDK exposes `Nationality *string` (`nationality`, `omitempty`) and the sandbox test must confirm it is accepted.
- Response: `Customer`. Errors listed: 400, 401, 404.

### 4.4 `POST /decodeQr` — `DecodeQR`

Decodes a scanned QR string, creates a `PENDING` transaction, returns `tx_id`. Typed destinations (PIX key, phone, bank account, BREB key) must use lookupPaymentCode; sending them here returns 400 `INVALID_REQUEST`.

| Where | Field | Type | Required | Notes |
|---|---|---|---|---|
| query | `dp` | int | no | |
| body | `qr_string` | string | yes | EMV / local QR payload; for KE/TZ a numeric till/paybill may be passed |
| body | `customer_id` | string | yes | registered customer; must not contain `/` |
| body | `partner_transaction_id` | string | no | set once; cannot change later |
| body | `partner_app_id` | string | no | 1 to 64 chars `[a-zA-Z0-9_-]`; body wins over the customer's stored value |
| body | `payment_currency` | string | no | ISO 4217 source currency when it cannot be resolved from the QR |
| body | `wallet_identifier` | string | no | stored on the transaction |
| body | `country_code` | string | no | ISO alpha-2 override; required when the QR is not EMV-parseable (`BO` encrypted QR; `KE`/`TZ` with `merchant_id`) |
| body | `merchant_id` | string | no | till or paybill number for KE/TZ |
| body | `account_reference` | string | no | paybill account reference; required for Kenya paybills (else returned in `missing_fields`), optional for TZ |

Response: `DecodeResult`. Errors listed: 400 (`CUSTOMER_NOT_ELIGIBLE`, `INVALID_COUNTRY_CODE`, `INVALID_CURRENCY`, `MISSING_REQUIRED_FIELD`, `INVALID_REQUEST`, `UNSUPPORTED_QR_TYPE`, corridor amount limit violations).

### 4.5 `POST /decodeQrUnregistered` — `DecodeQRUnregistered`

Same request body as decodeQr, but `customer_id` is not required and is ignored if sent (Go request struct has no `CustomerID`). Same `dp` query. Response: `DecodeResult`.

- The customer is attached at executePayout.
- LATAM (AR, BR, PE, BO, CO): `merchant` is the QR's own label, not the verified account holder.
- Thailand: not available; 400 `INVALID_REQUEST` with message `customer_id is required to decode this QR`.
- `missing_fields` may be omitted on some corridors.

### 4.6 `POST /lookupPaymentCode` — `LookupPaymentCode`

Resolves a typed destination (not a QR), creates a `PENDING` transaction, same response as decodeQr. Sender is never accepted in the request.

| Where | Field | Type | Required | Notes |
|---|---|---|---|---|
| query | `dp` | int | no | |
| body | `customer_id` | string | yes | |
| body | `country_code` | string | yes | ISO alpha-2. Contract 1: `ID`, `AR`, `BR`, `PE`, `BO`, `CO`, `NG`, `GH`, `ZM`. Contract 2: `KE`, `TZ`, `VN` |
| body | `payment_method` | string enum | contract 1 | `payment_key`, `ewallet`, `bank_transfer` |
| body | `payment_destination` | string | contract 1 | PIX key, phone, bank account, BREB key, wallet phone, 6 to 30 digit bank account, mobile-money number |
| body | `payment_destination_code` | string | per corridor | see table below |
| body | `account_type` | string enum | BO only | `SAVINGS`, `CHECKING` |
| body | `recipient` | object | BO only | `name_first`, `name_last`, `ic_number` (strings) |
| body | `merchant_id` | string | contract 2 | till/paybill (KE/TZ) or bank account number (VN) |
| body | `account_reference` | string | no | paybill reference (KE/TZ, required for Kenya paybills) or bank BIN (VN, example `970436`) |
| body | `amount` | number (money) | PE only | local-currency amount; when > 0 pricing may be returned |
| body | `currency` | string | no | ISO 4217; defaults to the corridor currency |
| body | `partner_transaction_id` | string | no | |
| body | `partner_app_id` | string | no | 1 to 64 chars `[a-zA-Z0-9_-]` |
| body | `wallet_identifier` | string | no | |
| body | `qr_string` | string | no | legacy alias of `payment_destination`; the SDK does not expose it |

Contract 1, per corridor (OpenAPI description):

| `country_code` | `payment_method` | `payment_destination_code` |
|---|---|---|
| `AR` | `payment_key` | not listed |
| `BR` | `payment_key` | must be `PIX` |
| `CO` | `payment_key` | must be `BREB` |
| `ID` | `ewallet` | `DANA`, `OVO`, `GOPAY`, `SHOPEEPAY` (OVO minimum 10,000 IDR) |
| `ID` | `bank_transfer` | 3-digit national bank code or SWIFT (`code` / `swift_code` from `listBanks?country_code=ID`; any 3-digit code or SWIFT accepted) |
| `PE` | `ewallet` | `YAPE` or `PLIN`; `amount` in PEN required |
| `BO` | `bank_transfer` | bank code from `listBanks?country_code=BO`; plus `account_type` and `recipient` |
| `NG`, `GH`, `ZM` | `bank_transfer` | bank or mobile-money network code (for example `058`, `MTN`) |

UNCONFIRMED (the "Bank / E-wallet Transfer" guide disagrees with the OpenAPI): the guide also lists `PH` (`bank_transfer`, `payment_destination_code` = `bank_list_code` from `listBanks?country_code=PH`, `recipient.name` required) and lists `KE`, `TZ`, `UG` as `payment_method` `payment_key` with the till/paybill in `payment_destination` and optional `payment_destination_code` `TILL` or `PAYBILL`; the OpenAPI uses `merchant_id` for KE/TZ and does not mention PH or UG. The SDK sends whatever the caller sets and validates only `customer_id` and `country_code`.

Response: `DecodeResult`. Errors listed: 400.

### 4.7 `POST /getQuotation` — `GetQuotation`

Prices a `PENDING` transaction and stores the quotation (valid 30 minutes).

| Where | Field | Type | Required | Notes |
|---|---|---|---|---|
| query | `dp` | int | no | |
| body | `tx_id` | string | yes | |
| body | `customer_id` | string | yes | must match the transaction's customer; must not contain `/` |
| body | `amount` | number (money) | conditional | must be > 0 when sent; required when the transaction has no saved amount > 0; ignored when it has one |
| body | `partner_transaction_id` | string | no | sets it once; a different value later returns 400 `INVALID_REQUEST` |

| Response field | Type | Notes |
|---|---|---|
| `tx_id` | string | |
| `amount` | Decimal | local currency |
| `currency` | string | |
| `exchange_rate` | Decimal | USD per one unit of the transaction currency; 10 decimal places |
| `amount_usd` | Decimal | USD, 2 dp by default |
| `fee` | Decimal | total fee in USD = `percentage_fee` + `fixed_fee` (within rounding) |
| `percentage_fee` | Decimal | USD |
| `fixed_fee` | Decimal | USD |
| `expires_at` | string | ISO 8601 |

Example: `{"tx_id":"{{tx_id}}","amount":100000,"currency":"VND","exchange_rate":0.0000639967,"amount_usd":6.4,"fee":0.45,"percentage_fee":0.35,"fixed_fee":0.1,"expires_at":"2024-01-01T00:30:00.000Z"}`

Errors: 400 (`INVALID_AMOUNT`, `TRANSACTION_NOT_PENDING`, `INVALID_CURRENCY`, `INVALID_REQUEST`, `CUSTOMER_NOT_ELIGIBLE`, `DAILY_LIMIT_EXCEEDED`, `MONTHLY_LIMIT_EXCEEDED`), 401 `AUTHENTICATION_REQUIRED`, 403 `TRANSACTION_NOT_OWNED` (in the description, not in the OpenAPI responses), 404 `TRANSACTION_NOT_FOUND`.

### 4.8 `POST /previewQuotations` — `PreviewQuotations`

Stateless preview for every corridor enabled on the account. Creates no transaction. Figures are indicative. Rate limit 60/min.

| Where | Field | Type | Required | Notes |
|---|---|---|---|---|
| query | `dp` | int | no | |
| body | `amount` | number (money) | yes | amount in `currency`, > 0 |
| body | `currency` | string | yes | ISO 4217, case-insensitive, for example `USD` |

Response: `quotations` (array). Each item: `amount` (Decimal, local), `currency` (string), `exchange_rate` (Decimal), `amount_usd` (Decimal), `fee` (Decimal), `percentage_fee` (Decimal), `fixed_fee` (Decimal), `expires_at` (string). A corridor is omitted when it has no FX rate or the converted amount is outside its limits, so a short or empty list is a normal response.

Errors: 400 (`INVALID_AMOUNT`, `MISSING_REQUIRED_FIELD`, `INVALID_CURRENCY`), 401 `AUTHENTICATION_REQUIRED`, 429 `RATE_LIMIT_EXCEEDED`.

### 4.9 `POST /executePayout` — `ExecutePayout`

Moves money. Returns 202 with `status` `PROCESSING`; the final status arrives by webhook or `getTransaction`. Rate limit 60/min.

| Where | Field | Type | Required | Notes |
|---|---|---|---|---|
| header | `X-Idempotency-Key` | string | yes | Go: `IdempotencyKey string` on the request struct (`json:"-"`); the SDK never generates it |
| query | `dp` | int | no | |
| body | `tx_id` | string | yes | |
| body | `amount_confirmed` | number (money) | yes | must equal the saved amount when the transaction has one |
| body | `currency` | string | yes | must equal the saved currency |
| body | `customer_id` | string | yes | must match the transaction's customer; must not contain `/` |
| body | `partner_transaction_id` | string | no | settable once |
| body | `recipient_name` | string | no | only when decode listed `recipient.name` in `missing_fields`. Also accepted as dotted key `recipient.name` or nested `recipient: {"name": ...}`; the SDK sends `recipient_name` only |

`sender` must not be sent (taken from the registered customer).

| Response field | Type | Notes |
|---|---|---|
| `status` | string enum | `PROCESSING` |
| `tx_id` | string | |
| `customer_id` | string | |
| `message` | string | example `Payout initiated successfully` |
| `exchange_rate` | Decimal | 10 dp |
| `fee` | Decimal | USD |
| `percentage_fee` | Decimal | USD |
| `fixed_fee` | Decimal | USD |
| `amount` | Decimal | local currency |
| `amount_usd` | Decimal | USD |
| `currency` | string | |
| `sender` | `Party`, optional | example keys: `name_first`, `name_last`, `ic_number`, `ic_type`, `ic_country`, `phone`, `email` |
| `recipient` | `Party`, optional | example keys: `name`, `account_no`, `bank_no` |
| `partner_transaction_id` | string, optional | |
| `updated_at` | string, optional | ISO 8601 |

Example: `{"status":"PROCESSING","tx_id":"{{tx_id}}","customer_id":"{{customer_id}}","message":"Payout initiated successfully","exchange_rate":0.0000666667,"fee":0.07,"percentage_fee":0.05,"fixed_fee":0.02,"amount":100000,"amount_usd":6.67,"currency":"VND","sender":{...},"recipient":{"name":"MERCHANT NAME","account_no":"1234567890","bank_no":"014"},"partner_transaction_id":"partner_tx_123","updated_at":"2024-01-01T00:00:00.000Z"}`

Errors: 400 (`IDEMPOTENCY_KEY_REQUIRED`, `QUOTATION_EXPIRED`, `TRANSACTION_NOT_PENDING`, `MISSING_REQUIRED_FIELDS`, `INVALID_AMOUNT`, `DAILY_LIMIT_EXCEEDED`, `MONTHLY_LIMIT_EXCEEDED`), 401 (auth, HMAC), 402 (`INSUFFICIENT_FUNDS`, `ACCOUNT_PAYOUTS_LOCKED`), 403 (`TRANSACTION_NOT_OWNED`, `CUSTOMER_BLOCKED`; in the description, not in the OpenAPI responses), 404 (customer not found under the account), 429, 503 `RECIPIENT_LOOKUP_UNAVAILABLE`.

Caller guidance for the README: a network error or timeout on this call is indeterminate; resolve with `GetTransaction(tx_id)` or retry with the same idempotency key.

### 4.10 `GET /listOccupations` — `ListOccupations`

| Query | Type | Required | Notes |
|---|---|---|---|
| `format` | string enum | no | `isco` (default), `legacy`, `all` |
| `level` | int 1 to 4 | no | filters ISCO entries by hierarchy level; ignored for `legacy` |

The response shape depends on `format`:

| `format` | Fields |
|---|---|
| `isco` | `occupations[]` of `{code: string, level: int, title: string, legacy_occ: string}`; `format`; `count` (int); `level` (int, only when sent) |
| `legacy` | `occupations[]` of `{code: string, title: string, isco_code: string or null}`; `format`; `count` (int). `isco_code` is `null` for `OCC12` |
| `all` | `isco[]` and `legacy[]` arrays (item shapes as above), `count` (object), optional `level` |

Go response: `Occupations []Occupation`, `ISCO []Occupation`, `Legacy []Occupation`, `Format`, `Count json.RawMessage` plus helper, `Level *int`; `Occupation` = `Code string`, `Title string`, `Level *int`, `LegacyOCC *string`, `ISCOCode *string`. UNCONFIRMED: the keys of the `count` object for `format=all` (no example; the OpenAPI types `count` as integer and `occupations` as required, which contradicts the description); whether `legacy_occ` can be `null`.

Examples: `{"occupations":[{"code":"1","level":1,"title":"Managers","legacy_occ":"OCC1"},{"code":"2","level":1,"title":"Professionals","legacy_occ":"OCC9"}],"format":"isco","count":2,"level":1}` and `{"occupations":[{"code":"OCC1","title":"C-Suite executive / Board member","isco_code":"1120"},{"code":"OCC12","title":"Others","isco_code":null}],"format":"legacy","count":12}`.

Errors listed: 400, 401.

### 4.11 `GET /listTransactions` — `ListTransactions`

| Query | Type | Required | Notes |
|---|---|---|---|
| `status` | string | no | description lists `PENDING`, `PROCESSING`, `SUCCESS`, `FAILED`. UNCONFIRMED whether `EXPIRED` and `REFUNDED` are accepted as filters |
| `customer_id` | string | no | one id, or up to 10 comma-separated ids; cannot be combined with `status` |
| `limit` | int 1 to 100 | no | default 50 |
| `start_after` | string | no | cursor: a transaction id (`next_cursor` of the previous page) |
| `start_date` | string date-time | no | ISO 8601, for example `2024-01-01T00:00:00Z`; inclusive |
| `end_date` | string date-time | no | inclusive; `start_date` <= `end_date` |
| `date_field` | string enum | no | `created_at` (default), `completed_at`, `refunded_at`: the date the range filters on and the list is ordered by. The last two list only transactions that have that date and cannot be combined with `status`, `partner_app_id` or `customer_id` |
| `dp` | int | no | |

UNCONFIRMED: the description also mentions a `detail` parameter (`full` returns the legacy wide objects with full `qr_data`; default is slim) and a `partner_app_id` filter; neither is declared as a parameter in the OpenAPI. Not exposed by the SDK until confirmed.

| Response field | Type | Notes |
|---|---|---|
| `transactions` | `[]Transaction` | required; newest first |
| `total` | int | required; "number of transactions returned" |
| `limit` | int | required |
| `has_more` | bool | required |
| `next_cursor` | string, nullable | present when `has_more` is true. Go `*string` |

Errors listed: 400 (`INVALID_STATUS`, `INVALID_LIMIT`, `INVALID_START_AFTER`), 401.

### 4.12 `GET /getTransaction` — `GetTransaction`

| Query | Type | Required |
|---|---|---|
| `transaction_id` | string | yes (this is the `tx_id`) |
| `dp` | int | no |

Response: `transaction` (`Transaction`, required). Errors listed: 400, 401, 404 `TRANSACTION_NOT_FOUND` (also when the transaction belongs to another account).

### 4.13 `GET /getAccountBalances` — `GetAccountBalances`

No parameters. Response: `balances` (required): object keyed by currency code (`USD` plus the account's `supported_currencies`), each value `{available: Decimal, locked: Decimal, total: Decimal, currency: string}`. Go: `map[string]Balance`. Errors listed: 401.

Example: `{"balances":{"USD":{"available":100,"locked":0,"total":100,"currency":"USD"},"VND":{"available":5000000,"locked":0,"total":5000000,"currency":"VND"}}}`

### 4.14 `GET /listBanks` — `ListBanks`

| Query | Type | Required | Notes |
|---|---|---|---|
| `country_code` | string enum | yes | `ID`, `PH`, `VN`, `BO`, `NG`, `GH`, `ZM`. `ID` requires IDR on the account, else 403 `COUNTRY_NOT_SUPPORTED_BY_ACCOUNT` |

| Response field | Type | Notes |
|---|---|---|
| `banks` | `[]Bank` | required |
| `country_code` | string | required |
| `total` | int | required |

`Bank`: `code` (string), `name` (string), `short_name` (string, optional, Indonesia rows), `swift_code` (string, optional, Indonesia rows that have one), `type` (string, optional), `country_code` (string, optional). Indonesia: codes `013`, `022`, `200` appear twice (same `code`, different `swift_code`); code `688` has no `swift_code`.

UNCONFIRMED: for NG/GH/ZM the text says "`code` carries `channel_category` to tell them apart" and for PH the docs refer to a `bank_list_code`; neither field is in the schema or the example. The values of `type` are not listed.

Errors listed: 400, 401 (403 as above).

## 5. Types

Enums are named `string` types with constants; an unknown value from the server must decode without error.

| Enum (Go) | Values |
|---|---|
| `TransactionStatus` | `PENDING`, `PROCESSING`, `SUCCESS`, `FAILED`, `EXPIRED`, `REFUNDED` |
| `ICType` | `NIC` (national ID card), `PP` (passport), `WEP` (work permit / employment pass), `DL` (driver's license) |
| `PaymentMethod` | `payment_key`, `ewallet`, `bank_transfer` |
| `AccountType` | `SAVINGS`, `CHECKING` |
| `PaymentType` | `till`, `paybill` |
| `OccupationFormat` | `isco`, `legacy`, `all` |
| `DateField` | `created_at`, `completed_at`, `refunded_at` |
| Legacy occupation codes (constants, `occupation` stays `string`) | `OCC1` C-Suite executive / Board member, `OCC2` Employee, `OCC3` Entrepreneur / Business owner, `OCC4` Foreign worker, `OCC5` Foreign domestic worker, `OCC6` Home-maker, `OCC7` Middle management employee, `OCC8` Pilot, `OCC9` Professional, `OCC10` Senior management employee, `OCC11` Student, `OCC12` Others |

### `Customer` (defined in `customer_check.go`; response of register, check, update)

| Field | Type | Notes |
|---|---|---|
| `customer_id` | string | |
| `name_first`, `name_last` | string | |
| `gender` | string | optional |
| `ic_number` | string | |
| `ic_type` | `ICType` | |
| `ic_country` | string | |
| `occupation` | string | |
| `country_of_residence` | string | |
| `phone` | string | optional |
| `email` | string | optional |
| `nationality` | string | optional |
| `dob` | string | `YYYY-MM-DD` |
| `address` | string | |
| `ic_expiry_date` | string | `YYYY-MM-DD` |
| `created_at`, `updated_at` | string | "when available" |
| `allowed` | `map[string]bool`, optional | ISO alpha-2 corridor code to `true` when the stored profile satisfies that corridor |

No field is documented as nullable; all are optional in the OpenAPI.

### `DecodeResult` (defined in `qr_decode.go`; response of decodeQr, decodeQrUnregistered, lookupPaymentCode)

| Field | Type | Notes |
|---|---|---|
| `is_dynamic` | bool | LATAM open-amount QRs return `false` with `amount` 0 |
| `is_business` | bool | |
| `amount` | Decimal | local currency; `0` when the amount is still to be supplied |
| `merchant` | string | may be `""` |
| `merchant_id` | string, optional | KE/TZ till or paybill |
| `merchant_name` | string, optional | KE/TZ verified display name |
| `payment_type` | `PaymentType`, optional | KE/TZ |
| `merchant_info` | `MerchantInfo`, optional | KE/TZ |
| `country` | string | ISO alpha-2 |
| `tx_id` | string | example `tx_1234567890_abc123` |
| `currency` | string | ISO 4217 |
| `partner_transaction_id` | string | `""` when not sent |
| `amount_usd` | Decimal, optional | only when `amount` > 0 and pricing succeeded. Go `*Decimal` |
| `fee` | Decimal, optional | same condition. Go `*Decimal` |
| `percentage_fee` | Decimal, optional | same. Go `*Decimal` |
| `fixed_fee` | Decimal, optional | same. Go `*Decimal` |
| `missing_fields` | `[]MissingField`, optional | data still required before payout |
| `suggested_values` | object, optional | hints keyed by field path; Go `map[string]any` |
| `sender` | `Party`, optional | echo of the registered sender (PII) |
| `recipient` | `Party`, optional | |
| `qr_type` | string, optional | Philippines: `qrph_p2m`, `qrph_p2p`. In the description and one example, not in the schema |

`exchange_rate` is never returned by decode/lookup.

`MerchantInfo`: `merchant_id`, `display_name`, `country_code`, `payment_type` (`PaymentType`), `lipa_number`, `account_reference` (all string, optional).

`MissingField`: free-form object in the OpenAPI. Keys seen in examples: `field` (string, for example `amount`, `recipient.name`, `recipient.account`, `recipient.account_no`, `sender.occupation`), `type` (string: `numeric`, `text`, `email`, `phone`, `select`), `label` (string, optional), `required` (bool, optional). UNCONFIRMED: the complete key set. Go: struct with these four, unknown keys ignored.

`Party` (sender / recipient): free-form object in the OpenAPI ("Any type"). Go: `map[string]any`. Keys seen: sender `name_first`, `name_last`, `ic_number`, `ic_type`, `ic_country`, `country`, `phone`, `email`; recipient `name`, `name_first`, `name_last`, `account_no`, `account`, `bank_no`, `bank_code`, `bank_name`, `bank_list_code`, `country`. No money inside.

### `Transaction` (defined in `transaction_get.go`; used by getTransaction and listTransactions)

| Field | Type | Notes |
|---|---|---|
| `id` | string | the `tx_id` |
| `exchange_app_id` | string | |
| `customer_id` | string | |
| `status` | `TransactionStatus` | |
| `amount` | Decimal | local currency |
| `amount_usd` | Decimal, optional | absent before a quotation. Go `*Decimal` |
| `fee` | Decimal, optional | USD. Go `*Decimal` |
| `percentage_fee` | Decimal, optional | USD; older transactions may omit the breakdown. Go `*Decimal` |
| `fixed_fee` | Decimal, optional | USD. Go `*Decimal` |
| `total` | Decimal, optional | USD (example: `amount_usd` 6.67 + `fee` 0.10 = 6.77). Go `*Decimal` |
| `exchange_rate` | Decimal, optional | Go `*Decimal` |
| `currency` | string | |
| `country_code` | string | |
| `partner_transaction_id` | string, optional | |
| `recipient_bank_no` | string, optional | legacy flat column |
| `recipient_account_no` | string, optional | legacy flat column |
| `recipient_account_name` | string, optional | legacy flat column |
| `sender` | `Party`, optional | |
| `recipient` | `Party`, optional | |
| `qr_data` | object, optional | free-form; Go `map[string]any` |
| `missing_fields` | `[]MissingField`, optional | |
| `quotation_expires_at` | string, optional | ISO 8601 |
| `created_at`, `updated_at` | string | ISO 8601 |
| `completed_at` | string, optional | when the payout succeeded; kept if refunded later |
| `refunded_at` | string, optional | when refunded (`REFUNDED`) |
| `error_message` | string, optional | failure reason of a `FAILED` payout. Stated on the Error Codes page; not in the OpenAPI schema. UNCONFIRMED exact field name on the transaction object |

Status flow: `PENDING` (decode/lookup) to `PROCESSING` (executePayout accepted) to `SUCCESS` or `FAILED`; `PENDING` to `EXPIRED` (quotation lapsed, nothing locked); `SUCCESS` to `REFUNDED` (manual refund by SQRIL ops). On `FAILED` the full amount returns to the balance.

### Others

- `Balance`: `available`, `locked`, `total` (Decimal), `currency` (string).
- `Bank`, `Occupation`, quotation item: see 4.14, 4.10, 4.8.
- `APIError`: `StatusCode int`, `Code string` (`error_code`), `Message string` (`error_message`, fallback `error`), `Details map[string]json.RawMessage`, `RequestID string` (always empty for SQRIL), `RawBody []byte`.

## 6. Money

SQRIL sends and expects amounts as JSON numbers in standard currency units (not minor units). SDK decision: money and rate fields use a named type `Decimal` (`money.go`), a decimal string in Go.

| Rule | Detail |
|---|---|
| Go type | `type Decimal string` |
| Marshal | emits the string as a raw JSON number (no quotes). Must match `^-?\d+(\.\d+)?$`, otherwise the marshal fails; no float conversion |
| Unmarshal | accepts a JSON number or a JSON string; keeps the digits as sent; never goes through `float64`. Exponent notation (for example `6.4e-8`, possible for small `exchange_rate` values) is expanded to plain decimal by digit shifting. JSON `null` leaves a `*Decimal` nil |
| Optional money | `*Decimal` with `omitempty` |
| No arithmetic | the SDK adds no math helpers; callers use their own decimal library |

Money and rate fields per endpoint (every field below is `Decimal`; "opt" means `*Decimal`):

| Endpoint | Request | Response |
|---|---|---|
| registerCustomer, checkCustomer, updateCustomer | none | none |
| decodeQr, decodeQrUnregistered | none | `amount`; opt `amount_usd`, `fee`, `percentage_fee`, `fixed_fee` |
| lookupPaymentCode | opt `amount` | `amount`; opt `amount_usd`, `fee`, `percentage_fee`, `fixed_fee` |
| getQuotation | opt `amount` | `amount`, `exchange_rate`, `amount_usd`, `fee`, `percentage_fee`, `fixed_fee` |
| previewQuotations | `amount` | per item: `amount`, `exchange_rate`, `amount_usd`, `fee`, `percentage_fee`, `fixed_fee` |
| executePayout | `amount_confirmed` | `exchange_rate`, `fee`, `percentage_fee`, `fixed_fee`, `amount`, `amount_usd` |
| listOccupations, listBanks | none | none |
| listTransactions, getTransaction | none | per transaction: `amount`; opt `amount_usd`, `fee`, `percentage_fee`, `fixed_fee`, `total`, `exchange_rate` |
| getAccountBalances | none | per currency: `available`, `locked`, `total` |
| webhooks | n/a | opt `amount`, `fee`, `percentage_fee`, `fixed_fee` |

Units: `amount` and `amount_confirmed` are in the local (transaction) currency. `amount_usd`, `fee`, `percentage_fee`, `fixed_fee`, `total` are USD. `exchange_rate` is USD per one unit of the transaction currency. Error `details` values `limit_usd`, `used_usd`, `remaining_usd`, `available` stay raw JSON.

Response formatting: USD fields 2 decimal places by default (banker-style rounding), `exchange_rate` 10 decimal places; `dp` changes this. At full precision `amount_usd + percentage_fee + fixed_fee` equals the amount deducted from the balance; at the default `dp` it may not.

Currency precision for request amounts ("About" page):

| Currency | Decimal places | Example |
|---|---|---|
| PHP | 2 | `1000.50` = 1,000.50 PHP |
| VND | 0 | `1000000.00` = 1,000,000 VND |
| THB | 2 | `500.00` = 500.00 THB |
| IDR | 0 | `100000.00` = 100,000 IDR |
| USD | 2 | `100.00` = 100.00 USD |

Decimal places enforced at `getQuotation` and `executePayout` ("Availability" page; too many decimals returns `INVALID_AMOUNT`):

| Decimal places | Currencies |
|---|---|
| 0 (whole numbers only) | VND, IDR |
| 2 | PHP, ARS, BRL, PEN, BOB, KES, TZS |
| not validated yet (send whole units to be safe) | THB, UGX, ZAR, NGN, GHS, ZMW, COP, PYG |

The two pages disagree on THB (2 vs "not validated"). The SDK does not validate decimals per currency; it sends what the caller gives.

Per-transaction limits in local currency (min / max): VND 10,000 / 17,000,000; PHP 1 / 40,000; THB 55 / 22,500; IDR 1,000 / 10,000,000; TZS 1,000 / 1,000,000; KES 50 / 81,000; UGX 1,000 / 2,500,000; ZAR 1 / 10,000; NGN 13 / 900,000; GHS 1 / 7,000; ZMW 1 / 12,000; BRL 0.6 / 3,300; ARS 175 / 950,000; COP 390 / 2,100,000; BOB 1.25 / 6,700; PEN 1 / 500; PYG 60 / 3,800,000. Per sender (`customer_id`): 6,500 USD per UTC day, 100,000 USD per UTC month.

## 7. Webhooks

SQRIL sends an HTTP `POST` with a JSON body to the partner's URL when a transaction is finalized. The three OpenAPI paths (`/webhooks/transactions/success`, `/webhooks/transactions`, `/webhooks/transactions/refunded`) are documentation placeholders, not SDK endpoints.

| Event filter name | Sent when | `status` |
|---|---|---|
| `transaction.success` | inbound processing marks the transaction successful | `SUCCESS` |
| `transaction.failed` | inbound processing marks the transaction failed | `FAILED` |
| `transaction.refunded` | an admin marks a `SUCCESS` transaction as `REFUNDED` (ledger credit) | `REFUNDED` |

The payload has no event-name field: the SDK tells the three apart by `status`.

| Request header | Value |
|---|---|
| `Content-Type` | `application/json` |
| `User-Agent` | `SQRIL-Webhook/<APP_VERSION>` |
| `X-SQRIL-Signature` | `base64(HMAC-SHA256(webhook_secret, exact_raw_request_body))`; present only when a webhook secret is configured |

Payload fields (one Go struct `WebhookEvent`):

| Field | Type | success | failed | refunded | Notes |
|---|---|---|---|---|---|
| `tx_id` | string | required | required | required | |
| `status` | `TransactionStatus` | `SUCCESS` | `FAILED` | `REFUNDED` | required |
| `amount` | Decimal, optional | yes | "if present" | yes | local currency (example `100000`); the payload has no `currency` |
| `fee` | Decimal, optional | yes | "if present" | yes | USD |
| `percentage_fee` | Decimal, optional | yes | "if present" | yes | USD |
| `fixed_fee` | Decimal, optional | yes | "if present" | yes | USD |
| `error_message` | string, optional | no | yes | no | one of the payout failure reasons of section 3 |
| `refund_reason` | string, optional | no | no | yes | only when ops supplied a reason |
| `sender` | `Party`, optional | yes | yes | yes | example keys: `name_first`, `name_last`, `country`, `ic_number`, `ic_type`, `phone`, `email` |
| `recipient` | `Party`, optional | yes | yes | yes | example keys: `name_first`, `name_last`, `country`, `account_no`, `bank_code`, `bank_name` |

The failed payload schema declares only `tx_id`, `status`, `error_message`, `sender`, `recipient`; the prose adds "fee fields (if present)".

Examples from the docs:

```json
{"tx_id":"{{tx_id}}","status":"SUCCESS","amount":100000,"fee":0.35,"percentage_fee":0.25,"fixed_fee":0.1,"sender":{"name_first":"John","name_last":"Doe","country":"MY","ic_number":"123456789012","ic_type":"NIC","phone":"+60123456789","email":"john.doe@example.com"},"recipient":{"name_first":"Jane","name_last":"Smith","country":"VN","account_no":"1234567890","bank_code":"970436","bank_name":"Vietcombank"}}
```

```json
{"tx_id":"{{tx_id}}","status":"FAILED","error_message":"Transaction failed","sender":{...},"recipient":{...}}
```

```json
{"tx_id":"{{tx_id}}","status":"REFUNDED","amount":100000,"fee":0.35,"percentage_fee":0.25,"fixed_fee":0.1,"refund_reason":"Payment reversal","sender":{...},"recipient":{...}}
```

Signature verification rule: compute HMAC-SHA256 with the webhook secret over the exact raw HTTP body bytes (never a re-serialized JSON), encode as standard base64 (not hex), compare with the header in constant time (`hmac.Equal`).

Expected response from the receiver: HTTP 200, body `{"received": true, "message": "Webhook processed successfully"}`.

Go API in `webhook.go`:

| Name | Signature | Behaviour |
|---|---|---|
| `WebhookSignatureHeader` | const `"X-SQRIL-Signature"` | |
| `VerifyWebhookSignature` | `(secret string, rawBody []byte, signature string) error` | returns `ErrInvalidWebhookSignature` when the signature is empty or does not match |
| `ParseWebhookEvent` | `(rawBody []byte) (*WebhookEvent, error)` | decodes; errors on invalid JSON, missing `tx_id` or missing `status`; an unknown `status` is kept as is |
| `WebhookEvent` | struct of the table above | no network call, no client needed |

UNCONFIRMED: retry policy and timeout when the receiver does not answer 200; whether deliveries can repeat (the receiver should be idempotent on `tx_id` + `status`); no timestamp or replay protection is documented; the OpenAPI attaches `BasicAuth` to the webhook operations, most likely a generator artifact (whether SQRIL sends an `Authorization` header to the receiver is not stated); the "Node.js verification example" the pages refer to is not in the local docs.

## 8. Sandbox

Sandbox = staging, `https://stg-api.sqril.io`. The test helper must refuse to run when the base URL is `https://api.sqril.io`.

| Env var | Purpose | Needed by |
|---|---|---|
| `SQRIL_BASE_URL` | must be the staging URL | all sandbox tests |
| `SQRIL_CLIENT_ID` | Basic auth user | all |
| `SQRIL_CLIENT_SECRET` | Basic auth password | all |
| `SQRIL_TEST_CUSTOMER_ID` | optional existing customer; when empty the flow registers one | read tests, main flow |
| `SQRIL_TEST_QR_STRING` | optional real QR payload for `DecodeQR` (the docs give placeholders only); test skips when empty | decode test |
| `SQRIL_TEST_EXECUTE_PAYOUT` | set to `1` to run the `ExecutePayout` step (moves sandbox balance) | main flow, last step |
| `SQRIL_WEBHOOK_SECRET` | not used by sandbox tests (webhook tests are unit tests with a fixed secret); listed for the example program | example only |

Missing credentials: tests skip, they do not fail. Test names start with `TestSandbox_`, in `*_integration_test.go`.

Safe read-only calls (create nothing):

| Call | Notes |
|---|---|
| `GetAccountBalances` | |
| `ListBanks` | `country_code=VN` (no account precondition documented; `ID` needs IDR) |
| `ListOccupations` | `format=legacy` returns `count` 12 |
| `ListTransactions` | `limit=1` |
| `GetTransaction` | needs an existing `tx_id` (take one from `ListTransactions`) |
| `CheckCustomer` | needs a customer id |
| `PreviewQuotations` | POST but stateless; `{"amount":100,"currency":"USD"}`; rate limit 60/min |

Main flow and what it leaves behind:

| Step | Call | Input | Leaves on sandbox |
|---|---|---|---|
| 1 | `RegisterCustomer` | docs sample profile, `name_first` `sdk-sandbox-test`; include `phone`, `email`, `gender`, `nationality` | one customer (upsert; there is no delete endpoint) |
| 2a | `LookupPaymentCode` | `country_code` `CO`, `payment_method` `payment_key`, `payment_destination` `sqril-breb-qa-manual-amount` (docs: "sandbox static") or `sqril-breb-qa` ("sandbox dynamic"), `payment_destination_code` `BREB` | one `PENDING` transaction (becomes `EXPIRED` if not executed) |
| 2b | `DecodeQR` (alternative) | `SQRIL_TEST_QR_STRING` | one `PENDING` transaction |
| 3 | `GetQuotation` | `tx_id`, `customer_id`, `amount` within the corridor limits (COP: 390 to 2,100,000, whole units) | quotation stored, expires in 30 minutes |
| 4 | `ExecutePayout` (only with `SQRIL_TEST_EXECUTE_PAYOUT=1`) | `amount_confirmed` and `currency` from step 3, fresh idempotency key, `partner_transaction_id` `sdk-sandbox-test-<time>` | one `PROCESSING` transaction, sandbox balance locked/deducted; final status by webhook or `GetTransaction` |
| 5 | `GetTransaction` | `tx_id` | nothing |

UNCONFIRMED: whether the sandbox account has balance (else step 4 returns 402 `INSUFFICIENT_FUNDS`), whether sandbox payouts reach `SUCCESS`, whether CO is enabled on our sandbox account, and whether the sample profile (`ic_country` `SG`, `ic_type` `PP`) passes the CO eligibility check.

Real error cases to assert:

| Case | Expect |
|---|---|
| `GetTransaction` with `transaction_id` `tx_sdk_sandbox_test_missing` | `*APIError`, status 404, code `TRANSACTION_NOT_FOUND`, `IsNotFound` true |
| any read call with a wrong secret | `*APIError`, status 401, `IsUnauthorized` true (code `AUTHENTICATION_REQUIRED` or `INVALID_CREDENTIALS`; assert the status only) |
| `PreviewQuotations` with `amount` `0` | status 400, code `INVALID_AMOUNT` |

## 9. Discrepancies with existing integration

Cross-check only against existing integrations (one Go client with customer calls only, one TypeScript client); the docs win.

| # | Where | Existing code | Docs |
|---|---|---|---|
| 1 | Go `ErrorResponse` | reads `error`, `error_code`, `validation_errors[]{field,message}`; never reads `error_message` or `details` | `error_code`, `error_message`, `error`, `details`; `validation_errors` is not documented |
| 2 | Go `UpdateCustomerRequest` | sends `nationality`; pointer fields without `omitempty` (sends `null`) | body schema has `email`, `phone`, `occupation`, `address`, `gender` only; prose says nationality can be backfilled with updateCustomer |
| 3 | Go gender mapping | sends `M` / `F` | registerCustomer says "M or F"; Peru notes and updateCustomer say `MALE` or `FEMALE` |
| 4 | Go `RegisterCustomerResponse` | no `allowed`; timestamps parsed as `time.Time` | `allowed` map exists; timestamps "when available" |
| 5 | TS `SqrilIcType`, `SqrilOccupation`, gender | no `DL`; no `OCC12`; gender allows `O` | `NIC`, `PP`, `WEP`, `DL`; `OCC1`..`OCC12` (and ISCO-08 codes); gender M/F |
| 6 | TS `SqrilTransactionStatus` | no `REFUNDED` | `PENDING`, `PROCESSING`, `SUCCESS`, `FAILED`, `EXPIRED`, `REFUNDED` |
| 7 | TS decodeQr | sends `payment_destination_code` (and maps `network` to it) on `/decodeQr`; comment says staging requires it for PE and `BREB` for CO | `/decodeQr` has no such field; typed destinations must use `/lookupPaymentCode`, where `payment_destination_code` lives |
| 8 | TS decodeQr request | lacks `partner_app_id`, `wallet_identifier`, `merchant_id`, `account_reference` | documented |
| 9 | TS client | no `lookupPaymentCode`, `decodeQrUnregistered`, `listOccupations` | 14 endpoints |
| 10 | TS executePayout | request lacks `partner_transaction_id`, `recipient_name`; response lacks `customer_id`; `sender`/`recipient` typed as required with keys `ic`, `country` | request fields documented; response has `customer_id`; `sender`/`recipient` optional, example keys `ic_number`, `ic_country` |
| 11 | TS `dp` | only on getQuotation (uses 6) | also on decodeQr, decodeQrUnregistered, lookupPaymentCode, previewQuotations, executePayout, listTransactions, getTransaction |
| 12 | TS listTransactions / `SqrilTransaction` | no `customer_id`, `date_field` filters; no `customer_id`, `exchange_rate`, `partner_transaction_id`, `qr_data`, `missing_fields`, `quotation_expires_at`, `completed_at`, `refunded_at`; many fields typed as required | documented, all optional |
| 13 | TS `SqrilBank` | no `short_name`, `swift_code` | documented for Indonesia |
| 14 | TS missing-field shape | `field`, `type`, `required`, `label`, `placeholder`, `description` | examples show `field`, `type`, `label` (decode) and `field`, `type`, `required` (error details) |
| 15 | TS webhook | not implemented ("signature scheme is not yet configured"); status is polled | `X-SQRIL-Signature` scheme and three payloads documented |
| 16 | TS money | `number` everywhere | JSON numbers on the wire; the SDK uses `Decimal` |
| 17 | Both | config names `username` / `password` | `client_id` / `client_secret` |

## 10. Open questions

1. UNCONFIRMED: where a developer obtains `client_id` / `client_secret`, and how the webhook URL, event filter and `webhook_secret` are configured (no portal URL in the docs).
2. UNCONFIRMED: production base URL `https://api.sqril.io` is not in the docs (given by the user).
3. UNCONFIRMED: `executePayout` HMAC when the account has `signing_secret` (`X-Signature` or body `signature`): encoding, exact signed bytes, and whether our accounts have it. Not implemented; no `signer/` package.
4. UNCONFIRMED: no request id / trace id field or header is documented.
5. UNCONFIRMED: when the error field `error` is sent, and whether `current_status`, `missing_fields`, `available`, `limit_usd`, `used_usd`, `remaining_usd`, `resets_at` sit inside `details` or at the top level.
6. UNCONFIRMED: error body shape of `previewQuotations` (typed "Any type").
7. UNCONFIRMED: `registerCustomer` upsert key (which field identifies an existing customer) and the 200 vs 201 split (OpenAPI lists 201 only).
8. UNCONFIRMED: accepted `gender` values (`M`/`F` vs `MALE`/`FEMALE`).
9. UNCONFIRMED: whether `updateCustomer` accepts `nationality` (prose yes, schema no).
10. UNCONFIRMED: `occupation` accepted values (register page says `OCC1`..`OCC12` only; listOccupations says ISCO-08 codes are preferred).
11. UNCONFIRMED: `lookupPaymentCode` for `PH`, `UG`, and the KE/TZ contract (`merchant_id` in the OpenAPI vs `payment_method: payment_key` + `payment_destination` + `TILL`/`PAYBILL` in the guide).
12. UNCONFIRMED: `listOccupations` with `format=all`: keys of the `count` object; whether `legacy_occ` can be null.
13. UNCONFIRMED: `listTransactions` parameters `detail` (`full`) and `partner_app_id` (in prose, not declared); whether `status` accepts `EXPIRED` and `REFUNDED`.
14. UNCONFIRMED: the field carrying the failure reason on a `Transaction` (`error_message` per the Error Codes page; not in the schema), and whether `refund_reason` appears on it.
15. UNCONFIRMED: `listBanks` extra fields `channel_category` (NG/GH/ZM) and `bank_list_code` (PH), and the values of `type`.
16. UNCONFIRMED: complete key sets of `missing_fields` items, `sender`, `recipient`, `suggested_values`, `qr_data` (free-form in the OpenAPI).
17. UNCONFIRMED: THB decimal places (2 on the About page, "not validated" on the Availability page).
18. UNCONFIRMED: webhook retry policy, duplicate deliveries, replay protection, and whether an `Authorization` header is sent to the receiver.
19. UNCONFIRMED: sandbox state: account balance, whether payouts settle, which corridors are enabled, and a usable QR string fixture.

## 11. Files

`gosqril/go.mod`
`gosqril/doc.go`
`gosqril/client.go`
`gosqril/request.go`
`gosqril/response.go`
`gosqril/errors.go`
`gosqril/money.go`
`gosqril/webhook.go`
`gosqril/customer_register.go`
`gosqril/customer_check.go`
`gosqril/customer_update.go`
`gosqril/qr_decode.go`
`gosqril/qr_decode_unregistered.go`
`gosqril/payment_code_lookup.go`
`gosqril/quotation_get.go`
`gosqril/quotation_preview.go`
`gosqril/payout_execute.go`
`gosqril/occupation_list.go`
`gosqril/transaction_list.go`
`gosqril/transaction_get.go`
`gosqril/account_balance_get.go`
`gosqril/bank_list.go`
`gosqril/example/quickstart/main.go`
`gosqril/README.md`
`gosqril/.env.example`
`gosqril/.gitignore`
`gosqril/.golangci.yml`
`gosqril/api-spec.md`
