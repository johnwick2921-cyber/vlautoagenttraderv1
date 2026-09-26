package updaterworker

// PARITY PIN (#206 fold verdict on 5a6492a4): the boot-line matchers must stay
// in lockstep with the app's REAL producers — kernel.BootIntegrity.Line() for
// the message and logger's compactFormatter for the prefix. The other cases
// use hand-typed fixtures, so a separator change in Line() or the formatter
// would silently stop recognising real OK/REFUSED lines while every worker
// test stays green. Here the OK and REFUSED lines are RENDERED by the
// production producers, with a real pid (os.Getpid() inside Line()). The one
// substituted piece is the caller token: the formatter derives it from the
// call stack, which in a test is this file, not main.go — its shape is pinned
// separately by the fixture cases. RED = change one separator in Line() or the
// formatter.

import (
	"os"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/logger"

	"github.com/sirupsen/logrus"
)

// renderBoot renders b through the production formatter and swaps the caller
// token for a main.go-shaped one.
func renderBoot(t *testing.T, b kernel.BootIntegrity, level logrus.Level) string {
	t.Helper()
	entry := &logrus.Entry{Level: level, Message: b.Line(), Time: time.Now()}
	out, err := logger.Log.Formatter.Format(entry)
	if err != nil {
		t.Fatalf("the production formatter failed: %v", err)
	}
	line := string(out)
	// The real prefix is "MM-DD HH:MM:SS [LEVEL] <caller> <msg>": replace only
	// the caller token between the level and the 🔐 message.
	idx := strings.Index(line, "] ")
	if idx < 0 {
		t.Fatalf("the formatter output %q is not <ts> [LEVEL] <caller> <msg>", line)
	}
	rest := line[idx+2:]
	sp := strings.IndexByte(rest, ' ')
	if sp < 0 {
		t.Fatalf("the formatter output %q has no space after the caller", line)
	}
	return line[:idx+2] + "nofx/main.go:322" + rest[sp:]
}

func TestBootLineParityWithTheAppProducer(t *testing.T) {
	sha := strings.Repeat("ab", 20)
	pid := os.Getpid() // kernel.BootIntegrity.Line() prints the REAL pid — the one the matcher must see
	ok := kernel.BootIntegrity{Revision: sha, Expected: sha, BuildTime: "2026-09-24T00:00:00Z", GoldensOK: true}
	refused := ok
	refused.Refused = true

	okLine := renderBoot(t, ok, logrus.InfoLevel)
	refusedLine := renderBoot(t, refused, logrus.ErrorLevel)
	t.Run("the app's OK boot line", func(t *testing.T) {
		if !isBootOKLine(okLine, sha[:12], pid) {
			t.Fatalf("the matcher rejects the app's own rendered boot line:\n%q", okLine)
		}
		if isBootOKLine(okLine, sha[:12], pid+1) {
			t.Fatalf("the matcher accepts the app's boot line for pid+1:\n%q", okLine)
		}
	})
	t.Run("the app's REFUSED boot line", func(t *testing.T) {
		if !isBootRefusedLine(refusedLine, pid) {
			t.Fatalf("the matcher rejects the app's own rendered boot line:\n%q", refusedLine)
		}
		if isBootRefusedLine(refusedLine, pid+1) {
			t.Fatalf("the matcher accepts the app's boot line for pid+1:\n%q", refusedLine)
		}
	})
	// Sanity: the rendered line really does carry the production separator
	// shape the regexes demand (guards the caller-substitution above).
	line := renderBoot(t, ok, logrus.InfoLevel)
	for _, want := range []string{" · pid ", " · built ", " · expected " + sha[:12] + " ·"} {
		if !strings.Contains(line, want) {
			t.Fatalf("the rendered boot line lacks %q:\n%q", want, line)
		}
	}
}
