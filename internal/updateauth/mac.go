package updateauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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

// ErrClockBehindSeenStore: Authorize refused to mint because the code it
// would mint expires at or below the seen store's pruned-through watermark
// (the server would answer 409 for a job never used) or its clock floor (the
// server would answer 403) — this box's clock is behind a reading the store
// already recorded (red-team red-3 #5).
var ErrClockBehindSeenStore = errors.New("updateauth: refusing to mint: the clock is behind the seen-job store (a code minted now would be refused) — fix the clock, or wait until it passes the store's floor")

var macHexRe = regexp.MustCompile(`^[0-9a-f]{64}$`)

// MACPurpose is the purpose/version tag every install-authorization MAC
// message starts with (red-team red-3 #7). A device-key MAC minted for any
// other action or layout — a future rollback authorization, a receipt, the
// pre-tag release|job|exp shape — never verifies as an install. Introduced
// before M3 shipped, so no untagged code was ever issued. A new layout gets a
// new version, never a reinterpretation of v1.
const MACPurpose = "nofx-update-install/v1"

// Message is the canonical MAC input:
// MACPurpose|user_id|release_id|job_id|expires_at, with user_id the ENROLLED
// administrator's id (admin.json — red-team red-4 #4: a code is bound to the
// administrator it was minted for) and expires_at in unix seconds as a
// canonical decimal. Every field is validated first — the allow-lists
// exclude '|', so no two field tuples share a message.
func Message(userID, releaseID, jobID string, expiresAt int64) ([]byte, error) {
	if !validUserID(userID) {
		return nil, malformed("user_id")
	}
	if !ValidReleaseID(releaseID) {
		return nil, malformed("release_id")
	}
	if !ValidJobID(jobID) {
		return nil, malformed("job_id")
	}
	if expiresAt <= 0 {
		return nil, malformed("expires_at")
	}
	return []byte(MACPurpose + "|" + userID + "|" + releaseID + "|" + jobID + "|" + strconv.FormatInt(expiresAt, 10)), nil
}

// ComputeMAC returns the lowercase-hex HMAC-SHA256 of Message under key.
// Callers: the attended `updater-bootstrap authorize` ONLY (CTO ruling Q1(a):
// nothing on the API side mints a MAC). A census test pins it.
func ComputeMAC(key []byte, userID, releaseID, jobID string, expiresAt int64) (string, error) {
	if len(key) != DeviceKeyLen || degenerateKey(key) {
		return "", errors.New("updateauth: bad key (wrong length or degenerate)")
	}
	msg, err := Message(userID, releaseID, jobID, expiresAt)
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
func VerifyMAC(key []byte, userID, releaseID, jobID string, expiresAt int64, macHex string) bool {
	if len(key) != DeviceKeyLen || degenerateKey(key) || !macHexRe.MatchString(macHex) {
		return false
	}
	msg, err := Message(userID, releaseID, jobID, expiresAt)
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

// grantRedacted replaces the MAC wherever a Grant is formatted.
const grantRedacted = "<redacted>"

// String, GoString and Format make every fmt verb print the Grant with its
// MAC redacted (red-team red-3 #6): a formatted grant reaches logs
// (data/nofx_*.log, log_events), and a logged unused grant is a live code for
// up to MaxAuthorizationWindow. json.Marshal is unaffected — the JSON IS the
// grant (the CLI's output and the install body). Known limit: a Grant held
// in an UNEXPORTED struct field is printed by reflection without these
// methods; never embed one in a type that is logged.
func (g Grant) String() string {
	return fmt.Sprintf("{release_id:%s job_id:%s expires_at:%d hmac:%s}", g.ReleaseID, g.JobID, g.ExpiresAt, grantRedacted)
}

// GoString is the %#v form, redacted.
func (g Grant) GoString() string {
	return fmt.Sprintf("updateauth.Grant{ReleaseID:%q, JobID:%q, ExpiresAt:%d, HMAC:%q}", g.ReleaseID, g.JobID, g.ExpiresAt, grantRedacted)
}

// Format routes EVERY verb (%v %+v %s %q %x %d …, any width/flags) to the
// redacted forms, so no verb falls through to fmt's field-by-field printer.
func (g Grant) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('#') {
		_, _ = io.WriteString(f, g.GoString())
		return
	}
	_, _ = io.WriteString(f, g.String())
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
// the same rules the API gate applies) and unless the server could accept
// the code (the seen-job store is readable and the code expires above its
// watermark and clock floor — ErrClockBehindSeenStore otherwise). The key
// itself is never returned.
//
// Callers: the attended CLI ONLY (census-pinned).
func Authorize(dataDir, releaseID string, now time.Time) (Grant, error) {
	if !ValidReleaseID(releaseID) {
		return Grant{}, malformed("release_id")
	}
	admin, err := LoadAdmin(dataDir)
	if err != nil {
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
	// Red-team red-3 #5: never hand out a code the server can only refuse.
	// Read the seen store (read-only: no lock — writers replace it by rename)
	// under the same rules Consume reads it: unreadable, or missing once
	// enrolled, refuses; a code expiring at or below the pruned-through
	// watermark (server: 409 for a job never used) or the clock floor
	// (server: 403) refuses; a job id already present refuses.
	st, err := readSeen(dataDir)
	if err != nil {
		return Grant{}, err
	}
	if exp <= st.PrunedThrough || exp <= st.ClockFloor {
		return Grant{}, ErrClockBehindSeenStore
	}
	for _, e := range st.IDs {
		if e.JobID == job {
			return Grant{}, errors.New("updateauth: refusing to mint: the fresh job id is already in the seen-job store")
		}
	}
	mac, err := ComputeMAC(key, admin.UserID, releaseID, job, exp)
	if err != nil {
		return Grant{}, err
	}
	return Grant{ReleaseID: releaseID, JobID: job, ExpiresAt: exp, HMAC: mac}, nil
}
