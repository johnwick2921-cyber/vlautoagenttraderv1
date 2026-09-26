package auth

import (
	"strings"
	"testing"
)

// M3 red-team M2 (red-1 #4): ValidateJWT decodes base64url STRICTLY, so each
// token has exactly one accepted spelling — the logout blacklist (an
// exact-string map) cannot be dodged by flipping the unused low bits of the
// signature's last character.
func TestValidateJWTRefusesNonCanonicalSignatureSpellings(t *testing.T) {
	prev := JWTSecret
	SetJWTSecret("strict-decoding-test-secret-0123456789")
	t.Cleanup(func() { JWTSecret = prev })
	tok, err := GenerateJWT("u-strict", "strict@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateJWT(tok); err != nil {
		t.Fatalf("control: the canonical token does not validate: %v", err)
	}
	const alpha = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	v := strings.IndexByte(alpha, tok[len(tok)-1])
	n := 0
	for low := 0; low < 4; low++ {
		nv := (v &^ 3) | low
		if nv == v {
			continue
		}
		n++
		if _, err := ValidateJWT(tok[:len(tok)-1] + string(alpha[nv])); err == nil {
			t.Errorf("ValidateJWT accepted a non-canonical respelling of the signature (last char %q → %q)", alpha[v], alpha[nv])
		}
	}
	if n != 3 {
		t.Fatalf("control: %d respellings, want 3", n)
	}
}
