package auth

import (
	"crypto/sha256"
	"encoding/hex"
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

// TokenBlacklistStore is the OPTIONAL persistence backend for the logout
// blacklist (P2-10, audit 0926-system). With no store set the blacklist is
// memory-only, exactly as before. With a store set, BlacklistToken writes a
// fingerprint+expiry row and IsTokenBlacklisted falls back to it after the
// in-memory map — a restart can no longer revive a logged-out token.
type TokenBlacklistStore interface {
	Save(tokenID string, expiresAt time.Time) error
	IsRevoked(tokenID string) (bool, error)
	PruneExpired(now time.Time) error
}

var blacklistStoreMu sync.RWMutex
var blacklistStore TokenBlacklistStore

// SetTokenBlacklistStore installs the persistence backend (called once at
// boot from main.go with the DB-backed store).
func SetTokenBlacklistStore(s TokenBlacklistStore) {
	blacklistStoreMu.Lock()
	defer blacklistStoreMu.Unlock()
	blacklistStore = s
}

func currentBlacklistStore() TokenBlacklistStore {
	blacklistStoreMu.RLock()
	defer blacklistStoreMu.RUnlock()
	return blacklistStore
}

// BlacklistStoreEnabled reports whether a persistence backend is installed —
// READ from live state for the boot line (B6).
func BlacklistStoreEnabled() bool {
	return currentBlacklistStore() != nil
}

// TokenFingerprint is the key a revoked token is stored under: the hex SHA-256
// of the token string. The full token is never persisted.
func TokenFingerprint(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// ClockLeeway is the clock skew ValidateJWT forgives on iat, nbf and exp
// (CTO ruling 1790243040753): a token whose iat is more than ClockLeeway
// ahead of the server's clock is refused everywhere; a clock step back of up
// to ClockLeeway costs nothing, a larger one refuses the sessions issued in
// the skipped interval until the clock catches up. The same leeway admits a
// token up to ClockLeeway past its exp — bounded, pinned (api
// TestClockLeewayOnExpAndNbfIsBoundedAtSixtySeconds).
const ClockLeeway = 60 * time.Second

// SetJWTSecret sets the JWT secret key
func SetJWTSecret(secret string) {
	JWTSecret = []byte(secret)
}

// BlacklistToken adds token to blacklist until expiration — its exp PLUS
// ClockLeeway, the last instant ValidateJWT can still admit it (an entry
// dropped at exp would bring a logged-out token back for that last minute).
// With a persistence store set, the fingerprint+expiry is ALSO written to the
// DB (best effort — a write failure logs and the memory entry still holds for
// this process).
func BlacklistToken(token string, exp time.Time) {
	until := exp.Add(ClockLeeway)
	tokenBlacklist.Lock()
	tokenBlacklist.items[token] = until
	tokenBlacklist.Unlock()

	if store := currentBlacklistStore(); store != nil {
		if err := store.Save(TokenFingerprint(token), until); err != nil {
			log.Printf("auth: persist logout revocation failed: fp=%s err=%v", TokenFingerprint(token), err)
		} else {
			// Opportunistic prune (P2-10): expired rows never accumulate
			// without a restart.
			if err := store.PruneExpired(time.Now()); err != nil {
				log.Printf("auth: prune expired revocations failed: err=%v", err)
			}
		}
	}

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

// IsTokenBlacklisted checks if token is in blacklist (auto cleanup on expiration).
// The in-memory map is authoritative for tokens revoked in this process; the
// persistence store (if set) catches tokens revoked before a restart. A store
// READ error degrades to the memory answer with a WARN — a transient DB blip
// must not hard-lock every request out of the API.
func IsTokenBlacklisted(token string) bool {
	tokenBlacklist.Lock()
	if exp, ok := tokenBlacklist.items[token]; ok {
		tokenBlacklist.Unlock()
		if time.Now().After(exp) {
			tokenBlacklist.Lock()
			delete(tokenBlacklist.items, token)
			tokenBlacklist.Unlock()
			return false
		}
		return true
	}
	tokenBlacklist.Unlock()

	if store := currentBlacklistStore(); store != nil {
		revoked, err := store.IsRevoked(TokenFingerprint(token))
		if err != nil {
			log.Printf("auth: blacklist store read failed (%v) — falling back to memory-only", err)
			return false
		}
		if revoked {
			// Cache the store's answer so the next check stays in memory.
			tokenBlacklist.Lock()
			tokenBlacklist.items[token] = time.Now().Add(24 * time.Hour)
			tokenBlacklist.Unlock()
			return true
		}
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

// strictParser decodes every segment with STRICT base64url (M3 red-team M2,
// red-1 #4; CTO ruling 1790231205208): jwt v5's default lenient decoder
// ignores the 2 unused low bits of an HS256 signature's 43rd character, so
// a token had 4 accepted spellings and the logout blacklist — an exact-string
// map — knew only one. Strict decoding refuses non-zero padding bits, so each
// token has exactly one accepted spelling.
//
// It also refuses a token issued in the FUTURE (M3 verifier defect 4; CTO
// ruling 1790243040753): jwt.WithIssuedAt() compares iat with now — present
// only; a token with NO iat passes the parser and is refused by the H2 retire
// rule (RetiredBy) instead. Before, iat was never compared with now, so a
// token stamped ahead of the clock carried an iat AFTER a later password
// change's epoch and survived H2. jwt.WithLeeway(ClockLeeway) forgives 60 s of
// clock step, and jwt v5 applies that ONE leeway to iat, nbf AND exp: a token
// is admitted up to ClockLeeway past its exp, so the logout blacklist holds an
// entry that long too (BlacklistToken).
var strictParser = jwt.NewParser(jwt.WithStrictDecoding(), jwt.WithIssuedAt(), jwt.WithLeeway(ClockLeeway))

// ValidateJWT validates JWT token
func ValidateJWT(tokenString string) (*Claims, error) {
	token, err := strictParser.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
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
