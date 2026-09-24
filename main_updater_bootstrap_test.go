package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"nofx/config"
	"nofx/internal/updateauth"
	"nofx/internal/updaterbootstrap"
	"nofx/store"
)

// W-ONE-BUTTON M3 — the production call-site proof for the enrollment CLI
// (the M2 MUST-2 shape): with DB_PATH set ONLY in .env, the data dir the bot
// resolves (loadDotEnv → config.Init → resolveMaintenanceDataDir, the calls
// main makes before api.NewServer; the API gate reads the same dir through
// trader.MaintenanceDataDir) is the dir the attended CLI enrolls into — run
// from another directory, through its real Run, on a real terminal (a pty),
// with no test seam.
func TestUpdaterBootstrapEnrollsWhereTheBotReads(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the CLI refuses root by design")
	}
	const email = "owner@example.test"
	inst := t.TempDir()
	if err := os.WriteFile(filepath.Join(inst, ".env"), []byte("DB_PATH=var/db/data.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dbFile := filepath.Join(inst, "var", "db", "data.db")
	if err := os.MkdirAll(filepath.Dir(dbFile), 0o700); err != nil {
		t.Fatal(err)
	}
	st, err := store.New(dbFile)
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour)
	if err := st.User().Create(&store.User{ID: "dddddddd-1111-2222-3333-444444444444", Email: email, PasswordHash: "non-empty-hash", CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	st.Plan().Close()
	_ = st.Close()
	t.Setenv("DB_PATH", "placeholder")
	os.Unsetenv("DB_PATH") // DB_PATH comes ONLY from .env, as on a real install

	// ── the bot, from its WorkingDirectory ──
	t.Chdir(inst)
	loadDotEnv(".env")
	config.Init()
	botDir := resolveMaintenanceDataDir(config.Get().DBPath)
	if want := filepath.Join(inst, "var", "db"); botDir != want {
		t.Fatalf("bot data dir = %q, want %q", botDir, want)
	}
	if _, err := updateauth.LoadAdmin(botDir); err == nil {
		t.Fatal("precondition: nothing enrolled")
	}

	// ── the operator, attended, from somewhere else ──
	t.Chdir(t.TempDir())
	os.Unsetenv("DB_PATH")
	master, slave := openPTY(t)
	if _, err := master.Write([]byte("ENROLL " + email + "\n")); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	if rc := updaterbootstrap.Run([]string{"--install-dir", inst, "enroll", email}, slave, &out, &errb); rc != 0 {
		t.Fatalf("enroll rc=%d: %s", rc, errb.String())
	}
	a, err := updateauth.LoadAdmin(botDir)
	if err != nil {
		t.Fatalf("the bot's dir %s does not hold the enrollment the CLI wrote: %v", botDir, err)
	}
	if a.Email != email || a.UserID != "dddddddd-1111-2222-3333-444444444444" {
		t.Fatalf("admin = %+v", a)
	}
	if _, err := updateauth.LoadDeviceKey(botDir); err != nil {
		t.Fatalf("device key: %v", err)
	}
}

// openPTY returns a pty master and its slave (a real terminal); skips when
// the environment has none.
func openPTY(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("no /dev/ptmx: %v", err)
	}
	t.Cleanup(func() { m.Close() })
	var unlock int32
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), syscall.TIOCSPTLCK, uintptr(unsafe.Pointer(&unlock))); e != 0 {
		t.Skipf("unlockpt: %v", e)
	}
	var n uint32
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, m.Fd(), syscall.TIOCGPTN, uintptr(unsafe.Pointer(&n))); e != 0 {
		t.Skipf("ptsname: %v", e)
	}
	s, err := os.OpenFile("/dev/pts/"+strconv.Itoa(int(n)), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		t.Skipf("open pty slave: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return m, s
}
