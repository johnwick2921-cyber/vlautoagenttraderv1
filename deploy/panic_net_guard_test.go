package deploy

// panic-net-complete (2026-09-26) — the guard that keeps the net complete.
//
// Every bare `go func` / `go x()` in PRODUCT non-test code must sit under a
// panic net (safe.GoNet / safe.Go / safe.GoNamed / the telemetry goTracked
// inline net) or be in the allow-list below WITH A REASON. A NEW long-lived
// goroutine launched without the net fails this test and names itself —
// that is the regression guard, not a style check.
//
// Allow-list (each entry names the reason):
//   - safe/*.go                          the net's own launches (the net cannot
//                                        wrap itself)
//   - telemetry/experience.go goTracked  the inline net's launch (safe would be
//                                        an import cycle: safe counts via
//                                        telemetry)
//   - provider/ninjatrader/*.go and trader/ninjatrader/tcp_trader.go
//                                        DS-102's in-flight double-entry wave —
//                                        DO NOT TOUCH; listed for DS-102 to net
//                                        in their wave.

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var netGuardBareGoRe = regexp.MustCompile(`^\s*go (func|\w[\w.]*\()`)

// netGuardAllowedDS102 are DS-102's in-flight files: every bare launch there is
// tolerated and LISTED for DS-102 (never edited by this wave).
var netGuardAllowedDS102 = map[string]bool{
	"provider/ninjatrader/tcp_server.go":      true,
	"provider/ninjatrader/tcp_trader.go":      true,
	"provider/ninjatrader/ordered_exec.go":    true,
	"provider/ninjatrader/bar_persist.go":     true,
	"provider/ninjatrader/bar_source.go":      true,
	"provider/ninjatrader/bar_live_sink.go":  true,
	"provider/ninjatrader/tcp_client_mock.go": true,
	"provider/ninjatrader/mock_nt.go":         true,
	"trader/ninjatrader/tcp_trader.go":        true,
}

func TestNoBareGoroutineOutsideTheNet(t *testing.T) {
	root := ".."
	var hits []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			switch path {
			case filepath.Join(root, "docs"), filepath.Join(root, ".audit"),
				filepath.Join(root, ".git"), filepath.Join(root, "web", "node_modules"):
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		src := string(content)

		// safe/*.go is the net itself — every launch inside is the wrapper.
		if strings.HasPrefix(rel, "safe/") {
			return nil
		}
		// telemetry: allow only the launch inside goTracked (the inline net).
		telemetryAllowFrom, telemetryAllowTo := -1, -1
		if rel == "telemetry/experience.go" {
			if i := strings.Index(src, "func goTracked("); i >= 0 {
				telemetryAllowFrom = strings.Count(src[:i], "\n")
				if j := strings.Index(src[i:], "\n}\n"); j >= 0 {
					telemetryAllowTo = strings.Count(src[:i+j+1], "\n")
				}
			}
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		line := 0
		for sc.Scan() {
			line++
			txt := sc.Text()
			trim := strings.TrimSpace(txt)
			if strings.HasPrefix(trim, "//") || strings.HasPrefix(trim, "*") {
				continue
			}
			if !netGuardBareGoRe.MatchString(txt) {
				continue
			}
			if rel == "telemetry/experience.go" && line > telemetryAllowFrom && line <= telemetryAllowTo {
				continue
			}
			if netGuardAllowedDS102[rel] {
				continue
			}
			hits = append(hits, rel+":"+strconvItoa(line)+" — "+trim)
		}
		return sc.Err()
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) > 0 {
		t.Fatalf("bare goroutine launches outside the panic net (%d) — wrap in safe.GoNet/GoNamed or allow-list with a reason:\n  %s",
			len(hits), strings.Join(hits, "\n  "))
	}
}

func strconvItoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
