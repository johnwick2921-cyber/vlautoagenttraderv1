package updateauth

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// ── M3 census repair, verifier D3: the LOADED device key may be used only to
// verify ──────────────────────────────────────────────────────────────────
//
// The primitive rule (4) sees only crypto/hmac. api/handler_updates.go is
// admitted LoadDeviceKey, and golang-jwt/v5 is already a direct dependency:
// probe V2 appended to it `jwt.SigningMethodHS256.Sign(release|job|exp, key)`,
// hex-encoded — a MAC production VerifyMAC accepts — with `go build ./api/`
// OK and TestUpdateAuthCensus green (red-team 3 #1(c)'s "most likely drift",
// through a different HMAC). Rule 6: in every file that references
// updateauth.LoadDeviceKey, the call is bound `key, err :=
// updateauth.LoadDeviceKey(dir)` and the key variable appears ONLY as the
// first argument of updateauth.VerifyMAC, of <a LoadAdmin result>.
// PasswordStillBound (the H1 belt's constant-time check), or of the builtin
// clear. Whatever the primitive — crypto/hmac, a JWT signer, a hand-rolled
// SHA-256 — the key has to be NAMED to reach it, and a name anywhere else is
// an offence.

// keyFlowGate is production's shape of the two key-holding functions in
// api/handler_updates.go (gate + install) — the positive control.
const keyFlowGate = `package api

import (
	"nofx/internal/updateauth"
	"nofx/trader"
)

func gate(hash string) string {
	d := trader.MaintenanceDataDir()
	admin, err := updateauth.LoadAdmin(d)
	if err != nil {
		return "not enrolled"
	}
	key, err := updateauth.LoadDeviceKey(d)
	if err != nil {
		return "device key unreadable"
	}
	defer clear(key)
	if admin.UserID == "" {
		return "no admin"
	}
	if !admin.PasswordStillBound(key, hash) {
		return "password changed"
	}
	return ""
}

func install(g updateauth.Grant) bool {
	d := trader.MaintenanceDataDir()
	key, err := updateauth.LoadDeviceKey(d)
	if err != nil {
		return false
	}
	ok := updateauth.VerifyMAC(key, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC)
	clear(key)
	return ok
}
`

func keyFlowOffendersFor(t *testing.T, files map[string]string) []string {
	t.Helper()
	root := mintBase(t)
	for rel, body := range files {
		mintWrite(t, root, rel, body)
	}
	return mintOffenders(t, root)
}

func TestUpdateAuthKeyFlowCleanBaseline(t *testing.T) {
	if off := keyFlowOffendersFor(t, map[string]string{"api/handler_updates.go": keyFlowGate}); len(off) != 0 {
		t.Fatalf("production's key-holding shape must be clean:\n%s", strings.Join(off, "\n"))
	}
}

// PIN (probe V2 verbatim, appended to the admitted file): the key signs a
// grant through golang-jwt's HS256 — no crypto/hmac import, no restricted name.
func TestUpdateAuthCensusRefusesTheKeyHolderMintingViaJWT(t *testing.T) {
	const rel = "api/handler_updates.go"
	body := strings.Replace(keyFlowGate, "import (\n", "import (\n\t\"encoding/hex\"\n\t\"strconv\"\n\n\t\"github.com/golang-jwt/jwt/v5\"\n", 1) + `
func v2MintViaJWT(dataDir, releaseID, jobID string, expiresAt int64) (string, error) {
	key, err := updateauth.LoadDeviceKey(dataDir)
	if err != nil {
		return "", err
	}
	sig, err := jwt.SigningMethodHS256.Sign(releaseID+"|"+jobID+"|"+strconv.FormatInt(expiresAt, 10), key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}
`
	requirePrefixes(t, keyFlowOffendersFor(t, map[string]string{rel: body}),
		rel+`: the loaded device key "key" is used outside`)
}

