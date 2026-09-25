package auth

import (
	"os"
	"path/filepath"
	"testing"
)

// ── DS-105 CENSUS-AUTH [14] (skeptic 4c05158b) ─────────────────────────────
//
// The future-iat census counts a minter only at a jwt.NewWithClaims /
// jwt.New CALL, and the unscoped census counts only GenerateJWT references.
// A &jwt.Token{…} composite literal signed with the exported auth.JWTSecret
// — or jwt.NewWithClaims held as a function value — mints an unscoped USER
// token with any email and any iat, both censuses green, and the real
// ValidateJWT admits it. These two tests plant both shapes in a synthetic
// module and pin that each census COUNTS them (RED before the census sees
// them, GREEN after — one commit per half).

func writeSynthetic(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The admitted minting site's shape: one NewWithClaims call, two now-dated
// iat/nbf sites — the future-iat census's own positive control.
const syntheticSignToken = `package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func signToken() (string, error) {
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		NotBefore: jwt.NewNumericDate(time.Now()),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(nil)
}
`

// The two evading minters: a jwt.Token composite literal signed directly,
// and jwt.NewWithClaims held as a function value and then called.
const syntheticEvadingMinters = `package api

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secret = []byte("synthetic")

// mintLiteral constructs the token by hand — no NewWithClaims call anywhere.
func mintLiteral(iat time.Time) (string, error) {
	rc := jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now())}
	tok := &jwt.Token{
		Header: map[string]any{"typ": "JWT", "alg": "HS256"},
		Claims: rc,
		Method: jwt.SigningMethodHS256,
	}
	return tok.SignedString(secret)
}

// mk holds the constructor as a value; the call is mk(…), invisible to a
// CallExpr census over jwt.NewWithClaims.
var mk = jwt.NewWithClaims

func mintFuncValue() (string, error) {
	rc := jwt.RegisteredClaims{IssuedAt: jwt.NewNumericDate(time.Now())}
	return mk(jwt.SigningMethodHS256, rc).SignedString(secret)
}
`

func TestFutureIatCensusCountsTokenLiteralAndFunctionValueMinters(t *testing.T) {
	root := t.TempDir()
	writeSynthetic(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	writeSynthetic(t, root, "auth/auth.go", syntheticSignToken)
	writeSynthetic(t, root, "api/zz_mint.go", syntheticEvadingMinters)
	minters, iatSites, offenders, _, err := futureIatCensus(root)
	if err != nil {
		t.Fatal(err)
	}
	if minters["auth/auth.go"] != 1 {
		t.Fatalf("positive control: synthetic signToken = %d minter, want 1", minters["auth/auth.go"])
	}
	if minters["api/zz_mint.go"] != 2 {
		t.Fatalf("the token literal and the function-value minter are invisible to the future-iat census: minters = %v", minters)
	}
	if iatSites["auth/auth.go"] != 2 {
		t.Fatalf("positive control: synthetic signToken iatSites = %d, want 2", iatSites["auth/auth.go"])
	}
	if len(offenders) != 0 {
		t.Fatalf("unexpected non-now iat/nbf offenders: %v", offenders)
	}
}

const syntheticGenerateJWT = `package auth

import "github.com/golang-jwt/jwt/v5"

var JWTSecret []byte

type Claims struct {
	UserID string
	Email  string
}

func GenerateJWT(userID, email string) (string, error) {
	claims := Claims{UserID: userID, Email: email}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(JWTSecret)
}
`

const syntheticLoginRegister = `package api

import "nofx/auth"

func login() (string, error)    { return auth.GenerateJWT("u", "e") }

func register() (string, error) { return auth.GenerateJWT("u", "e") }
`

func TestUnscopedMintCensusCountsTokenLiteralAndFunctionValueMinters(t *testing.T) {
	root := t.TempDir()
	writeSynthetic(t, root, "go.mod", "module nofx\n\ngo 1.25\n")
	writeSynthetic(t, root, "auth/auth.go", syntheticGenerateJWT)
	writeSynthetic(t, root, "api/handler_user.go", syntheticLoginRegister)
	writeSynthetic(t, root, "api/zz_mint.go", syntheticEvadingMinters)
	seen, _, err := unscopedMintSites(root)
	if err != nil {
		t.Fatal(err)
	}
	if seen["api/handler_user.go"] != 2 {
		t.Fatalf("positive control: synthetic login+register = %d unscoped mints, want 2", seen["api/handler_user.go"])
	}
	if seen["api/zz_mint.go"] != 2 {
		t.Fatalf("the token literal and the function-value minter are invisible to the unscoped census: seen = %v", seen)
	}
}
