package crypto

import (
	"bytes"
	"strings"
	"testing"
)

func TestHashAndVerifyPassword(t *testing.T) {
	h, err := HashPassword("kata-sandi-rahasia")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected phc prefix: %s", h)
	}
	ok, err := VerifyPassword(h, "kata-sandi-rahasia")
	if err != nil || !ok {
		t.Fatalf("expected match, ok=%v err=%v", ok, err)
	}
	ok, err = VerifyPassword(h, "kata-sandi-salah")
	if err != nil || ok {
		t.Fatalf("expected mismatch, ok=%v err=%v", ok, err)
	}
	if _, err := VerifyPassword("$bcrypt$abc", "x"); err == nil {
		t.Fatal("expected format error")
	}
}

func TestDummyHashVerifiesNothingUseful(t *testing.T) {
	if DummyHash() == "" {
		t.Fatal("dummy hash empty")
	}
	ok, err := VerifyPassword(DummyHash(), "")
	if err != nil || ok {
		t.Fatalf("dummy should not match empty, ok=%v err=%v", ok, err)
	}
}

func TestHashTokenIsDeterministic(t *testing.T) {
	if !bytes.Equal(HashToken("abc"), HashToken("abc")) || bytes.Equal(HashToken("abc"), HashToken("abd")) {
		t.Fatal("hash token mismatch")
	}
	if len(HashToken("x")) != 32 {
		t.Fatal("expected 32 bytes")
	}
}

func TestSealerRoundTrip(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	s, err := NewSealer(key)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := s.Seal([]byte("token-rahasia"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := s.Open(sealed)
	if err != nil || string(plain) != "token-rahasia" {
		t.Fatalf("round trip failed: %q %v", plain, err)
	}
	sealed[len(sealed)-1] ^= 1
	if _, err := s.Open(sealed); err == nil {
		t.Fatal("tampered data should fail")
	}
	if _, err := NewSealer([]byte("short")); err == nil {
		t.Fatal("short key should fail")
	}
}

func TestConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("a", "a") || ConstantTimeEqual("a", "b") || ConstantTimeEqual("a", "aa") {
		t.Fatal("compare broken")
	}
}
