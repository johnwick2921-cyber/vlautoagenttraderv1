package updateauth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

const (
	tUser  = "11111111-2222-3333-4444-555555555555"
	tEmail = "owner@example.test"
)

var tNow = time.Unix(1_800_000_000, 0)

func enrolled(t *testing.T) string {
	t.Helper()
	d := t.TempDir()
	if err := Enroll(d, tUser, tEmail, tNow, false); err != nil {
		t.Fatalf("enroll: %v", err)
	}
	return d
}

func mustLoad(t *testing.T, d string) {
	t.Helper()
	if _, err := LoadAdmin(d); err != nil {
		t.Fatalf("positive control: LoadAdmin: %v", err)
	}
	if _, err := LoadDeviceKey(d); err != nil {
		t.Fatalf("positive control: LoadDeviceKey: %v", err)
	}
}

func mode(t *testing.T, p string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode()
}

// ── enrollment files ──────────────────────────────────────────────────────

func TestEnrollWritesPrivateFilesAndLoads(t *testing.T) {
	d := enrolled(t)
	if m := mode(t, Dir(d)).Perm(); m != 0o700 {
		t.Fatalf("dir mode %04o, want 0700", m)
	}
	for _, p := range []string{AdminPath(d), DeviceKeyPath(d)} {
		if m := mode(t, p); !m.IsRegular() || m.Perm() != 0o600 {
			t.Fatalf("%s mode %v, want -rw-------", p, m)
		}
	}
	a, err := LoadAdmin(d)
	if err != nil {
		t.Fatal(err)
	}
	if a.UserID != tUser || a.Email != tEmail || a.EnrolledAt != tNow.UTC().Format(time.RFC3339) {
		t.Fatalf("admin = %+v", a)
	}
	k, err := LoadDeviceKey(d)
	if err != nil || len(k) != DeviceKeyLen {
		t.Fatalf("key len %d err %v", len(k), err)
	}
	if bytes.Equal(k, make([]byte, DeviceKeyLen)) {
		t.Fatal("device key is all zero")
	}
}

func TestEnrollRefusesAnExistingEnrollmentWithoutReplace(t *testing.T) {
	d := enrolled(t)
	a0, _ := os.ReadFile(AdminPath(d))
	k0, _ := os.ReadFile(DeviceKeyPath(d))
	if err := Enroll(d, "other-user-id", "x@example.test", tNow, false); !errors.Is(err, ErrAlreadyEnrolled) {
		t.Fatalf("err = %v, want ErrAlreadyEnrolled", err)
	}
	a1, _ := os.ReadFile(AdminPath(d))
	k1, _ := os.ReadFile(DeviceKeyPath(d))
	if !bytes.Equal(a0, a1) || !bytes.Equal(k0, k1) {
		t.Fatal("a refused enroll changed the files")
	}
	// a lone device.key (half-written enrollment) also requires --replace
	d2 := t.TempDir()
	_ = os.MkdirAll(Dir(d2), 0o700)
	_ = os.WriteFile(DeviceKeyPath(d2), make([]byte, 32), 0o600)
	if err := Enroll(d2, tUser, tEmail, tNow, false); !errors.Is(err, ErrAlreadyEnrolled) {
		t.Fatalf("lone key: err = %v", err)
	}
	// positive control: replace rotates key and identity
	if err := Enroll(d, "22222222-aaaa", "new@example.test", tNow, true); err != nil {
		t.Fatal(err)
	}
	k2, _ := LoadDeviceKey(d)
	a2, _ := LoadAdmin(d)
	if bytes.Equal(k0, k2) || a2.UserID != "22222222-aaaa" {
		t.Fatal("replace did not rotate key/identity")
	}
}

