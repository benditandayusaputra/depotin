package idgen

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

const base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func NewID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.New()
	}
	return id
}

func RandomToken(byteLen int) (string, error) {
	raw := make([]byte, byteLen)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("baca acak: %w", err)
	}
	return encodeBase62(raw), nil
}

func RandomBytes(n int) ([]byte, error) {
	raw := make([]byte, n)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("baca acak: %w", err)
	}
	return raw, nil
}

func OrderCode(day time.Time, seq int) string {
	return fmt.Sprintf("DP-%s-%03d", day.Format("060102"), seq)
}

func encodeBase62(raw []byte) string {
	num := new(big.Int).SetBytes(raw)
	base := big.NewInt(62)
	mod := new(big.Int)
	var out []byte
	for num.Sign() > 0 {
		num.DivMod(num, base, mod)
		out = append(out, base62Alphabet[mod.Int64()])
	}
	for i := 0; i < len(raw) && raw[i] == 0; i++ {
		out = append(out, base62Alphabet[0])
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return string(out)
}
