package updateauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

// DeviceKeyLen is the device key's exact length in bytes.
const DeviceKeyLen = 32

// randRead is a seam so Enroll's degenerate-key refusal is testable.
var randRead = rand.Read

// degenerateKey reports whether key is empty or every byte of it equals the
// first (32 zero bytes from a zero-filled restore or sparse copy, 32 × 0xFF,
// …): a key anyone can enumerate in 256 guesses, so a MAC under it proves
// nothing about possession (M3-RT-F2). The scan touches every byte and
// branches only on the verdict. crypto/rand yields such a key with
// probability 2^-248, so no real enrollment is ever refused.
func degenerateKey(key []byte) bool {
	if len(key) == 0 {
		return true
	}
	var acc byte
	for _, b := range key[1:] {
		acc |= b ^ key[0]
	}
	return subtle.ConstantTimeByteEq(acc, 0) == 1
}

const maxAdminFileBytes = 4096

// Admin is the enrolled installation administrator (admin.json).
type Admin struct {
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	EnrolledAt string `json:"enrolled_at"` // RFC3339 UTC
	// PasswordBinding (M3 red-team H1/H2 belt) is the lowercase-hex
	// HMAC-SHA256, under device.key, of passwordBindingMessage(user_id,
	// users.password_hash) as it stood at enrollment. The /updates gate
	// recomputes it from the CURRENT row: any password change — the owner's,
	// or one forced through a stolen/machine token — un-enrolls (403 until
	// `updater-bootstrap enroll --replace`), exactly as a reset-account does.
	// It is a MAC over a bcrypt hash under a key only this box holds: it
	// reveals nothing about the password.
	PasswordBinding string `json:"password_binding"`
}

// passwordBindingDomain separates this MAC from the install-grant MAC
// (release|job|exp) under the same device.key: the NUL bytes cannot occur in
// any grant message (the id allow-lists exclude them), so no binding can ever
// be replayed as a grant or the reverse.
const passwordBindingDomain = "nofx-updater/password-binding/v1\x00"

func passwordBindingMessage(userID, passwordHash string) []byte {
	return []byte(passwordBindingDomain + userID + "\x00" + passwordHash)
}

// passwordBinding computes the binding; it refuses a bad key and an empty
// hash (a user with no password is not a login identity). Unexported: only
// Enroll mints one.
func passwordBinding(key []byte, userID, passwordHash string) (string, error) {
	if len(key) != DeviceKeyLen || degenerateKey(key) {
		return "", errors.New("updateauth: bad key (wrong length or degenerate)")
	}
	if passwordHash == "" {
		return "", malformed("password hash is empty")
	}
	m := hmac.New(sha256.New, key)
	_, _ = m.Write(passwordBindingMessage(userID, passwordHash))
	return hex.EncodeToString(m.Sum(nil)), nil
}

// PasswordStillBound reports whether currentPasswordHash (the admin row's
// password_hash NOW) is the one this enrollment was bound to, under key.
// Constant-time on the MAC; false for a bad key, an empty hash or a
// malformed stored binding. The /updates gate refuses on false.
func (a Admin) PasswordStillBound(key []byte, currentPasswordHash string) bool {
	if !macHexRe.MatchString(a.PasswordBinding) {
		return false
	}
	want, err := passwordBinding(key, a.UserID, currentPasswordHash)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(want), []byte(a.PasswordBinding))
}

// validateAdmin is the ONE predicate a stored enrollment record must pass:
// LoadAdmin applies it to what it reads and Enroll to what it is about to
// write (write-through-the-read-validator — a writer can never wedge its own
// reader).
func validateAdmin(a Admin) error {
	if !validUserID(a.UserID) {
		return malformed("user_id")
	}
	if !validEmail(a.Email) {
		return malformed("email")
	}
	if _, err := time.Parse(time.RFC3339, a.EnrolledAt); err != nil {
		return malformed("enrolled_at")
	}
	if !macHexRe.MatchString(a.PasswordBinding) {
		return malformed("password_binding")
	}
	return nil
}

// ErrNotEnrolled: admin.json is absent (the OFF state — nothing enrolled).
var ErrNotEnrolled = errors.New("updateauth: not enrolled")

// ErrAlreadyEnrolled: Enroll without replace over an existing enrollment
// (admin.json OR device.key present).
var ErrAlreadyEnrolled = errors.New("updateauth: already enrolled (re-enroll requires --replace)")