func TestEnrollRefusesAnUnsafeUpdaterDir(t *testing.T) {
	d := t.TempDir()
	if err := os.MkdirAll(Dir(d), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(Dir(d), 0o755)
	if err := Enroll(d, tUser, tEmail, tNow, false); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("0755 dir: err = %v, want ErrUnsafe", err)
	}
	if _, err := os.Stat(AdminPath(d)); !os.IsNotExist(err) {
		t.Fatal("admin.json written into an unsafe dir")
	}
	_ = os.Chmod(Dir(d), 0o700) // positive control
	if err := Enroll(d, tUser, tEmail, tNow, false); err != nil {
		t.Fatalf("positive control: %v", err)
	}
	if err := Enroll("relative/data", tUser, tEmail, tNow, false); !errors.Is(err, ErrNoDataDir) {
		t.Fatalf("relative data dir: %v", err)
	}
}

func TestLoadAdminRefusesUnsafeFiles(t *testing.T) {
	cases := []struct {
		name string
		mut  func(t *testing.T, d string)
	}{
		{"absent", func(t *testing.T, d string) { _ = os.Remove(AdminPath(d)) }},
		{"mode 0644", func(t *testing.T, d string) { _ = os.Chmod(AdminPath(d), 0o644) }},
		{"mode 0640", func(t *testing.T, d string) { _ = os.Chmod(AdminPath(d), 0o640) }},
		{"mode 0604", func(t *testing.T, d string) { _ = os.Chmod(AdminPath(d), 0o604) }},
		{"dir 0755", func(t *testing.T, d string) { _ = os.Chmod(Dir(d), 0o755) }},
		{"symlink to a valid copy", func(t *testing.T, d string) {
			b, _ := os.ReadFile(AdminPath(d))
			cp := filepath.Join(d, "copy.json")
			_ = os.WriteFile(cp, b, 0o600)
			_ = os.Remove(AdminPath(d))
			_ = os.Symlink(cp, AdminPath(d))
		}},
		{"updater dir is a symlink", func(t *testing.T, d string) {
			real := filepath.Join(d, "real-updater")
			_ = os.Rename(Dir(d), real)
			_ = os.Symlink(real, Dir(d))
		}},
		{"fifo", func(t *testing.T, d string) {
			_ = os.Remove(AdminPath(d))
			_ = syscall.Mkfifo(AdminPath(d), 0o600)
		}},
		{"directory", func(t *testing.T, d string) { _ = os.Remove(AdminPath(d)); _ = os.Mkdir(AdminPath(d), 0o700) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := enrolled(t)
			mustLoad(t, d) // positive control: same dir without the defect
			c.mut(t, d)
			if _, err := LoadAdmin(d); err == nil {
				t.Fatal("LoadAdmin accepted an unsafe admin.json")
			}
		})
	}
}

func TestLoadRefusesAFileOwnedByAnotherUID(t *testing.T) {
	d := enrolled(t)
	mustLoad(t, d)
	prev := geteuid
	geteuid = func() int { return prev() + 1 }
	t.Cleanup(func() { geteuid = prev })
	if _, err := LoadAdmin(d); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("LoadAdmin: %v, want ErrUnsafe", err)
	}
	if _, err := LoadDeviceKey(d); !errors.Is(err, ErrUnsafe) {
		t.Fatalf("LoadDeviceKey: %v, want ErrUnsafe", err)
	}
}