// PIN: every other way out for the loaded key, one per case, each in an
// otherwise production-shaped admitted file.
func TestUpdateAuthLoadedKeyFlowsOnlyIntoVerification(t *testing.T) {
	const rel = "api/handler_updates.go"
	const used = rel + `: the loaded device key "key" is used outside`
	const unbound = rel + ": updateauth.LoadDeviceKey must be bound"
	fn := func(src string) map[string]string {
		return map[string]string{rel: keyFlowGate + "\n" + src}
	}
	for name, c := range map[string]struct {
		files map[string]string
		want  string
	}{
		"crypto/hmac over the key":       {fn("func m(d string) []byte {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\th := hmac.New(sha256.New, key)\n\treturn h.Sum(nil)\n}\n"), used},
		"copied to another variable":     {fn("func m(d string) []byte {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\tk2 := key\n\treturn k2\n}\n"), used},
		"returned":                       {fn("func m(d string) []byte {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\treturn key\n}\n"), used},
		"sliced into VerifyMAC":          {fn("func m(d string, g updateauth.Grant) bool {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\treturn updateauth.VerifyMAC(key[:], g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC)\n}\n"), used},
		"converted into the MAC message": {fn("func m(d string, g updateauth.Grant) bool {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\treturn updateauth.VerifyMAC(key, string(key), g.JobID, g.ExpiresAt, g.HMAC)\n}\n"), used},
		"captured by a closure":          {fn("func m(d string, sink func([]byte)) {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\tgo func() { sink(key) }()\n}\n"), used},
		"stored in a field":              {fn("type box struct{ k []byte }\n\nfunc m(d string) box {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\treturn box{k: key}\n}\n"), used},
		"named result, bare return":      {fn("func m(d string) (key []byte, err error) {\n\tkey, e2 := updateauth.LoadDeviceKey(d)\n\t_ = e2\n\treturn\n}\n"), used},
		"a local clear":                  {fn("func m(d string, sink func([]byte)) {\n\tclear := func(b []byte) { sink(b) }\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\tclear(key)\n}\n"), used},
		"a package-level clear":          {map[string]string{rel: keyFlowGate, "api/util.go": "package api\n\nvar leaked []byte\n\nfunc clear(b []byte) { leaked = b }\n"}, used},
		"PasswordStillBound on a fake":   {fn("type fake struct{}\n\nfunc (fake) PasswordStillBound(k []byte, h string) bool { return len(k) > 0 }\n\nfunc m(d string) bool {\n\tvar admin fake\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\treturn admin.PasswordStillBound(key, \"h\")\n}\n"), used},
		"LoadAdmin result shadowed":      {fn("type fake struct{}\n\nfunc (fake) PasswordStillBound(k []byte, h string) bool { return len(k) > 0 }\n\nfunc m(d string) bool {\n\tadmin, _ := updateauth.LoadAdmin(d)\n\t_ = admin\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\t{\n\t\tadmin := fake{}\n\t\treturn admin.PasswordStillBound(key, \"h\")\n\t}\n}\n"), used},
		"admin name is package-level":    {fn("type fake struct{}\n\nfunc (fake) PasswordStillBound(k []byte, h string) bool { return len(k) > 0 }\n\nvar owner fake\n\nfunc m(d string) bool {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\tif owner, err := updateauth.LoadAdmin(d); err == nil {\n\t\t_ = owner\n\t}\n\treturn owner.PasswordStillBound(key, \"h\")\n}\n"), used},
		"import name shadowed":           {fn("type ns struct{ VerifyMAC func([]byte, string, string, int64, string) bool }\n\nfunc m(d string, g updateauth.Grant, sink ns) bool {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\tupdateauth := sink\n\treturn updateauth.VerifyMAC(key, g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC)\n}\n"), used},
		"loader as a function value":     {fn("var load = updateauth.LoadDeviceKey\n"), unbound},
		"bound with = to a package var":  {fn("var cached []byte\n\nfunc m(d string) {\n\tvar err error\n\tcached, err = updateauth.LoadDeviceKey(d)\n\t_ = err\n}\n"), unbound},
		"bound with var":                 {fn("func m(d string) {\n\tvar key, err = updateauth.LoadDeviceKey(d)\n\t_, _ = key, err\n}\n"), unbound},
		"bound to the blank identifier":  {fn("func m(d string) error {\n\t_, err := updateauth.LoadDeviceKey(d)\n\treturn err\n}\n"), unbound},
		"used inline, never bound":       {fn("func must(b []byte, _ error) []byte { return b }\n\nfunc m(d string, g updateauth.Grant) bool {\n\treturn updateauth.VerifyMAC(must(updateauth.LoadDeviceKey(d)), g.ReleaseID, g.JobID, g.ExpiresAt, g.HMAC)\n}\n"), unbound},
		"through a second import name":   {map[string]string{rel: strings.Replace(keyFlowGate, "\t\"nofx/internal/updateauth\"\n", "\t\"nofx/internal/updateauth\"\n\tua \"nofx/internal/updateauth\"\n", 1) + "\nfunc m(d string) []byte {\n\tkey, _ := ua.LoadDeviceKey(d)\n\treturn key\n}\n"}, used},
		// census-repair verify P2: the import name is shadowed AFTER the real
		// LoadDeviceKey, so `admin` is a fake's result and its
		// PasswordStillBound receives the real key.
		"LoadAdmin through a shadowed import name": {fn("type fakeAdmin struct{ sink *[]byte }\n\nfunc (a fakeAdmin) PasswordStillBound(k []byte, _ string) bool {\n\t*a.sink = append([]byte(nil), k...)\n\treturn true\n}\n\ntype fakeNS struct{ LoadAdmin func(string) (fakeAdmin, error) }\n\nfunc m(d string) []byte {\n\tkey, _ := updateauth.LoadDeviceKey(d)\n\tvar stolen []byte\n\tupdateauth := fakeNS{LoadAdmin: func(string) (fakeAdmin, error) { return fakeAdmin{sink: &stolen}, nil }}\n\tadmin, _ := updateauth.LoadAdmin(d)\n\tadmin.PasswordStillBound(key, \"\")\n\treturn stolen\n}\n"), used},
	} {
		t.Run(name, func(t *testing.T) {
			requirePrefixes(t, keyFlowOffendersFor(t, c.files), c.want)
		})
	}
}

