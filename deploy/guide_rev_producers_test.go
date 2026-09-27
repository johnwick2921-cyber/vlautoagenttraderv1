package deploy

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// CLASS 250. VITE_GUIDE_BUILT_REV is a REQUIRED build input: web/vite.config.ts
// refuses a production build without it. It shipped with a NEGATIVE proof (the
// refusal fires) and no POSITIVE one (every producer of the artifact supplies
// it), so the first producer to run — CI — failed on five checks.
//
// A guard is only as good as the census of its producers. This test IS that
// census, and it fails when a new producer appears without the input, which is
// the only way the next person to add one finds out before CI does.
//
// It deliberately does not try to parse YAML or Dockerfiles: it asks, for every
// place that runs a production frontend build, whether the input is supplied
// near enough to reach it (a step's `env:`, an `ARG`/`ENV` above the `RUN`, or
// on the command line itself).

// scanLookback is how far above an invocation the input may be declared: a
// step's env: block or an ARG/ENV pair sits within a few lines, never 25.
const scanLookback = 25

const guideRevVar = "VITE_GUIDE_BUILT_REV"

// notAProducer lists files that CONTAIN the string "npm run build" without ever
// running one. Each entry says why, because an unexplained exemption is how a
// real producer gets waved through later (CLASS 242's lesson about exemptions).
var notAProducer = map[string]string{
	// Emits advice text into a PR comment; the string is documentation for a
	// human, not a build this repo performs.
	".github/workflows/pr-checks-comment.yml": "generates comment text, runs no build",
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	return filepath.Dir(wd) // deploy/ -> repo root
}

// collectBuildSites scans one file for production frontend build invocations
// and reports which lack the guide rev. Two invocation kinds:
//   - "npm run build" lines: the rev must appear within scanLookback lines
//     (a step's env: or an ARG/ENV above the RUN);
//   - "Dockerfile.frontend" references (workflows/compose that build the image
//     through build-args): the rev must appear ANYWHERE in the file — the
//     build-args block can sit far from the dockerfile key (PR B [12]).
//
// docMode counts only copy-pastable commands (lines that also say "cd web") —
// a prose mention is not a producer.
func collectBuildSites(rel, content string, docMode bool) (scanned, missing []string) {
	lines := strings.Split(content, "\n")
	for i, ln := range lines {
		if strings.Contains(ln, "npm run build") {
			if strings.HasPrefix(strings.TrimSpace(ln), "#") {
				continue
			}
			if docMode && !strings.Contains(ln, "cd web") {
				continue
			}
			scanned = append(scanned, fmt.Sprintf("%s:%d", rel, i+1))
			lo := i - scanLookback
			if lo < 0 {
				lo = 0
			}
			if strings.Contains(strings.Join(lines[lo:i+1], "\n"), guideRevVar) {
				continue
			}
			missing = append(missing, fmt.Sprintf("%s:%d  %s", rel, i+1, strings.TrimSpace(ln)))
			continue
		}
		if !docMode && strings.Contains(ln, "Dockerfile.frontend") {
			scanned = append(scanned, fmt.Sprintf("%s:%d", rel, i+1))
			if strings.Contains(content, guideRevVar) {
				continue
			}
			missing = append(missing, fmt.Sprintf("%s:%d  %s", rel, i+1, strings.TrimSpace(ln)))
		}
	}
	return
}

// scanProducerFile reads one file at the repo root and folds its sites into the
// census — the Makefile and the documented install/contributor commands live
// OUTSIDE the three walked dirs, which was the [12] blind spot.
func scanProducerFile(t *testing.T, root, rel string, docMode bool, scanned, missing *[]string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	s, m := collectBuildSites(rel, string(b), docMode)
	*scanned = append(*scanned, s...)
	*missing = append(*missing, m...)
}

