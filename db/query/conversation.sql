-- Window/last-interaction store (ADR-006 §4.2/§5.3). Postgres, not Redis — the
-- 24h messaging window is compliance-critical state (§4c), not a fana counter.

-- name: UpsertConversationInteraction :one
-- Buka/refresh window. GREATEST menjaga monoton bila event tiba out-of-order.
INSERT INTO conversation (account_id, contact_ig_user_id, last_interaction_at, last_source)
VALUES (@account_id, @contact_ig_user_id, @last_interaction_at, @last_source)
ON CONFLICT (account_id, contact_ig_user_id) DO UPDATE SET
    last_interaction_at = GREATEST(conversation.last_interaction_at, EXCLUDED.last_interaction_at),
    last_source         = EXCLUDED.last_source,
    updated_at          = now()
RETURNING *;

-- name: GetConversation :one
SELECT * FROM conversation WHERE account_id = @account_id AND contact_ig_user_id = @contact_ig_user_id;
