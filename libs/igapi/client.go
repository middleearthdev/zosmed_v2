package igapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// defaultBaseURL is the Meta Graph API v21.0 base URL.
const defaultBaseURL = "https://graph.facebook.com/v21.0"

// Client is a thin, stateless wrapper around the Meta Instagram Graph API.
// It holds an access token and reuses a single http.Client across calls.
// Each IG Business account that connects to Zosmed gets its own Client instance.
type Client struct {
	httpClient  *http.Client
	accessToken string
	baseURL     string
}

// New returns a Client for the given IG page access token.
func New(accessToken string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		accessToken: accessToken,
		baseURL:     defaultBaseURL,
	}
}

// NewWithBaseURL returns a Client with a custom base URL, intended for testing.
func NewWithBaseURL(accessToken, baseURL string) *Client {
	return &Client{
		httpClient:  &http.Client{},
		accessToken: accessToken,
		baseURL:     baseURL,
	}
}

// post sends a JSON-encoded POST request to baseURL+path.
// It sets Authorization: Bearer <token> and Content-Type: application/json.
// On HTTP ≥400 or a Graph API error envelope it returns a non-nil error.
// If dst is non-nil, the response body is decoded into it on success.
func (c *Client) post(ctx context.Context, path string, body any, dst any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("igapi: marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+path, bytes.NewReader(encoded))
	if err != nil {
		return fmt.Errorf("igapi: build request for %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("igapi: do request %s: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("igapi: read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ge GraphErrorResponse
		if jsonErr := json.Unmarshal(raw, &ge); jsonErr == nil && ge.Error.Message != "" {
			return fmt.Errorf("igapi: %s: %w", path, ge.Error)
		}
		return fmt.Errorf("igapi: %s: http %d: %s", path, resp.StatusCode, string(raw))
	}

	if dst != nil {
		if err := json.Unmarshal(raw, dst); err != nil {
			return fmt.Errorf("igapi: decode response from %s: %w", path, err)
		}
	}
	return nil
}
