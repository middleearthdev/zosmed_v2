// Package httpx provides shared HTTP transport utilities for the Zosmed API server.
package httpx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"
)

type requestIDKey struct{}

// RequestID is a middleware that generates a unique request ID per request,
// stores it in the context, and sets the X-Request-ID response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := genRequestID()
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func genRequestID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// RequestIDFromContext retrieves the request ID stored by RequestID middleware.
// Returns an empty string if none was set.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// responseRecorder wraps http.ResponseWriter to capture the HTTP status code
// written by the next handler. Defaults to 200 if WriteHeader is never called.
type responseRecorder struct {
	http.ResponseWriter
	status int
}

func (rr *responseRecorder) WriteHeader(status int) {
	rr.status = status
	rr.ResponseWriter.WriteHeader(status)
}

// Recover is a middleware that catches panics, logs them with slog,
// and responds with a 500 JSON envelope. Prevents server crashes.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				id := RequestIDFromContext(r.Context())
				slog.Error("httpx: panic recovered",
					slog.Any("panic", rec),
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("request_id", id),
				)
				Err(w, http.StatusInternalServerError, "internal_error", "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Logger is a middleware that logs each completed request via slog.
// It records method, path, status, latency, and request ID.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rr := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rr, r)
		slog.Info("httpx: request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", rr.status),
			slog.Duration("latency", time.Since(start)),
			slog.String("request_id", RequestIDFromContext(r.Context())),
		)
	})
}

// AuthStub is a no-op authentication middleware for MVP.
// TODO(auth): replace with JWT/session validation before production.
func AuthStub(next http.Handler) http.Handler {
	return next
}
