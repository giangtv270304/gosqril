package gosqril_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/giangtv270304/gosqril"
)

func TestNewClient_RequiredArguments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		baseURL      string
		clientID     string
		clientSecret string
		want         error
	}{
		{"empty base url", "", "id", "secret", gosqril.ErrEmptyBaseURL},
		{"empty client id", "https://x.example", "", "secret", gosqril.ErrEmptyClientID},
		{"empty client secret", "https://x.example", "id", "", gosqril.ErrEmptyClientSecret},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			_, err := gosqril.NewClient(testCase.baseURL, testCase.clientID, testCase.clientSecret)
			if !errors.Is(err, testCase.want) {
				t.Errorf("error = %v, want %v", err, testCase.want)
			}
		})
	}
}

func TestBaseURLConstants(t *testing.T) {
	t.Parallel()

	if gosqril.SandboxBaseURL != "https://stg-api.sqril.io" || gosqril.ProductionBaseURL != "https://api.sqril.io" {
		t.Errorf("base URLs = %s, %s", gosqril.SandboxBaseURL, gosqril.ProductionBaseURL)
	}
}

func TestClient_SendsAuthAndUserAgent(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, `{"balances":{}}`)

	if _, err := client.GetAccountBalances(context.Background(), nil); err != nil {
		t.Fatalf("GetAccountBalances: %v", err)
	}

	got := recorded.snapshot()

	if auth := got.header.Get("Authorization"); auth != "Basic dGVzdC1jbGllbnQtaWQ6dGVzdC1jbGllbnQtc2VjcmV0" {
		t.Errorf("Authorization = %q, want Basic base64(test-client-id:test-client-secret)", auth)
	}

	if userAgent := got.header.Get("User-Agent"); userAgent != "gosqril/"+gosqril.Version {
		t.Errorf("User-Agent = %q, want gosqril/%s", userAgent, gosqril.Version)
	}

	if accept := got.header.Get("Accept"); accept != "application/json" {
		t.Errorf("Accept = %q, want application/json", accept)
	}

	if key := got.header.Get("X-Idempotency-Key"); key != "" {
		t.Errorf("X-Idempotency-Key = %q, want none outside ExecutePayout", key)
	}
}

func TestClient_TrimsTrailingSlashOfBaseURL(t *testing.T) {
	t.Parallel()

	paths := make(chan string, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path

		_, _ = w.Write([]byte(`{"balances":{}}`))
	}))
	t.Cleanup(server.Close)

	client, err := gosqril.NewClient(server.URL+"/", testClientID, testClientSecret)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	if _, err := client.GetAccountBalances(context.Background(), nil); err != nil {
		t.Fatalf("GetAccountBalances: %v", err)
	}

	if path := <-paths; path != "/getAccountBalances" {
		t.Errorf("path = %q, want /getAccountBalances", path)
	}
}

func TestClient_NonJSONErrorBody(t *testing.T) {
	t.Parallel()

	const html = "<html><body>502 Bad Gateway</body></html>"

	client, _ := newFakeServer(t, http.StatusBadGateway, html)

	_, err := client.GetAccountBalances(context.Background(), nil)

	apiErr := requireAPIError(t, err, http.StatusBadGateway, "")
	if string(apiErr.RawBody) != html || apiErr.Message != html {
		t.Errorf("raw body = %q, message = %q, want the HTML page in both", apiErr.RawBody, apiErr.Message)
	}
}

func TestClient_BrokenSuccessBody(t *testing.T) {
	t.Parallel()

	for _, body := range []string{`{"balances":`, ``, `<html>ok</html>`} {
		client, _ := newFakeServer(t, http.StatusOK, body)

		resp, err := client.GetAccountBalances(context.Background(), nil)
		if err == nil || resp != nil {
			t.Errorf("body %q: resp = %v, err = %v, want nil response and an error", body, resp, err)
		}

		var apiErr *gosqril.APIError
		if errors.As(err, &apiErr) || !errors.Is(err, gosqril.ErrInvalidResponse) {
			t.Errorf("body %q: error = %v, want ErrInvalidResponse and no *APIError", body, err)
		}
	}
}

