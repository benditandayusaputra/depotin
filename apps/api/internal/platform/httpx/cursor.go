package httpx

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const (
	DefaultLimit = 25
	MaxLimit     = 100
)

type Cursor struct {
	At time.Time
	ID uuid.UUID
}

func EncodeCursor(at time.Time, id uuid.UUID) string {
	raw := strconv.FormatInt(at.UnixMicro(), 10) + ":" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func DecodeCursor(s string) (Cursor, bool) {
	if s == "" {
		return Cursor{}, true
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, false
	}
	parts := strings.SplitN(string(raw), ":", 2)
	if len(parts) != 2 {
		return Cursor{}, false
	}
	micros, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return Cursor{}, false
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return Cursor{}, false
	}
	return Cursor{At: time.UnixMicro(micros).UTC(), ID: id}, true
}

type Page struct {
	Limit     int
	Cursor    Cursor
	HasCursor bool
}

func ParsePage(c fiber.Ctx) (Page, error) {
	limit := DefaultLimit
	if raw := c.Query("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return Page{}, Validation(map[string]string{"limit": "Harus angka positif."})
		}
		limit = min(n, MaxLimit)
	}
	cursor, ok := DecodeCursor(c.Query("cursor"))
	if !ok {
		return Page{}, Validation(map[string]string{"cursor": "Kursor tidak valid."})
	}
	return Page{Limit: limit, Cursor: cursor, HasCursor: c.Query("cursor") != ""}, nil
}
