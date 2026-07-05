package httpx

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

// Routes holds the concrete handler functions wired by main.go.
// Using http.HandlerFunc avoids an import cycle between httpx and its sibling
// packages (webhook, commentorder, auth) that themselves import httpx for helpers.
type Routes struct {
	// Meta webhook (called by Meta — no auth)
	WebhookChallenge http.HandlerFunc
	WebhookReceive   http.HandlerFunc

	// Instagram Login connect flow (ADR-002 §3.3). Start now requires a
	// logged-in Zosmed user (ADR-003 AC-9) — mounted behind RequireUser below.
	// Callback stays public: Instagram calls it back directly, and the
	// user's identity travels in the signed state, not a cookie.
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

	// Zosmed login (ADR-003 §4.1). Register/Login/Logout are public;
	// Me requires RequireUser (mounted in the protected group below).
	Register http.HandlerFunc
	Login    http.HandlerFunc
	Logout   http.HandlerFunc
	Me       http.HandlerFunc

	// Onboarding (ADR-003 §4.2) — both require RequireUser.
	PutSegment         http.HandlerFunc
	CompleteOnboarding http.HandlerFunc

	// RequireUser is injected from main.go (it needs a DB-backed Store, which
	// httpx must not depend on directly — hindari import cycle httpx<->auth).
	RequireUser func(http.Handler) http.Handler
}

// NewRouter builds and returns the main chi router.
// Mount order: global middleware → public routes (webhook, connect callback,
// auth) → /api/v1 + /connect/instagram protected group (RequireUser).
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

	// Instagram OAuth callback: public — Instagram calls this back directly,
	// and the user identity travels in the signed state (ADR-002 §3.3 / ADR-003 §6).
	r.Get("/connect/instagram/callback", routes.ConnectCallback)

	// Zosmed auth: register/login/logout are public by definition.
	r.Post("/api/v1/auth/register", routes.Register)
	r.Post("/api/v1/auth/login", routes.Login)
	r.Post("/api/v1/auth/logout", routes.Logout)

	// Protected group: requires a valid "zsid" session (ADR-003 AC-6).
	r.Group(func(r chi.Router) {
		r.Use(routes.RequireUser)

		r.Get("/connect/instagram", routes.ConnectStart)

		r.Get("/api/v1/auth/me", routes.Me)

		r.Put("/api/v1/onboarding/segment", routes.PutSegment)
		r.Post("/api/v1/onboarding/complete", routes.CompleteOnboarding)

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
