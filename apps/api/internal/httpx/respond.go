// Package httpx provides shared HTTP transport utilities for the Zosmed API server.
package httpx

import (
	"encoding/json"
	"net/http"
)

// Envelope is the standard JSON response wrapper for all API responses (ADR-001 §4).
//
//	Success: {"data": <T>, "error": null}
//	Failure: {"data": null, "error": {"code": "...", "message": "..."}}
type Envelope struct {
	Data  any       `json:"data"`
	Error *APIError `json:"error"`
}

// APIError is the structured error payload in the envelope.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JSON writes a JSON success response with the given HTTP status code.
func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: data, Error: nil})
}

// Err writes a JSON error response.
func Err(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope{Data: nil, Error: &APIError{Code: code, Message: message}})
}
