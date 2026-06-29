// Package commentorder provides the HTTP transport for the Comment-to-Order
// screen (ADR-001 §4.2). It reads from dbgen (sqlc-generated) and maps to
// JSON DTOs that are aligned with packages/types/src/comment-order.ts.
//
// SoC (§12a-3): this package reads normalised DB data — it does NOT call the
// Instagram Graph API, modify reservations in bulk, or import seller kit logic.
// Business rules (keep-code detection, state machine) live in apps/worker.
package commentorder

import "time"

// IncomingCommentDTO is one row in the live comment feed panel.
// JSON field names match packages/types/src/comment-order.ts (camelCase).
type IncomingCommentDTO struct {
	ID          string  `json:"id"`
	User        string  `json:"user"`        // handle without '@'
	Text        string  `json:"text"`
	Ago         string  `json:"ago"`         // server-rendered relative time (e.g. "5 mnt lalu")
	MatchedCode *string `json:"matchedCode"` // null when no keep code was detected
	Reserved    bool    `json:"reserved"`    // true when a non-expired reservation exists for this comment
	Duplicate   bool    `json:"duplicate"`   // true when comment was an ingest duplicate (see handler notes)
}

// ReservationDTO is one row in the reservation panel.
type ReservationDTO struct {
	ID             string    `json:"id"`
	Code           string    `json:"code"`
	BuyerHandle    string    `json:"buyerHandle"`
	Product        string    `json:"product"`
	PriceLabel     string    `json:"priceLabel"`     // formatted: "Rp 189rb", "Rp 2jt"
	Status         string    `json:"status"`         // canonical: reserved|waiting-pay|closed-wa|expired-released
	CountdownLabel string    `json:"countdownLabel"` // "—" (live), "✓ closed", "— released"
	ExpiresAt      time.Time `json:"expiresAt"`      // FE uses this for live countdown when status is active
}

// CatalogProductDTO is one row in the product stock panel.
type CatalogProductDTO struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	StockLeft  int32  `json:"stockLeft"`
	StockTotal int32  `json:"stockTotal"`
}

// CommentOrderStatDTO is one stat tile (e.g. "total detected", "reserved now").
type CommentOrderStatDTO struct {
	Key   string `json:"key"`   // canonical: "code-detected", "reserved-now", "closed-wa", "expired"
	Label string `json:"label"` // display label in Bahasa Indonesia
	Value string `json:"value"` // formatted integer
}

// CommentOrderResponse is the aggregate response for GET /api/v1/comment-order.
// One fetch covers all panels in the Comment-to-Order screen (ADR-001 §4.2).
type CommentOrderResponse struct {
	PostCommentsLabel string               `json:"postCommentsLabel"` // e.g. "1.234 komentar"
	Comments          []IncomingCommentDTO `json:"comments"`
	Stats             []CommentOrderStatDTO `json:"stats"`
	Reservations      []ReservationDTO     `json:"reservations"`
	Products          []CatalogProductDTO  `json:"products"`
}

// SettingsDTO is the shape for GET/PUT /api/v1/comment-order/settings.
type SettingsDTO struct {
	Keywords      []string `json:"keywords"`
	HoldSeconds   int32    `json:"holdSeconds"`
	ReplyTemplate string   `json:"replyTemplate"`
}
