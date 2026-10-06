package sessions

import (
	"testing"
)

func TestHashTokenStable(t *testing.T) {
	first := HashToken("abc123")
	second := HashToken("abc123")
	if first != second {
		t.Fatalf("expected stable hash, got %q and %q", first, second)
	}
	if len(first) != 64 {
		t.Fatalf("expected 64 char hash, got %q", first)
	}
}

func TestHashTokenDiffers(t *testing.T) {
	if HashToken("token-a") == HashToken("token-b") {
		t.Fatalf("expected different tokens to hash differently")
	}
}

func TestHashTokenEmpty(t *testing.T) {
	if HashToken("") == HashToken("x") {
		t.Fatalf("expected empty token hash to differ")
	}
}
