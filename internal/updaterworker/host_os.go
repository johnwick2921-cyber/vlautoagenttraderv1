package updaterworker

import (
	"bytes"
	"context"
	"debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ── the real machine ────────────────────────────────────────────────────────

// Seams for the start refusals (tests play root, the bot's cgroup and a TZ
// without being any of them).
var (
	geteuid    = os.Geteuid
	readCgroup = func() ([]byte, error) { return os.ReadFile("/proc/self/cgroup") }
	lookupTZ   = func() (string, bool) { return os.LookupEnv("TZ") }
)

var (
	// ErrRoot: the worker never runs as root (a root-owned data/updater would
	// lock the bot into a hold it cannot read — the hold CLI's M2.1 N4 rule).
	ErrRoot = errors.New("updaterworker: refusing to run as root: run nofx-updater as the bot's own user")
	// ErrBotCgroup: the worker must never share the bot's control group — the
	// unit is KillMode=control-group, so the activation's kill would kill the
	// worker with the bot, mid-job.
	ErrBotCgroup = errors.New("updaterworker: refusing to run inside the nofx.service control group")
	// ErrTZ: the library parses log line times in time.Local; a worker whose
	// zone differs from the bot's would misjudge every boot line's time.
	ErrTZ = errors.New("updaterworker: refusing to run with TZ set: the worker must read log times in the bot's own local zone")
)

// CheckProcess is the worker's start refusals: root, the bot's cgroup, TZ.
func CheckProcess() error {
	if geteuid() == 0 {
		return ErrRoot
	}
	if b, err := readCgroup(); err == nil && bytes.Contains(b, []byte("/nofx.service")) {
		return ErrBotCgroup
	}
	if _, set := lookupTZ(); set {
		return ErrTZ
	}
	return nil
}

// OSHost is the production Host.
type OSHost struct {
	// LockScript is <install>/deploy/nofx-lock.sh; only its `check` verb is
	// ever run (rc 1 = held, per the lock-verbs canon). Never acquire.
	LockScript string
}

func (OSHost) Now() time.Time { return time.Now() }

func (OSHost) Sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// MainTreeLockHeld runs `bash <LockScript> check` and reads ONLY its exit
// code: 0 free · 1 held · 2 stale · 3 incomplete · 4 abandoned-incomplete. Only
// rc 1 is "held" (a stale or incomplete lock is not a working owner).
func (h OSHost) MainTreeLockHeld() (bool, string, error) {
	if h.LockScript == "" || !filepath.IsAbs(h.LockScript) {
		return false, "", errors.New("no lock script configured")
	}
	if _, err := os.Stat(h.LockScript); err != nil {
		return false, "", fmt.Errorf("lock script: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", h.LockScript, "check")
	cmd.Stdout, cmd.Stderr = nil, nil
	err := cmd.Run()
	rc := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			return false, "", fmt.Errorf("lock check: %w", err)
		}
		rc = ee.ExitCode()
	}
	return rc == 1, "check rc=" + strconv.Itoa(rc), nil
}

// BuildInfo reads the binary's vcs stamps with debug/buildinfo (never by
// parsing `go version -m` text — CLASS 241).
func (OSHost) BuildInfo(binary string) (string, string, error) {
	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		return "", "", err
	}
	var rev, mod string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			mod = s.Value
		}
	}
	return rev, mod, nil
}

// ExeOf is the target of /proc/<pid>/exe (" (deleted)" kept: a replaced
// binary is not the install's binary).
func (OSHost) ExeOf(pid int) (string, error) {
	if pid <= 0 {
		return "", fmt.Errorf("pid %d", pid)
	}
	return os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
}

// firstMarkerLine is the first non-empty, non-comment line of a RELEASE
// marker (kernel/boot_integrity.go expectedRevision's rule).
func firstMarkerLine(b []byte) string {
	for _, ln := range strings.Split(string(b), "\n") {
		ln = strings.TrimSpace(ln)
		if ln != "" && !strings.HasPrefix(ln, "#") {
			return ln
		}
	}
	return ""
}

// revisionsAgree: the REPORTED revision may abbreviate the EXPECTED one (≥7
// chars, never longer) — activation's rule (health reports 12 chars, a RELEASE
// marker may be short).
func revisionsAgree(reported, expected string) bool {
	if reported == "" || expected == "" {
		return false
	}
	if reported == expected {
		return true
	}
	if len(reported) < 7 || len(reported) >= len(expected) {
		return false
	}
	return strings.HasPrefix(expected, reported)
}
