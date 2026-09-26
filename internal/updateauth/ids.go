package updateauth

import (
	"crypto/rand"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"strings"
)

// Length caps (bytes).
const (
	MaxReleaseIDLen = 64
	MaxJobIDLen     = 64
	MaxUserIDLen    = 128
	MaxEmailLen     = 254
)

var (
	// A release id is an IDENTIFIER, never a URL, a path or a command:
	// ASCII letters/digits first, then letters/digits/'.'/'_'/'-'. No '/',
	// '\\', ':', '%', '|', NUL, whitespace; no ".." anywhere.
	releaseIDRe = regexp.MustCompile(`^[0-9A-Za-z][0-9A-Za-z._-]{0,63}$`)
	// A job id: lowercase letters/digits/'-', 8..64, alnum first.
	jobIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{7,63}$`)
)

// ValidReleaseID reports whether s is a well-formed release id. Every
// character outside the allow-list — including the MAC field separator '|' —
// is refused, so release_id|job_id|expires_at can never be re-framed.
func ValidReleaseID(s string) bool {
	return len(s) <= MaxReleaseIDLen && releaseIDRe.MatchString(s) &&
		!strings.Contains(s, "..") && filepath.IsLocal(s)
}

// ValidJobID reports whether s is a well-formed job id (see jobIDRe).
func ValidJobID(s string) bool {
	return len(s) <= MaxJobIDLen && jobIDRe.MatchString(s)
}

// validUserID: 1..128 printable ASCII, no space, no '|'.
func validUserID(s string) bool {
	if s == "" || len(s) > MaxUserIDLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= ' ' || c >= 0x7f || c == '|' {
			return false
		}
	}
	return true
}

// validEmail: 3..254 printable ASCII, no space, exactly one '@' with text on
// both sides. (Enrollment compares it EXACTLY to users.email — the same
// predicate login uses — so this only refuses the unprintable.)
func validEmail(s string) bool {
	if len(s) < 3 || len(s) > MaxEmailLen {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= ' ' || c >= 0x7f {
			return false
		}
	}
	at := strings.IndexByte(s, '@')
	return at > 0 && at < len(s)-1 && strings.Count(s, "@") == 1
}

// NewJobID returns a fresh random job id (32 lowercase hex chars).
func NewJobID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
