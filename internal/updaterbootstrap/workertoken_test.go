package updaterbootstrap

// P-E E2/E3 + G2 — the worker credential written by enroll, the --replace
// revocation, the epoch bump, and the last-authz file (never overwritten,
// refused when the config dir is missing/mis-owned).

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"

	"vl/auth"
)

// workerEnvForTest returns the env path under the throwaway home the
// package's attended(t) seam installed — never the real ~/.config/vl-updater
// (the package guard fails the run if the real dir changes).
func workerEnvForTest(t *testing.T) string {
	t.Helper()
	home, err := userHomeDir()
	if err != nil {
		t.Fatalf("userHomeDir: %v", err)
	}
	return workerEnvPath(home)
}

// parseWorkerToken parses a cutover-worker token with the in-process secret
// the last bootstrap Run installed (the mint uses the SAME resolution).
func parseWorkerToken(t *testing.T, tok string) *auth.Claims {
	t.Helper()
	var claims auth.Claims
	if _, err := jwt.ParseWithClaims(tok, &claims, func(*jwt.Token) (any, error) {
		return auth.JWTSecret, nil
	}); err != nil {
		t.Fatalf("parse worker token: %v", err)
	}
	return &claims
}

func TestEnrollWritesTheWorkerTokenEnvFile(t *testing.T) {
	attended(t)
	inst := install(t)
	envFile := workerEnvForTest(t)
	// The installer's other required line must survive byte-identically.
	pre := "VL_RELEASE_DIR=/srv/vl-releases\n# a comment that must stay\n"
	if err := os.MkdirAll(filepath.Dir(envFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, []byte(pre), 0o600); err != nil {
		t.Fatal(err)
	}
	rc, out, errb := run(inst, enrollLine(bEmail), "enroll", bEmail)
	if rc != 0 {
		t.Fatalf("rc=%d stderr=%s", rc, errb)
	}
	b, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, pre) {
		t.Fatalf("the VL_RELEASE_DIR line and comment were not preserved:\n%s", s)
	}
	fi, _ := os.Stat(envFile)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("env file mode = %04o — want 0600", fi.Mode().Perm())
	}
	tok := ""
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "VL_CUTOVER_TOKEN=") {
			tok = strings.TrimPrefix(line, "VL_CUTOVER_TOKEN=")
		}
	}
	if tok == "" {
		t.Fatalf("no VL_CUTOVER_TOKEN line in env:\n%s", s)
	}
	claims := parseWorkerToken(t, tok)
	if claims.Scope != auth.ScopeCutoverWorker {
		t.Fatalf("scope = %q — want %q", claims.Scope, auth.ScopeCutoverWorker)
	}
	if claims.WTE != 0 {
		t.Fatalf("wte = %d — want 0 on a fresh install", claims.WTE)
	}
	if claims.Email != bEmail {
		t.Fatalf("email = %q — want %q", claims.Email, bEmail)
	}
	if strings.Contains(out, tok) || strings.Contains(errb, tok) {
		t.Fatal("the worker token leaked to stdout/stderr")
	}
}

func TestEnrollReplaceRevokesTheOldWorkerToken(t *testing.T) {
	attended(t)
	inst := install(t)
	envFile := workerEnvForTest(t)
	if err := os.MkdirAll(filepath.Dir(envFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("first enroll rc=%d %s", rc, errb)
	}
	b, _ := os.ReadFile(envFile)
	oldTok := ""
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VL_CUTOVER_TOKEN=") {
			oldTok = strings.TrimPrefix(line, "VL_CUTOVER_TOKEN=")
		}
	}
	if oldTok == "" {
		t.Fatal("no token after the first enroll")
	}
	oldFP := auth.TokenFingerprint(oldTok)

	rc, _, errb := run(inst, replaceLine(bEmail, bEmail), "enroll", "--replace", bEmail)
	if rc != 0 {
		t.Fatalf("replace rc=%d %s", rc, errb)
	}
	b, _ = os.ReadFile(envFile)
	newTok := ""
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VL_CUTOVER_TOKEN=") {
			newTok = strings.TrimPrefix(line, "VL_CUTOVER_TOKEN=")
		}
	}
	if newTok == "" {
		t.Fatal("--replace did not write a worker token")
	}
	// Note: the token STRING may equal the old one when both mints land in the
	// same whole second (iat/exp granularity). What guarantees the rotation is
	// the revoked_tokens row below — the old fingerprint is refused regardless.
	_ = oldTok
	// The old token's fingerprint must sit in the bot DB's revoked_tokens.
	db, err := openReadOnly(filepath.Join(inst, "data", "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM revoked_tokens WHERE id = ?", oldFP).Scan(&n); err != nil {
		t.Fatalf("read revoked_tokens: %v", err)
	}
	if n != 1 {
		t.Fatalf("revoked_tokens rows for the old token = %d — want 1", n)
	}
}

