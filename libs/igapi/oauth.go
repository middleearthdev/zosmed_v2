package igapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// OAuth endpoint hosts for Business Login for Instagram (RESOLVED G6).
// Kept as package vars (not consts) solely so tests can redirect them at an
// httptest server — see export_test.go. Production code must never mutate
// these; there is exactly one App per Zosmed deployment (DRY §12a-1).
var (
	oauthAuthorizeURL         = "https://www.instagram.com/oauth/authorize"
	oauthExchangeCodeURL      = "https://api.instagram.com/oauth/access_token"
	oauthExchangeLongLivedURL = "https://graph.instagram.com/access_token"
	oauthRefreshLongLivedURL  = "https://graph.instagram.com/refresh_access_token"
)

// DefaultScopes is the scope set requested during Business Login for
// Instagram (RESOLVED G7). instagram_business_content_publish and
// instagram_business_manage_insights are deliberately excluded from the MVP
// default — add them only when a feature actually needs them.
var DefaultScopes = []string{
	"instagram_business_basic",
	"instagram_business_manage_comments",
	"instagram_business_manage_messages",
}

// OAuthConfig holds the app-level credentials for Business Login for
// Instagram (ADR-002 §2.5). It is stateless: no DB, no HTTP handler, no
// session — those live in apps/api/internal/connect. Exposed as explicit
// methods rather than an interface: there is exactly one implementation
// (Instagram) today (§12a-4 — no premature abstraction).
type OAuthConfig struct {
	AppID       string
	AppSecret   string
	RedirectURI string
}

// AuthorizeURL builds the redirect URL for step 1 of Business Login for
// Instagram (RESOLVED G6). It performs no network call.
func (o OAuthConfig) AuthorizeURL(state string, scopes []string) string {
	if len(scopes) == 0 {
		scopes = DefaultScopes
	}
	q := url.Values{}
	q.Set("client_id", o.AppID)
	q.Set("redirect_uri", o.RedirectURI)
	q.Set("response_type", "code")
	q.Set("scope", strings.Join(scopes, ","))
	q.Set("state", state)
	return oauthAuthorizeURL + "?" + q.Encode()
}

// ShortLivedToken is the result of exchanging an OAuth authorization code
// (step 2, RESOLVED G6). The code is valid for one hour and single-use.
type ShortLivedToken struct {
	AccessToken string
	UserID      string
}

// shortLivedTokenResponse is tolerant of the response shapes documented for
// this endpoint across Meta's Instagram Login materials: either a flat
// object, or one wrapped in a `data` array (mirrors the same "don't guess
// blindly" tolerant-parser approach ADR-002 §6.3/G10 uses for the webhook
// comment id ambiguity). Confirm the exact shape against a live sandbox
// payload before go-live — documented as a residual risk in ADR-002 §11-R.
type shortLivedTokenResponse struct {
	AccessToken string      `json:"access_token"`
	UserID      json.Number `json:"user_id"`
	Data        []struct {
		AccessToken string      `json:"access_token"`
		UserID      json.Number `json:"user_id"`
	} `json:"data"`
}

func (r shortLivedTokenResponse) resolve() ShortLivedToken {
	if r.AccessToken != "" {
		return ShortLivedToken{AccessToken: r.AccessToken, UserID: r.UserID.String()}
	}
	if len(r.Data) > 0 {
		return ShortLivedToken{AccessToken: r.Data[0].AccessToken, UserID: r.Data[0].UserID.String()}
	}
	return ShortLivedToken{}
}