func TestNewClient_BaseURLRules(t *testing.T) {
	t.Parallel()

	rejected := []string{
		"http://stg-api.sqril.io",
		"http://example.com:8080",
		"http://127.0.0.1.example.com",
		"ftp://stg-api.sqril.io",
		"stg-api.sqril.io",
		"//stg-api.sqril.io",
		"https://",
		"https://user:pass@stg-api.sqril.io",
		"https://user@stg-api.sqril.io",
		"https://stg-api.sqril.io?x=1",
		"https://stg-api.sqril.io?",
		"https://stg-api.sqril.io#frag",
		"http://localhost:8080?x=1",
		"https://stg-api.sqril.io/%zz",
	}
	for _, baseURL := range rejected {
		client, err := gosqril.NewClient(baseURL, testClientID, testClientSecret)
		if !errors.Is(err, gosqril.ErrInsecureBaseURL) || client != nil {
			t.Errorf("%q: client = %v, error = %v, want ErrInsecureBaseURL", baseURL, client, err)
		}

		if err != nil && strings.Contains(err.Error(), "pass") {
			t.Errorf("%q: error text %q repeats the URL", baseURL, err)
		}
	}

	accepted := []string{
		gosqril.SandboxBaseURL,
		gosqril.ProductionBaseURL,
		"https://stg-api.sqril.io/",
		"https://proxy.example.com/sqril",
		"HTTPS://stg-api.sqril.io",
		"http://127.0.0.1:8080",
		"http://localhost:8080",
		"http://LOCALHOST",
		"http://[::1]:8080",
	}
	for _, baseURL := range accepted {
		if _, err := gosqril.NewClient(baseURL, testClientID, testClientSecret); err != nil {
			t.Errorf("%q: error = %v, want nil", baseURL, err)
		}
	}
}

func TestClient_FormattingNeverPrintsSecret(t *testing.T) {
	t.Parallel()

	client, err := gosqril.NewClient(gosqril.SandboxBaseURL, testClientID, testClientSecret)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	printed := map[string]string{
		"%v pointer":  fmt.Sprintf("%v", client),
		"%+v pointer": fmt.Sprintf("%+v", client),
		"%#v pointer": fmt.Sprintf("%#v", client),
		"%v value":    fmt.Sprintf("%v", *client),
		"%+v value":   fmt.Sprintf("%+v", *client),
		"%#v value":   fmt.Sprintf("%#v", *client),
		"%v nested":   fmt.Sprintf("%v", struct{ C *gosqril.Client }{client}),
		"%+v nested":  fmt.Sprintf("%+v", []*gosqril.Client{client}),
		"%#v nested":  fmt.Sprintf("%#v", map[string]*gosqril.Client{"c": client}),
		"String":      client.String(),
		"GoString":    client.GoString(),
	}

	for name, text := range printed {
		if strings.Contains(text, testClientSecret) {
			t.Errorf("%s leaks the client secret: %s", name, text)
		}
	}

	want := `gosqril.Client{baseURL: "https://stg-api.sqril.io", clientID: "test-client-id", clientSecret: REDACTED}`
	if printed["%v pointer"] != want || printed["%+v pointer"] != want || printed["%#v pointer"] != want {
		t.Errorf("printed = %q, %q, %q, want %q", printed["%v pointer"], printed["%+v pointer"], printed["%#v pointer"], want)
	}
}

