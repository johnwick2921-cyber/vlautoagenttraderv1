package updaterbootstrap

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"nofx/internal/holdcli"
	"nofx/internal/updateauth"
	"nofx/store"
)

// W-ONE-BUTTON M3 — the attended enrollment / authorization CLI. Every
// refusal test carries a positive control: the same invocation with the one
// defect removed succeeds, so the refusal is caused by the defect.

const (
	bUser  = "cccccccc-1111-2222-3333-444444444444"
	bEmail = "owner@example.test"
	bRel   = "v2026.09.24-1"
)

var bNow = time.Unix(1_800_000_000, 0)

// install lays out a real installation: <inst>/data/data.db created by the
// production store (schema + migrations), seeded, CLOSED — then returns the
// install dir. The CLI must find the same data dir the bot resolves.
func install(t *testing.T) string {
	t.Helper()
	inst := t.TempDir()
	dbFile := filepath.Join(inst, "data", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbFile), 0o700); err != nil {
		t.Fatal(err)
	}
	st, err := store.New(dbFile)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	past := bNow.Add(-time.Hour)
	for _, u := range []*store.User{
		{ID: bUser, Email: bEmail, PasswordHash: "$2a$10$not-a-real-hash-but-non-empty", CreatedAt: past, UpdatedAt: past},
		{ID: "admin", Email: "admin@localhost", PasswordHash: "", CreatedAt: past, UpdatedAt: past}, // EnsureAdmin's shape
	} {
		if err := st.User().Create(u); err != nil {
			t.Fatal(err)
		}
	}
	st.Plan().Close()
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	return inst
}

// attended sets the seams to an attended, non-root session at bNow.
func attended(t *testing.T) {
	t.Helper()
	pT, pE, pN := isTerminal, geteuid, now
	isTerminal = func(io.Reader) bool { return true }
	geteuid = func() int { return 1000 }
	now = func() time.Time { return bNow }
	t.Cleanup(func() { isTerminal, geteuid, now = pT, pE, pN })
}

func run(inst string, stdin string, args ...string) (int, string, string) {
	var out, errb bytes.Buffer
	rc := Run(append([]string{"--install-dir", inst}, args...), strings.NewReader(stdin), &out, &errb)
	return rc, out.String(), errb.String()
}

func enrollLine(email string) string { return "ENROLL " + email + "\n" }

func fileSig(t *testing.T, p string) string {
	t.Helper()
	fi, err := os.Lstat(p)
	if err != nil {
		return "absent"
	}
	b, _ := os.ReadFile(p)
	sum := sha256.Sum256(b)
	st := fi.Sys().(*syscall.Stat_t)
	return fmt.Sprintf("%x|%v|%d|%d", sum, fi.Mode(), fi.ModTime().UnixNano(), st.Ino)
}

func noEnrollment(t *testing.T, inst string) {
	t.Helper()
	d := DataDirFor(inst)
	for _, p := range []string{updateauth.AdminPath(d), updateauth.DeviceKeyPath(d)} {
		if _, err := os.Lstat(p); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("%s exists after a refused enroll", filepath.Base(p))
		}
	}
}

// ── the resolver ─────────────────────────────────────────────────────────

// ONE resolver: the CLI's data dir is the maintenance CLI's (and so the
// bot's — main_maintenance_test pins holdcli == bot) for every layout.
func TestDataDirIsTheMaintenanceResolver(t *testing.T) {
	for _, env := range []string{"", "DB_PATH=var/db/data.db\n", "DB_PATH=/abs/elsewhere/x.db\n"} {
		inst := t.TempDir()
		if env != "" {
			_ = os.WriteFile(filepath.Join(inst, ".env"), []byte(env), 0o600)
		}
		t.Setenv("DB_PATH", "x")
		os.Unsetenv("DB_PATH")
		if a, b := DataDirFor(inst), holdcli.DataDirFor(inst); a != b {
			t.Errorf(".env %q: updater-bootstrap %q != maintenance-hold %q", env, a, b)
		}
		if a, b := DBFileFor(inst), holdcli.DBFileFor(inst); a != b {
			t.Errorf(".env %q: db %q != %q", env, a, b)
		}
	}
}

