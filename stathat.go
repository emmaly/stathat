// Package stathat provides a complete client for the StatHat API,
// covering stat posting (EZ and Classic), data export, stat management,
// and alert configuration.
package stathat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

const (
	defaultPostURL   = "https://api.stathat.com"
	defaultExportURL = "https://www.stathat.com"
)

// Client is a StatHat API client. Create one with New.
type Client struct {
	httpClient  *http.Client
	postURL     string
	exportURL   string
	ezKey       string
	userKey     string
	accessToken string
	logger      *slog.Logger
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets a custom HTTP client for all requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.httpClient = c }
}

// WithEZKey sets the EZ API key (typically the account email address).
func WithEZKey(key string) Option {
	return func(cl *Client) { cl.ezKey = key }
}

// WithUserKey sets the Classic API user key.
func WithUserKey(userKey string) Option {
	return func(cl *Client) { cl.userKey = userKey }
}

// WithAccessToken sets the access token for export, stat management, and alerts.
func WithAccessToken(token string) Option {
	return func(cl *Client) { cl.accessToken = token }
}

// WithLogger sets a structured logger. Defaults to slog.Default().
func WithLogger(l *slog.Logger) Option {
	return func(cl *Client) { cl.logger = l }
}

// WithPostURL overrides the posting API base URL (useful for testing).
func WithPostURL(u string) Option {
	return func(cl *Client) { cl.postURL = strings.TrimRight(u, "/") }
}

// WithExportURL overrides the export API base URL (useful for testing).
func WithExportURL(u string) Option {
	return func(cl *Client) { cl.exportURL = strings.TrimRight(u, "/") }
}

// New creates a new StatHat client with the given options.
func New(opts ...Option) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		postURL:    defaultPostURL,
		exportURL:  defaultExportURL,
		logger:     slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// exportPath builds the full URL for an export/management API path.
// The path should not include the /x/{token} prefix.
func (c *Client) exportPath(path string) (string, error) {
	if c.accessToken == "" {
		return "", ErrNoAccessToken
	}
	return fmt.Sprintf("%s/x/%s%s", c.exportURL, url.PathEscape(c.accessToken), path), nil
}

// doJSON performs an HTTP request and decodes the JSON response body into dst.
// If dst is nil, the response body is discarded.
func (c *Client) doJSON(ctx context.Context, req *http.Request, dst any) error {
	req = req.WithContext(ctx)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("stathat: HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if err := checkResponse(resp); err != nil {
		return err
	}

	if dst == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("stathat: decoding response: %w", err)
	}
	return nil
}

// checkResponse returns an *APIError for non-2xx responses.
func checkResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	apiErr := &APIError{StatusCode: resp.StatusCode}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err == nil && len(body) > 0 {
		var msg struct {
			Msg string `json:"msg"`
		}
		if json.Unmarshal(body, &msg) == nil && msg.Msg != "" {
			apiErr.Message = msg.Msg
		} else {
			apiErr.Message = strings.TrimSpace(string(body))
		}
	}

	return apiErr
}

// postForm sends a POST request with form-encoded body and returns the response.
func (c *Client) postForm(ctx context.Context, rawURL string, vals url.Values) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(vals.Encode()))
	if err != nil {
		return fmt.Errorf("stathat: creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return c.doJSON(ctx, req, nil)
}