func TestLoadAdminRefusesMalformedJSON(t *testing.T) {
	valid := `{"user_id":"` + tUser + `","email":"` + tEmail + `","enrolled_at":"2026-09-24T01:00:00Z"}`
	bad := map[string]string{
		"unknown field":   `{"user_id":"` + tUser + `","email":"` + tEmail + `","enrolled_at":"2026-09-24T01:00:00Z","role":"admin"}`,
		"re-cased key":    `{"User_ID":"` + tUser + `","email":"` + tEmail + `","enrolled_at":"2026-09-24T01:00:00Z"}`,
		"duplicate key":   `{"user_id":"x","user_id":"` + tUser + `","email":"` + tEmail + `","enrolled_at":"2026-09-24T01:00:00Z"}`,
		"missing email":   `{"user_id":"` + tUser + `","enrolled_at":"2026-09-24T01:00:00Z"}`,
		"empty user_id":   `{"user_id":"","email":"` + tEmail + `","enrolled_at":"2026-09-24T01:00:00Z"}`,
		"numeric user_id": `{"user_id":7,"email":"` + tEmail + `","enrolled_at":"2026-09-24T01:00:00Z"}`,
		"null email":      `{"user_id":"` + tUser + `","email":null,"enrolled_at":"2026-09-24T01:00:00Z"}`,
		"bad enrolled_at": `{"user_id":"` + tUser + `","email":"` + tEmail + `","enrolled_at":"yesterday"}`,
		"trailing data":   valid + `{}`,
		"array":           `[` + valid + `]`,
		"empty":           ``,
		"pipe in user_id": `{"user_id":"a|b","email":"` + tEmail + `","enrolled_at":"2026-09-24T01:00:00Z"}`,
	}
	d := enrolled(t)
	if err := os.WriteFile(AdminPath(d), []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	mustLoad(t, d) // positive control: the canonical file loads
	for name, body := range bad {
		if err := os.WriteFile(AdminPath(d), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadAdmin(d); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestLoadDeviceKeyRequiresExactly32PrivateBytes(t *testing.T) {
	for _, n := range []int{0, 31, 33, 64} {
		d := enrolled(t)
		mustLoad(t, d)
		_ = os.WriteFile(DeviceKeyPath(d), make([]byte, n), 0o600)
		if _, err := LoadDeviceKey(d); err == nil {
			t.Errorf("%d-byte key accepted", n)
		}
	}
	d := enrolled(t)
	_ = os.Chmod(DeviceKeyPath(d), 0o640)
	if _, err := LoadDeviceKey(d); !errors.Is(err, ErrUnsafe) {
		t.Errorf("0640 key: %v", err)
	}
	d = enrolled(t)
	cp := filepath.Join(d, "k")
	b, _ := os.ReadFile(DeviceKeyPath(d))
	_ = os.WriteFile(cp, b, 0o600)
	_ = os.Remove(DeviceKeyPath(d))
	_ = os.Symlink(cp, DeviceKeyPath(d))
	if _, err := LoadDeviceKey(d); !errors.Is(err, ErrUnsafe) {
		t.Errorf("symlinked key: %v", err)
	}
}

// ── MAC ───────────────────────────────────────────────────────────────────

func rawMAC(key []byte, msg string) string {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func TestMACIsHMACSHA256OverTheCanonicalMessage(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	msg, err := Message("v1.2.3", "0123456789abcdef", 1800000300)
	if err != nil || string(msg) != "v1.2.3|0123456789abcdef|1800000300" {
		t.Fatalf("message %q err %v", msg, err)
	}
	mac, err := ComputeMAC(key, "v1.2.3", "0123456789abcdef", 1800000300)
	if err != nil || mac != rawMAC(key, "v1.2.3|0123456789abcdef|1800000300") {
		t.Fatalf("mac mismatch: %s", mac)
	}
	if !VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000300, mac) {
		t.Fatal("positive control: valid MAC refused")
	}
	flip := []byte(mac)
	if flip[0] == 'a' {
		flip[0] = 'b'
	} else {
		flip[0] = 'a'
	}
	other := bytes.Repeat([]byte{8}, 32)
	bad := map[string]func() bool{
		"wrong key":      func() bool { return VerifyMAC(other, "v1.2.3", "0123456789abcdef", 1800000300, mac) },
		"flipped":        func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000300, string(flip)) },
		"uppercase":      func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000300, strings.ToUpper(mac)) },
		"truncated":      func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000300, mac[:63]) },
		"empty":          func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000300, "") },
		"other release":  func() bool { return VerifyMAC(key, "v1.2.4", "0123456789abcdef", 1800000300, mac) },
		"other job":      func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdee", 1800000300, mac) },
		"other expiry":   func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000301, mac) },
		"short key":      func() bool { return VerifyMAC(key[:31], "v1.2.3", "0123456789abcdef", 1800000300, mac) },
		"reordered msg":  func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000300, rawMAC(key, "0123456789abcdef|v1.2.3|1800000300")) },
		"trailing space": func() bool { return VerifyMAC(key, "v1.2.3", "0123456789abcdef", 1800000300, mac+" ") },
	}
	for name, f := range bad {
		if f() {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestMACFieldsCannotBeReframed(t *testing.T) {
	// "a|b" + "|" + "c-job-0001" == "a" + "|" + "b|c-job-0001": both must be refused.
	if _, err := Message("a|b", "c-job-0001", 1); err == nil {
		t.Error("'|' in release_id accepted")
	}
	if _, err := Message("a", "b|c-job-0001", 1); err == nil {
		t.Error("'|' in job_id accepted")
	}
	if _, err := Message("a", "c-job-0001", 0); err == nil {
		t.Error("expires_at 0 accepted")
	}
	if _, err := Message("a", "c-job-0001", 1); err != nil { // positive control
		t.Errorf("positive control: %v", err)
	}
}

func TestIDValidators(t *testing.T) {
	goodRel := []string{"v1.2.3", "2026.09.24-1", "a", "r_1", strings.Repeat("a", 64)}
	badRel := []string{"", "..", "../../etc/passwd", "/abs", "a/b", `a\b`, "https://x/y", "file:x",
		"%2e%2e", "a\x00b", strings.Repeat("a", 65), ".hidden", "a..b", "a b", "a|b", "-a", "v1\n"}
	for _, s := range goodRel {
		if !ValidReleaseID(s) {
			t.Errorf("release %q refused", s)
		}
	}
	for _, s := range badRel {
		if ValidReleaseID(s) {
			t.Errorf("release %q accepted", s)
		}
	}
	goodJob := []string{"0123456789abcdef", "job-0001", strings.Repeat("a", 64)}
	badJob := []string{"", "short", "ABCDEF0123", "../x-00000", "a/b-000000", "a|b-000000", strings.Repeat("a", 65), "-abcdefgh", "abc def gh"}
	for _, s := range goodJob {
		if !ValidJobID(s) {
			t.Errorf("job %q refused", s)
		}
	}
	for _, s := range badJob {
		if ValidJobID(s) {
			t.Errorf("job %q accepted", s)
		}
	}
	id, err := NewJobID()
	if err != nil || !ValidJobID(id) {
		t.Fatalf("NewJobID %q %v", id, err)
	}
}

func TestCheckExpiryWindow(t *testing.T) {
	n := tNow.Unix()
	for exp, ok := range map[int64]bool{n - 1: false, n: false, n + 1: true, n + 300: true, n + 301: false, 0: false} {
		if got := CheckExpiry(exp, tNow) == nil; got != ok {
			t.Errorf("exp=now%+d: ok=%v want %v", exp-n, got, ok)
		}
	}
}

func TestParseInstallRequestIsStrict(t *testing.T) {
	mac := strings.Repeat("a", 64)
	good := `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`
	if g, err := ParseInstallRequest(strings.NewReader(good)); err != nil || g.ExpiresAt != 1800000300 {
		t.Fatalf("positive control: %+v %v", g, err)
	}
	bad := map[string]string{
		"unknown field":   `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `","url":"x"}`,
		"re-cased":        `{"Release_ID":"v1","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`,
		"duplicate":       `{"release_id":"v1","release_id":"v2","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`,
		"missing hmac":    `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1800000300}`,
		"exp string":      `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":"1800000300","hmac":"` + mac + `"}`,
		"exp float":       `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1800000300.0,"hmac":"` + mac + `"}`,
		"exp exponent":    `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1.8e9,"hmac":"` + mac + `"}`,
		"exp negative":    `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":-5,"hmac":"` + mac + `"}`,
		"exp leading 0":   `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":01800000300,"hmac":"` + mac + `"}`,
		"exp zero":        `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":0,"hmac":"` + mac + `"}`,
		"hmac number":     `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":1}`,
		"path release":    `{"release_id":"../x","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`,
		"pipe job":        `{"release_id":"v1","job_id":"0123|456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`,
		"two objects":     good + good,
		"trailing junk":   good + "x",
		"not an object":   `"x"`,
		"empty":           ``,
		"release null":    `{"release_id":null,"job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`,
		"nested object":   `{"release_id":{"a":1},"job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`,
		"unterminated":    `{"release_id":"v1"`,
		"exp 19 digits":   `{"release_id":"v1","job_id":"0123456789abcdef","expires_at":9999999999999999999,"hmac":"` + mac + `"}`,
		"escaped release": `{"release_id":"v/1","job_id":"0123456789abcdef","expires_at":1800000300,"hmac":"` + mac + `"}`,
	}
	for name, body := range bad {
		if _, err := ParseInstallRequest(strings.NewReader(body)); !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: err = %v, want ErrMalformed", name, err)
		}
	}
}

// ── single-use job ids ────────────────────────────────────────────────────

func TestConsumeIsSingleUseAcrossCalls(t *testing.T) {
	d := t.TempDir()
	exp := tNow.Unix() + 60
	if err := Consume(d, "0123456789abcdef", exp, tNow); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, "0123456789abcdef", exp, tNow); !errors.Is(err, ErrReplay) {
		t.Fatalf("replay: %v", err)
	}
	if err := Consume(d, "0123456789abcdee", exp, tNow); err != nil { // positive control
		t.Fatalf("different id: %v", err)
	}
	if m := mode(t, SeenPath(d)).Perm(); m != 0o600 {
		t.Fatalf("seen mode %04o", m)
	}
}

