package activation

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// system is the seam between this package and the machine. It exists so the
// refusals can be TESTED — a recycled pid, a process that never comes back, a
// boot line that predates the kill — none of which can be produced reliably by
// signalling real processes in a test.
//
// It is a package-level var rather than a parameter because §2's signatures are
// the contract the worker (3b-B) imports; the seam must not leak into them.
type system struct {
	// ReadStat returns the raw /proc/<pid>/stat line.
	ReadStat func(pid int) (string, error)
	// Kill sends SIGKILL. SIGTERM exits 0 and systemd's Restart=on-failure
	// does NOT relaunch, so the process would simply stay down.
	Kill func(pid int) error
	// MainPID reads systemd's notion of the unit's main process. NEVER pgrep:
	// `pgrep -f nofx-bin` also matches `go version -m nofx-bin`, and a pattern
	// can match the very shell that runs it (CLASS 242).
	MainPID func() (int, error)
	Now     func() time.Time
	Sleep   func(time.Duration)
}

func defaultSystem() *system {
	return &system{
		ReadStat: func(pid int) (string, error) {
			b, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
			return string(b), err
		},
		Kill: func(pid int) error { return syscall.Kill(pid, syscall.SIGKILL) },
		MainPID: func() (int, error) {
			out, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", "nofx").Output()
			if err != nil {
				return 0, fmt.Errorf("systemctl show MainPID: %w", err)
			}
			s := strings.TrimSpace(string(out))
			pid, err := strconv.Atoi(s)
			if err != nil {
				return 0, fmt.Errorf("MainPID %q is not a pid: %w", s, err)
			}
			if pid == 0 {
				return 0, fmt.Errorf("MainPID is 0 — the unit is not running")
			}
			return pid, nil
		},
		Now:   time.Now,
		Sleep: time.Sleep,
	}
}

var sys = defaultSystem()

// CurrentIdentity reads the running unit's identity: its MainPID together with
// the start time that distinguishes it from any later process reusing that pid.
func CurrentIdentity() (Identity, error) {
	pid, err := sys.MainPID()
	if err != nil {
		return Identity{}, err
	}
	line, err := sys.ReadStat(pid)
	if err != nil {
		return Identity{}, fmt.Errorf("cannot read /proc/%d/stat: %w", pid, err)
	}
	ticks, err := statField22(line)
	if err != nil {
		return Identity{}, fmt.Errorf("pid %d: %w", pid, err)
	}
	return Identity{PID: pid, StartTicks: ticks}, nil
}

// stillAlive reports whether the pid is STILL the process this Identity names.
// A pid that has been recycled is a DIFFERENT process wearing the same number,
// and signalling it would kill an innocent bystander.
func (id Identity) stillAlive() (bool, error) {
	line, err := sys.ReadStat(id.PID)
	if err != nil {
		return false, nil // gone
	}
	ticks, err := statField22(line)
	if err != nil {
		return false, err
	}
	return ticks == id.StartTicks, nil
}
