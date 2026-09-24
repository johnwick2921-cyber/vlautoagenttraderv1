package updateauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"regexp"
	"strconv"
	"time"
)

// MaxAuthorizationWindow is how far in the future expires_at may lie: an
// authorization is valid for at most 5 minutes from the server's now.
const MaxAuthorizationWindow = 5 * time.Minute

// ErrExpired: expires_at is not in (now, now+MaxAuthorizationWindow].
var ErrExpired = errors.New("updateauth: authorization outside its validity window")

var macHexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// Message is the canonical MAC input: release_id|job_id|expires_at, with
// expires_at in unix seconds as a canonical decimal. Both ids are validated
// first — the allow-lists exclude '|', so no two field triples share a
// message.
func Message(releaseID, jobID string, expiresAt int64) ([]byte, error) {
	if !ValidReleaseID(releaseID) {
		return nil, malformed("release_id")
	}
	if !ValidJobID(jobID) {
		return nil, malformed("job_id")
	}
	if expiresAt <= 0 {
		return nil, malformed("expires_at")
	}
	return []byte(releaseID + "|" + jobID + "|" + strconv.FormatInt(expiresAt, 10)), nil
}

// ComputeMAC returns the lowercase-hex HMAC-SHA256 of Message under key.
// Callers: the attended `updater-bootstrap authorize` ONLY (CTO ruling Q1(a):
// nothing on the API side mints a MAC). A census test pins it.
func ComputeMAC(key []byte, releaseID, jobID string, expiresAt int64) (string, error) {
	if len(key) != DeviceKeyLen || degenerateKey(key) {
		return "", errors.New("updateauth: bad key (wrong length or degenerate)")
	}
	msg, err := Message(releaseID, jobID, expiresAt)
	if err != nil {
		return "", err
	}
	m := hmac.New(sha256.New, key)
	_, _ = m.Write(msg)
	return hex.EncodeToString(m.Sum(nil)), nil
}

// VerifyMAC reports whether macHex (exactly 64 lowercase hex chars) is the
// HMAC-SHA256 of Message under key. The comparison is hmac.Equal (constant
// time). A degenerate key (every byte equal) verifies nothing (M3-RT-F2).
func VerifyMAC(key []byte, releaseID, jobID string, expiresAt int64, macHex string) bool {
	if len(key) != DeviceKeyLen || degenerateKey(key) || !macHexRe.MatchString(macHex) {
		return false
	}
	msg, err := Message(releaseID, jobID, expiresAt)
	if err != nil {
		return false
	}
	got, err := hex.DecodeString(macHex)
	if err != nil {
		return false
	}
	m := hmac.New(sha256.New, key)
	_, _ = m.Write(msg)
	return hmac.Equal(m.Sum(nil), got)
}

// CheckExpiry accepts expiresAt only in (now, now+MaxAuthorizationWindow],
// both in whole unix seconds.
func CheckExpiry(expiresAt int64, now time.Time) error {
	n := now.Unix()
	if expiresAt <= n || expiresAt > n+int64(MaxAuthorizationWindow/time.Second) {
		return ErrExpired
	}
	return nil
}

// Grant is one install authorization: exactly the POST /api/updates/install
// body, and exactly what `updater-bootstrap authorize` prints.
type Grant struct {
	ReleaseID string `json:"release_id"`
	JobID     string `json:"job_id"`
	ExpiresAt int64  `json:"expires_at"`
	HMAC      string `json:"hmac"`
}

// ParseInstallRequest strictly parses an install body: one JSON object with
// exactly release_id (string, ValidReleaseID), job_id (string, ValidJobID),
// expires_at (canonical positive integer) and hmac (string; its CONTENT is
// judged by VerifyMAC, not here, so a bad MAC and a malformed one refuse
// identically). Any other shape ⇒ ErrMalformed (400).
func ParseInstallRequest(r io.Reader) (Grant, error) {
	m, err := decodeStrictObject(r, "release_id", "job_id", "expires_at", "hmac")
	if err != nil {
		return Grant{}, err
	}
	var g Grant
	if g.ReleaseID, err = rawString(m["release_id"]); err != nil {
		return Grant{}, err
	}
	if g.JobID, err = rawString(m["job_id"]); err != nil {
		return Grant{}, err
	}
	if g.ExpiresAt, err = rawUnixSeconds(m["expires_at"]); err != nil {
		return Grant{}, err
	}
	if g.HMAC, err = rawString(m["hmac"]); err != nil {
		return Grant{}, err
	}
	if !ValidReleaseID(g.ReleaseID) {
		return Grant{}, malformed("release_id")
	}
	if !ValidJobID(g.JobID) {
		return Grant{}, malformed("job_id")
	}
	return g, nil
}

// Authorize mints one Grant for releaseID from the enrolled device key:
// a fresh random job id, expires_at = now+MaxAuthorizationWindow. It refuses
// unless the installation is enrolled (admin.json AND device.key load under
// the same rules the API gate applies). The key itself is never returned.
//
// Callers: the attended CLI ONLY (census-pinned).
func Authorize(dataDir, releaseID string, now time.Time) (Grant, error) {
	if !ValidReleaseID(releaseID) {
		return Grant{}, malformed("release_id")
	}
	if _, err := LoadAdmin(dataDir); err != nil {
		return Grant{}, err
	}
	key, err := LoadDeviceKey(dataDir)
	if err != nil {
		return Grant{}, err
	}
	job, err := NewJobID()
	if err != nil {
		return Grant{}, err
	}
	exp := now.Add(MaxAuthorizationWindow).Unix()
	mac, err := ComputeMAC(key, releaseID, job, exp)
	if err != nil {
		return Grant{}, err
	}
	return Grant{ReleaseID: releaseID, JobID: job, ExpiresAt: exp, HMAC: mac}, nil
}
