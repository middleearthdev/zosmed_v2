package igapi

import (
	"context"
	"fmt"
)

// MeResult is the identity payload returned by GET /me (RESOLVED G8).
// UserID is the Instagram-scoped user ID (IGSID) to store as account.ig_user_id.
type MeResult struct {
	UserID      string `json:"user_id"`
	Username    string `json:"username"`
	AccountType string `json:"account_type"`
}

// Me resolves the identity of the account behind the Client's access token.
//
// Endpoint: GET /me?fields=user_id,username,account_type (RESOLVED G8).
// Called once during the OAuth connect callback (apps/api/internal/connect)
// right after the long-lived token is obtained, to learn the IGSID that
// anchors account.ig_user_id.
func (c *Client) Me(ctx context.Context) (MeResult, error) {
	var result MeResult
	if err := c.get(ctx, "/me?fields=user_id,username,account_type", &result); err != nil {
		return MeResult{}, fmt.Errorf("igapi: Me: %w", err)
	}
	if result.UserID == "" {
		return MeResult{}, fmt.Errorf("igapi: Me: empty user_id in response")
	}
	return result, nil
}