func TestRevokeWorkerBumpsTheEpoch(t *testing.T) {
	attended(t)
	inst := install(t)
	for want := int64(1); want <= 2; want++ {
		rc, out, errb := run(inst, "REVOKE ALL WORKERS\n", "revoke-worker", "--all")
		if rc != 0 {
			t.Fatalf("revoke-worker rc=%d %s", rc, errb)
		}
		epoch, err := workerEpoch(filepath.Join(inst, "data", "data.db"))
		if err != nil {
			t.Fatal(err)
		}
		if epoch != want {
			t.Fatalf("epoch = %d — want %d", epoch, want)
		}
		if !strings.Contains(out, "worker epoch:") {
			t.Fatalf("output does not report the epoch: %q", out)
		}
	}
	// Without --all the command refuses (it invalidates every token at once).
	rc, _, _ := run(inst, "", "revoke-worker")
	if rc == 0 {
		t.Fatal("revoke-worker without --all must refuse")
	}
}

func TestAuthorizeWritesLastAuthzAndNeverOverwrites(t *testing.T) {
	attended(t)
	inst := install(t)
	home, err := userHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".config", "vl-updater"), 0o700); err != nil {
		t.Fatal(err)
	}
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("enroll rc=%d %s", rc, errb)
	}
	rc, out, errb := run(inst, authorizeLine(bRel), "authorize", bRel)
	if rc != 0 {
		t.Fatalf("authorize rc=%d %s", rc, errb)
	}
	p := lastAuthzPath(home, bRel)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("last-authz not written: %v", err)
	}
	fi, _ := os.Stat(p)
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("last-authz mode = %04o — want 0600", fi.Mode().Perm())
	}
	if strings.TrimSpace(string(b)) != strings.TrimSpace(out) {
		t.Fatalf("last-authz != the printed line:\nfile: %s\nout:  %s", b, out)
	}
	// A second authorize of the SAME release never overwrites the file.
	rc2, out2, errb2 := run(inst, authorizeLine(bRel), "authorize", bRel)
	if rc2 != 0 {
		t.Fatalf("second authorize rc=%d %s", rc2, errb2)
	}
	if after, _ := os.ReadFile(p); string(after) != string(b) {
		t.Fatal("second authorize overwrote last-authz")
	}
	if !strings.Contains(errb2, "already exists") || strings.TrimSpace(out2) == "" {
		t.Fatalf("second authorize must print the fresh line + a refusal note; stderr=%q", errb2)
	}
}

func TestAuthorizeWorksAfterEnrollBootstrappedTheConfigDir(t *testing.T) {
	attended(t)
	inst := install(t)
	home, err := userHomeDir() // no ~/.config/vl-updater in the seam home yet
	if err != nil {
		t.Fatal(err)
	}
	if rc, _, errb := run(inst, enrollLine(bEmail), "enroll", bEmail); rc != 0 {
		t.Fatalf("enroll rc=%d %s", rc, errb)
	}
	// enroll created the config dir for the worker env file; authorize then
	// writes the G2 file into it and prints the line.
	rc, out, errb := run(inst, authorizeLine(bRel), "authorize", bRel)
	if rc != 0 {
		t.Fatalf("authorize rc=%d %s", rc, errb)
	}
	if strings.TrimSpace(out) == "" {
		t.Fatal("the grant line must print")
	}
	if _, err := os.Lstat(lastAuthzPath(home, bRel)); err != nil {
		t.Fatalf("last-authz missing after enroll created the dir: %v", err)
	}
}

func TestWriteLastAuthzUnitGuards(t *testing.T) {
	home := t.TempDir()
	if err := writeLastAuthz(home, "v1.2.3", []byte("x")); err == nil {
		t.Fatal("missing config dir must refuse")
	}
	dir := filepath.Join(home, ".config", "vl-updater")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeLastAuthz(home, "v1.2.3", []byte("x")); err == nil {
		t.Fatal("a 0755 config dir must refuse (0700 required)")
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := writeLastAuthz(home, "v1.2.3", []byte("x")); err != nil {
		t.Fatalf("0700 dir must admit the write: %v", err)
	}
	before, _ := os.ReadFile(lastAuthzPath(home, "v1.2.3"))
	if err := writeLastAuthz(home, "v1.2.3", []byte("overwrite")); err == nil {
		t.Fatal("an existing last-authz file must refuse (never overwrite)")
	}
	after, _ := os.ReadFile(lastAuthzPath(home, "v1.2.3"))
	if string(after) != string(before) {
		t.Fatal("the existing file changed")
	}
}