func TestConsumeCorruptStoreFailsClosedAndIsNeverReset(t *testing.T) {
	for name, body := range map[string]string{
		"garbage":       "not json",
		"empty":         "",
		"null ids":      `{"v":1,"ids":null}`,
		"wrong version": `{"v":2,"ids":[]}`,
		"unknown field": `{"v":1,"ids":[],"x":1}`,
		"bad entry":     `{"v":1,"ids":[{"job_id":"../x","expires_at":1,"consumed_at":1}]}`,
	} {
		d := t.TempDir()
		_ = os.MkdirAll(Dir(d), 0o700)
		_ = os.WriteFile(SeenPath(d), []byte(body), 0o600)
		if err := Consume(d, "0123456789abcdef", tNow.Unix()+60, tNow); !errors.Is(err, ErrSeenCorrupt) {
			t.Errorf("%s: err = %v, want ErrSeenCorrupt", name, err)
		}
		if b, _ := os.ReadFile(SeenPath(d)); string(b) != body {
			t.Errorf("%s: the corrupt store was rewritten to %q", name, b)
		}
	}
	// positive control: an empty-but-valid store accepts
	d := t.TempDir()
	_ = os.MkdirAll(Dir(d), 0o700)
	_ = os.WriteFile(SeenPath(d), []byte(`{"v":1,"ids":[]}`), 0o600)
	if err := Consume(d, "0123456789abcdef", tNow.Unix()+60, tNow); err != nil {
		t.Fatalf("positive control: %v", err)
	}
	// a loose-mode store is unreadable → corrupt, not empty
	_ = os.Chmod(SeenPath(d), 0o644)
	if err := Consume(d, "0123456789abcdee", tNow.Unix()+60, tNow); !errors.Is(err, ErrSeenCorrupt) {
		t.Fatalf("0644 store: %v", err)
	}
}