// PIN at the production file: the REAL api/handler_updates.go, copied into a
// synthetic module with probe V2 appended, yields exactly one offence — the
// V2 line. Its own key uses (defer clear, PasswordStillBound, VerifyMAC,
// clear) are admitted, so rule 6 is neither vacuous on the real handler nor
// over-reporting it.
func TestUpdateAuthKeyFlowOnTheRealHandler(t *testing.T) {
	real, err := os.ReadFile(filepath.Join("..", "..", "api", "handler_updates.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(real)
	if !strings.Contains(src, "updateauth.LoadDeviceKey(") {
		t.Fatal("the real handler no longer loads the device key — re-point this pin")
	}
	body := strings.Replace(src, "import (\n", "import (\n\t\"encoding/hex\"\n\t\"strconv\"\n\n\t\"github.com/golang-jwt/jwt/v5\"\n", 1) + `
func v2MintViaJWT(dataDir, releaseID, jobID string, expiresAt int64) (string, error) {
	key, err := updateauth.LoadDeviceKey(dataDir)
	if err != nil {
		return "", err
	}
	sig, err := jwt.SigningMethodHS256.Sign(releaseID+"|"+jobID+"|"+strconv.FormatInt(expiresAt, 10), key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}
`
	signLine := 0
	for i, l := range strings.Split(body, "\n") {
		if strings.Contains(l, "jwt.SigningMethodHS256.Sign(") {
			signLine = i + 1
		}
	}
	off := keyFlowOffendersFor(t, map[string]string{"api/handler_updates.go": body})
	want := `api/handler_updates.go: the loaded device key "key" is used outside updateauth.VerifyMAC / <LoadAdmin result>.PasswordStillBound / clear (line ` + strconv.Itoa(signLine) + `)`
	if len(off) != 1 || off[0] != want {
		t.Fatalf("census offenders =\n%s\nwant exactly\n%s", strings.Join(off, "\n"), want)
	}
}

// keyFlowProbeP2 is the census-repair verifier's probe P2, verbatim: the
// import name `updateauth` is shadowed AFTER the real LoadDeviceKey, so
// `admin, _ := updateauth.LoadAdmin(d)` binds a FAKE's result, and its
// PasswordStillBound — admitted by name — copies the key out. With it
// appended to the real handler, `go build ./api/` and `go vet ./api/` pass
// and production VerifyMAC accepts the HS256 grant it signs [A, 2026-09-24].
const keyFlowProbeP2 = `
// VCC-P2 (verifier, census repair): rule 6 records a LoadAdmin binding for
// ` + "`admin, _ := <import name>.LoadAdmin(d)`" + ` by NAME, without asking whether the
// import name was shadowed (it asks that only for VerifyMAC). Shadow the
// import name AFTER the real LoadDeviceKey, bind a fake "LoadAdmin" result,
// and its PasswordStillBound receives the key as an admitted first argument.
type vccP2Admin struct{ sink *[]byte }

func (a vccP2Admin) PasswordStillBound(k []byte, _ string) bool {
	*a.sink = append([]byte(nil), k...)
	return true
}

type vccP2NS struct {
	LoadAdmin func(string) (vccP2Admin, error)
}

func vccP2Mint(dataDir, releaseID, jobID string, expiresAt int64) (string, error) {
	key, err := updateauth.LoadDeviceKey(dataDir)
	if err != nil {
		return "", err
	}
	var stolen []byte
	updateauth := vccP2NS{LoadAdmin: func(string) (vccP2Admin, error) { return vccP2Admin{sink: &stolen}, nil }}
	admin, _ := updateauth.LoadAdmin(dataDir)
	admin.PasswordStillBound(key, "")
	sig, err := jwt.SigningMethodHS256.Sign("nofx-update-install/v1|"+releaseID+"|"+jobID+"|"+strconv.FormatInt(expiresAt, 10), stolen)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sig), nil
}
`

// PIN at the production file (census-repair verify): the REAL
// api/handler_updates.go, copied into a synthetic module with ONE verifier
// probe appended (with the imports it needs), yields exactly one offence —
// the probe's key use. Each probe compiled against the real tree and minted a
// grant production VerifyMAC accepted while TestUpdateAuthCensus stayed green.
func TestUpdateAuthKeyFlowRefusesTheVerifierProbesOnTheRealHandler(t *testing.T) {
	real, err := os.ReadFile(filepath.Join("..", "..", "api", "handler_updates.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(real)
	if strings.Count(src, "import (\n") != 1 || !strings.Contains(src, "updateauth.LoadDeviceKey(") {
		t.Fatal("the real handler's import block or key load moved — re-point this pin")
	}
	withImports := strings.Replace(src, "import (\n", "import (\n\t\"encoding/hex\"\n\t\"strconv\"\n\n\t\"github.com/golang-jwt/jwt/v5\"\n", 1)
	for name, c := range map[string]struct {
		probe, leakLine string // leakLine: the one line whose key use must be reported
	}{
		"P2 LoadAdmin through a shadowed import name": {keyFlowProbeP2, "\tadmin.PasswordStillBound(key, \"\")"},
	} {
		t.Run(name, func(t *testing.T) {
			body := withImports + c.probe
			line := 0
			for i, l := range strings.Split(body, "\n") {
				if l == c.leakLine {
					if line != 0 {
						t.Fatalf("leak line %q appears twice", c.leakLine)
					}
					line = i + 1
				}
			}
			if line == 0 {
				t.Fatalf("leak line %q not in the probe", c.leakLine)
			}
			off := keyFlowOffendersFor(t, map[string]string{"api/handler_updates.go": body})
			want := `api/handler_updates.go: the loaded device key "key" is used outside updateauth.VerifyMAC / <LoadAdmin result>.PasswordStillBound / clear (line ` + strconv.Itoa(line) + `)`
			if len(off) != 1 || off[0] != want {
				t.Fatalf("census offenders =\n%s\nwant exactly\n%s", strings.Join(off, "\n"), want)
			}
		})
	}
}
