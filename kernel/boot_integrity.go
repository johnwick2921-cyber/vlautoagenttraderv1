package kernel

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync/atomic"
)

// P1 — BOOT INTEGRITY ASSERTION (the Knight-Capital control).
//
// This system shipped a stale binary twice in one weekend: a rebuild that the
// running process never picked up, and a restart that silently didn't take. Both
// times the operator believed new code was live when it was not. The fix is to
// make the binary state an ASSERTION at startup rather than something a human has
// to remember to check:
//
//	· read the revision + build time EMBEDDED in this binary (runtime/debug)
//	· compare against the INTENDED release (NOFX_EXPECTED_REVISION, or
//	  deploy/RELEASE — whichever is set; a prefix match so short SHAs work)
//	· re-render the prompt goldens embedded in this same binary
//
// Any failure ⇒ TRADING IS REFUSED: entries are blocked, a loud P0 alert is
// raised, and the rest of the process runs read-only (the dashboard, the API and
// the planner reads all keep working, so the owner can see WHY it refused).
//
// Setting no expectation is allowed and is NOT a failure — it logs the revision
// and proceeds. Assertion only bites when the owner has declared an intent.

// tradingRefused is set at boot and read by the entry gate. atomic so the gate
// can read it from any goroutine without a lock.
var tradingRefused atomic.Bool

// refusalReason explains the refusal for logs, alerts and the API.
var refusalReason atomic.Value // string

// BootIntegrity is the result of the startup assertion.
type BootIntegrity struct {
	Revision   string // vcs.revision embedded in this binary ("" if built without VCS)
	BuildTime  string // vcs.time
	Modified   bool   // vcs.modified (dirty tree at build time)
	Expected   string // the intended release, if one was declared
	RevisionOK bool
	GoldensOK  bool
	Goldens    []GoldenSelfCheckResult
	Refused    bool
	Reason     string
}

// Line renders the single boot log line the owner should look for.
func (b BootIntegrity) Line() string {
	rev := b.Revision
	if rev == "" {
		rev = "<no-vcs>"
	} else if len(rev) > 12 {
		rev = rev[:12]
	}
	status := "OK"
	if b.Refused {
		status = "REFUSED"
	}
	exp := b.Expected
	if exp == "" {
		exp = "<unset>"
	} else if len(exp) > 12 {
		exp = exp[:12]
	}
	dirty := ""
	if b.Modified {
		dirty = " +dirty"
	}
	return fmt.Sprintf("🔐 BOOT INTEGRITY %s — rev %s%s · built %s · expected %s · goldens %s",
		status, rev, dirty, b.BuildTime, exp, goldensWord(b.GoldensOK))
}

func goldensWord(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

// expectedRevision resolves the intended release: NOFX_EXPECTED_REVISION wins,
// else the first line of deploy/RELEASE. Empty means "no expectation declared".
func expectedRevision() string {
	if v := strings.TrimSpace(os.Getenv("NOFX_EXPECTED_REVISION")); v != "" {
		return v
	}
	if b, err := os.ReadFile("deploy/RELEASE"); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			ln = strings.TrimSpace(ln)
			if ln != "" && !strings.HasPrefix(ln, "#") {
				return ln
			}
		}
	}
	return ""
}

// buildStamp reads the revision/time/dirty flag embedded by the Go toolchain.
func buildStamp() (rev, when string, modified bool) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", "", false
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.time":
			when = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return rev, when, modified
}

// AssertBootIntegrity runs the assertion and latches the refusal state.
// Call once at startup, before traders begin cycling.
func AssertBootIntegrity() BootIntegrity {
	rev, when, modified := buildStamp()
	exp := expectedRevision()
	goldens, goldensOK := VerifyPromptGoldens()

	b := BootIntegrity{
		Revision: rev, BuildTime: when, Modified: modified, Expected: exp,
		Goldens: goldens, GoldensOK: goldensOK,
		RevisionOK: true, // no expectation declared ⇒ nothing to violate
	}
	var reasons []string
	if exp != "" {
		// prefix match so a short SHA in deploy/RELEASE matches the full one
		b.RevisionOK = rev != "" && strings.HasPrefix(rev, exp)
		if !b.RevisionOK {
			reasons = append(reasons, fmt.Sprintf("binary is revision %q but the intended release is %q — a stale binary is running", short(rev), short(exp)))
		}
	}
	if !goldensOK {
		for _, g := range goldens {
			if !g.OK {
				reasons = append(reasons, "prompt golden "+g.Name+" drifted: "+g.Detail)
			}
		}
	}
	if len(reasons) > 0 {
		b.Refused = true
		b.Reason = strings.Join(reasons, " · ")
		tradingRefused.Store(true)
		refusalReason.Store(b.Reason)
	} else {
		tradingRefused.Store(false)
		refusalReason.Store("")
	}
	lastBootRevision.Store(shortRev(b.Revision))
	return b
}

func short(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	if s == "" {
		return "<none>"
	}
	return s
}

// TradingRefused reports whether boot integrity failed. The ENTRY GATE reads this:
// true ⇒ no new position may be opened, for any trader, until the operator fixes
// the deploy and restarts. Closes and read-only paths are never gated.
func TradingRefused() (string, bool) {
	if !tradingRefused.Load() {
		return "", false
	}
	r, _ := refusalReason.Load().(string)
	if r == "" {
		r = "boot integrity assertion failed"
	}
	return r, true
}

// SetTradingRefusedForTest lets tests drive both paths. Test-only by convention.
func SetTradingRefusedForTest(refused bool, reason string) {
	tradingRefused.Store(refused)
	refusalReason.Store(reason)
}


// lastBootRevision caches the asserted revision for API exposure (v1 audit
// §5.6 — bug reports could not be checked against the running rev without a
// shell). Set once at boot.
var lastBootRevision atomic.Value // string

// RunningRevision returns the short revision of the running binary ("" before
// the boot assertion runs).
func RunningRevision() string {
	if v, ok := lastBootRevision.Load().(string); ok {
		return v
	}
	return ""
}


func shortRev(r string) string {
	if len(r) > 12 {
		return r[:12]
	}
	return r
}