func TestConsumePrunesOnlyLongExpiredIDs(t *testing.T) {
	d := t.TempDir()
	old := "0000000000000001"
	recent := "0000000000000002"
	if err := Consume(d, old, tNow.Unix()-11*60, tNow.Add(-15*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, recent, tNow.Unix()-9*60, tNow.Add(-14*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := Consume(d, "0000000000000003", tNow.Unix()+60, tNow); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(SeenPath(d))
	if strings.Contains(string(b), old) {
		t.Error("an id expired >10 min ago was kept")
	}
	if !strings.Contains(string(b), recent) {
		t.Error("an id expired <10 min ago was pruned")
	}
	if err := Consume(d, recent, tNow.Unix()+60, tNow); !errors.Is(err, ErrReplay) {
		t.Errorf("recent replay: %v", err)
	}
}

func TestConsumeHardCapRefuses(t *testing.T) {
	d := t.TempDir()
	var sb strings.Builder
	sb.WriteString(`{"v":1,"ids":[`)
	for i := 0; i < MaxSeenEntries; i++ {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(`{"job_id":"` + hex.EncodeToString([]byte{byte(i >> 8), byte(i)}) + `-cap-job","expires_at":` +
			"1800000900" + `,"consumed_at":1800000000}`)
	}
	sb.WriteString(`]}`)
	_ = os.MkdirAll(Dir(d), 0o700)
	_ = os.WriteFile(SeenPath(d), []byte(sb.String()), 0o600)
	if err := Consume(d, "0123456789abcdef", tNow.Unix()+60, tNow); !errors.Is(err, ErrSeenFull) {
		t.Fatalf("err = %v, want ErrSeenFull", err)
	}
	// positive control: once they are long expired they prune and it accepts
	if err := Consume(d, "0123456789abcdef", tNow.Unix()+60, tNow.Add(30*time.Minute)); err != nil {
		t.Fatalf("positive control: %v", err)
	}
}

func TestConcurrentConsumeAdmitsExactlyOne(t *testing.T) {
	d := t.TempDir()
	var wg sync.WaitGroup
	var mu sync.Mutex
	ok, replay := 0, 0
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := Consume(d, "0123456789abcdef", tNow.Unix()+60, tNow)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil:
				ok++
			case errors.Is(err, ErrReplay):
				replay++
			default:
				t.Errorf("unexpected: %v", err)
			}
		}()
	}
	wg.Wait()
	if ok != 1 || replay != 7 {
		t.Fatalf("ok=%d replay=%d, want 1/7", ok, replay)
	}
}