func TestClient_DoesNotFollowRedirects(t *testing.T) {
	t.Parallel()

	targetHits := make(chan struct{}, 1)

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetHits <- struct{}{}

		_, _ = w.Write([]byte(`{"balances":{"USD":{"available":1}}}`))
	}))
	t.Cleanup(target.Close)

	for _, status := range []int{http.StatusFound, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, target.URL+r.URL.Path, status)
		}))
		t.Cleanup(redirector.Close)

		client, err := gosqril.NewClient(redirector.URL, testClientID, testClientSecret)
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}

		resp, err := client.GetAccountBalances(context.Background(), nil)
		if !errors.Is(err, gosqril.ErrInvalidResponse) || resp != nil {
			t.Errorf("status %d: resp = %v, error = %v, want ErrInvalidResponse", status, resp, err)
		}
	}

	jsonRedirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", target.URL)
		w.WriteHeader(http.StatusFound)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(jsonRedirect.Close)

	client, err := gosqril.NewClient(jsonRedirect.URL, testClientID, testClientSecret)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.GetAccountBalances(context.Background(), nil)
	if !errors.Is(err, gosqril.ErrInvalidResponse) || resp != nil {
		t.Errorf("302 with a JSON body: resp = %v, error = %v, want ErrInvalidResponse", resp, err)
	}

	select {
	case <-targetHits:
		t.Error("the redirect target was called; the default client must not follow redirects")
	default:
	}
}

func TestClient_ResponseBodyCap(t *testing.T) {
	t.Parallel()

	const capBytes = 10 << 20

	padding := func(total int) string {
		const prefix, suffix = `{"balances":{},"pad":"`, `"}`

		return prefix + strings.Repeat("a", total-len(prefix)-len(suffix)) + suffix
	}

	atCap, _ := newFakeServer(t, http.StatusOK, padding(capBytes))
	if _, err := atCap.GetAccountBalances(context.Background(), nil); err != nil {
		t.Errorf("body of exactly 10 MiB: error = %v, want nil", err)
	}

	overCap, _ := newFakeServer(t, http.StatusOK, padding(capBytes+1))

	resp, err := overCap.GetAccountBalances(context.Background(), nil)
	if !errors.Is(err, gosqril.ErrInvalidResponse) || resp != nil {
		t.Errorf("body of 10 MiB + 1: resp = %v, error = %v, want ErrInvalidResponse", resp, err)
	}

	overCapError, _ := newFakeServer(t, http.StatusBadGateway, strings.Repeat("x", capBytes+1))

	_, err = overCapError.GetAccountBalances(context.Background(), nil)

	var apiErr *gosqril.APIError
	if !errors.Is(err, gosqril.ErrInvalidResponse) || errors.As(err, &apiErr) {
		t.Errorf("error body of 10 MiB + 1: error = %v, want ErrInvalidResponse and no *APIError", err)
	}
}

func TestClient_WithTimeoutIgnoresNonPositive(t *testing.T) {
	t.Parallel()

	for _, timeout := range []time.Duration{0, -time.Second} {
		client, _ := newFakeServer(t, http.StatusOK, `{"balances":{}}`, gosqril.WithTimeout(timeout))

		if _, err := client.GetAccountBalances(context.Background(), nil); err != nil {
			t.Errorf("WithTimeout(%s): error = %v, want the default timeout to stay", timeout, err)
		}
	}
}

