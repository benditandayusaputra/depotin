package httpx

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

type envelope struct {
	Data any   `json:"data"`
	Meta *meta `json:"meta,omitempty"`
}

type meta struct {
	NextCursor *string `json:"next_cursor"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func OK(c fiber.Ctx, data any) error {
	return c.Status(http.StatusOK).JSON(envelope{Data: data})
}

func Created(c fiber.Ctx, data any) error {
	return c.Status(http.StatusCreated).JSON(envelope{Data: data})
}

func List(c fiber.Ctx, data any, nextCursor string) error {
	m := &meta{}
	if nextCursor != "" {
		m.NextCursor = &nextCursor
	}
	return c.Status(http.StatusOK).JSON(envelope{Data: data, Meta: m})
}

func Fail(c fiber.Ctx, err error) error {
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) {
			apiErr = fromFiberError(fiberErr)
		} else {
			apiErr = Internal(err)
		}
	}
	if apiErr.Status >= http.StatusInternalServerError {
		slog.ErrorContext(c.Context(), "permintaan gagal",
			"request_id", RequestID(c),
			"route", routePattern(c),
			"err", apiErr.Error(),
		)
	}
	if apiErr.RetryAfter > 0 {
		c.Set(fiber.HeaderRetryAfter, strconv.Itoa(apiErr.RetryAfter))
	}
	return c.Status(apiErr.Status).JSON(errorEnvelope{Error: errorBody{
		Code:    apiErr.Code,
		Message: apiErr.Message,
		Fields:  apiErr.Fields,
	}})
}

func fromFiberError(err *fiber.Error) *Error {
	switch err.Code {
	case http.StatusNotFound:
		return NotFound("Alamat tidak ditemukan.")
	case http.StatusMethodNotAllowed:
		return &Error{Status: http.StatusMethodNotAllowed, Code: CodeNotFound, Message: "Metode tidak didukung."}
	case http.StatusRequestEntityTooLarge:
		return ValidationMessage("Data yang dikirim terlalu besar.")
	case http.StatusRequestTimeout:
		return &Error{Status: http.StatusRequestTimeout, Code: CodeUnavailable, Message: "Permintaan terlalu lama. Coba lagi."}
	}
	if err.Code >= http.StatusInternalServerError {
		return Internal(err)
	}
	return &Error{Status: err.Code, Code: CodeValidationFailed, Message: "Permintaan tidak bisa diproses."}
}

func ErrorHandler(c fiber.Ctx, err error) error {
	return Fail(c, err)
}

func routePattern(c fiber.Ctx) string {
	if r := c.Route(); r != nil {
		return r.Path
	}
	return ""
}
