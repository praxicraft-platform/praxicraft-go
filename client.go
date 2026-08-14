package praxicraft

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client is the synchronous HTTP client for the Assess Public API.
type Client struct {
	APIKey     string
	BaseURL    string
	APIPrefix  string
	HTTPClient *http.Client
	MaxRetries int

	Assessments *AssessmentsResource
	Invites     *InvitesResource
	Results     *ResultsResource
	Org         *OrgResource
	Webhooks    *WebhooksResource
	Pipelines   *PipelinesResource
}

// ClientOption configures a Client.
type ClientOption func(*Client) error

// WithAPIKey sets the API key explicitly.
func WithAPIKey(key string) ClientOption {
	return func(c *Client) error {
		c.APIKey = strings.TrimSpace(key)
		return nil
	}
}

// WithBaseURL overrides the Assess API host.
func WithBaseURL(base string) ClientOption {
	return func(c *Client) error {
		c.BaseURL = strings.TrimRight(strings.TrimSpace(base), "/")
		return nil
	}
}

// WithHTTPClient injects a custom *http.Client (timeouts, transport, etc.).
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) error {
		c.HTTPClient = httpClient
		return nil
	}
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) error {
		if c.HTTPClient == nil {
			c.HTTPClient = &http.Client{}
		}
		c.HTTPClient.Timeout = d
		return nil
	}
}

// WithMaxRetries sets how many times to retry after the first attempt on
// 429 / 5xx / transport failures (default 2 → up to 3 total attempts).
// Pass 0 to disable retries.
func WithMaxRetries(n int) ClientOption {
	return func(c *Client) error {
		if n < 0 {
			n = 0
		}
		c.MaxRetries = n
		return nil
	}
}

// New creates a Client. Pass WithAPIKey(...) or set PRAXICRAFT_API_KEY.
func New(opts ...ClientOption) (*Client, error) {
	c := &Client{
		APIKey:     strings.TrimSpace(os.Getenv("PRAXICRAFT_API_KEY")),
		BaseURL:    strings.TrimRight(strings.TrimSpace(firstNonEmpty(os.Getenv("PRAXICRAFT_API_BASE_URL"), defaultBaseURL)), "/"),
		APIPrefix:  defaultAPIPrefix,
		MaxRetries: defaultMaxRetries,
		HTTPClient: &http.Client{
			Timeout: time.Duration(defaultTimeout) * time.Second,
		},
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	c.APIKey = strings.TrimSpace(c.APIKey)
	if c.APIKey == "" {
		return nil, &APIError{
			Message: "No API key provided. Pass WithAPIKey(...) or set PRAXICRAFT_API_KEY.",
			ErrCode: "MISSING_API_KEY",
		}
	}
	if c.BaseURL == "" {
		return nil, &APIError{Message: "base URL must be a non-empty URL.", ErrCode: "INVALID_BASE_URL"}
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: time.Duration(defaultTimeout) * time.Second}
	}

	c.Assessments = &AssessmentsResource{client: c}
	c.Invites = &InvitesResource{client: c}
	c.Results = &ResultsResource{client: c}
	c.Org = &OrgResource{client: c}
	c.Webhooks = &WebhooksResource{client: c}
	c.Pipelines = &PipelinesResource{client: c}
	return c, nil
}

// RequestOptions configure a single HTTP call.
type RequestOptions struct {
	Params  map[string]any
	JSON    any
	Headers map[string]string
	Context context.Context
}

// Do sends a request to a Public API path (e.g. "/assessments/") and returns
// decoded flat JSON (map/slice) or nil for 204. Prefer typed resource methods
// or DoAs for structured responses.
func (c *Client) Do(method, path string, opts *RequestOptions) (any, error) {
	raw, status, err := c.doRaw(method, path, opts)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNoContent || len(bytes.TrimSpace(raw)) == 0 {
		return nil, nil
	}
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, &APIError{
			Message: fmt.Sprintf("Invalid JSON response (HTTP %d).", status),
			ErrCode: "INVALID_JSON",
		}
	}
	return parsed, nil
}

// DoAs is like Do but decodes the JSON body into T.
func DoAs[T any](c *Client, method, path string, opts *RequestOptions) (T, error) {
	var out T
	raw, status, err := c.doRaw(method, path, opts)
	if err != nil {
		return out, err
	}
	if status == http.StatusNoContent || len(bytes.TrimSpace(raw)) == 0 {
		return out, nil
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, &APIError{
			Message: fmt.Sprintf("Invalid JSON response (HTTP %d).", status),
			ErrCode: "INVALID_JSON",
		}
	}
	return out, nil
}

