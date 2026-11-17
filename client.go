package gosqril

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// Version is the SDK version. It is sent as "User-Agent: gosqril/<Version>".
	Version = "0.1.1"

	// SandboxBaseURL is the SQRIL staging environment.
	SandboxBaseURL = "https://stg-api.sqril.io"
	// ProductionBaseURL is the SQRIL live environment. Calls there move real money.
	ProductionBaseURL = "https://api.sqril.io"

	// DefaultTimeout is the timeout of the default HTTP client.
	DefaultTimeout = 30 * time.Second

	maxResponseBytes = 10 << 20

	userAgent            = "gosqril/" + Version
	idempotencyKeyHeader = "X-Idempotency-Key"
)

// Errors returned before any request is sent. Compare with errors.Is.
var (
	ErrEmptyBaseURL = errors.New("gosqril: baseURL is required")
	// ErrInsecureBaseURL rejects a base URL that is not https (http is allowed on loopback only),
	// has no host, or carries userinfo, a query or a fragment.
	ErrInsecureBaseURL   = errors.New("gosqril: baseURL is not allowed")
	ErrEmptyClientID     = errors.New("gosqril: clientID is required")
	ErrEmptyClientSecret = errors.New("gosqril: clientSecret is required")
	ErrNilRequest        = errors.New("gosqril: request is nil")
	ErrMissingField      = errors.New("gosqril: required field is empty")
	ErrInvalidField      = errors.New("gosqril: field value is invalid")
)

// HTTPClient is satisfied by *http.Client and by a test double.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Logger is satisfied by *slog.Logger.
type Logger interface {
	Debug(msg string, args ...any)
}

// Option changes an optional setting of the Client.
type Option func(*Client)

// Client calls the SQRIL API. Create one with NewClient and reuse it; it is safe for concurrent use.
type Client struct {
	baseURL      string
	clientID     string
	clientSecret string
	httpClient   HTTPClient
	timeout      time.Duration
	logger       Logger
}

// WithHTTPClient replaces the default client. The caller's client must set its own timeout and
// should not follow redirects (the default client does not).
func WithHTTPClient(httpClient HTTPClient) Option {
	return func(c *Client) { c.httpClient = httpClient }
}

// WithTimeout changes the timeout of the default HTTP client (DefaultTimeout). Zero and negative
// values are ignored. It has no effect together with WithHTTPClient.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithLogger turns on debug logs of method, path and status. The query string, headers, keys and
// bodies are never logged.
func WithLogger(logger Logger) Option {
	return func(c *Client) { c.logger = logger }
}

// NewClient returns a client that authenticates with HTTP Basic (clientID:clientSecret).
// It fails when baseURL, clientID or clientSecret is empty, and with ErrInsecureBaseURL when
// baseURL is not https (http is accepted for localhost, 127.0.0.1 and ::1 only).
// Keep clientSecret on a server: never ship it in a browser or a mobile app.
func NewClient(baseURL, clientID, clientSecret string, opts ...Option) (*Client, error) {
	if baseURL == "" {
		return nil, ErrEmptyBaseURL
	}

	if clientID == "" {
		return nil, ErrEmptyClientID
	}

	if clientSecret == "" {
		return nil, ErrEmptyClientSecret
	}

	if err := validateBaseURL(baseURL); err != nil {
		return nil, err
	}

	client := &Client{
		baseURL:      strings.TrimRight(baseURL, "/"),
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   nil,
		timeout:      DefaultTimeout,
		logger:       nil,
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.httpClient == nil {
		client.httpClient = &http.Client{
			Timeout:       client.timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
		}
	}

	return client, nil
}

// String returns the base URL and client id. The client secret is never included.
func (c Client) String() string {
	return fmt.Sprintf("gosqril.Client{baseURL: %q, clientID: %q, clientSecret: REDACTED}", c.baseURL, c.clientID)
}

// GoString is String, so that %#v does not print the client secret either.
func (c Client) GoString() string { return c.String() }

func validateBaseURL(baseURL string) error {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("%w: cannot be parsed", ErrInsecureBaseURL)
	}

	switch {
	case parsed.Hostname() == "":
		return fmt.Errorf("%w: host is missing", ErrInsecureBaseURL)
	case parsed.User != nil:
		return fmt.Errorf("%w: userinfo is not accepted", ErrInsecureBaseURL)
	case parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "":
		return fmt.Errorf("%w: query and fragment are not accepted", ErrInsecureBaseURL)
	case parsed.Scheme == "https":
		return nil
	case parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()):
		return nil
	}

	return fmt.Errorf("%w: scheme must be https", ErrInsecureBaseURL)
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

func call[T any](ctx context.Context, c *Client, req request) (*T, error) {
	resp := new(T)
	if err := c.execute(ctx, req, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Client) execute(ctx context.Context, req request, out any) error {
	var body []byte

	if req.Body != nil {
		encoded, err := json.Marshal(req.Body)
		if err != nil {
			return fmt.Errorf("gosqril: encode request body of %s %s: %w", req.Method, req.Path, err)
		}

		body = encoded
	}

	fullURL := c.baseURL + req.Path
	if query := req.Query.Encode(); query != "" {
		fullURL += "?" + query
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, fullURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("gosqril: build request %s %s: %w", req.Method, req.Path, err)
	}

	httpReq.SetBasicAuth(c.clientID, c.clientSecret)
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", userAgent)

	if body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	if req.IdempotencyKey != "" {
		httpReq.Header.Set(idempotencyKeyHeader, req.IdempotencyKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("gosqril: send %s %s: %w", req.Method, req.Path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("gosqril: read response of %s %s: %w", req.Method, req.Path, err)
	}

	if c.logger != nil {
		c.logger.Debug("gosqril request", "method", req.Method, "path", req.Path, "status", resp.StatusCode)
	}

	if len(raw) > maxResponseBytes {
		return fmt.Errorf("gosqril: response of %s %s (status %d) is larger than %d bytes: %w",
			req.Method, req.Path, resp.StatusCode, maxResponseBytes, ErrInvalidResponse)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return newAPIError(resp.StatusCode, raw)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("gosqril: unexpected status %d for %s %s: %w",
			resp.StatusCode, req.Method, req.Path, ErrInvalidResponse)
	}

	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("gosqril: decode response of %s %s (status %d): %w: %w",
			req.Method, req.Path, resp.StatusCode, ErrInvalidResponse, err)
	}

	return nil
}