// ── enroll ───────────────────────────────────────────────────────────────

func TestEnrollWritesTheEnrollmentForTheExactEmail(t *testing.T) {
	attended(t)
	inst := install(t)
	rc, out, errb := run(inst, enrollLine(bEmail), "enroll", bEmail)
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, errb)
	}
	d := DataDirFor(inst)
	a, err := updateauth.LoadAdmin(d)
	if err != nil {
		t.Fatal(err)
	}
	if a.UserID != bUser || a.Email != bEmail || a.EnrolledAt != bNow.UTC().Format(time.RFC3339) {
		t.Fatalf("admin = %+v", a)
	}
	key, err := updateauth.LoadDeviceKey(d)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{out, errb} {
		if strings.Contains(s, hex.EncodeToString(key)) || strings.Contains(s, string(key)) || strings.Contains(s, bUser) {
			t.Fatal("enroll printed the key or the full user id")
		}
	}
	fi, _ := os.Stat(updateauth.Dir(d))
	if fi.Mode().Perm() != 0o700 {
		t.Fatalf("updater dir %v", fi.Mode())
	}
}

// H1/H2 belt: the CLI binds the enrollment to the row's password_hash it
// READ (read-only) — the one the /updates gate later compares against — and
// a re-enroll after the row's hash changed binds the NEW hash.
func TestEnrollBindsTheRowsPasswordHash(t *testing.T) {
	attended(t)
	inst := install(t)
	const fixtureHash = "$2a$10$not-a-real-hash-but-non-empty" // install()'s row
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("rc=%d %s", rc, errb)
	}
	d := DataDirFor(inst)
	a, err := updateauth.LoadAdmin(d)
	if err != nil {
		t.Fatal(err)
	}
	key, err := updateauth.LoadDeviceKey(d)
	if err != nil {
		t.Fatal(err)
	}
	if !a.PasswordStillBound(key, fixtureHash) {
		t.Fatal("enroll did not bind the row's password_hash")
	}
	// The owner changes the password (the web app writes the row) …
	st, err := store.New(filepath.Join(inst, "data", "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := st.User().UpdatePassword(bUser, "$2a$10$another-hash-after-a-change"); err != nil {
		t.Fatal(err)
	}
	st.Plan().Close()
	_ = st.Close()
	if !a.PasswordStillBound(key, fixtureHash) || a.PasswordStillBound(key, "$2a$10$another-hash-after-a-change") {
		t.Fatal("the stored binding must name the OLD hash only")
	}
	// … and re-enrolls with --replace: the new binding names the new hash.
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", "--replace", bEmail); rc != 0 {
		t.Fatalf("replace rc=%d %s", rc, errb)
	}
	a2, _ := updateauth.LoadAdmin(d)
	key2, _ := updateauth.LoadDeviceKey(d)
	if !a2.PasswordStillBound(key2, "$2a$10$another-hash-after-a-change") || a2.PasswordStillBound(key2, fixtureHash) {
		t.Fatal("--replace did not bind the row's CURRENT password_hash")
	}
}

func TestEnrollMatchesTheEmailExactlyLikeLogin(t *testing.T) {
	attended(t)
	inst := install(t)
	for _, e := range []string{"Owner@example.test", "owner@example.tes", "nobody@example.test", "owner@example.test.", "%@example.test"} {
		if rc, _, _ := run(inst, enrollLine(e), "enroll", e); rc == 0 {
			t.Errorf("%q enrolled", e)
		}
		noEnrollment(t, inst)
	}
	// a user row with no password (EnsureAdmin's shape) is not an identity
	if rc, _, _ := run(inst, enrollLine("admin@localhost"), "enroll", "admin@localhost"); rc == 0 {
		t.Error("an empty-password user enrolled")
	}
	noEnrollment(t, inst)
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 { // positive control
		t.Fatalf("positive control rc=%d %s", rc, errb)
	}
}

