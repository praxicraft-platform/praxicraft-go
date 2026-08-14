package praxicraft

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// APIError is raised for client-side / unexpected failures without an HTTP status.
type APIError struct {
	Message string
	ErrCode string
}

func (e *APIError) Error() string {
	if e.ErrCode != "" {
		return fmt.Sprintf("%s (%s)", e.Message, e.ErrCode)
	}
	return e.Message
}

func (e *APIError) Code() string { return e.ErrCode }

// APIConnectionError is raised when the HTTP request could not be completed.
type APIConnectionError struct {
	Message string
}

func (e *APIConnectionError) Error() string {
	if e.Message == "" {
		return "Failed to connect to the Praxicraft API."
	}
	return e.Message
}

func (e *APIConnectionError) Code() string { return "CONNECTION_ERROR" }

// APIStatusError is raised for non-success HTTP responses.
type APIStatusError struct {
	Message      string
	ErrCode      string
	StatusCode   int
	Details      any
	ResponseBody any
	Headers      http.Header
	RequiredPlan string
}

func (e *APIStatusError) Error() string {
	if e.ErrCode != "" {
		return fmt.Sprintf("HTTP %d: %s (%s)", e.StatusCode, e.Message, e.ErrCode)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, e.Message)
}

func (e *APIStatusError) Code() string { return e.ErrCode }

// AuthenticationError is 401 — invalid, missing, or expired API key.
type AuthenticationError struct{ APIStatusError }

// InsufficientScopeError is 403 — missing scope or related denial.
type InsufficientScopeError struct{ APIStatusError }

// NotFoundError is 404.
type NotFoundError struct{ APIStatusError }

// ValidationError is 400-class client/business-rule failure.
type ValidationError struct{ APIStatusError }

// RateLimitError is 429.
type RateLimitError struct {
	APIStatusError
	RetryAfter *float64
}

type publicErrorEnvelope struct {
	Error *struct {
		Code         string          `json:"code"`
		Message      string          `json:"message"`
		Details      json.RawMessage `json:"details"`
		RequiredPlan string          `json:"required_plan"`
	} `json:"error"`
}

func raiseForStatus(statusCode int, body []byte, headers http.Header) error {
	var envelope publicErrorEnvelope
	var parsed any
	_ = json.Unmarshal(body, &parsed)

	code := ""
	message := ""
	var details any
	requiredPlan := ""

	if err := json.Unmarshal(body, &envelope); err == nil && envelope.Error != nil {
		code = envelope.Error.Code
		message = envelope.Error.Message
		requiredPlan = envelope.Error.RequiredPlan
		if len(envelope.Error.Details) > 0 {
			_ = json.Unmarshal(envelope.Error.Details, &details)
		}
	}

	if message == "" {
		trimmed := strings.TrimSpace(string(body))
		if len(trimmed) > 500 {
			trimmed = trimmed[:500]
		}
		if trimmed != "" && !json.Valid(body) {
			message = trimmed
		} else {
			message = fmt.Sprintf("API request failed with status %d.", statusCode)
		}
	}

	base := APIStatusError{
		Message:      message,
		ErrCode:      code,
		StatusCode:   statusCode,
		Details:      details,
		ResponseBody: parsed,
		Headers:      headers.Clone(),
		RequiredPlan: requiredPlan,
	}

	switch {
	case statusCode == 401:
		return &AuthenticationError{APIStatusError: base}
	case statusCode == 403:
		return &InsufficientScopeError{APIStatusError: base}
	case statusCode == 404:
		return &NotFoundError{APIStatusError: base}
	case statusCode == 429:
		return &RateLimitError{APIStatusError: base, RetryAfter: parseRetryAfterSeconds(headers.Get("Retry-After"))}
	case statusCode >= 400 && statusCode < 500:
		return &ValidationError{APIStatusError: base}
	default:
		return &base
	}
}
