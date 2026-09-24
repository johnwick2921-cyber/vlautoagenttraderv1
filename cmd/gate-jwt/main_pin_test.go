package main

// M3 hc (verifier ha2 defect 3, L8 / canon 53): main_test.go drives
// mintGateToken — not main(). A main() that inlined its own mint
// (auth.GenerateScopedJWT(..., auth.ScopeTelegram), revert MB3) stayed GREEN.
// Two pins close that, from both sides:
//
//   - TestGateJWTMainMintsOnlyThroughMintGateToken (source): every mint in
//     package main — a reference to jwt.*, to auth.JWTSecret, or to any
//     function or method of the packages main can reach that reaches the
//     signer (auth.GenerateJWT / GenerateScopedJWT / signToken,
//     agent.GenerateBotToken, … derived, not listed) — is attributed to its
//     enclosing declaration. mintGateToken is the ONE declaration that mints
//     directly, with exactly one auth.GenerateScopedJWT; main's ONLY mint is
//     one call `mintGateToken(`; nothing else in the package mints.
//   - TestGateJWTBinaryPrintsAGateScopedTokenLast (behaviour): the BUILT tool
//     runs main() on a temp database from a scratch directory (no .env), and
//     the token it prints is the gate-jwt machine token — admitted to
//     /api/cutover-gate exactly as mintGateToken's own, refused 403 on a
//     machine-denied route — and it is the LAST line of stdout, the only
//     eyJ… segment, after whatever the logger printed first.

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
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
	"nofx/store"
)

// gjGoCommand is the go command running this test (PATH, else $GOROOT/bin).
func gjGoCommand() (string, error) {
	if p, err := exec.LookPath("go"); err == nil {
		return p, nil
	}
	if root := os.Getenv("GOROOT"); root != "" {
		p := filepath.Join(root, "bin", "go")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("no go command on PATH or in $GOROOT/bin")
}

type gjListedPkg struct {
	path, name, dir string
	files           []string
	self            bool
}

// gjListDeps: package main and every package it can reach, as the go command
// builds them (one `go list -deps`, no compile).
func gjListDeps(t *testing.T) []gjListedPkg {
	t.Helper()
	goBin, err := gjGoCommand()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(goBin, "list", "-deps", "-f",
		"{{.ImportPath}}\t{{.Name}}\t{{.Dir}}\t{{.DepOnly}}\t{{join .GoFiles \",\"}}\t{{join .CgoFiles \",\"}}", ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps .: %v\n%s", err, stderr.String())
	}
	var pkgs []gjListedPkg
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 6 {
			t.Fatalf("go list: unexpected line %q", sc.Text())
		}
		p := gjListedPkg{path: f[0], name: f[1], dir: f[2], self: f[3] == "false"}
		for _, n := range strings.Split(f[4]+","+f[5], ",") {
			if n != "" {
				p.files = append(p.files, n)
			}
		}
		pkgs = append(pkgs, p)
	}
	return pkgs
}

func gjIsJWTPath(p string) bool { return strings.HasPrefix(p, "github.com/golang-jwt/jwt") }

// gjRef is one reference made inside a top-level declaration.
type gjRef struct {
	pkg, name string // package-qualified (pkg = import path; "" = a method/field by name)
	local     bool   // a bare identifier: same package (or a dot import, pkg set)
	call      bool   // the Fun of a call expression
	pos       token.Pos
}

