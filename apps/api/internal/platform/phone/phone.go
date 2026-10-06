package phone

import (
	"errors"
	"strings"
)

var ErrInvalid = errors.New("nomor HP tidak valid")

const (
	minDigits = 9
	maxDigits = 15
)

func Normalize(raw string) (string, error) {
	var digits strings.Builder
	for _, r := range raw {
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	switch {
	case strings.HasPrefix(d, "62"):
	case strings.HasPrefix(d, "0"):
		d = "62" + d[1:]
	case strings.HasPrefix(d, "8"):
		d = "62" + d
	default:
		return "", ErrInvalid
	}
	if len(d) < minDigits || len(d) > maxDigits {
		return "", ErrInvalid
	}
	return d, nil
}

func Mask(normalized string) string {
	if len(normalized) <= 4 {
		return normalized
	}
	return strings.Repeat("•", len(normalized)-4) + normalized[len(normalized)-4:]
}
