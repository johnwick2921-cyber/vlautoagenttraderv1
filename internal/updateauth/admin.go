package updateauth

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
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
}

// ErrNotEnrolled: admin.json is absent (the OFF state — nothing enrolled).
var ErrNotEnrolled = errors.New("updateauth: not enrolled")

// ErrAlreadyEnrolled: Enroll without replace over an existing enrollment
// (admin.json OR device.key present).
var ErrAlreadyEnrolled = errors.New("updateauth: already enrolled (re-enroll requires --replace)")

// LoadAdmin reads admin.json strictly: the updater dir must be 0700 and ours,
// the file a regular 0600 file we own (never a symlink), one JSON object with
// EXACTLY the keys user_id, email, enrolled_at (byte-exact, each once), all
// strings, user_id and email well-formed, enrolled_at RFC3339. Any deviation
// is an error; the caller refuses (403). Absent ⇒ ErrNotEnrolled.
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
	m, err := decodeStrictObject(bytes.NewReader(b), "user_id", "email", "enrolled_at")
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
	if !validUserID(a.UserID) {
		return Admin{}, malformed("user_id")
	}
	if !validEmail(a.Email) {
		return Admin{}, malformed("email")
	}
	if _, err := time.Parse(time.RFC3339, a.EnrolledAt); err != nil {
		return Admin{}, malformed("enrolled_at")
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
// replace rotates the key too. A crash between the two writes leaves a new
// key beside the old/no admin.json — every MAC then fails (fail closed) until
// enrollment is re-run.
//
// Callers: the attended CLI ONLY (internal/updaterbootstrap). A census test
// pins that no other non-test file calls it.
func Enroll(dataDir, userID, email string, now time.Time, replace bool) error {
	if err := checkDataDir(dataDir); err != nil {
		return err
	}
	if !validUserID(userID) {
		return malformed("user_id")
	}
	if !validEmail(email) {
		return malformed("email")
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
	a := Admin{UserID: userID, Email: email, EnrolledAt: now.UTC().Format(time.RFC3339)}
	ab, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	if err := writeAtomic(dir, DeviceKeyPath(dataDir), key); err != nil {
		return err
	}
	return writeAtomic(dir, AdminPath(dataDir), append(ab, '\n'))
}