func TestEnrollOpensTheBotDBReadOnly(t *testing.T) {
	attended(t)
	inst := install(t)
	dbFile := DBFileFor(inst)
	before := fileSig(t, dbFile)
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("rc=%d %s", rc, errb)
	}
	if after := fileSig(t, dbFile); after != before {
		t.Fatalf("data.db changed: %s -> %s", before, after)
	}
	// SQLite may create an EMPTY -wal beside a WAL-mode DB it opens read-only
	// (the wal-index needs it); a non-empty one would be a written frame.
	if fi, err := os.Lstat(dbFile + "-wal"); err == nil && fi.Size() != 0 {
		t.Fatalf("the -wal holds %d bytes: something was written", fi.Size())
	}
}

// The lookup connection itself cannot write: mode=ro + query_only. Positive
// control: it reads.
func TestTheLookupConnectionCannotWrite(t *testing.T) {
	inst := install(t)
	db, err := openReadOnly(DBFileFor(inst))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("positive control: read users n=%d err=%v", n, err)
	}
	for _, stmt := range []string{
		`UPDATE users SET email = 'x@example.test'`,
		`INSERT INTO users (id, email, password_hash) VALUES ('z', 'z@example.test', 'h')`,
		`CREATE TABLE pwned (a)`,
		`PRAGMA query_only = 0`,
	} {
		if _, err := db.Exec(stmt); err == nil && !strings.HasPrefix(stmt, "PRAGMA") {
			t.Errorf("the read-only connection executed %q", stmt)
		}
	}
	// even with query_only lifted on that connection, mode=ro refuses
	if _, err := db.Exec(`DELETE FROM users`); err == nil {
		t.Error("the read-only connection deleted users")
	}
	for _, bad := range []string{"/x/data?.db", "/x/da#ta.db", "/x/d%41ta.db"} {
		if _, err := openReadOnly(bad); err == nil {
			t.Errorf("path %q accepted", bad)
		}
	}
}

func TestEnrollRefusesRoot(t *testing.T) {
	attended(t)
	inst := install(t)
	geteuid = func() int { return 0 }
	if rc, _, _ := run(inst, enrollLine(bEmail), "enroll", bEmail); rc == 0 {
		t.Fatal("root enrolled")
	}
	noEnrollment(t, inst)
	geteuid = func() int { return 1000 } // positive control
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("positive control rc=%d %s", rc, errb)
	}
}

func TestEnrollRefusesWithoutTheBotDatabase(t *testing.T) {
	attended(t)
	inst := install(t)
	empty := t.TempDir()
	if rc, _, errb := run(empty, enrollLine(bEmail), "enroll", bEmail); rc != 2 || !strings.Contains(errb, "no bot database") {
		t.Fatalf("no bot database: rc=%d %q, want the precondition refusal (rc 2)", rc, errb)
	}
	noEnrollment(t, empty)
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 { // positive control
		t.Fatalf("positive control rc=%d %s", rc, errb)
	}
}

func TestEnrollRequiresATerminal(t *testing.T) {
	attended(t)
	inst := install(t)
	isTerminal = stdinIsTerminal // the production check
	if rc, _, _ := run(inst, enrollLine(bEmail), "enroll", bEmail); rc == 0 {
		t.Fatal("a piped (non-terminal) stdin enrolled")
	}
	f, _ := os.CreateTemp(t.TempDir(), "stdin")
	_, _ = f.WriteString(enrollLine(bEmail))
	_, _ = f.Seek(0, 0)
	var out, errb bytes.Buffer
	if rc := Run([]string{"--install-dir", inst, "enroll", bEmail}, f, &out, &errb); rc == 0 {
		t.Fatal("a regular-file stdin enrolled")
	}
	f.Close()
	noEnrollment(t, inst)
	isTerminal = func(io.Reader) bool { return true } // positive control
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("positive control rc=%d %s", rc, errb)
	}
}