// ── authorize (the CLI's minting path) ────────────────────────────────────

func TestAuthorizeMintsAGrantTheVerifierAccepts(t *testing.T) {
	d := enrolled(t)
	g, err := Authorize(d, "v1.2.3", tNow)
	if err != nil {
		t.Fatal(err)
	}
	if g.ExpiresAt != tNow.Add(5*time.Minute).Unix() || !ValidJobID(g.JobID) {
		t.Fatalf("grant %+v", g)
	}
	key, _ := LoadDeviceKey(d)
	if !VerifyMAC(key, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) {
		t.Fatal("grant MAC does not verify")
	}
	if CheckExpiry(g.ExpiresAt, tNow) != nil {
		t.Fatal("grant expiry outside the window at mint time")
	}
	if strings.Contains(g.HMAC, hex.EncodeToString(key)) {
		t.Fatal("grant leaks the key")
	}
	// not enrolled ⇒ refused
	if _, err := Authorize(t.TempDir(), "v1.2.3", tNow); !errors.Is(err, ErrNotEnrolled) {
		t.Fatalf("unenrolled: %v", err)
	}
	if _, err := Authorize(d, "../v1", tNow); !errors.Is(err, ErrMalformed) {
		t.Fatalf("path release: %v", err)
	}
}

// ── stub verifier ─────────────────────────────────────────────────────────

func TestStubVerifierRefusesEverythingAndTouchesNoFS(t *testing.T) {
	for _, id := range []string{"v1.2.3", "", "../../etc/passwd", "/etc/passwd", "https://x/y", strings.Repeat("a", 5000)} {
		if _, err := (StubVerifier{}).VerifiedManifest(id); !errors.Is(err, ErrNoVerifiedManifest) {
			t.Errorf("%q: %v", id, err)
		}
	}
	// The stub's file may import nothing but "errors": no os, no io, no net.
	src, err := os.ReadFile("verifier.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "import \"errors\"\n") || strings.Contains(string(src), "import (") {
		t.Fatal("verifier.go imports more than \"errors\" — the stub must not be able to reach the filesystem")
	}
}