func TestClient_CustomerIDRuleIsTheSameEverywhere(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, `{}`)
	ctx := context.Background()

	const bad = "cus_1/../admin"

	calls := map[string]func() error{
		"CheckCustomer": func() error {
			_, err := client.CheckCustomer(ctx, &gosqril.CheckCustomerRequest{CustomerID: bad})
			return err
		},
		"UpdateCustomer": func() error {
			_, err := client.UpdateCustomer(ctx, &gosqril.UpdateCustomerRequest{CustomerID: bad, Email: ptr("a@example.com")})
			return err
		},
		"DecodeQR": func() error {
			_, err := client.DecodeQR(ctx, &gosqril.DecodeQRRequest{QRString: "0002", CustomerID: bad})
			return err
		},
		"LookupPaymentCode": func() error {
			_, err := client.LookupPaymentCode(ctx, &gosqril.LookupPaymentCodeRequest{CustomerID: bad, CountryCode: "CO"})
			return err
		},
		"GetQuotation": func() error {
			_, err := client.GetQuotation(ctx, &gosqril.GetQuotationRequest{TxID: "tx_1", CustomerID: bad})
			return err
		},
		"ExecutePayout": func() error {
			_, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
				IdempotencyKey: "key", TxID: "tx_1", AmountConfirmed: "1", Currency: "VND", CustomerID: bad,
			})
			return err
		},
		"ListTransactions": func() error {
			_, err := client.ListTransactions(ctx, &gosqril.ListTransactionsRequest{CustomerID: "cus_1," + bad})
			return err
		},
	}

	for name, call := range calls {
		if err := call(); !errors.Is(err, gosqril.ErrInvalidField) {
			t.Errorf("%s: error = %v, want ErrInvalidField", name, err)
		}
	}

	assertNoCall(t, recorded)
}

type failingHTTPClient struct {
	err error
}

func (f failingHTTPClient) Do(*http.Request) (*http.Response, error) { return nil, f.err }

func TestClient_NetworkErrorIsWrapped(t *testing.T) {
	t.Parallel()

	errConnection := errors.New("connection refused")

	client, err := gosqril.NewClient("https://fake.invalid", testClientID, testClientSecret,
		gosqril.WithHTTPClient(failingHTTPClient{err: errConnection}))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	_, err = client.GetAccountBalances(context.Background(), nil)
	if !errors.Is(err, errConnection) {
		t.Errorf("error = %v, want it to wrap the network error", err)
	}

	var apiErr *gosqril.APIError
	if errors.As(err, &apiErr) {
		t.Errorf("error = %v, want no *APIError for a network error", err)
	}

	if strings.Contains(err.Error(), testClientSecret) {
		t.Errorf("error text contains the client secret: %v", err)
	}
}

func newSlowServer(t *testing.T) *httptest.Server {
	t.Helper()

	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	t.Cleanup(func() {
		close(release)
		server.Close()
	})

	return server
}

func TestClient_ContextTimeout(t *testing.T) {
	t.Parallel()

	server := newSlowServer(t)

	client, err := gosqril.NewClient(server.URL, testClientID, testClientSecret)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = client.GetAccountBalances(ctx, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}
}

func TestClient_WithTimeout(t *testing.T) {
	t.Parallel()

	server := newSlowServer(t)

	client, err := gosqril.NewClient(server.URL, testClientID, testClientSecret, gosqril.WithTimeout(50*time.Millisecond))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	started := time.Now()

	_, err = client.GetAccountBalances(context.Background(), nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("error = %v, want context.DeadlineExceeded", err)
	}

	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Errorf("call took %s, want about 50ms", elapsed)
	}
}

func TestClient_NilRequest(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, `{}`)
	ctx := context.Background()

	calls := map[string]func() error{
		"RegisterCustomer":     func() error { _, err := client.RegisterCustomer(ctx, nil); return err },
		"CheckCustomer":        func() error { _, err := client.CheckCustomer(ctx, nil); return err },
		"UpdateCustomer":       func() error { _, err := client.UpdateCustomer(ctx, nil); return err },
		"DecodeQR":             func() error { _, err := client.DecodeQR(ctx, nil); return err },
		"DecodeQRUnregistered": func() error { _, err := client.DecodeQRUnregistered(ctx, nil); return err },
		"LookupPaymentCode":    func() error { _, err := client.LookupPaymentCode(ctx, nil); return err },
		"GetQuotation":         func() error { _, err := client.GetQuotation(ctx, nil); return err },
		"PreviewQuotations":    func() error { _, err := client.PreviewQuotations(ctx, nil); return err },
		"ExecutePayout":        func() error { _, err := client.ExecutePayout(ctx, nil); return err },
		"GetTransaction":       func() error { _, err := client.GetTransaction(ctx, nil); return err },
		"ListBanks":            func() error { _, err := client.ListBanks(ctx, nil); return err },
	}

	for name, call := range calls {
		if err := call(); !errors.Is(err, gosqril.ErrNilRequest) {
			t.Errorf("%s(nil) error = %v, want ErrNilRequest", name, err)
		}
	}

	assertNoCall(t, recorded)
}