// gjDeclRefs lists, per top-level declaration of one file, every reference
// it makes: a qualified identifier through the file's imports (an alias or a
// dot import included), a bare identifier, a selector's name (a method or
// field, by name). Keys: "Name", "(T).Name" for a method, "package scope"
// for a var/const/type declaration.
func gjDeclRefs(f *ast.File, names map[string]string) map[string][]gjRef {
	imports := map[string]string{} // local name → import path
	var dots []string
	for _, im := range f.Imports {
		p := strings.Trim(im.Path.Value, `"`)
		switch {
		case im.Name == nil:
			if n := names[p]; n != "" {
				imports[n] = p
			} else {
				imports[p[strings.LastIndex(p, "/")+1:]] = p
			}
		case im.Name.Name == ".":
			dots = append(dots, p)
		case im.Name.Name != "_":
			imports[im.Name.Name] = p
		}
	}
	out := map[string][]gjRef{}
	collect := func(key string, n ast.Node) {
		calls := map[ast.Expr]bool{}
		ast.Inspect(n, func(x ast.Node) bool {
			if c, ok := x.(*ast.CallExpr); ok {
				calls[ast.Unparen(c.Fun)] = true
			}
			return true
		})
		var visit func(ast.Node) bool
		visit = func(x ast.Node) bool {
			switch v := x.(type) {
			case *ast.SelectorExpr:
				if id, ok := v.X.(*ast.Ident); ok {
					if p, ok := imports[id.Name]; ok {
						out[key] = append(out[key], gjRef{pkg: p, name: v.Sel.Name, call: calls[v], pos: v.Pos()})
						return false
					}
				}
				out[key] = append(out[key], gjRef{name: v.Sel.Name, call: calls[v], pos: v.Sel.Pos()})
				ast.Inspect(v.X, visit)
				return false
			case *ast.KeyValueExpr:
				if _, ok := v.Key.(*ast.Ident); ok { // a struct literal's field key names no function
					ast.Inspect(v.Value, visit)
					return false
				}
			case *ast.Ident:
				out[key] = append(out[key], gjRef{name: v.Name, local: true, call: calls[v], pos: v.Pos()})
				for _, p := range dots {
					out[key] = append(out[key], gjRef{pkg: p, name: v.Name, local: true, call: calls[v], pos: v.Pos()})
				}
			}
			return true
		}
		ast.Inspect(n, visit)
	}
	for _, d := range f.Decls {
		switch v := d.(type) {
		case *ast.FuncDecl:
			if v.Body == nil {
				continue
			}
			collect(gjDeclKey(v), v.Body)
		case *ast.GenDecl:
			if v.Tok != token.IMPORT {
				collect("package scope", v)
			}
		}
	}
	return out
}

func gjDeclKey(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return fd.Name.Name
	}
	t := fd.Recv.List[0].Type
	for {
		switch v := t.(type) {
		case *ast.StarExpr:
			t = v.X
			continue
		case *ast.IndexExpr:
			t = v.X
			continue
		case *ast.IndexListExpr:
			t = v.X
			continue
		case *ast.ParenExpr:
			t = v.X
			continue
		}
		break
	}
	if id, ok := t.(*ast.Ident); ok {
		return "(" + id.Name + ")." + fd.Name.Name
	}
	return "(?)." + fd.Name.Name
}

