package crypto

import (
	"errors"
	"testing"
)

func testKey() [32]byte {
	var key [32]byte
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	encrypted, err := Encrypt("gho-secret-value", testKey())
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if encrypted == "gho-secret-value" {
		t.Fatalf("expected ciphertext to differ from plaintext")
	}
	decrypted, err := Decrypt(encrypted, testKey())
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != "gho-secret-value" {
		t.Fatalf("expected roundtrip, got %q", decrypted)
	}
}

func TestEncryptRandomized(t *testing.T) {
	first, err := Encrypt("same", testKey())
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	second, err := Encrypt("same", testKey())
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if first == second {
		t.Fatalf("expected randomized nonces to differ")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	encrypted, err := Encrypt("gho-secret-value", testKey())
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if _, err := Decrypt(encrypted, [32]byte{}); !errors.Is(err, ErrBadCiphertext) {
		t.Fatalf("expected ErrBadCiphertext, got %v", err)
	}
}

func TestDecryptCorrupt(t *testing.T) {
	for _, cipherHex := range []string{"not-hex!!", "abcd", ""} {
		if _, err := Decrypt(cipherHex, testKey()); !errors.Is(err, ErrBadCiphertext) {
			t.Fatalf("expected ErrBadCiphertext for %q, got %v", cipherHex, err)
		}
	}
}