func TestClient_EmptyRequiredParams(t *testing.T) {
	t.Parallel()

	client, recorded := newFakeServer(t, http.StatusOK, `{}`)
	ctx := context.Background()
	amount := gosqril.Decimal("100")

	testCases := []struct {
		name string
		want error
		call func() error
	}{
		{"CheckCustomer without CustomerID", gosqril.ErrMissingField, func() error {
			_, err := client.CheckCustomer(ctx, &gosqril.CheckCustomerRequest{})
			return err
		}},
		{"UpdateCustomer without CustomerID", gosqril.ErrMissingField, func() error {
			_, err := client.UpdateCustomer(ctx, &gosqril.UpdateCustomerRequest{Email: ptr("a@example.com")})
			return err
		}},
		{"DecodeQR without QRString", gosqril.ErrMissingField, func() error {
			_, err := client.DecodeQR(ctx, &gosqril.DecodeQRRequest{CustomerID: "cus_1"})
			return err
		}},
		{"DecodeQR without CustomerID", gosqril.ErrMissingField, func() error {
			_, err := client.DecodeQR(ctx, &gosqril.DecodeQRRequest{QRString: "0002"})
			return err
		}},
		{"DecodeQR with slash in CustomerID", gosqril.ErrInvalidField, func() error {
			_, err := client.DecodeQR(ctx, &gosqril.DecodeQRRequest{QRString: "0002", CustomerID: "a/b"})
			return err
		}},
		{"DecodeQRUnregistered without QRString", gosqril.ErrMissingField, func() error {
			_, err := client.DecodeQRUnregistered(ctx, &gosqril.DecodeQRUnregisteredRequest{})
			return err
		}},
		{"LookupPaymentCode without CustomerID", gosqril.ErrMissingField, func() error {
			_, err := client.LookupPaymentCode(ctx, &gosqril.LookupPaymentCodeRequest{CountryCode: "CO"})
			return err
		}},
		{"LookupPaymentCode without CountryCode", gosqril.ErrMissingField, func() error {
			_, err := client.LookupPaymentCode(ctx, &gosqril.LookupPaymentCodeRequest{CustomerID: "cus_1"})
			return err
		}},
		{"LookupPaymentCode with slash in CustomerID", gosqril.ErrInvalidField, func() error {
			_, err := client.LookupPaymentCode(ctx, &gosqril.LookupPaymentCodeRequest{CustomerID: "a/b", CountryCode: "CO"})
			return err
		}},
		{"GetQuotation without TxID", gosqril.ErrMissingField, func() error {
			_, err := client.GetQuotation(ctx, &gosqril.GetQuotationRequest{CustomerID: "cus_1"})
			return err
		}},
		{"GetQuotation without CustomerID", gosqril.ErrMissingField, func() error {
			_, err := client.GetQuotation(ctx, &gosqril.GetQuotationRequest{TxID: "tx_1"})
			return err
		}},
		{"GetQuotation with slash in CustomerID", gosqril.ErrInvalidField, func() error {
			_, err := client.GetQuotation(ctx, &gosqril.GetQuotationRequest{TxID: "tx_1", CustomerID: "a/b"})
			return err
		}},
		{"PreviewQuotations without Amount", gosqril.ErrInvalidDecimal, func() error {
			_, err := client.PreviewQuotations(ctx, &gosqril.PreviewQuotationsRequest{Currency: "USD"})
			return err
		}},
		{"ExecutePayout without IdempotencyKey", gosqril.ErrMissingField, func() error {
			_, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
				TxID: "tx_1", AmountConfirmed: amount, Currency: "VND", CustomerID: "cus_1",
			})
			return err
		}},
		{"ExecutePayout without TxID", gosqril.ErrMissingField, func() error {
			_, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
				IdempotencyKey: "key", AmountConfirmed: amount, Currency: "VND", CustomerID: "cus_1",
			})
			return err
		}},
		{"ExecutePayout without CustomerID", gosqril.ErrMissingField, func() error {
			_, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
				IdempotencyKey: "key", TxID: "tx_1", AmountConfirmed: amount, Currency: "VND",
			})
			return err
		}},
		{"ExecutePayout with slash in CustomerID", gosqril.ErrInvalidField, func() error {
			_, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
				IdempotencyKey: "key", TxID: "tx_1", AmountConfirmed: amount, Currency: "VND", CustomerID: "a/b",
			})
			return err
		}},
		{"ExecutePayout without AmountConfirmed", gosqril.ErrInvalidDecimal, func() error {
			_, err := client.ExecutePayout(ctx, &gosqril.ExecutePayoutRequest{
				IdempotencyKey: "key", TxID: "tx_1", Currency: "VND", CustomerID: "cus_1",
			})
			return err
		}},
		{"GetTransaction without TransactionID", gosqril.ErrMissingField, func() error {
			_, err := client.GetTransaction(ctx, &gosqril.GetTransactionRequest{})
			return err
		}},
		{"ListBanks without CountryCode", gosqril.ErrMissingField, func() error {
			_, err := client.ListBanks(ctx, &gosqril.ListBanksRequest{})
			return err
		}},
	}

	for _, testCase := range testCases {
		if err := testCase.call(); !errors.Is(err, testCase.want) {
			t.Errorf("%s: error = %v, want %v", testCase.name, err, testCase.want)
		}
	}

	assertNoCall(t, recorded)
}

