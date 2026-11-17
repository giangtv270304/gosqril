package gosqril_test

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/giangtv270304/gosqril"
)

const (
	envBaseURL       = "SQRIL_BASE_URL"
	envClientID      = "SQRIL_CLIENT_ID"
	envClientSecret  = "SQRIL_CLIENT_SECRET"
	envCustomerID    = "SQRIL_TEST_CUSTOMER_ID"
	envQRString      = "SQRIL_TEST_QR_STRING"
	envExecutePayout = "SQRIL_TEST_EXECUTE_PAYOUT"

	testClientID     = "test-client-id"
	testClientSecret = "test-client-secret"
)

type recordedRequest struct {
	mu     sync.Mutex
	calls  int
	method string
	path   string
	query  string
	header http.Header
	body   string
}

func (r *recordedRequest) snapshot() recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	return recordedRequest{
		calls:  r.calls,
		method: r.method,
		path:   r.path,
		query:  r.query,
		header: r.header,
		body:   r.body,
	}
}

func newFakeServer(t *testing.T, status int, body string, opts ...gosqril.Option) (*gosqril.Client, *recordedRequest) {
	t.Helper()

	recorded := new(recordedRequest)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)

		recorded.mu.Lock()
		recorded.calls++
		recorded.method = r.Method
		recorded.path = r.URL.Path
		recorded.query = r.URL.RawQuery
		recorded.header = r.Header.Clone()
		recorded.body = string(raw)
		recorded.mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(server.Close)

	client, err := gosqril.NewClient(server.URL, testClientID, testClientSecret, opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client, recorded
}

func basicAuth(clientID, clientSecret string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(clientID+":"+clientSecret))
}

func assertRequest(t *testing.T, recorded *recordedRequest, method, path, query, body string) {
	t.Helper()

	got := recorded.snapshot()

	if got.calls != 1 {
		t.Fatalf("calls = %d, want 1", got.calls)
	}

	if got.method != method || got.path != path {
		t.Errorf("request = %s %s, want %s %s", got.method, got.path, method, path)
	}

	if got.query != query {
		t.Errorf("query = %q, want %q", got.query, query)
	}

	if got.body != body {
		t.Errorf("body = %s, want %s", got.body, body)
	}

	if auth := got.header.Get("Authorization"); auth != basicAuth(testClientID, testClientSecret) {
		t.Errorf("Authorization = %q, want Basic of the test credentials", auth)
	}

	wantContentType := ""
	if body != "" {
		wantContentType = "application/json"
	}

	if contentType := got.header.Get("Content-Type"); contentType != wantContentType {
		t.Errorf("Content-Type = %q, want %q", contentType, wantContentType)
	}
}

func assertNoCall(t *testing.T, recorded *recordedRequest) {
	t.Helper()

	if calls := recorded.snapshot().calls; calls != 0 {
		t.Errorf("calls = %d, want 0: the SDK must fail before calling the API", calls)
	}
}

func assertAPIError(t *testing.T, err error, status int, code string) {
	t.Helper()

	_ = requireAPIError(t, err, status, code)
}

func requireAPIError(t *testing.T, err error, status int, code string) *gosqril.APIError {
	t.Helper()

	var apiErr *gosqril.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *gosqril.APIError", err)
	}

	if apiErr.StatusCode != status || apiErr.Code != code {
		t.Errorf("api error = status %d code %q, want status %d code %q", apiErr.StatusCode, apiErr.Code, status, code)
	}

	return apiErr
}

func assertDecimal(t *testing.T, name string, got *gosqril.Decimal, want string) {
	t.Helper()

	if got == nil {
		t.Errorf("%s = nil, want %s", name, want)

		return
	}

	if string(*got) != want {
		t.Errorf("%s = %s, want %s", name, *got, want)
	}
}

func ptr[T any](value T) *T { return &value }

func isProductionURL(baseURL string) bool {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return true
	}

	production, _ := url.Parse(gosqril.ProductionBaseURL)

	return strings.EqualFold(parsed.Hostname(), production.Hostname())
}

var errSandboxPointsAtProduction = errors.New("points at production; sandbox tests never run there")

func sandboxBaseURL() (string, error) {
	baseURL := os.Getenv(envBaseURL)
	if baseURL == "" {
		baseURL = gosqril.SandboxBaseURL
	}

	if isProductionURL(baseURL) {
		return "", fmt.Errorf("%s %w", envBaseURL, errSandboxPointsAtProduction)
	}

	return baseURL, nil
}

func sandboxClientWith(clientID, clientSecret string) (*gosqril.Client, error) {
	baseURL, err := sandboxBaseURL()
	if err != nil {
		return nil, err
	}

	return gosqril.NewClient(baseURL, clientID, clientSecret)
}

func newSandboxClient(t *testing.T) *gosqril.Client {
	t.Helper()

	clientID := os.Getenv(envClientID)
	clientSecret := os.Getenv(envClientSecret)

	if clientID == "" || clientSecret == "" {
		t.Skipf("sandbox test skipped: set %s and %s", envClientID, envClientSecret)
	}

	client, err := sandboxClientWith(clientID, clientSecret)
	if err != nil {
		t.Fatalf("sandbox client: %v", err)
	}

	return client
}
