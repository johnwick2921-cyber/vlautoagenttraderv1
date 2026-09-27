package updateauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

// M3 red-team H1/H2 belt: admin.json carries HMAC(device.key, domain |
// user_id | users.password_hash) as it stood at enrollment; the /updates gate
// refuses when the row's CURRENT hash no longer matches (api pin:
// TestPasswordChangeUnbindsTheEnrollment).

func TestEnrollBindsTheCurrentPasswordHash(t *testing.T) {
	d := enrolled(t)
	a, err := LoadAdmin(d)
	if err != nil {
		t.Fatal(err)
	}
	key, err := LoadDeviceKey(d)
	if err != nil {
		t.Fatal(err)
	}
	if !a.PasswordStillBound(key, tHash) {
		t.Fatal("positive control: the enrollment is not bound to the hash it was enrolled with")
	}
	other := append([]byte(nil), key...)
	other[0] ^= 0x01
	tampered := a
	tampered.PasswordBinding = strings.Repeat("0", 64)
	otherUser := a
	otherUser.UserID = "99999999-2222-3333-4444-555555555555"
	for name, ok := range map[string]bool{
		"a changed password hash":              a.PasswordStillBound(key, tHash+"x"),
		"an empty password hash":               a.PasswordStillBound(key, ""),
		"another device key":                   a.PasswordStillBound(other, tHash),
		"a degenerate key":                     a.PasswordStillBound(make([]byte, DeviceKeyLen), tHash),
		"a short key":                          a.PasswordStillBound(key[:31], tHash),
		"a forged binding":                     tampered.PasswordStillBound(key, tHash),
		"another user id under the same MAC":   otherUser.PasswordStillBound(key, tHash),
		"an upper-cased copy of the real MAC":  Admin{UserID: a.UserID, PasswordBinding: strings.ToUpper(a.PasswordBinding)}.PasswordStillBound(key, tHash),
		"a malformed (non-hex) stored binding": Admin{UserID: a.UserID, PasswordBinding: "not-hex"}.PasswordStillBound(key, tHash),
	} {
		if ok {
			t.Errorf("%s still verifies as bound", name)
		}
	}
	// A re-enrollment rotates the key, so the same hash yields a new binding.
	if err := Enroll(d, tUser, tEmail, tHash, tNow, true); err != nil {
		t.Fatal(err)
	}
	a2, _ := LoadAdmin(d)
	if a2.PasswordBinding == a.PasswordBinding {
		t.Fatal("re-enrollment kept the old binding (the key did not rotate?)")
	}
}

// A user with no password is not a login identity: nothing is written.
func TestEnrollRefusesAnEmptyPasswordHash(t *testing.T) {
	d := t.TempDir()
	if err := Enroll(d, tUser, tEmail, "", tNow, false); !errors.Is(err, ErrMalformed) {
		t.Fatalf("empty hash: err = %v, want ErrMalformed", err)
	}
	for _, p := range []string{AdminPath(d), DeviceKeyPath(d)} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s written by a refused enroll", p)
		}
	}
	if err := Enroll(d, tUser, tEmail, tHash, tNow, false); err != nil { // positive control
		t.Fatal(err)
	}
	mustLoad(t, d)
}

// Write-through-the-read-validator: what Enroll writes is exactly the four
// keys LoadAdmin demands, and LoadAdmin accepts it.
func TestEnrollWritesTheRecordItsReaderAccepts(t *testing.T) {
	d := enrolled(t)
	b, err := os.ReadFile(AdminPath(d))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if len(m) != 4 || m["password_binding"] == nil {
		t.Fatalf("admin.json keys = %v, want user_id/email/enrolled_at/password_binding", m)
	}
	mustLoad(t, d)
	// The stored binding reveals neither the hash nor the key.
	key, _ := LoadDeviceKey(d)
	if bytes.Contains(b, []byte(tHash)) || bytes.Contains(b, key) {
		t.Fatal("admin.json carries the password hash or the key")
	}
}

// Domain separation from the install-grant MAC under the same key: a binding
// message carries NUL bytes, which no grant field can (the id allow-lists
// refuse them), so no binding can pass as a grant MAC or the reverse.
func TestPasswordBindingIsDomainSeparatedFromGrants(t *testing.T) {
	msg := passwordBindingMessage(tUser, tHash)
	if !bytes.HasPrefix(msg, []byte(passwordBindingDomain)) || !bytes.Contains(msg, []byte{0}) {
		t.Fatalf("binding message %q lacks its domain tag / NUL separators", msg)
	}
	if ValidReleaseID("v1\x00x") || ValidJobID("0123456789abcdef\x00") {
		t.Fatal("a grant id admits NUL — the binding/grant domains could collide")
	}
}