func TestEveryProductionFrontendBuildSuppliesTheGuideRev(t *testing.T) {
	root := repoRoot(t)
	var scanned, missing []string

	walk := func(dir string, keep func(string) bool) {
		_ = filepath.Walk(filepath.Join(root, dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !keep(p) {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			if why, skip := notAProducer[filepath.ToSlash(rel)]; skip {
				t.Logf("skipping %s — %s", rel, why)
				return nil
			}
			b, rerr := os.ReadFile(p)
			if rerr != nil {
				return nil
			}
			s, m := collectBuildSites(rel, string(b), false)
			scanned = append(scanned, s...)
			missing = append(missing, m...)
			return nil
		})
	}

	walk(".github/workflows", func(p string) bool { return strings.HasSuffix(p, ".yml") })
	walk("docker", func(p string) bool { return strings.Contains(filepath.Base(p), "Dockerfile") })
	walk("deploy", func(p string) bool { return strings.HasSuffix(p, ".sh") })
	// PR B [12]: producers outside the three walked dirs — the repo-root
	// Makefile and the DOCUMENTED copy-pastable build commands. A producer the
	// census cannot see is one that fails at build time with no warning here.
	scanProducerFile(t, root, "Makefile", false, &scanned, &missing)
	scanProducerFile(t, root, "INSTALL.md", true, &scanned, &missing)
	scanProducerFile(t, root, "CONTRIBUTING.md", true, &scanned, &missing)

	if len(scanned) == 0 {
		t.Fatal("scanned no production build call sites at all — the census is looking in the wrong place")
	}
	if len(missing) > 0 {
		t.Fatalf("%d production frontend build(s) do not supply %s (of %d scanned):\n  %s\n\n"+
			"Each must get it from the step's env:, an ARG/ENV above the RUN, or the command line. "+
			"web/vite.config.ts REFUSES a production build without it, so an unsupplied producer fails at build time.",
			len(missing), guideRevVar, len(scanned), strings.Join(missing, "\n  "))
	}
}

// TestComposeSuppliesTheGuideRevToTheFrontendImage covers the producer that
// runs no `npm run build` itself: compose builds Dockerfile.frontend, so the
// input has to cross the build-args boundary or the image is built unstamped.
func TestComposeSuppliesTheGuideRevToTheFrontendImage(t *testing.T) {
	root := repoRoot(t)
	hits, _ := filepath.Glob(filepath.Join(root, "docker-compose*.yml"))
	if len(hits) == 0 {
		t.Skip("no docker-compose files in this tree")
	}
	var bad []string
	for _, p := range hits {
		rel, _ := filepath.Rel(root, p)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", rel, err)
		}
		s := string(b)
		if !strings.Contains(s, "Dockerfile.frontend") {
			continue
		}
		if !strings.Contains(s, guideRevVar) {
			bad = append(bad, rel)
		}
	}
	if len(bad) > 0 {
		t.Fatalf("%d compose file(s) build Dockerfile.frontend without passing %s — they would build a "+
			"guide whose revision cannot be checked, or fail at build time:\n  %s",
			len(bad), guideRevVar, strings.Join(bad, "\n  "))
	}
}

// TestWorkflowsThatBuildTheFrontendImagePassTheBuildArg is the third boundary:
// a workflow can build the frontend image without ever typing "npm run build".
func TestWorkflowsThatBuildTheFrontendImagePassTheBuildArg(t *testing.T) {
	root := repoRoot(t)
	var bad []string
	hits, _ := filepath.Glob(filepath.Join(root, ".github/workflows/*.yml"))
	for _, p := range hits {
		rel, _ := filepath.Rel(root, p)
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := string(b)
		if !strings.Contains(s, "Dockerfile.frontend") {
			continue
		}
		if !strings.Contains(s, guideRevVar) {
			bad = append(bad, rel)
		}
	}
	if len(bad) > 0 {
		t.Fatalf("%d workflow(s) build Dockerfile.frontend without a %s build-arg:\n  %s",
			len(bad), guideRevVar, strings.Join(bad, "\n  "))
	}
}

// pathTokenRe pulls backtick-quoted path tokens out of a README table cell.
var pathTokenRe = regexp.MustCompile("`([^`]+)`")

// globRoot returns repo-relative paths matching a root-level glob.
func globRoot(t *testing.T, pattern string) []string {
	t.Helper()
	hits, err := filepath.Glob(filepath.Join(repoRoot(t), pattern))
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	var rel []string
	for _, p := range hits {
		r, _ := filepath.Rel(repoRoot(t), p)
		rel = append(rel, filepath.ToSlash(r))
	}
	return rel
}

