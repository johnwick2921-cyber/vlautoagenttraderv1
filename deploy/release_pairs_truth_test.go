package deploy

// W-PR-B fold [4] — the dbcompat job's rollback pair must be DERIVED from the
// tag being released, never a hard-coded literal.
//
// Class-250 probe: THIS TEST IS THE CALLER. It extracts the REAL `run:` body
// of the "Prove each rollback pair" step out of .github/workflows/release.yml
// and executes those exact lines against a fixture git repo with v* tags.
// The mutation that proves it: reintroducing a hard-coded
// `PAIRS: '0e490e44:662c79bd'` line (a) fails the no-literal gate below and
// (b) fails the derived-pair assertion, because the fixture's tag shas are
// not 0e490e44/662c79bd. A workflow that hard-codes pairs cannot pass.
//
// Truth rules pinned (canon 49/53 — unprovable ⇒ [], never fabricated):
//   new = the tag being released, read from refs/tags (40-hex, never assumed)
//   old = the last released sha (ROLLBACK_FROM input, else the highest v* tag
//         below the one being released), 40-hex on BOTH sides
//   no provable old/new ⇒ the step emits [] and exits 0

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stepRunBody returns the `run: |` body of the workflow step whose `name:`
// matches, indentation stripped. The workflow file is the production source —
// nothing is re-typed here.
func stepRunBody(t *testing.T, yaml, stepName string) string {
	t.Helper()
	lines := strings.Split(yaml, "\n")
	stepIdx := -1
	for i, l := range lines {
		if strings.Contains(l, "- name: "+stepName) {
			stepIdx = i
			break
		}
	}
	if stepIdx < 0 {
		t.Fatalf("workflow step %q not found", stepName)
	}
	runIdx := -1
	for i := stepIdx; i < len(lines); i++ {
		if strings.HasPrefix(strings.TrimSpace(lines[i]), "run: |") {
			runIdx = i
			break
		}
	}
	if runIdx < 0 {
		t.Fatalf("step %q has no run block", stepName)
	}
	indent := len(lines[runIdx]) - len(strings.TrimLeft(lines[runIdx], " "))
	var body []string
	for i := runIdx + 1; i < len(lines); i++ {
		l := lines[i]
		if strings.TrimSpace(l) == "" {
			continue
		}
		cur := len(l) - len(strings.TrimLeft(l, " "))
		if cur <= indent {
			break
		}
		body = append(body, l)
	}
	if len(body) == 0 {
		t.Fatalf("step %q run body is empty", stepName)
	}
	// strip the run block's own indentation (YAML run bodies nest 2 past run:)
	out := make([]string, len(body))
	for i, l := range body {
		if len(l) >= indent+2 {
			l = l[indent+2:]
		}
		out[i] = l
	}
	return strings.Join(out, "\n")
}