// The production TTY check answers true for a real terminal (a pty slave):
// the positive control for TestEnrollRequiresATerminal's refusals.
func TestTTYCheckRecognisesARealTerminal(t *testing.T) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no /dev/ptmx: %v", err)
	}
	defer m.Close()
	var unlock int32
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); e != 0 {
		t.Skipf("unlockpt: %v", e)
	}
	var n uint32
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); e != 0 {
		t.Skipf("ptsname: %v", e)
	}
	s, err := os.OpenFile("/dev/pts/"+itoa(int(n)), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("open slave: %v", err)
	}
	defer s.Close()
	if !stdinIsTerminal(s) {
		t.Fatal("a pty slave is not recognised as a terminal")
	}
	if stdinIsTerminal(strings.NewReader("x")) {
		t.Fatal("a non-file reader is a terminal")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func TestEnrollRequiresTheExactTypedConfirmation(t *testing.T) {
	attended(t)
	inst := install(t)
	for _, line := range []string{"", "\n", "y\n", "yes\n", "ENROLL\n", "enroll " + bEmail + "\n", "ENROLL " + bEmail + " \n",
		"ENROLL someone@example.test\n", " ENROLL " + bEmail + "\n", "ENROLL " + bEmail} {
		if rc, _, _ := run(inst, line, "enroll", bEmail); rc == 0 {
			t.Errorf("confirmation %q accepted", line)
		}
		noEnrollment(t, inst)
	}
	if rc, _, errb := run(inst, "ENROLL "+bEmail+"\r\n", "enroll", bEmail); rc != 0 { // positive control (CRLF terminal)
		t.Fatalf("positive control rc=%d %s", rc, errb)
	}
}

func TestReEnrollRequiresReplaceAndRotatesTheKey(t *testing.T) {
	attended(t)
	inst := install(t)
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("first enroll rc=%d %s", rc, errb)
	}
	d := DataDirFor(inst)
	a0, k0 := fileSig(t, updateauth.AdminPath(d)), fileSig(t, updateauth.DeviceKeyPath(d))
	key0, _ := updateauth.LoadDeviceKey(d)
	if rc, _, _ := run(inst, enrollLine(bEmail), "enroll", bEmail); rc == 0 {
		t.Fatal("re-enroll without --replace succeeded")
	}
	if fileSig(t, updateauth.AdminPath(d)) != a0 || fileSig(t, updateauth.DeviceKeyPath(d)) != k0 {
		t.Fatal("a refused re-enroll changed the enrollment")
	}
	for _, args := range [][]string{{"enroll", "--replace", bEmail}, {"enroll", bEmail, "--replace"}} {
		if rc, _, errb := run(inst, enrollLine(bEmail), args...); rc != 0 {
			t.Fatalf("%v rc=%d %s", args, rc, errb)
		}
		key1, _ := updateauth.LoadDeviceKey(d)
		if bytes.Equal(key0, key1) {
			t.Fatalf("%v did not rotate the key", args)
		}
		key0 = key1
	}
}

func TestEnrollUsage(t *testing.T) {
	attended(t)
	inst := install(t)
	for _, args := range [][]string{{}, {"enroll"}, {"enroll", bEmail, "extra"}, {"bogus"}, {"enroll", "--nope", bEmail}} {
		if rc, _, _ := run(inst, enrollLine(bEmail), args...); rc == 0 {
			t.Errorf("%v: rc 0", args)
		}
	}
	noEnrollment(t, inst)
}

// ── authorize ────────────────────────────────────────────────────────────

func enrolledInstall(t *testing.T) string {
	t.Helper()
	inst := install(t)
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("enroll rc=%d %s", rc, errb)
	}
	return inst
}

func authorizeLine(rel string) string { return "AUTHORIZE " + rel + "\n" }

// movedDB: an enrolled installation whose bot database is no longer where
// the resolver points (the --install-dir is not the bot's installation).
func movedDB(t *testing.T) string {
	t.Helper()
	inst := enrolledInstall(t)
	if err := os.Rename(DBFileFor(inst), DBFileFor(inst)+".moved"); err != nil {
		t.Fatal(err)
	}
	return inst
}