// TestReadmeProducerTableMatchesTheCensus (PR B [12]) — the README's producers
// table must never drift from the census: every repo path the census scans as
// a producer must be named in the table, and every path row in the table must
// be a path the census actually scans. A table maintained by hand beside the
// test is how "make build-frontend" shipped unstamped while the table claimed
// completeness.
func TestReadmeProducerTableMatchesTheCensus(t *testing.T) {
	root := repoRoot(t)

	// census paths: the files the census scans for build invocations.
	census := map[string]bool{}
	walk := func(dir string, keep func(string) bool) {
		_ = filepath.Walk(filepath.Join(root, dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() || !keep(p) {
				return nil
			}
			rel, _ := filepath.Rel(root, p)
			if why, skip := notAProducer[filepath.ToSlash(rel)]; skip {
				_ = why
				return nil
			}
			b, rerr := os.ReadFile(p)
			if rerr != nil {
				return nil
			}
			if strings.Contains(string(b), "npm run build") || strings.Contains(string(b), "Dockerfile.frontend") {
				census[filepath.ToSlash(rel)] = true
			}
			return nil
		})
	}
	walk(".github/workflows", func(p string) bool { return strings.HasSuffix(p, ".yml") })
	walk("docker", func(p string) bool { return strings.Contains(filepath.Base(p), "Dockerfile") })
	walk("deploy", func(p string) bool { return strings.HasSuffix(p, ".sh") })
	for _, rel := range []string{"Makefile", "INSTALL.md", "CONTRIBUTING.md"} {
		if b, err := os.ReadFile(filepath.Join(root, rel)); err == nil && strings.Contains(string(b), "npm run build") {
			census[rel] = true
		}
	}
	// root compose files are producers only when they build the frontend image
	// (the compose supply test uses the same rule); the README names
	// docker-compose.yml, so the parity census must see it.
	for _, p := range globRoot(t, "docker-compose*.yml") {
		if b, err := os.ReadFile(filepath.Join(root, p)); err == nil && strings.Contains(string(b), "Dockerfile.frontend") {
			census[p] = true
		}
	}

	// README path rows: parse the producers table between the CLASS 250 header
	// and the next heading; a cell that names a repo path (contains "/" or
	// ".md" or "Makefile") is a path row.
	b, err := os.ReadFile(filepath.Join(root, "deploy/release/README.md"))
	if err != nil {
		t.Fatalf("read README: %v", err)
	}
	inTable := false
	readme := map[string]bool{}
	for _, ln := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(ln, "## Producers of ") {
			inTable = true
			continue
		}
		if inTable && strings.HasPrefix(ln, "## ") {
			break
		}
		if !inTable || !strings.HasPrefix(strings.TrimSpace(ln), "|") {
			continue
		}
		cell := strings.TrimSpace(strings.Split(strings.Trim(ln, "|"), "|")[0])
		// path tokens are backtick-quoted; a cell like "`Makefile` (`make
		// build-frontend`)" carries the path in the first quoted token.
		toks := pathTokenRe.FindAllStringSubmatch(cell, -1)
		if len(toks) == 0 {
			continue // descriptive row, not a path
		}
		for _, m := range toks {
			tok := m[1]
			// only path-like tokens count: "make build-frontend" in the same
			// cell is a command, not a producer path.
			if !strings.Contains(tok, "/") && !strings.Contains(tok, ".") && tok != "Makefile" {
				continue
			}
			readme[tok] = true
		}
	}

	var drift []string
	for p := range readme {
		if !census[p] {
			drift = append(drift, "README names "+p+" but the census does not scan it")
		}
	}
	for p := range census {
		if !readme[p] {
			drift = append(drift, "the census scans "+p+" but the README table does not name it")
		}
	}
	if len(drift) > 0 {
		t.Fatalf("README producers table drifted from the census (%d):\n  %s",
			len(drift), strings.Join(drift, "\n  "))
	}
}
