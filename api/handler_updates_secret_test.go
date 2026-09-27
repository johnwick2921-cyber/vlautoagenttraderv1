package api

// W-ONE-BUTTON M3 red-team fold M1 (red-1 #3): /api/updates* refuses EVERY
// JWT secret this public repository publishes — the loader's default literal,
// the .env.example placeholder (INSTALL.md tells users to `cp .env.example
// .env`, and running the binary directly keeps it), the CI literal — and any
// secret shorter than 32 bytes. A token forged under a public secret is not
// an identity. Driven at the production router (canon 53).
//
// The public set is not a hand-kept list in the test: it is a census of every
// literal JWT_SECRET assignment in the TRACKED tree (git ls-files — never an
// untracked .env), so a future edit of .env.example or a workflow that
// introduces a new placeholder is covered the day it lands.

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"nofx/auth"
	"nofx/config"
)

// jwtSecretLiteralRe matches a literal JWT_SECRET assignment: KEY=value
// (dotenv/shell) or KEY: value (YAML). A value starting with '$' is an
// expansion, not a literal, and is skipped.
var jwtSecretLiteralRe = regexp.MustCompile(`JWT_SECRET(?:=|:[ \t]*)("[^"\n]*"|'[^'\n]*'|[^\s"'$#]\S*)`)

// notASecretShape: a value carrying regex or template syntax is a pattern
// or a documentation placeholder (e.g. the runbook's secret-scan grep
// `JWT_SECRET=..|DATA_ENCRYPTION_KEY=..|…`), not a deployable literal.
const notASecretShape = "|()[]{}\\<>"

// trackedJWTSecretLiterals returns every literal JWT_SECRET value in the
// tracked tree, value → "file:line" of its first occurrence.
func trackedJWTSecretLiterals(t *testing.T) map[string]string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("git", "-C", root, "ls-files", "-z").Output()
	if err != nil {
		t.Fatalf("git ls-files: %v — the census needs the tracked file list (it must never read an untracked .env)", err)
	}
	found := map[string]string{}
	files := 0
	for _, rel := range strings.Split(string(out), "\x00") {
		if rel == "" {
			continue
		}
		p := filepath.Join(root, rel)
		fi, err := os.Lstat(p)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > 2<<20 {
			continue
		}
		b, err := os.ReadFile(p)
		if err != nil || bytes.IndexByte(b, 0) >= 0 {
			continue
		}
		files++
		for i, line := range strings.Split(string(b), "\n") {
			for _, m := range jwtSecretLiteralRe.FindAllStringSubmatch(line, -1) {
				v := strings.TrimSpace(m[1])
				if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
					v = v[1 : len(v)-1]
				}
				if v == "" || strings.ContainsAny(v, notASecretShape) {
					continue
				}
				if _, seen := found[v]; !seen {
					found[v] = rel + ":" + strconv.Itoa(i+1)
				}
			}
		}
	}
	if files < 500 {
		t.Fatalf("the census read only %d tracked files — it is not covering the tree", files)
	}
	return found
}

// The census finds the two placeholders the red team named (so the test
// provably READS .env.example and the CI workflow), and the production gate
// refuses a correctly signed admin token under every literal it finds, under
// the loader's default, and under any secret shorter than 32 bytes — with
// the byte-identical 403 on every route. Positive control: the same token
// shape under a private 32-byte secret is admitted.
func TestUpdatesRefuseEveryPublicPlaceholderAndShortJWTSecret(t *testing.T) {
	lits := trackedJWTSecretLiterals(t)
	for _, want := range []string{".env.example:", ".github/workflows/pr-docker-compose-healthcheck.yml:"} {
		ok := false
		for _, where := range lits {
			if strings.HasPrefix(where, want) {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("the census found no JWT_SECRET literal in %s — it no longer reads the file the finding named (found: %v)", strings.TrimSuffix(want, ":"), lits)
		}
	}
	secrets := map[string]string{
		config.InsecureDefaultJWTSecret: "the loader's default (config.InsecureDefaultJWTSecret)",
		"x":                             "a 1-byte secret",
		strings.Repeat("k", 31):         "a 31-byte secret",
	}
	for v, where := range lits {
		secrets[v] = "tracked literal at " + where
	}
	names := make([]string, 0, len(secrets))
	for v := range secrets {
		names = append(names, v)
	}
	sort.Strings(names)

	e := newUpdEnv(t)
	e.expectAllAdmitted("private 32+-byte secret")
	for _, sec := range names {
		auth.SetJWTSecret(sec)
		forged := mintJWT(t, updAdminID, updAdminEmail, time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), sec)
		e.expectAllForbidden("JWT secret "+secrets[sec]+" (len "+strconv.Itoa(len(sec))+")", withToken(forged))
	}
	// positive control: exactly 32 private bytes is enough
	priv := "0123456789abcdef-private-secret!"
	if len(priv) != 32 {
		t.Fatalf("positive control secret is %d bytes", len(priv))
	}
	auth.SetJWTSecret(priv)
	e.expectAllAdmitted("private exactly-32-byte secret", withToken(mintJWT(t, updAdminID, updAdminEmail, time.Now().Add(-5*time.Second), time.Now().Add(time.Hour), priv)))
}
