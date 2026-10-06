package httpx

import (
	"fmt"
	"net/http"
	"strings"
)

const (
	CodeValidationFailed  = "validation_failed"
	CodeUnauthenticated   = "unauthenticated"
	CodeForbidden         = "forbidden"
	CodeNotFound          = "not_found"
	CodeConflict          = "conflict"
	CodeInvalidTransition = "invalid_transition"
	CodeRateLimited       = "rate_limited"
	CodeInternal          = "internal"
	CodeUnavailable       = "unavailable"
)

type Error struct {
	Status     int
	Code       string
	Message    string
	Fields     map[string]string
	RetryAfter int
	cause      error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.cause)
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) WithCause(err error) *Error {
	clone := *e
	clone.cause = err
	return &clone
}

func Validation(fields map[string]string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: CodeValidationFailed, Message: "Data yang dikirim belum lengkap atau tidak valid.", Fields: fields}
}

func ValidationMessage(message string) *Error {
	return &Error{Status: http.StatusUnprocessableEntity, Code: CodeValidationFailed, Message: message}
}

func Unauthenticated() *Error {
	return &Error{Status: http.StatusUnauthorized, Code: CodeUnauthenticated, Message: "Silakan masuk dulu."}
}

func Forbidden() *Error {
	return &Error{Status: http.StatusForbidden, Code: CodeForbidden, Message: "Akses ditolak."}
}

func NotFound(message string) *Error {
	if message == "" {
		message = "Data tidak ditemukan."
	}
	return &Error{Status: http.StatusNotFound, Code: CodeNotFound, Message: message}
}

func Conflict(message string) *Error {
	return &Error{Status: http.StatusConflict, Code: CodeConflict, Message: message}
}

func InvalidTransition(message string) *Error {
	if message == "" {
		message = "Status pesanan sudah berubah. Muat ulang halaman."
	}
	return &Error{Status: http.StatusConflict, Code: CodeInvalidTransition, Message: message}
}

func RateLimited(retryAfterSeconds int) *Error {
	return &Error{Status: http.StatusTooManyRequests, Code: CodeRateLimited, Message: "Terlalu banyak permintaan. Coba lagi sebentar.", RetryAfter: retryAfterSeconds}
}

func Internal(err error) *Error {
	return &Error{Status: http.StatusInternalServerError, Code: CodeInternal, Message: "Terjadi gangguan di server. Coba lagi.", cause: err}
}

func Unavailable(err error) *Error {
	return &Error{Status: http.StatusServiceUnavailable, Code: CodeUnavailable, Message: "Layanan sedang tidak siap. Coba lagi sebentar.", cause: err}
}

func (e *Error) WithMessage(err error) *Error {
	clone := *e
	clone.Message = capitalize(err.Error()) + "."
	return &clone
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