// ExchangeCode exchanges an authorization code for a short-lived access
// token (POST api.instagram.com/oauth/access_token, RESOLVED G6).
func (o OAuthConfig) ExchangeCode(ctx context.Context, code string) (ShortLivedToken, error) {
	form := url.Values{}
	form.Set("client_id", o.AppID)
	form.Set("client_secret", o.AppSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", o.RedirectURI)
	form.Set("code", code)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		oauthExchangeCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return ShortLivedToken{}, fmt.Errorf("igapi: ExchangeCode: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var raw shortLivedTokenResponse
	if err := doOAuthRequest(req, &raw); err != nil {
		return ShortLivedToken{}, fmt.Errorf("igapi: ExchangeCode: %w", err)
	}
	tok := raw.resolve()
	if tok.AccessToken == "" {
		return ShortLivedToken{}, fmt.Errorf("igapi: ExchangeCode: empty access_token in response")
	}
	return tok, nil
}

// LongLivedToken is an IG-user-scoped long-lived token, valid ~60 days
// (RESOLVED G9). ExpiresIn is in seconds; the caller (connect handler or
// refresh scheduler) converts it to an absolute token_expires_at — igapi
// itself never stores or interprets timestamps.
type LongLivedToken struct {
	AccessToken string
	TokenType   string
	ExpiresIn   int64
}

type longLivedTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

// ExchangeLongLived exchanges a short-lived token for a long-lived one
// (GET graph.instagram.com/access_token, grant_type=ig_exchange_token,
// RESOLVED G6/G9).
func (o OAuthConfig) ExchangeLongLived(ctx context.Context, shortToken string) (LongLivedToken, error) {
	q := url.Values{}
	q.Set("grant_type", "ig_exchange_token")
	q.Set("client_secret", o.AppSecret)
	q.Set("access_token", shortToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		oauthExchangeLongLivedURL+"?"+q.Encode(), nil)
	if err != nil {
		return LongLivedToken{}, fmt.Errorf("igapi: ExchangeLongLived: build request: %w", err)
	}

	var raw longLivedTokenResponse
	if err := doOAuthRequest(req, &raw); err != nil {
		return LongLivedToken{}, fmt.Errorf("igapi: ExchangeLongLived: %w", err)
	}
	if raw.AccessToken == "" {
		return LongLivedToken{}, fmt.Errorf("igapi: ExchangeLongLived: empty access_token in response")
	}
	if raw.ExpiresIn <= 0 {
		return LongLivedToken{}, fmt.Errorf("igapi: ExchangeLongLived: missing/invalid expires_in (%d)", raw.ExpiresIn)
	}
	return LongLivedToken{AccessToken: raw.AccessToken, TokenType: raw.TokenType, ExpiresIn: raw.ExpiresIn}, nil
}

// RefreshLongLived extends a long-lived token's validity by another ~60 days
// (GET graph.instagram.com/refresh_access_token, grant_type=ig_refresh_token,
// RESOLVED G6/G9). Meta requires the token to be at least 24 hours old and
// still valid; the caller (apps/worker/internal/tasks/token_refresh.go) is
// responsible for only calling this on tokens that satisfy that constraint.
func (o OAuthConfig) RefreshLongLived(ctx context.Context, longToken string) (LongLivedToken, error) {
	q := url.Values{}
	q.Set("grant_type", "ig_refresh_token")
	q.Set("access_token", longToken)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		oauthRefreshLongLivedURL+"?"+q.Encode(), nil)
	if err != nil {
		return LongLivedToken{}, fmt.Errorf("igapi: RefreshLongLived: build request: %w", err)
	}

	var raw longLivedTokenResponse
	if err := doOAuthRequest(req, &raw); err != nil {
		return LongLivedToken{}, fmt.Errorf("igapi: RefreshLongLived: %w", err)
	}
	if raw.AccessToken == "" {
		return LongLivedToken{}, fmt.Errorf("igapi: RefreshLongLived: empty access_token in response")
	}
	if raw.ExpiresIn <= 0 {
		return LongLivedToken{}, fmt.Errorf("igapi: RefreshLongLived: missing/invalid expires_in (%d)", raw.ExpiresIn)
	}
	return LongLivedToken{AccessToken: raw.AccessToken, TokenType: raw.TokenType, ExpiresIn: raw.ExpiresIn}, nil
}

// doOAuthRequest executes req against one of the OAuth hosts (which are not
// graph.instagram.com/v25.0, so Client.do cannot be reused directly) and
// decodes the JSON body into dst. On HTTP >=400 it attempts to decode a
// Graph-style error envelope, falling back to the raw body.
func doOAuthRequest(req *http.Request, dst any) error {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// The long-lived exchange/refresh calls carry access_token in the query
		// string; *url.Error.Error() reflects the full URL (token included) and
		// would leak the token into any logger that prints err (§12 "jangan log
		// token"). Strip the URL — keep only the path + underlying network error.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return fmt.Errorf("%s: %w", req.URL.Path, urlErr.Err)
		}
		return fmt.Errorf("%s: do request: %w", req.URL.Path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var ge GraphErrorResponse
		if jsonErr := json.Unmarshal(raw, &ge); jsonErr == nil && ge.Error.Message != "" {
			return fmt.Errorf("%s: %w", req.URL.Path, ge.Error)
		}
		return fmt.Errorf("%s: http %d: %s", req.URL.Path, resp.StatusCode, string(raw))
	}

	if err := json.Unmarshal(raw, dst); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