type captureLogger struct {
	lines []string
}

func (l *captureLogger) Debug(msg string, args ...any) {
	l.lines = append(l.lines, msg+" "+fmt.Sprint(args...))
}

func TestClient_DebugLogHasNoSecrets(t *testing.T) {
	t.Parallel()

	logger := new(captureLogger)
	client, _ := newFakeServer(t, http.StatusAccepted,
		`{"status":"PROCESSING","tx_id":"tx_secret_response","message":"Payout initiated successfully"}`,
		gosqril.WithLogger(logger))

	_, err := client.ExecutePayout(context.Background(), &gosqril.ExecutePayoutRequest{
		IdempotencyKey:  "idem-key-123",
		DP:              ptr(4),
		TxID:            "tx_body_value",
		AmountConfirmed: "100000",
		Currency:        "VND",
		CustomerID:      "cus_body_value",
		RecipientName:   "Jane Recipient",
	})
	if err != nil {
		t.Fatalf("ExecutePayout: %v", err)
	}

	if len(logger.lines) != 1 {
		t.Fatalf("log lines = %d, want 1", len(logger.lines))
	}

	line := logger.lines[0]

	for _, want := range []string{"POST", "/executePayout", "202"} {
		if !strings.Contains(line, want) {
			t.Errorf("log line %q does not contain %q", line, want)
		}
	}

	forbidden := []string{
		testClientID, testClientSecret, "Basic", "dGVzdC1jbGllbnQtaWQ6dGVzdC1jbGllbnQtc2VjcmV0", "Authorization",
		"idem-key-123", "X-Idempotency-Key", "tx_body_value", "cus_body_value", "Jane Recipient", "100000",
		"tx_secret_response", "Payout initiated", "dp=4", "?", "127.0.0.1",
	}
	for _, text := range forbidden {
		if strings.Contains(line, text) {
			t.Errorf("log line %q contains %q", line, text)
		}
	}
}
