package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/gofiber/fiber/v3"
)

const MaxBodyBytes = 64 * 1024

func DecodeJSON(c fiber.Ctx, dst any) error {
	if !strings.HasPrefix(c.Get(fiber.HeaderContentType), fiber.MIMEApplicationJSON) {
		return ValidationMessage("Content-Type harus application/json.")
	}
	dec := json.NewDecoder(bytes.NewReader(c.Body()))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return ValidationMessage("Format data tidak dikenali.").WithCause(err)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ValidationMessage("Format data tidak dikenali.")
	}
	return nil
}

func DecodeAndValidate(c fiber.Ctx, dst any) error {
	if err := DecodeJSON(c, dst); err != nil {
		return err
	}
	return Validate(dst)
}