// main's only mint is the one call to mintGateToken, and mintGateToken is the
// one declaration in package main that mints.
func TestGateJWTMainMintsOnlyThroughMintGateToken(t *testing.T) {
	pkgs := gjListDeps(t)
	names := map[string]string{}
	for _, p := range pkgs {
		names[p.path] = p.name
	}

	// Every function or method, in every module package main can reach, that
	// reaches the signer — derived to a fixpoint from the JWT signing calls
	// themselves (jwt.New / NewWithClaims / .Sign / .SignedString /
	// .SigningString), so a new minting helper anywhere is in the set.
	type decl struct {
		key  string // "path.Name" / "path.(T).Name"
		name string // bare method name ("" for a function)
		refs []gjRef
		path string
	}
	var decls []decl
	var self *gjListedPkg
	fset := token.NewFileSet()
	mainFiles := map[string]*ast.File{}
	for i, p := range pkgs {
		if p.self {
			self = &pkgs[i]
		}
		if !strings.HasPrefix(p.path, "nofx/") && !p.self {
			continue
		}
		for _, fn := range p.files {
			f, err := parser.ParseFile(fset, filepath.Join(p.dir, fn), nil, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			if p.self {
				mainFiles[fn] = f
			}
			fdecls := map[string]*ast.FuncDecl{}
			for _, d := range f.Decls {
				if fd, ok := d.(*ast.FuncDecl); ok {
					fdecls[gjDeclKey(fd)] = fd
				}
			}
			for key, refs := range gjDeclRefs(f, names) {
				d := decl{key: p.path + "." + key, refs: refs, path: p.path}
				if fd := fdecls[key]; fd != nil && fd.Recv != nil {
					d.name = fd.Name.Name
				}
				decls = append(decls, d)
			}
		}
	}
	if self == nil || self.name != "main" || len(mainFiles) == 0 {
		t.Fatalf("go list did not name package main for \".\" — the pin walked nothing")
	}
	signing := func(r gjRef) bool {
		if gjIsJWTPath(r.pkg) {
			return r.name == "New" || r.name == "NewWithClaims"
		}
		return r.pkg == "" && !r.local && (r.name == "Sign" || r.name == "SignedString" || r.name == "SigningString")
	}
	mint := map[string]bool{}        // decl key → reaches the signer
	mintMethods := map[string]bool{} // a minting method's bare name (method calls are matched by name: over-reports, never under)
	reaches := func(d decl, r gjRef) bool {
		switch {
		case signing(r):
			return true
		case r.pkg != "":
			return mint[r.pkg+"."+r.name]
		case r.local:
			return mint[d.path+"."+r.name]
		default:
			return mintMethods[r.name]
		}
	}
	for changed := true; changed; {
		changed = false
		for _, d := range decls {
			if mint[d.key] || strings.HasSuffix(d.key, ".package scope") {
				continue
			}
			for _, r := range d.refs {
				if reaches(d, r) {
					mint[d.key] = true
					if d.name != "" {
						mintMethods[d.name] = true
					}
					changed = true
					break
				}
			}
		}
	}
	// Positive control: the derivation finds the signer's own entry points.
	for _, k := range []string{"nofx/auth.signToken", "nofx/auth.GenerateJWT", "nofx/auth.GenerateScopedJWT"} {
		if !mint[k] {
			t.Fatalf("positive control: %s is not derived as a minting function — the derivation is broken (minting set: %v)", k, sortedKeys(mint))
		}
	}

	t.Logf("minting declarations derived over %d packages: %v (methods, matched by name: %v)", len(pkgs), sortedKeys(mint), sortedKeys(mintMethods))

	// Package main: every reference that mints, per declaration.
	pos := func(p token.Pos) string {
		ps := fset.Position(p)
		return filepath.Base(ps.Filename) + ":" + strconv.Itoa(ps.Line) + ":" + strconv.Itoa(ps.Column)
	}
	directMints := map[string][]string{} // main-package decl → its direct mints
	localRefs := map[string][]gjRef{}    // main-package decl → bare references
	mainDecls := map[string]bool{}
	for _, d := range decls {
		if d.path != self.path {
			continue
		}
		key := strings.TrimPrefix(d.key, self.path+".")
		mainDecls[key] = true
		for _, r := range d.refs {
			switch {
			case gjIsJWTPath(r.pkg):
				directMints[key] = append(directMints[key], pos(r.pos)+" "+r.pkg+"."+r.name+" (jwt)")
			case r.pkg == "nofx/auth" && r.name == "JWTSecret":
				directMints[key] = append(directMints[key], pos(r.pos)+" auth.JWTSecret (the signing secret)")
			case r.pkg != "" && mint[r.pkg+"."+r.name]:
				directMints[key] = append(directMints[key], pos(r.pos)+" "+r.pkg+"."+r.name)
			case r.pkg == "" && !r.local && (mintMethods[r.name] || signing(r)):
				directMints[key] = append(directMints[key], pos(r.pos)+" ."+r.name+" (a minting method, by name)")
			case r.local && r.pkg == "":
				localRefs[key] = append(localRefs[key], r)
			}
		}
	}
	for _, k := range []string{"main", "mintGateToken"} {
		if !mainDecls[k] {
			t.Fatalf("package main declares no %s — the pin guards nothing (re-anchor it)", k)
		}
	}
	// Minting within package main, to a fixpoint over its own declarations.
	mintsLocal := map[string]bool{}
	for k, v := range directMints {
		if len(v) > 0 {
			mintsLocal[k] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for k, refs := range localRefs {
			if mintsLocal[k] {
				continue
			}
			for _, r := range refs {
				if mintsLocal[r.name] && r.name != k {
					mintsLocal[k], changed = true, true
					break
				}
			}
		}
	}

	var bad []string
	// 1. mintGateToken is the one declaration that mints directly — with
	//    exactly one auth.GenerateScopedJWT.
	for k, v := range directMints {
		if k != "mintGateToken" {
			for _, m := range v {
				bad = append(bad, fmt.Sprintf("%s mints directly: %s — only mintGateToken may", k, m))
			}
		}
	}
	if got := directMints["mintGateToken"]; len(got) != 1 || !strings.HasSuffix(got[0], " nofx/auth.GenerateScopedJWT") {
		bad = append(bad, fmt.Sprintf("mintGateToken's direct mints = %v — want exactly one nofx/auth.GenerateScopedJWT", got))
	}
	// 2. main's ONLY mint is one call `mintGateToken(`.
	mainMints := 0
	for _, r := range localRefs["main"] {
		if !mintsLocal[r.name] {
			continue
		}
		mainMints++
		if r.name != "mintGateToken" || !r.call {
			bad = append(bad, fmt.Sprintf("main mints through %s at %s (call=%v) — its only mint must be the call mintGateToken(", r.name, pos(r.pos), r.call))
		}
	}
	if mainMints != 1 {
		bad = append(bad, fmt.Sprintf("main reaches a minting declaration %d times — want exactly once, the call mintGateToken(", mainMints))
	}
	// 3. Nothing else in package main mints.
	for k := range mintsLocal {
		if k != "main" && k != "mintGateToken" {
			bad = append(bad, fmt.Sprintf("%s mints (directly or through another declaration) — only mintGateToken may, and only main may call it", k))
		}
	}
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Fatalf("cmd/gate-jwt mints outside the one path main_test.go drives through the production server:\n  %s", strings.Join(bad, "\n  "))
	}
}

func sortedKeys(m map[string]bool) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

var gjJWT = regexp.MustCompile(`eyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+`)

func gjRedact(s string) string { return gjJWT.ReplaceAllString(s, "<JWT>") }

// The BUILT tool — main() itself, its production call site — prints the
// gate-jwt machine token as the last line of stdout, and the production
// server treats it exactly as it treats mintGateToken's.
func TestGateJWTBinaryPrintsAGateScopedTokenLast(t *testing.T) {
	goBin, err := gjGoCommand()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()

	// The tool's own database: a scratch cwd with NO .env, the owner's row.
	// Built BEFORE gjBoot: store.New repoints the package-global gorm handle,
	// which must end on the server's store, not this closed one.
	run := filepath.Join(tmp, "run")
	if err := os.MkdirAll(filepath.Join(run, "data"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(run, ".env")); err == nil {
		t.Fatal(".env in the scratch cwd")
	}
	tst, err := store.New(filepath.Join(run, "data", "data.db"))
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-time.Hour).UTC()
	if err := tst.User().Create(&store.User{ID: gjOwnerID, Email: gjOwnerEmail, PasswordHash: "x", CreatedAt: past, UpdatedAt: past}); err != nil {
		t.Fatal(err)
	}
	tst.Plan().Close()
	_ = tst.Close()

	base, st := gjBoot(t) // the production server; auth's secret = gjSecret; owner row gjOwnerID

	bin := filepath.Join(tmp, "gate-jwt")
	if out, err := exec.Command(goBin, "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/gate-jwt: %v\n%s", err, out)
	}

	var env []string
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "JWT_SECRET=") {
			env = append(env, kv)
		}
	}
	cmd := exec.Command(bin, gjOwnerEmail, "data/data.db")
	cmd.Dir = run
	cmd.Env = append(env, "JWT_SECRET="+gjSecret)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gate-jwt: %v\nstdout: %s\nstderr: %s", err, gjRedact(stdout.String()), gjRedact(stderr.String()))
	}

	// The comment's contract: the token is the LAST line of stdout (no
	// trailing newline), the only eyJ… segment; log lines may precede it.
	out := stdout.String()
	last := out[strings.LastIndexByte(out, '\n')+1:]
	all := gjJWT.FindAllString(out, -1)
	if len(all) != 1 || last != all[0] {
		t.Fatalf("gate-jwt stdout holds %d eyJ… segment(s), last line a token: %v — want exactly one, and it the last line (no trailing newline):\n%s", len(all), gjJWT.MatchString(last) && gjJWT.FindString(last) == last, gjRedact(out))
	}
	if strings.Contains(stderr.String(), "eyJ") {
		t.Fatalf("gate-jwt wrote a token to stderr: %s", gjRedact(stderr.String()))
	}
	tok := last

	cl, err := auth.ValidateJWT(tok)
	if err != nil {
		t.Fatalf("the printed token does not validate under the server's secret: %v", err)
	}
	if cl.Scope != auth.ScopeGateJWT || cl.UserID != gjOwnerID || cl.Email != gjOwnerEmail || !cl.IsMachine() {
		t.Fatalf("main() prints a token with scope=%q user=%q email-match=%v machine=%v — want the gate-jwt machine scope on the owner's id and email", cl.Scope, cl.UserID, cl.Email == gjOwnerEmail, cl.IsMachine())
	}

	// At the production server: admitted to the cutover gate exactly as
	// mintGateToken's own token, refused 403 on a machine-denied route.
	ref, err := mintGateToken(st, gjOwnerEmail)
	if err != nil {
		t.Fatal(err)
	}
	bCode, bBody := gjCall(t, base, "GET", "/api/cutover-gate", tok, "")
	rCode, rBody := gjCall(t, base, "GET", "/api/cutover-gate", ref, "")
	if bCode == http.StatusUnauthorized || bCode == http.StatusForbidden || bCode != rCode || bBody != rBody {
		t.Fatalf("main()'s token on GET /api/cutover-gate = %d %s — want mintGateToken's own answer (%d %s), never 401/403", bCode, bBody, rCode, rBody)
	}
	if code, body := gjCall(t, base, "PUT", "/api/user/password", tok, "{}"); code != http.StatusForbidden || body != gjForbidden {
		t.Fatalf("main()'s token on PUT /api/user/password = %d %s — want 403 %s (a machine token)", code, body, gjForbidden)
	}
}
