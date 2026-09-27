package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/config"
	"nofx/internal/updateauth"
	"nofx/internal/updaterbootstrap"
	"nofx/store"
)

// PR #200 fold F3 at the production call sites: the bot resolves its data
// dir the way main does (loadDotEnv → config.Init → resolveMaintenanceDataDir,
// with NO DB_PATH in its own environment, as the shipped systemd unit starts
// it). The operator's shell exports a DB_PATH naming ANOTHER existing,
// seeded bot database. The attended CLI (real Run, real pty, no seam) must
// refuse — writing nothing in either dir — and, with the variable unset,
// enroll exactly where the bot reads.
func TestUpdaterBootstrapRefusesAShellDBPathTheBotDoesNotUse(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("the CLI refuses root by design")
	}
	const email = "owner@example.test"
	const uid = "eeeeeeee-1111-2222-3333-444444444444"
	inst := t.TempDir()
	if err := os.WriteFile(filepath.Join(inst, ".env"), []byte("DB_PATH=var/db/data.db\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{"var/db/data.db", "data/data.db"} { // the bot's, and the diversion target
		dbFile := filepath.Join(inst, rel)
		if err := os.MkdirAll(filepath.Dir(dbFile), 0o700); err != nil {
			t.Fatal(err)
		}
		st, err := store.New(dbFile)
		if err != nil {
			t.Fatal(err)
		}
		past := time.Now().Add(-time.Hour)
		if err := st.User().Create(&store.User{ID: uid, Email: email, PasswordHash: "non-empty-hash", CreatedAt: past, UpdatedAt: past}); err != nil {
			t.Fatal(err)
		}
		st.Plan().Close()
		_ = st.Close()
	}
	t.Setenv("DB_PATH", "placeholder")
	os.Unsetenv("DB_PATH") // the bot's own environment has no DB_PATH

	// ── the bot, from its WorkingDirectory ──
	t.Chdir(inst)
	loadDotEnv(".env")
	config.Init()
	botDir := resolveMaintenanceDataDir(config.Get().DBPath)
	if want := filepath.Join(inst, "var", "db"); botDir != want {
		t.Fatalf("bot data dir = %q, want %q", botDir, want)
	}
	diverted := filepath.Join(inst, "data")

	// ── the operator, attended, from elsewhere, with DB_PATH exported ──
	t.Chdir(t.TempDir())
	os.Setenv("DB_PATH", "data/data.db")
	master, slave := openPTY(t)
	if _, err := master.Write([]byte("ENROLL " + email + "\n")); err != nil {
		t.Fatal(err)
	}
	var out, errb bytes.Buffer
	rc := updaterbootstrap.Run([]string{"--install-dir", inst, "enroll", email}, slave, &out, &errb)
	if rc != 2 {
		t.Fatalf("a shell DB_PATH the bot does not use: rc=%d stdout=%q stderr=%s", rc, out.String(), errb.String())
	}
	for _, want := range []string{`DB_PATH="data/data.db" in this process's environment`, filepath.Join(diverted, "data.db"),
		`DB_PATH="var/db/data.db" in ` + filepath.Join(inst, ".env"), filepath.Join(botDir, "data.db")} {
		if !strings.Contains(errb.String(), want) {
			t.Errorf("the refusal does not name %q:\n%s", want, errb.String())
		}
	}
	for _, d := range []string{botDir, diverted} {
		if _, err := os.Lstat(updateauth.Dir(d)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("a refused enroll created %s", updateauth.Dir(d))
		}
	}

	// positive control: the same operator with DB_PATH unset (the refused
	// run did not read the typed line; it is still pending on the pty).
	os.Unsetenv("DB_PATH")
	out.Reset()
	errb.Reset()
	if rc := updaterbootstrap.Run([]string{"--install-dir", inst, "enroll", email}, slave, &out, &errb); rc != 0 {
		t.Fatalf("positive control rc=%d: %s", rc, errb.String())
	}
	if a, err := updateauth.LoadAdmin(botDir); err != nil || a.UserID != uid {
		t.Fatalf("the bot's dir %s does not hold the enrollment: %+v %v", botDir, a, err)
	}
	if _, err := os.Lstat(updateauth.Dir(diverted)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the positive control wrote the diverted dir %s", updateauth.Dir(diverted))
	}
}
