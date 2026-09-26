package updaterworker

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// PIN: the worker refuses to run as root (a root-owned data/updater would lock
// the bot into a hold it cannot read), inside the bot's control group (the
// unit's KillMode=control-group would kill the worker with the bot, mid-job),
// and with TZ set (log line times are the bot's local zone). CheckProcess is
// what `nofx-updater serve` calls first (cmd TestEverySubcommandRefusesRoot).
func TestWorkerRefusesRoot(t *testing.T) {
	origE, origC, origT := geteuid, readCgroup, lookupTZ
	t.Cleanup(func() { geteuid, readCgroup, lookupTZ = origE, origC, origT })
	user := func() int { return 1000 }
	plainCgroup := func() ([]byte, error) { return []byte("0::/user.slice/user-1000.slice/session-3.scope\n"), nil }
	noTZ := func() (string, bool) { return "", false }
	for name, c := range map[string]struct {
		euid   func() int
		cgroup func() ([]byte, error)
		tz     func() (string, bool)
		want   error
	}{
		"an ordinary user":          {user, plainCgroup, noTZ, nil},
		"root":                      {func() int { return 0 }, plainCgroup, noTZ, ErrRoot},
		"inside nofx.service":       {user, func() ([]byte, error) { return []byte("0::/system.slice/nofx.service\n"), nil }, noTZ, ErrBotCgroup},
		"TZ set":                    {user, plainCgroup, func() (string, bool) { return "America/Chicago", true }, ErrTZ},
		"TZ set empty is still set": {user, plainCgroup, func() (string, bool) { return "", true }, ErrTZ},
	} {
		geteuid, readCgroup, lookupTZ = c.euid, c.cgroup, c.tz
		if err := CheckProcess(); !errors.Is(err, c.want) || (c.want == nil && err != nil) {
			t.Errorf("%s: CheckProcess = %v, want %v", name, err, c.want)
		}
	}
}

// C19 as ruled: the worker READS the main-tree lock and never takes it — the
// only verb it ever runs is `check`, and only rc 1 (held) passes; free (0),
// stale (2), incomplete (3) and abandoned-incomplete (4) all refuse.
func TestLockCheckRunsCheckOnlyAndOnlyRc1IsHeld(t *testing.T) {
	dir := t.TempDir()
	argsFile := filepath.Join(dir, "args")
	for rc, want := range map[int]bool{0: false, 1: true, 2: false, 3: false, 4: false} {
		script := filepath.Join(dir, "nofx-lock.sh")
		body := fmt.Sprintf("#!/bin/bash\necho \"$@\" >> %q\nexit %d\n", argsFile, rc)
		if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
		held, detail, err := OSHost{LockScript: script}.MainTreeLockHeld()
		if err != nil || held != want || detail != fmt.Sprintf("check rc=%d", rc) {
			t.Fatalf("rc %d: held=%v detail=%q err=%v, want held=%v", rc, held, detail, err, want)
		}
	}
	b, _ := os.ReadFile(argsFile)
	for _, ln := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		if ln != "check" {
			t.Fatalf("the worker ran the lock script with %q — only `check` is ever allowed", ln)
		}
	}
	if _, _, err := (OSHost{LockScript: filepath.Join(dir, "absent.sh")}).MainTreeLockHeld(); err == nil {
		t.Fatal("an absent lock script read as a verdict")
	}
	if _, _, err := (OSHost{}).MainTreeLockHeld(); err == nil {
		t.Fatal("no lock script read as a verdict")
	}
}