func fixtureTagRepo(t *testing.T, tags ...string) (dir string, shas map[string]string) {
	t.Helper()
	dir = t.TempDir()
	run := func(name string, args ...string) {
		t.Helper()
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s %v: %v\n%s", name, args, err, out)
		}
	}
	run("git", "init", "-q", "-b", "main")
	shas = map[string]string{}
	for _, tag := range tags {
		if err := os.WriteFile(filepath.Join(dir, "f"), []byte("c:"+tag+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		run("git", "add", "f")
		run("git", "commit", "-q", "-m", "commit for "+tag)
		run("git", "tag", tag)
		out, err := exec.Command("git", "-C", dir, "rev-parse", tag).Output()
		if err != nil {
			t.Fatalf("rev-parse %s: %v", tag, err)
		}
		shas[tag] = strings.TrimSpace(string(out))
	}
	// stub db-compat.sh so the step's proof call succeeds without real binaries
	stubDir := filepath.Join(dir, "deploy", "release")
	if err := os.MkdirAll(stubDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stubDir, "db-compat.sh"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, shas
}

func TestReleaseWorkflowPairsDeriveFromTags(t *testing.T) {
	y := repoFile(t, ".github/workflows/release.yml")

	if strings.Contains(y, "PAIRS: '") {
		t.Fatalf("[4] the dbcompat job carries a hard-coded PAIRS literal — rollback_pairs tested:true is FABRICATED for every later tag")
	}
	body := stepRunBody(t, y, "Prove each rollback pair")

	stubBin := t.TempDir()
	// the workflow line installs sqlite3 via sudo; stub sudo away (|| true must
	// keep the step alive either way) so the test never blocks on a password
	if err := os.WriteFile(filepath.Join(stubBin, "sudo"), []byte("#!/usr/bin/env bash\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runBody := func(dir, tag string, extraEnv ...string) (stdout, gitHubOutput string) {
		t.Helper()
		cmd := exec.Command("bash", "-c", body)
		cmd.Dir = dir
		ghOut := filepath.Join(dir, "ghout")
		env := append(os.Environ(),
			"PATH="+stubBin+":"+os.Getenv("PATH"),
			"TAG="+tag,
			"GITHUB_REF_NAME="+tag,
			"GITHUB_OUTPUT="+ghOut,
		)
		env = append(env, extraEnv...)
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("step body failed for TAG=%s: %v\n%s", tag, err, out)
		}
		b, rerr := os.ReadFile(ghOut)
		if rerr == nil {
			gitHubOutput = string(b)
		}
		return string(out), gitHubOutput
	}

	// 1. two tags: the pair must be [old sha, new sha] in 40-hex, tested by the
	//    stubbed proof.
	dir, shas := fixtureTagRepo(t, "v0.1.0", "v0.2.0")
	stdout, ghOut := runBody(dir, "v0.2.0")
	var pairs []struct {
		ToRelease string `json:"to_release"`
		Tested    bool   `json:"tested"`
	}
	line := strings.TrimSpace(stdout)
	if err := json.Unmarshal([]byte(line), &pairs); err != nil {
		t.Fatalf("step stdout is not a pairs array: %q (%v)", line, err)
	}
	if len(pairs) != 1 {
		t.Fatalf("want exactly 1 derived pair, got %d: %v", len(pairs), pairs)
	}
	if pairs[0].ToRelease != shas["v0.1.0"] {
		t.Fatalf("to_release must be the PREVIOUS v* tag sha (40-hex): want %s got %s", shas["v0.1.0"], pairs[0].ToRelease)
	}
	if len(pairs[0].ToRelease) != 40 || !isHex40(pairs[0].ToRelease) {
		t.Fatalf("to_release is not 40-hex: %q", pairs[0].ToRelease)
	}
	if !pairs[0].Tested {
		t.Fatalf("stubbed db-compat.sh exits 0, so the derived pair must be tested:true")
	}
	if !strings.Contains(ghOut, `json=[{"to_release":"`+shas["v0.1.0"]+`","tested":true}]`) {
		t.Fatalf("GITHUB_OUTPUT must carry the same derived pair; got %q", ghOut)
	}

	// 2. an unknown tag: no proof ⇒ [] (canon 49/53), never a fabricated pair.
	stdout, ghOut = runBody(dir, "v9.9.9")
	if strings.TrimSpace(stdout) != "[]" || !strings.Contains(ghOut, "json=[]") {
		t.Fatalf("unprovable tag must emit [] and exit 0; stdout=%q ghOut=%q", stdout, ghOut)
	}

	// 3. only one v* tag (nothing to roll back TO): [] again.
	dir1, _ := fixtureTagRepo(t, "v0.1.0")
	stdout, _ = runBody(dir1, "v0.1.0")
	if strings.TrimSpace(stdout) != "[]" {
		t.Fatalf("a repo with no previous release must emit []; stdout=%q", stdout)
	}
}

func isHex40(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}
