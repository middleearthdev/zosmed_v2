package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Routes holds the concrete handler functions wired by main.go.
// Using http.HandlerFunc avoids an import cycle between httpx and its sibling
// packages (webhook, commentorder) that themselves import httpx for helpers.
type Routes struct {
	// Meta webhook (called by Meta — no auth)
	WebhookChallenge http.HandlerFunc
	WebhookReceive   http.HandlerFunc

	// Instagram Login connect flow (ADR-002 §3.3). Both are public: Start
	// only redirects (no secrets exposed), and Callback is called back by
	// Instagram itself; the signed state param is the CSRF protection for MVP.
	ConnectStart    http.HandlerFunc
	ConnectCallback http.HandlerFunc

	// Comment-to-Order screen
	GetCommentOrder http.HandlerFunc

	// Reservation actions
	GetReservation   http.HandlerFunc
	CloseReservation http.HandlerFunc

	// Keyword settings
	GetSettings http.HandlerFunc
	PutSettings http.HandlerFunc
}

// NewRouter builds and returns the main chi router.
// Mount order: global middleware → webhook routes → /api/v1 group (auth-stubbed).
func NewRouter(routes Routes) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack.
	r.Use(RequestID)
	r.Use(Recover)
	r.Use(Logger)
	r.Use(chimw.StripSlashes) // normalise trailing slashes

	// Webhook endpoints: called by Meta, must not have auth middleware.
	r.Get("/webhooks/meta", routes.WebhookChallenge)
	r.Post("/webhooks/meta", routes.WebhookReceive)

	// Instagram Login connect flow: public, protected by signed state (ADR-002 §3.3).
	r.Get("/connect/instagram", routes.ConnectStart)
	r.Get("/connect/instagram/callback", routes.ConnectCallback)

	// API v1 group: auth stub for MVP.
	r.Group(func(r chi.Router) {
		r.Use(AuthStub)

		r.Get("/api/v1/comment-order", routes.GetCommentOrder)

		// /reservations/{id} routes — note the sub-route first so chi
		// does not swallow /close as a dynamic segment.
		r.Post("/api/v1/reservations/{id}/close", routes.CloseReservation)
		r.Get("/api/v1/reservations/{id}", routes.GetReservation)

		r.Get("/api/v1/comment-order/settings", routes.GetSettings)
		r.Put("/api/v1/comment-order/settings", routes.PutSettings)
	})

	return r
}