func TestAuthorizePrintsAGrantAndNeverTheKey(t *testing.T) {
	attended(t)
	inst := enrolledInstall(t)
	rc, out, errb := run(inst, authorizeLine(bRel), "authorize", bRel)
	if rc != 0 {
		t.Fatalf("rc=%d %s", rc, errb)
	}
	var m map[string]any
	dec := json.NewDecoder(strings.NewReader(out))
	if err := dec.Decode(&m); err != nil {
		t.Fatalf("stdout is not one JSON object: %q", out)
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if strings.Join(keys, ",") != "expires_at,hmac,job_id,release_id" {
		t.Fatalf("grant keys %v", keys)
	}
	g, err := updateauth.ParseInstallRequest(strings.NewReader(out))
	if err != nil {
		t.Fatalf("the printed grant is not a valid install body: %v", err)
	}
	if g.ReleaseID != bRel || g.ExpiresAt != bNow.Add(5*time.Minute).Unix() {
		t.Fatalf("grant %+v", g)
	}
	d := DataDirFor(inst)
	key, _ := updateauth.LoadDeviceKey(d)
	if !updateauth.VerifyMAC(key, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC) {
		t.Fatal("the printed MAC does not verify under device.key")
	}
	for _, s := range []string{out, errb} {
		if strings.Contains(s, hex.EncodeToString(key)) || strings.Contains(s, string(key)) {
			t.Fatal("authorize printed the device key")
		}
	}
	// two runs ⇒ two different job ids
	_, out2, _ := run(inst, authorizeLine(bRel), "authorize", bRel)
	g2, _ := updateauth.ParseInstallRequest(strings.NewReader(out2))
	if g2.JobID == g.JobID {
		t.Fatal("job id reused across authorizations")
	}
}

func TestAuthorizeRefusals(t *testing.T) {
	attended(t)
	inst := enrolledInstall(t)
	unenrolled := install(t)
	cases := []struct {
		name  string
		inst  string
		stdin string
		args  []string
		setup func()
	}{
		{"root", inst, authorizeLine(bRel), []string{"authorize", bRel}, func() { geteuid = func() int { return 0 } }},
		{"not a terminal", inst, authorizeLine(bRel), []string{"authorize", bRel}, func() { isTerminal = stdinIsTerminal }},
		{"no bot db (enrolled dir, DB moved away)", movedDB(t), authorizeLine(bRel), []string{"authorize", bRel}, nil},
		{"not enrolled", unenrolled, authorizeLine(bRel), []string{"authorize", bRel}, nil},
		{"wrong confirmation", inst, authorizeLine("v9"), []string{"authorize", bRel}, nil},
		{"no confirmation", inst, "", []string{"authorize", bRel}, nil},
		{"path-shaped release", inst, authorizeLine("../../x"), []string{"authorize", "../../x"}, nil},
		{"url release", inst, authorizeLine("https://x/y"), []string{"authorize", "https://x/y"}, nil},
		{"pipe release", inst, authorizeLine("a|b"), []string{"authorize", "a|b"}, nil},
		{"no release", inst, authorizeLine(""), []string{"authorize"}, nil},
		{"two releases", inst, authorizeLine(bRel), []string{"authorize", bRel, "v2"}, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			attended(t)
			if c.setup != nil {
				c.setup()
			}
			rc, out, _ := run(c.inst, c.stdin, c.args...)
			if rc == 0 || strings.Contains(out, "hmac") {
				t.Fatalf("rc=%d out=%q", rc, out)
			}
		})
	}
	// positive control: the same enrolled install, attended, exact confirmation
	if rc, _, errb := run(inst, authorizeLine(bRel), "authorize", bRel); rc != 0 {
		t.Fatalf("positive control rc=%d %s", rc, errb)
	}
	// a corrupted key file refuses (loaded under the gate's own rules)
	_ = os.Chmod(updateauth.DeviceKeyPath(DataDirFor(inst)), 0o644)
	if rc, out, _ := run(inst, authorizeLine(bRel), "authorize", bRel); rc == 0 || strings.Contains(out, "hmac") {
		t.Fatal("authorize minted from a 0644 key")
	}
}
