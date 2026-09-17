// Package httpresponse implements the response envelope shared by every
// REST endpoint, per docs/spec/09-api-contracts.md.
package httpresponse

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Envelope struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorBody  `json:"error,omitempty"`
	RequestID string      `json:"requestId"`
	Timestamp time.Time   `json:"timestamp"`
}

type ErrorBody struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Known error codes from docs/spec/09-api-contracts.md, plus
// AUTH_INVALID_CREDENTIALS for login failures (the spec list is the
// baseline, not exhaustive).
//
// #nosec G101 -- public API error-code identifiers, not credentials.
const (
	ErrAuthInvalidToken       = "AUTH_INVALID_TOKEN"       // #nosec G101
	ErrAuthInvalidCredentials = "AUTH_INVALID_CREDENTIALS" // #nosec G101
	ErrAuthExpired            = "AUTH_EXPIRED"
	ErrAccessDenied           = "ACCESS_DENIED"
	ErrResourceNotFound       = "RESOURCE_NOT_FOUND"
	ErrRateLimited            = "RATE_LIMITED"
	// ErrQuotaExceeded: a storage quota or the posts-per-hour limit (audit A-07).
	ErrQuotaExceeded = "QUOTA_EXCEEDED"
	// ErrE2EERequired: a private chat accepts text only end-to-end encrypted.
	ErrE2EERequired     = "E2EE_REQUIRED"
	ErrValidationFailed = "VALIDATION_FAILED"
	ErrInternal         = "INTERNAL_ERROR"
	// Display names (migration 46) get their own codes: the sign-up and rename
	// forms have to tell "taken" from "not allowed" without parsing a message.
	ErrDisplayNameTaken   = "DISPLAY_NAME_TAKEN"
	ErrDisplayNameInvalid = "DISPLAY_NAME_INVALID"
)

func requestID(r *http.Request) string {
	if id := r.Header.Get("X-Request-Id"); id != "" {
		return id
	}
	return uuid.NewString()
}

// OK writes a successful response envelope.
func OK(w http.ResponseWriter, r *http.Request, status int, data interface{}) {
	write(w, r, status, Envelope{Success: true, Data: data})
}

// Fail writes an error response envelope. Internal error details are never
// exposed to the client — only code and message are, per the security
// spec's requirement to hide internal errors.
func Fail(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	write(w, r, status, Envelope{Success: false, Error: &ErrorBody{Code: code, Message: message}})
}

func write(w http.ResponseWriter, r *http.Request, status int, env Envelope) {
	env.RequestID = requestID(r)
	env.Timestamp = time.Now().UTC()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}
