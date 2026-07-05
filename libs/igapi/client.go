package igapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// defaultBaseURL is the Instagram API with Instagram Login base URL
// (CLAUDE.md §4.0; ADR-002 §11-R RESOLVED G1). Version is segment-pinned
// (v25.0) per Meta's live documentation as of the ADR-002 reconciliation.
// This is the single source of truth for the host — every other igapi path
// is relative to it.
const defaultBaseURL = "https://graph.instagram.com/v25.0"

// Client is a thin, stateless wrapper around the Instagram API with
// Instagram Login (graph.instagram.com). It holds an IG user access token
// and reuses a single http.Client across calls. Each IG Business/Creator
// account connected to Zosmed via OAuth (apps/api/internal/connect) gets its
// own Client instance, built from that account's stored token.
type Client struct {
	httpClient  *http.Client
	accessToken string
	baseURL     string
}

// New returns a Client for the given IG user access token (IG-user-scoped
// long-lived token obtained via Business Login for Instagram, CLAUDE.md §4.0).
// This is NOT a Facebook Page access token.
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
	req.Header.Set("Content-Type", "application/json")

	return c.do(req, path, dst)
}

// get sends a GET request to baseURL+path with the bearer token set.
// Same error/decode semantics as post.
func (c *Client) get(ctx context.Context, path string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("igapi: build request for %s: %w", path, err)
	}
	return c.do(req, path, dst)
}

// do sets the Authorization header, executes req, and decodes the response.
// Shared by post and get to avoid duplicating error-envelope handling (§12a-1).
func (c *Client) do(req *http.Request, path string, dst any) error {
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

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
