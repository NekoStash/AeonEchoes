package respond

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

type requestIDContextKey struct{}

var requestIDSequence uint64

type metaDTO struct {
	RequestID string `json:"request_id"`
}

type pageDTO struct {
	Count int `json:"count"`
	Limit int `json:"limit,omitempty"`
}

type envelopeDTO struct {
	Data any      `json:"data"`
	Page *pageDTO `json:"page,omitempty"`
	Meta metaDTO  `json:"meta"`
}

type errorEnvelopeDTO struct {
	Error errorDTO `json:"error"`
}

type errorDTO struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Status    int    `json:"status"`
	RequestID string `json:"request_id"`
	Details   any    `json:"details,omitempty"`
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = fmt.Sprintf("req_%d_%d", time.Now().UTC().UnixNano(), atomic.AddUint64(&requestIDSequence, 1))
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDContextKey{}, requestID)))
	})
}

func RequestID(r *http.Request) string {
	if r == nil {
		return ""
	}
	if value, ok := r.Context().Value(requestIDContextKey{}).(string); ok && value != "" {
		return value
	}
	if header := strings.TrimSpace(r.Header.Get("X-Request-ID")); header != "" {
		return header
	}
	return fmt.Sprintf("req_%d_%d", time.Now().UTC().UnixNano(), atomic.AddUint64(&requestIDSequence, 1))
}

func Decode(w http.ResponseWriter, r *http.Request, out any) bool {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, http.ErrBodyReadAfterClose) {
			Error(w, r, http.StatusBadRequest, "invalid_request_body", "request body is not readable", map[string]any{"cause": err.Error()})
			return false
		}
		Error(w, r, http.StatusBadRequest, "invalid_json", "invalid JSON request body", map[string]any{"cause": err.Error()})
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			Error(w, r, http.StatusBadRequest, "invalid_json", "request body must contain exactly one JSON document", nil)
			return false
		}
		Error(w, r, http.StatusBadRequest, "invalid_json", "invalid trailing JSON request body", map[string]any{"cause": err.Error()})
		return false
	}
	return true
}

func Data(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeEnvelope(w, status, envelopeDTO{Data: data, Meta: metaDTO{RequestID: RequestID(r)}})
}

func List(w http.ResponseWriter, r *http.Request, status int, data any, count int, limit int) {
	writeEnvelope(w, status, envelopeDTO{Data: data, Page: &pageDTO{Count: count, Limit: limit}, Meta: metaDTO{RequestID: RequestID(r)}})
}

func writeEnvelope(w http.ResponseWriter, status int, envelope envelopeDTO) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope); err != nil {
		slog.Default().Error("encode v1 HTTP response failed", "error", err)
	}
}

func ErrorFromErr(w http.ResponseWriter, r *http.Request, status int, err error) {
	message := "request failed"
	if err != nil {
		message = err.Error()
	}
	Error(w, r, status, ErrorCodeForStatus(status), message, nil)
}

func Error(w http.ResponseWriter, r *http.Request, status int, code string, message string, details any) {
	if strings.TrimSpace(code) == "" {
		code = ErrorCodeForStatus(status)
	}
	if strings.TrimSpace(message) == "" {
		message = http.StatusText(status)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	payload := errorEnvelopeDTO{Error: errorDTO{Code: code, Message: message, Status: status, RequestID: RequestID(r), Details: details}}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Default().Error("encode v1 HTTP error response failed", "error", err)
	}
}

func ErrorCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusConflict:
		return "conflict"
	case http.StatusServiceUnavailable:
		return "service_unavailable"
	case http.StatusBadGateway:
		return "bad_gateway"
	default:
		if status >= 500 {
			return "internal_error"
		}
		return "request_error"
	}
}
