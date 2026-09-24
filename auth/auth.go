package auth

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JWTSecret is the JWT secret key, will be dynamically set from config
var JWTSecret []byte

// tokenBlacklist for logged out tokens (memory only, cleaned by expiration time)
var tokenBlacklist = struct {
	sync.RWMutex
	items map[string]time.Time
}{items: make(map[string]time.Time)}

// maxBlacklistEntries is the maximum capacity threshold for blacklist
const maxBlacklistEntries = 100_000

// SetJWTSecret sets the JWT secret key
func SetJWTSecret(secret string) {
	JWTSecret = []byte(secret)
}

// BlacklistToken adds token to blacklist until expiration
func BlacklistToken(token string, exp time.Time) {
	tokenBlacklist.Lock()
	defer tokenBlacklist.Unlock()
	tokenBlacklist.items[token] = exp

	// If exceeds capacity threshold, perform expired cleanup; if still over limit, log warning
	if len(tokenBlacklist.items) > maxBlacklistEntries {
		now := time.Now()
		for t, e := range tokenBlacklist.items {
			if now.After(e) {
				delete(tokenBlacklist.items, t)
			}
		}
		if len(tokenBlacklist.items) > maxBlacklistEntries {
			log.Printf("auth: token blacklist size (%d) exceeds limit (%d) after sweep; consider reducing JWT TTL or using a shared persistent store",
				len(tokenBlacklist.items), maxBlacklistEntries)
		}
	}
}

// IsTokenBlacklisted checks if token is in blacklist (auto cleanup on expiration)
func IsTokenBlacklisted(token string) bool {
	tokenBlacklist.Lock()
	defer tokenBlacklist.Unlock()
	if exp, ok := tokenBlacklist.items[token]; ok {
		if time.Now().After(exp) {
			delete(tokenBlacklist.items, token)
			return false
		}
		return true
	}
	return false
}

// Claims represents JWT claims
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	// Scope marks a MACHINE token (M3 red-team H1): one minted for a process
	// rather than by a user proving their password — the Telegram bot
	// (ScopeTelegram), cmd/gate-jwt (ScopeGateJWT). The API denies machine
	// tokens by default on the credential, Telegram-config and update routes
	// (api/credential_guard.go). A user token has NO scope key at all
	// (omitempty), so a login token's claim set is byte-identical to before.
	Scope string `json:"scope,omitempty"`
	jwt.RegisteredClaims
}

// BotInternalEmail is the email the Telegram bot's token has always carried.
// A token with it is a machine token even with no scope claim (a bot token
// minted by an older binary — fail closed).
const BotInternalEmail = "bot@internal"

// Machine-token scopes. Any non-empty scope is a machine scope; these are the
// ones this build mints.
const (
	ScopeTelegram = "telegram"
	ScopeGateJWT  = "gate-jwt"
)

// IsMachine reports whether the token is a machine token: it carries any
// scope, or the bot's email. nil claims are treated as a machine token (a
// caller that could not read the claims must not be granted a user's rights).
func (c *Claims) IsMachine() bool {
	if c == nil {
		return true
	}
	return c.Scope != "" || strings.EqualFold(strings.TrimSpace(c.Email), BotInternalEmail)
}

// HashPassword hashes the password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword verifies the password
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateJWT generates a USER token (no scope). Callers: the login and
// register handlers ONLY — a census test (auth/mint_census_test.go) pins it;
// every other minting site uses GenerateScopedJWT.
func GenerateJWT(userID, email string) (string, error) {
	return signToken(userID, email, "")
}

// GenerateScopedJWT generates a MACHINE token carrying scope (non-empty).
func GenerateScopedJWT(userID, email, scope string) (string, error) {
	if strings.TrimSpace(scope) == "" {
		return "", fmt.Errorf("auth: a machine token needs a non-empty scope")
	}
	return signToken(userID, email, scope)
}

func signToken(userID, email, scope string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		Scope:  scope,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // Expires in 24 hours
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "nofxAI",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ValidateJWT validates JWT token
func ValidateJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