// LoadAdmin reads admin.json strictly: the updater dir must be 0700 and ours,
// the file a regular 0600 file we own (never a symlink), one JSON object with
// EXACTLY the keys user_id, email, enrolled_at, password_binding (byte-exact,
// each once), all strings, passing validateAdmin. Any deviation is an error;
// the caller refuses (403). Absent ⇒ ErrNotEnrolled.
//
// MIGRATION NOTE: an admin.json without password_binding (the three-key shape
// written before the H1/H2 belt) is REFUSED — re-enroll with --replace. No
// shipped binary ever wrote one (M3 never shipped), so no box holds one.
func LoadAdmin(dataDir string) (Admin, error) {
	if err := checkDataDir(dataDir); err != nil {
		return Admin{}, err
	}
	b, err := readPrivateFile(AdminPath(dataDir), maxAdminFileBytes)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Admin{}, ErrNotEnrolled
		}
		return Admin{}, err
	}
	m, err := decodeStrictObject(bytes.NewReader(b), "user_id", "email", "enrolled_at", "password_binding")
	if err != nil {
		return Admin{}, err
	}
	var a Admin
	if a.UserID, err = rawString(m["user_id"]); err != nil {
		return Admin{}, err
	}
	if a.Email, err = rawString(m["email"]); err != nil {
		return Admin{}, err
	}
	if a.EnrolledAt, err = rawString(m["enrolled_at"]); err != nil {
		return Admin{}, err
	}
	if a.PasswordBinding, err = rawString(m["password_binding"]); err != nil {
		return Admin{}, err
	}
	if err := validateAdmin(a); err != nil {
		return Admin{}, err
	}
	return a, nil
}

// LoadDeviceKey reads device.key under the same file rules as LoadAdmin and
// requires exactly DeviceKeyLen bytes that are not degenerate (every byte
// equal — an all-zero key is ErrUnsafe; re-enroll with --replace). The key
// never leaves this process: no API returns it, logs it, or derives a
// response from it.
func LoadDeviceKey(dataDir string) ([]byte, error) {
	if err := checkDataDir(dataDir); err != nil {
		return nil, err
	}
	b, err := readPrivateFile(DeviceKeyPath(dataDir), DeviceKeyLen)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotEnrolled
		}
		return nil, err
	}
	if len(b) != DeviceKeyLen {
		return nil, unsafeErr(DeviceKeyPath(dataDir), fmt.Sprintf("%d bytes, want %d", len(b), DeviceKeyLen))
	}
	if degenerateKey(b) {
		clear(b)
		return nil, unsafeErr(DeviceKeyPath(dataDir), "degenerate key (every byte equal) — re-enroll with --replace")
	}
	return b, nil
}

// Enroll writes a new enrollment: a fresh random device.key, THEN admin.json
// (each by the temp+fsync+rename+fsync-dir sequence), under .enroll.lock. It
// refuses (ErrAlreadyEnrolled) when either file exists and replace is false;
// replace rotates the key too.
//
// The two files are two renames, not one transaction. If the process dies —
// or the second write fails (ENOSPC, EIO, admin.json replaced by a directory)
// — after the key rename and before the admin.json rename:
//   - FIRST enroll: a lone new key, no admin.json — nobody is enrolled (every
//     /api/updates* 403) and a plain enroll refuses until --replace;
//   - REPLACE: the NEW key beside the OLD admin.json — a WORKING enrollment
//     of the incumbent: every grant minted under the old key dies, but
//     `authorize` mints under the new key and the incumbent's install is
//     authorized. A replace meant to revoke the incumbent is NOT a revocation
//     until the CLI prints "enrolled:"; re-run it until it does.
//
// Pinned at the production router by
// api.TestEnrollCommentTruthACrashBetweenTheTwoRenames (red-team red-4 #4).
//
// passwordHash is the admin row's users.password_hash as the CLI read it
// (read-only) at enrollment: admin.json stores its binding under the new key
// (Admin.PasswordBinding), so any later password change un-enrolls.
//
// Callers: the attended CLI ONLY (internal/updaterbootstrap). A census test
// pins that no other non-test file calls it.
func Enroll(dataDir, userID, email, passwordHash string, now time.Time, replace bool) error {
	if err := checkDataDir(dataDir); err != nil {
		return err
	}
	if !validUserID(userID) {
		return malformed("user_id")
	}
	if !validEmail(email) {
		return malformed("email")
	}
	if passwordHash == "" {
		return malformed("password hash is empty (not a login identity)")
	}
	dir, err := ensurePrivateDir(dataDir)
	if err != nil {
		return err
	}
	unlock, err := lockFile(enrollLockPath(dataDir))
	if err != nil {
		return err
	}
	defer unlock()
	if !replace {
		for _, p := range []string{AdminPath(dataDir), DeviceKeyPath(dataDir)} {
			if _, err := os.Lstat(p); err == nil {
				return ErrAlreadyEnrolled
			} else if !errors.Is(err, os.ErrNotExist) {
				return err
			}
		}
	}
	key := make([]byte, DeviceKeyLen)
	if _, err := randRead(key); err != nil {
		return err
	}
	// Never write a key LoadDeviceKey refuses (the writer passes its own
	// reader's validator).
	if degenerateKey(key) {
		return errors.New("updateauth: the random source returned a degenerate key — nothing written")
	}
	binding, err := passwordBinding(key, userID, passwordHash)
	if err != nil {
		clear(key)
		return err
	}
	a := Admin{UserID: userID, Email: email, EnrolledAt: now.UTC().Format(time.RFC3339), PasswordBinding: binding}
	// The record passes the SAME predicate LoadAdmin applies before a byte is
	// written (write-through-the-read-validator).
	if err := validateAdmin(a); err != nil {
		clear(key)
		return err
	}
	ab, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(dir, DeviceKeyPath(dataDir), key); err != nil {
		return err
	}
	return writeAtomic(dir, AdminPath(dataDir), append(ab, '\n'))
}