func (c *Client) doRaw(method, path string, opts *RequestOptions) ([]byte, int, error) {
	if opts == nil {
		opts = &RequestOptions{}
	}
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	fullURL, err := c.buildURL(path, opts.Params)
	if err != nil {
		return nil, 0, err
	}

	var bodyBytes []byte
	if opts.JSON != nil {
		bodyBytes, err = json.Marshal(opts.JSON)
		if err != nil {
			return nil, 0, &APIError{Message: fmt.Sprintf("failed to encode JSON body: %v", err), ErrCode: "INVALID_JSON"}
		}
	}

	attempts := c.MaxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if err := ctx.Err(); err != nil {
				return nil, 0, &APIConnectionError{Message: err.Error()}
			}
			sleepFn(retryDelay(attempt-1, retryAfterFromErr(lastErr)))
			if err := ctx.Err(); err != nil {
				return nil, 0, &APIConnectionError{Message: err.Error()}
			}
		}

		raw, status, hdr, err := c.once(ctx, method, fullURL, bodyBytes, opts.Headers, opts.JSON != nil)
		if err != nil {
			lastErr = err
			if attempt < attempts-1 {
				continue
			}
			return nil, 0, err
		}

		if status == http.StatusNoContent {
			return nil, status, nil
		}

		if status >= 200 && status < 300 {
			return raw, status, nil
		}

		statusErr := raiseForStatus(status, raw, hdr)
		lastErr = statusErr
		if shouldRetryStatus(status) && attempt < attempts-1 {
			continue
		}
		return nil, status, statusErr
	}
	return nil, 0, lastErr
}

func retryAfterFromErr(err error) string {
	switch e := err.(type) {
	case *RateLimitError:
		return e.Headers.Get("Retry-After")
	case *APIStatusError:
		return e.Headers.Get("Retry-After")
	case *AuthenticationError:
		return e.Headers.Get("Retry-After")
	case *InsufficientScopeError:
		return e.Headers.Get("Retry-After")
	case *NotFoundError:
		return e.Headers.Get("Retry-After")
	case *ValidationError:
		return e.Headers.Get("Retry-After")
	default:
		return ""
	}
}

func (c *Client) once(
	ctx context.Context,
	method, fullURL string,
	bodyBytes []byte,
	extraHeaders map[string]string,
	hasJSON bool,
) ([]byte, int, http.Header, error) {
	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), fullURL, bodyReader)
	if err != nil {
		return nil, 0, nil, &APIConnectionError{Message: fmt.Sprintf("failed to create request: %v", err)}
	}

	req.Header.Set("Accept", "application/json")
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("User-Agent", userAgent)
	if hasJSON && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, 0, nil, &APIConnectionError{Message: fmt.Sprintf("transport error: %v", err)}
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, nil, &APIConnectionError{Message: fmt.Sprintf("failed to read response: %v", err)}
	}
	return raw, resp.StatusCode, resp.Header.Clone(), nil
}

func (c *Client) Get(path string, opts *RequestOptions) (any, error) {
	return c.Do(http.MethodGet, path, opts)
}

func (c *Client) Post(path string, opts *RequestOptions) (any, error) {
	return c.Do(http.MethodPost, path, opts)
}

func (c *Client) Patch(path string, opts *RequestOptions) (any, error) {
	return c.Do(http.MethodPatch, path, opts)
}

func (c *Client) Put(path string, opts *RequestOptions) (any, error) {
	return c.Do(http.MethodPut, path, opts)
}

func (c *Client) Delete(path string, opts *RequestOptions) (any, error) {
	return c.Do(http.MethodDelete, path, opts)
}

func (c *Client) buildURL(path string, params map[string]any) (string, error) {
	var full string
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		full = path
	} else {
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		rel := path
		if !strings.HasPrefix(path, c.APIPrefix) {
			rel = c.APIPrefix + path
		}
		full = c.BaseURL + rel
	}

	u, err := url.Parse(full)
	if err != nil {
		return "", &APIError{Message: fmt.Sprintf("invalid URL: %v", err), ErrCode: "INVALID_URL"}
	}
	if len(params) > 0 {
		q := u.Query()
		for k, v := range params {
			if v == nil {
				continue
			}
			switch t := v.(type) {
			case bool:
				if t {
					q.Set(k, "true")
				} else {
					q.Set(k, "false")
				}
			default:
				q.Set(k, fmt.Sprint(v))
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
