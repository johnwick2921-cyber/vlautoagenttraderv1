package telegram

// M3 ha verifier defect 3: runBot answers every AI message on its own
// goroutine, while the main loop's ident.refresh() — at start, on /start and
// before every AI call — REASSIGNS the identity's fields (agents, token,
// userID, email) whenever the bot re-mints. M3 H2 made the re-mint routine
// (every owner password change, every 24 h expiry), so a goroutine that reads
// a field of the identity races the next message's refresh.
//
// M3 hc (verifier ha2 defect 2): these pins are TYPE-based. Their first
// version tracked the identity by VARIABLE NAME, so `id := ident; go func(){
// agents := id.agents … }()` restored the race with both pins green. Every
// rule asks go/types what an expression IS, never what it is called: package
// telegram's production files are type-checked (imports from the go
// command's own export data), and "the identity" is any value whose type is
// botIdentity or *botIdentity — through an alias, or a named type declared
// from it.
//
// M3 hc repair (verifier vf-hc defects 1-2): the type rules then asked only
// "is this value the identity?", so a value that POINTS INTO it or HOLDS it
// got through with every pin green — &ident.agents captured or handed to the
// goroutine, a holder struct captured and read through a helper or its own
// method, a method value or `go h.m()` on a holder, the identity handed to a
// generic. Every rule now asks "does this value hold, or point into, the
// identity?". TestBotIdentityPinRulesCatchEveryRoad runs the SAME rule code
// over a synthetic package, one compiling road per case, beside the shapes
// that stay allowed — so a rule that stops catching a road fails there.
//
// The rules, over the production source, type-checked (canon 53 — no copy of
// the loop):
//
//   - TestRunBotGoroutinesReadNoBotIdentityField — no closure outside
//     botIdentity's methods uses the identity (a selector through it, a field
//     or method promoted through an embedded one, the bare value) or captures
//     a variable that HOLDS it (a struct, slice, map, channel, pointer or
//     generic value carrying one — whatever the closure then does with it: a
//     method, a helper, a call two levels down). A go statement binds no
//     method to a receiver that holds the identity (go ident.refresh(),
//     go h.outer()), starts no package function whose body uses it, and hands
//     the goroutine no argument that holds it. A go statement's ARGUMENTS are
//     evaluated on the main loop (Go spec), so `}(ident.agents, chatID, text)`
//     is the capture, and it is allowed; a POINTER into the identity is no
//     capture, and the third rule refuses it wherever it is formed.
//   - TestBotIdentityClosuresReadNoReceiverField — no closure a botIdentity
//     method builds uses the identity or captures a holder of it, and no
//     method value anywhere in the package is bound to a receiver that holds
//     it (b.llmClient, h.llm, f := h.touch, time.AfterFunc(d, h.touch)): a
//     method value IS a closure over its receiver. The LLM factory refresh
//     hands to agent.NewManager runs on the per-message goroutine
//     (Manager.Run → agent.New / Agent.Run), so it closes over locals.
//   - TestBotIdentityEscapesNoOtherWay — no package-level variable holds the
//     identity; no value holding it is converted to an interface or
//     unsafe.Pointer; no pointer INTO it is formed — &ident.f, ident.arr[:],
//     or implicitly, a pointer-receiver method selected on a value field of
//     it; and no generic is instantiated with a type that holds it (inside
//     the generic the identity is a type parameter no rule can see).
//
// Allowed, each a GREEN control in the synthetic proof: a field read on the
// main loop (into a local first, or as a go argument); go on a field's own
// method (go ident.agents.Run(…) — the receiver is evaluated on the main
// loop); a synchronous call of an identity method; a pointer into the memory
// a field POINTS TO (&ident.bx.n, &ident.sl[0]); a value-receiver method
// value on a value field (a copy). The pins guard the fields refresh
// REPLACES. A field's pointee mutated in place is outside what they can see:
// the pre-existing /start → ident.agents.Reset (bot.go) against an in-flight
// Agent.Run (agent/manager.go) is such a race.
//
// Known limits (fail-closed defaults): pointer arithmetic from memory that
// holds no identity (unsafe past a conversion of something else) is not
// traced; a production file behind a build constraint, or in cgo, fails the
// load rather than going unchecked. Flagged on purpose though race-free: a
// copied struct value; a closure that reads the identity, or a holder of it,
// synchronously; logging it through fmt (an interface conversion); a generic
// such as slices.Contains over identity pointers; and a value-typed field
// with pointer methods — a sync.Mutex added to botIdentity trips the
// implicit-address rule even at b.mu.Lock(): a locked field is a new model,
// so re-anchor the pins with it.
//
// NAMED, NOT CHASED (CTO ruling 1790248662535: after the hc repair, a
// further purely structural escape is named here, not another round; these
// pins are a BELT, and the boundary is the capture-before-go pattern in
// runBot). The interface-conversion rule reads single-value assignment
// contexts only. Each of these compiles, converts the identity to an
// interface, and passes all three pins [A, hc re-verify, compiling reverts
// T1–T9]:
//   - a TUPLE result: f(g()), a, b = g(), var a, b T = g(), return g() — e.g.
//     `go func(ag Ager, _ bool){…}(pair(ident))`, which also hands the
//     goroutine an argument that holds the identity;
//   - comma-ok: ag, _ = <-ch;
//   - a `:=` that re-declares an existing variable: ag, n := ident, 0;
//   - range with `=`: for _, ag = range []*botIdentity{ident} {};
//   - panic(b) then recover().(Ager).
// The fold, if one is ever wanted, is a spec-derived list of assignability
// contexts (tuples split per element, := as well as =, range =, panic as a
// slot of type any), each with one synthetic road in
// TestBotIdentityPinRulesCatchEveryRoad.
//
// The -race reproduction of the factory half is
// TestRaceBotRefreshAgainstInFlightManager (bot_refresh_race_test.go).

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

// typedTelegram is package telegram's production source, type-checked.
type typedTelegram struct {
	fset        *token.FileSet
	files       []*ast.File
	info        *types.Info
	pkg         *types.Package
	identNamed  types.Type                     // botIdentity
	identStruct types.Type                     // its underlying struct
	decls       map[types.Object]*ast.FuncDecl // package funcs and methods → declaration
}

var (
	typedTelegramOnce sync.Once
	typedTelegramPkg  *typedTelegram
	typedTelegramErr  error
	telegramExports   map[string]string // import path → export data, from the same go list
)

func loadTypedTelegram(t *testing.T) *typedTelegram {
	t.Helper()
	typedTelegramOnce.Do(func() { typedTelegramPkg, typedTelegramErr = typeCheckTelegram() })
	if typedTelegramErr != nil {
		t.Fatalf("the pin type-checks package telegram or it guards nothing: %v", typedTelegramErr)
	}
	return typedTelegramPkg
}

// goCommand is the go command running this test (PATH, else $GOROOT/bin).
func goCommand() (string, error) {
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

// typeCheckTelegram type-checks the production files of the package in the
// working directory (package telegram, under go test). The file list and
// every import's export data come from ONE `go list -export -deps` — the go
// command's own build view, so the files checked are the files compiled.
func typeCheckTelegram() (*typedTelegram, error) {
	goBin, err := goCommand()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(goBin, "list", "-export", "-deps", "-f",
		"{{.ImportPath}}\t{{.Export}}\t{{.DepOnly}}\t{{join .GoFiles \",\"}}\t{{join .CgoFiles \",\"}}\t{{join .IgnoredGoFiles \",\"}}", ".")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -export -deps .: %v\n%s", err, stderr.String())
	}
	exports := map[string]string{}
	var self []string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) != 6 {
			return nil, fmt.Errorf("go list: unexpected line %q", sc.Text())
		}
		exports[f[0]] = f[1]
		if f[2] == "false" {
			if self != nil {
				return nil, errors.New("go list named more than one package for \".\"")
			}
			self = f
		}
	}
	if self == nil || self[0] != "nofx/telegram" {
		return nil, fmt.Errorf("go list did not name nofx/telegram for \".\" (got %q)", self)
	}
	if self[4] != "" {
		return nil, fmt.Errorf("cgo files %s are not type-checked by this pin — extend it", self[4])
	}
	for _, n := range strings.Split(self[5], ",") {
		if n != "" && !strings.HasSuffix(n, "_test.go") {
			return nil, fmt.Errorf("production file %s sits behind a build constraint this pin does not type-check — extend it", n)
		}
	}
	telegramExports = exports
	fset := token.NewFileSet()
	var files []*ast.File
	for _, n := range strings.Split(self[3], ",") {
		if n == "" {
			continue
		}
		f, err := parser.ParseFile(fset, n, nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return nil, errors.New("no production file listed — the pin would walk nothing")
	}
	return checkTyped(fset, files, "nofx/telegram", exportImporter(fset, exports))
}

// exportImporter imports from the go command's export data.
func exportImporter(fset *token.FileSet, exports map[string]string) types.Importer {
	return importer.ForCompiler(fset, "gc", func(path string) (io.ReadCloser, error) {
		p := exports[path]
		if p == "" {
			return nil, fmt.Errorf("no export data for %q", path)
		}
		return os.Open(p)
	})
}

// checkTyped type-checks files as package path — the production package, or
// the synthetic proof's — and anchors the rules on its botIdentity.
func checkTyped(fset *token.FileSet, files []*ast.File, path string, imp types.Importer) (*typedTelegram, error) {
	info := &types.Info{
		Types:      map[ast.Expr]types.TypeAndValue{},
		Defs:       map[*ast.Ident]types.Object{},
		Uses:       map[*ast.Ident]types.Object{},
		Selections: map[*ast.SelectorExpr]*types.Selection{},
		Instances:  map[*ast.Ident]types.Instance{},
	}
	pkg, err := (&types.Config{Importer: imp}).Check(path, fset, files, info)
	if err != nil {
		return nil, err // any type error: the pin cannot trust what it did record
	}
	tn, ok := pkg.Scope().Lookup("botIdentity").(*types.TypeName)
	if !ok {
		return nil, fmt.Errorf("package %s declares no type botIdentity — the identity the pins guard is gone (re-anchor them)", path)
	}
	tp := &typedTelegram{fset: fset, files: files, info: info, pkg: pkg,
		identNamed: tn.Type(), identStruct: tn.Type().Underlying(), decls: map[types.Object]*ast.FuncDecl{}}
	for _, f := range files {
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok {
				tp.decls[info.Defs[fd.Name]] = fd
			}
		}
	}
	return tp, nil
}

// isIdentity: T is botIdentity or *botIdentity — through an alias, a named
// type declared from it (type twin botIdentity), or a named pointer type.
func (tp *typedTelegram) isIdentity(T types.Type) bool {
	if T == nil {
		return false
	}
	T = types.Unalias(T)
	if p, ok := T.Underlying().(*types.Pointer); ok {
		T = types.Unalias(p.Elem())
	}
	return types.Identical(T.Underlying(), tp.identStruct)
}

// holdsIdentity: a value of type T carries the identity — it is one, or
// reaches one through pointers, struct fields, arrays, slices, maps, channels
// or a generic type's arguments. (Interfaces: TestBotIdentityEscapesNoOtherWay.)
func (tp *typedTelegram) holdsIdentity(T types.Type) bool {
	seen := map[types.Type]bool{}
	var reach func(types.Type) bool
	reach = func(T types.Type) bool {
		if T == nil {
			return false
		}
		T = types.Unalias(T)
		if tp.isIdentity(T) {
			return true
		}
		if seen[T] {
			return false
		}
		seen[T] = true
		if n, ok := T.(*types.Named); ok {
			for i := 0; i < n.TypeArgs().Len(); i++ {
				if reach(n.TypeArgs().At(i)) {
					return true
				}
			}
		}
		switch u := T.Underlying().(type) {
		case *types.Pointer:
			return reach(u.Elem())
		case *types.Struct:
			for i := 0; i < u.NumFields(); i++ {
				if reach(u.Field(i).Type()) {
					return true
				}
			}
		case *types.Slice:
			return reach(u.Elem())
		case *types.Array:
			return reach(u.Elem())
		case *types.Map:
			return reach(u.Key()) || reach(u.Elem())
		case *types.Chan:
			return reach(u.Elem())
		}
		return false
	}
	return reach(T)
}

// valueType is the type of e when e denotes a VALUE — nil for a type, a
// package, a function name, or a struct literal's field key.
func (tp *typedTelegram) valueType(e ast.Expr) types.Type {
	if id, ok := e.(*ast.Ident); ok {
		if v, ok := tp.info.Uses[id].(*types.Var); ok && !v.IsField() {
			return v.Type()
		}
		return nil
	}
	if tv, ok := tp.info.Types[e]; ok && tv.IsValue() {
		return tv.Type
	}
	return nil
}

// viaEmbeddedIdentity: the selection reaches its field or method through an
// embedded field of the identity's type (w.agents, w embedding *botIdentity) —
// a use of the identity no expression in the source spells.
func (tp *typedTelegram) viaEmbeddedIdentity(sel *types.Selection) bool {
	T := sel.Recv()
	idx := sel.Index()
	for _, i := range idx[:len(idx)-1] {
		T = types.Unalias(T)
		if p, ok := T.Underlying().(*types.Pointer); ok {
			T = p.Elem()
		}
		st, ok := types.Unalias(T).Underlying().(*types.Struct)
		if !ok {
			return false
		}
		f := st.Field(i)
		if tp.isIdentity(f.Type()) {
			return true
		}
		T = f.Type()
	}
	return false
}

// boundToIdentity: sel is a method bound to a receiver that holds the
// identity — the identity itself (ident.refresh), one promoted through an
// embedded identity (w.refresh), or a holder of it (h.touch, hp.outer,
// vfW{h}.cur).
func (tp *typedTelegram) boundToIdentity(sel *types.Selection) bool {
	return sel != nil && sel.Kind() == types.MethodVal && (tp.holdsIdentity(sel.Recv()) || tp.viaEmbeddedIdentity(sel))
}

// selWalk follows the operand x of the selection x.f through the embedded
// fields on the selection's path. in: the struct that finally holds f lies in
// a botIdentity's own memory. viaPtr: that struct was reached through a
// pointer (x is one, or the last embedded field on the path is), so a
// pointer-receiver method binds that pointer, not a new address.
func (tp *typedTelegram) selWalk(x ast.Expr, sel *types.Selection) (in, viaPtr bool) {
	T := types.Unalias(sel.Recv())
	if p, ok := T.Underlying().(*types.Pointer); ok {
		T = types.Unalias(p.Elem())
		in, viaPtr = tp.isIdentity(T), true
	} else {
		in = tp.isIdentity(T) || tp.inIdentity(x)
	}
	idx := sel.Index()
	for _, i := range idx[:len(idx)-1] {
		st, ok := T.Underlying().(*types.Struct)
		if !ok {
			return false, false
		}
		ft := types.Unalias(st.Field(i).Type())
		if p, ok := ft.Underlying().(*types.Pointer); ok {
			T = types.Unalias(p.Elem())
			in, viaPtr = tp.isIdentity(T), true
		} else {
			T = ft
			in, viaPtr = in || tp.isIdentity(T), false
		}
	}
	return in, viaPtr
}

// inIdentity: the variable e denotes lies inside a botIdentity's own memory —
// a field path through the identity with no pointer indirection after it
// (ident.agents, (*ident).userID, ident.arr[1], a field promoted through an
// embedded value). The memory a pointer field POINTS TO is not
// (ident.bx.n, ident.sl[0]): refresh replaces the field, not its pointee.
func (tp *typedTelegram) inIdentity(e ast.Expr) bool {
	switch v := ast.Unparen(e).(type) {
	case *ast.SelectorExpr:
		if sel := tp.info.Selections[v]; sel != nil && sel.Kind() == types.FieldVal {
			in, _ := tp.selWalk(v.X, sel)
			return in
		}
	case *ast.IndexExpr:
		if T := tp.valueType(v.X); T != nil {
			if _, ok := types.Unalias(T).Underlying().(*types.Array); ok {
				return tp.inIdentity(v.X)
			}
		}
	case *ast.StarExpr:
		if T := tp.valueType(v.X); T != nil {
			if p, ok := types.Unalias(T).Underlying().(*types.Pointer); ok {
				return tp.isIdentity(p.Elem())
			}
		}
	}
	return false
}

func (tp *typedTelegram) typeString(T types.Type) string {
	return types.TypeString(T, types.RelativeTo(tp.pkg))
}

func (tp *typedTelegram) at(pos token.Pos) string { return tp.fset.Position(pos).String() }

// identityUses lists every use of the identity inside n, one line each: a
// selector through it (reported whole: ident.agents), a field or method
// promoted through an embedded one, or the bare value (an alias, <-ch, a
// conversion, *ident).
func (tp *typedTelegram) identityUses(n ast.Node) []string {
	var out []string
	var visit func(ast.Node) bool
	visit = func(x ast.Node) bool {
		switch v := x.(type) {
		case *ast.SelectorExpr:
			if T := tp.valueType(v.X); tp.isIdentity(T) {
				out = append(out, fmt.Sprintf("%s: %s — through a %s", tp.at(v.X.Pos()), types.ExprString(v), tp.typeString(T)))
				return false
			}
			if sel := tp.info.Selections[v]; sel != nil && tp.viaEmbeddedIdentity(sel) {
				out = append(out, fmt.Sprintf("%s: %s — promoted through an embedded %s", tp.at(v.Pos()), types.ExprString(v), tp.typeString(tp.identNamed)))
				return false
			}
			ast.Inspect(v.X, visit) // never v.Sel: a field NAMED like a variable is not a use of it
			return false
		case ast.Expr:
			if T := tp.valueType(v); tp.isIdentity(T) {
				out = append(out, fmt.Sprintf("%s: %s — a %s value", tp.at(v.Pos()), types.ExprString(v), tp.typeString(T)))
				return false
			}
		}
		return true
	}
	ast.Inspect(n, visit)
	return out
}

// capturedHolders: the variables fl captures (declared outside it, not at
// package scope) whose type HOLDS the identity without being it — a struct,
// slice, map, channel, pointer or generic value carrying one. What the
// closure then does with the holder (its method, a helper, a call two levels
// down) is out of any one rule's sight, so the capture is the use.
func (tp *typedTelegram) capturedHolders(fl *ast.FuncLit) []string {
	var out []string
	ast.Inspect(fl.Body, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		if !ok {
			return true
		}
		v, ok := tp.info.Uses[id].(*types.Var)
		if !ok || v.IsField() || v.Pkg() != tp.pkg || v.Parent() == tp.pkg.Scope() {
			return true
		}
		if v.Pos() >= fl.Pos() && v.Pos() < fl.End() {
			return true // declared inside the closure: not a capture
		}
		if T := v.Type(); !tp.isIdentity(T) && tp.holdsIdentity(T) {
			out = append(out, fmt.Sprintf("%s: %s — a closure captures a %s, which holds the identity", tp.at(id.Pos()), id.Name, tp.typeString(T)))
		}
		return true
	})
	return out
}

// isIdentityMethod: fd is a method of the identity (pointer or value receiver).
func (tp *typedTelegram) isIdentityMethod(fd *ast.FuncDecl) bool {
	if fd == nil || fd.Recv == nil {
		return false
	}
	fn, ok := tp.info.Defs[fd.Name].(*types.Func)
	if !ok {
		return false
	}
	return tp.isIdentity(fn.Type().(*types.Signature).Recv().Type())
}

// eachFunc calls fn for every function body in the package's production
// files — every declared function or method — and for every package-level
// declaration (a func literal in a var initialiser) as "package scope".
func (tp *typedTelegram) eachFunc(fn func(name string, fd *ast.FuncDecl, body ast.Node)) {
	for _, f := range tp.files {
		for _, d := range f.Decls {
			switch v := d.(type) {
			case *ast.FuncDecl:
				if v.Body == nil {
					continue
				}
				name := v.Name.Name
				if obj, ok := tp.info.Defs[v.Name].(*types.Func); ok {
					name = strings.ReplaceAll(obj.FullName(), tp.pkg.Path()+".", "")
				}
				fn(name, v, v.Body)
			case *ast.GenDecl:
				fn("package scope", nil, v)
			}
		}
	}
}

// closureUses: every identity use, and every captured holder of it, inside a
// func literal in body — deduped (a use inside nested literals is one use).
func (tp *typedTelegram) closureUses(body ast.Node) []string {
	seen := map[string]bool{}
	var out []string
	add := func(us []string) {
		for _, u := range us {
			if !seen[u] {
				seen[u] = true
				out = append(out, u)
			}
		}
	}
	ast.Inspect(body, func(n ast.Node) bool {
		if fl, ok := n.(*ast.FuncLit); ok {
			add(tp.identityUses(fl.Body))
			add(tp.capturedHolders(fl))
		}
		return true
	})
	return out
}

// declOf: the declaration of a function or method object, through a generic
// instantiation to its origin.
func (tp *typedTelegram) declOf(obj types.Object) *ast.FuncDecl {
	if fn, ok := obj.(*types.Func); ok {
		return tp.decls[fn.Origin()]
	}
	return nil
}

// goroutineRule is TestRunBotGoroutinesReadNoBotIdentityField's rule: no
// closure outside botIdentity's methods uses or captures the identity, and
// no go statement binds, starts or hands over what reaches it.
func (tp *typedTelegram) goroutineRule() []string {
	var bad []string
	tp.eachFunc(func(name string, fd *ast.FuncDecl, body ast.Node) {
		// Closures outside botIdentity's methods (those are closureRule's).
		if !tp.isIdentityMethod(fd) {
			for _, u := range tp.closureUses(body) {
				bad = append(bad, u+" — used inside a closure in "+name)
			}
		}
		// Every go statement: what it binds, what it starts, what it hands over.
		ast.Inspect(body, func(n ast.Node) bool {
			g, ok := n.(*ast.GoStmt)
			if !ok {
				return true
			}
			switch fun := ast.Unparen(g.Call.Fun).(type) {
			case *ast.FuncLit:
				// its body: the closure rule above (or closureRule)
			case *ast.SelectorExpr:
				sel := tp.info.Selections[fun]
				if tp.boundToIdentity(sel) {
					bad = append(bad, fmt.Sprintf("%s: go %s — binds a method to a %s, which holds the identity; its body runs on the goroutine (in %s)", tp.at(fun.Pos()), types.ExprString(fun), tp.typeString(sel.Recv()), name))
				} else if sel != nil && sel.Kind() == types.MethodVal {
					if decl := tp.declOf(sel.Obj()); decl != nil {
						for _, u := range tp.identityUses(decl.Body) {
							bad = append(bad, u+" — in "+types.ExprString(fun)+", which `go` runs on a goroutine (in "+name+")")
						}
					}
				}
			case *ast.Ident:
				if decl := tp.declOf(tp.info.Uses[fun]); decl != nil {
					for _, u := range tp.identityUses(decl.Body) {
						bad = append(bad, u+" — in "+fun.Name+", which `go` runs on a goroutine (in "+name+")")
					}
				}
			}
			// Arguments are evaluated on the main loop (Go spec): a field
			// read THERE is the capture. A value that still HOLDS the
			// identity hands the goroutine the fields refresh rewrites (a
			// pointer INTO it: escapeRule, wherever it is formed).
			for _, a := range g.Call.Args {
				if T := tp.valueType(a); tp.holdsIdentity(T) {
					bad = append(bad, fmt.Sprintf("%s: %s — a %s handed to a goroutine (in %s)", tp.at(a.Pos()), types.ExprString(a), tp.typeString(T), name))
				}
			}
			return true
		})
	})
	return bad
}

// closureRule is TestBotIdentityClosuresReadNoReceiverField's rule: the
// closures botIdentity's methods build neither use nor capture the identity,
// and no method value anywhere is bound to a receiver that holds it.
func (tp *typedTelegram) closureRule() []string {
	var bad []string
	tp.eachFunc(func(name string, fd *ast.FuncDecl, body ast.Node) {
		if tp.isIdentityMethod(fd) {
			for _, u := range tp.closureUses(body) {
				bad = append(bad, u+" — in a closure built by "+name)
			}
		}
		// A method value is a closure over its receiver. Called on the spot
		// it runs here; anywhere else it carries its receiver — and what
		// that holds — with it. (A `go` statement's call runs elsewhere:
		// goroutineRule.)
		called := map[*ast.SelectorExpr]bool{}
		ast.Inspect(body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if se, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
					called[se] = true
				}
			}
			return true
		})
		ast.Inspect(body, func(n ast.Node) bool {
			if se, ok := n.(*ast.SelectorExpr); ok && !called[se] {
				if sel := tp.info.Selections[se]; tp.boundToIdentity(sel) {
					bad = append(bad, fmt.Sprintf("%s: %s — a method value bound to a %s, which holds the identity (a closure over it) in %s", tp.at(se.Pos()), types.ExprString(se), tp.typeString(sel.Recv()), name))
				}
			}
			return true
		})
	})
	return bad
}

// escapeRule is TestBotIdentityEscapesNoOtherWay's rule: no package-level
// variable holds the identity, nothing holding it is converted to an
// interface or unsafe.Pointer, no pointer into it is formed, and no generic
// is instantiated with it. slots counts the interface slots checked.
func (tp *typedTelegram) escapeRule() (bad []string, slots int) {
	for _, n := range tp.pkg.Scope().Names() {
		if v, ok := tp.pkg.Scope().Lookup(n).(*types.Var); ok && tp.holdsIdentity(v.Type()) {
			bad = append(bad, fmt.Sprintf("%s: var %s %s — a package-level variable holding the identity", tp.at(v.Pos()), n, tp.typeString(v.Type())))
		}
	}
	// A generic instantiated with a type that holds the identity: inside it
	// the identity is a type parameter, and every rule here asks a type.
	var inst []string
	for id, in := range tp.info.Instances {
		for i := 0; i < in.TypeArgs.Len(); i++ {
			if a := in.TypeArgs.At(i); tp.holdsIdentity(a) {
				inst = append(inst, fmt.Sprintf("%s: %s[…%s…] — a generic instantiated with a type that holds the identity; inside it the identity is a type parameter no rule can see", tp.at(id.Pos()), id.Name, tp.typeString(a)))
			}
		}
	}
	sort.Strings(inst)
	bad = append(bad, inst...)

	opaque := func(T types.Type) bool {
		if T == nil {
			return false
		}
		T = types.Unalias(T)
		if _, ok := T.(*types.TypeParam); ok {
			return false
		}
		if b, ok := T.Underlying().(*types.Basic); ok {
			return b.Kind() == types.UnsafePointer
		}
		return types.IsInterface(T)
	}
	check := func(val ast.Expr, slot types.Type, what, where string) {
		if val == nil || !opaque(slot) {
			return
		}
		slots++
		if T := tp.valueType(val); T != nil && !opaque(T) && tp.holdsIdentity(T) {
			bad = append(bad, fmt.Sprintf("%s: %s — a %s converted to %s (%s, in %s)", tp.at(val.Pos()), types.ExprString(val), tp.typeString(T), tp.typeString(slot), what, where))
		}
	}
	var walk func(body ast.Node, sig *types.Signature, where string)
	walk = func(body ast.Node, sig *types.Signature, where string) {
		ast.Inspect(body, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.FuncLit:
				s, _ := tp.valueType(v).(*types.Signature)
				walk(v.Body, s, where)
				return false
			// A pointer INTO the identity: whoever holds it reads or writes
			// the field refresh replaces, and its type no longer says so.
			case *ast.UnaryExpr:
				if v.Op == token.AND && tp.inIdentity(v.X) {
					bad = append(bad, fmt.Sprintf("%s: %s — a pointer into the identity (in %s)", tp.at(v.Pos()), types.ExprString(v), where))
				}
			case *ast.SliceExpr:
				if T := tp.valueType(v.X); T != nil {
					if _, ok := types.Unalias(T).Underlying().(*types.Array); ok && tp.inIdentity(v.X) {
						bad = append(bad, fmt.Sprintf("%s: %s — a slice of an array inside the identity: a pointer into it (in %s)", tp.at(v.Pos()), types.ExprString(v), where))
					}
				}
			case *ast.SelectorExpr:
				// x.m with a pointer receiver on an addressable x takes &x
				// with no & in the source.
				if sel := tp.info.Selections[v]; sel != nil && sel.Kind() == types.MethodVal {
					if fn, ok := sel.Obj().(*types.Func); ok {
						if r := fn.Type().(*types.Signature).Recv(); r != nil {
							if _, ptr := types.Unalias(r.Type()).Underlying().(*types.Pointer); ptr {
								if in, viaPtr := tp.selWalk(v.X, sel); in && !viaPtr {
									bad = append(bad, fmt.Sprintf("%s: %s — %s has a pointer receiver: a pointer into the identity, taken implicitly (in %s)", tp.at(v.Pos()), types.ExprString(v), fn.FullName(), where))
								}
							}
						}
					}
				}
			case *ast.CallExpr:
				tv := tp.info.Types[v.Fun]
				switch {
				case tv.IsType():
					if len(v.Args) == 1 {
						check(v.Args[0], tv.Type, "conversion", where)
					}
				case tv.IsBuiltin():
					if id, ok := ast.Unparen(v.Fun).(*ast.Ident); ok && id.Name == "append" && len(v.Args) > 1 && !v.Ellipsis.IsValid() {
						if T := tp.valueType(v.Args[0]); T != nil {
							if s, ok := T.Underlying().(*types.Slice); ok {
								for _, a := range v.Args[1:] {
									check(a, s.Elem(), "append", where)
								}
							}
						}
					}
				case tv.Type != nil:
					s, ok := tv.Type.Underlying().(*types.Signature)
					if !ok {
						break
					}
					ps := s.Params()
					for i, a := range v.Args {
						var pt types.Type
						switch {
						case s.Variadic() && i >= ps.Len()-1:
							last := ps.At(ps.Len() - 1).Type()
							if v.Ellipsis.IsValid() {
								pt = last
							} else if sl, ok := last.Underlying().(*types.Slice); ok {
								pt = sl.Elem()
							}
						case i < ps.Len():
							pt = ps.At(i).Type()
						}
						check(a, pt, "argument", where)
					}
				}
			case *ast.AssignStmt:
				if v.Tok == token.ASSIGN && len(v.Lhs) == len(v.Rhs) {
					for i := range v.Lhs {
						check(v.Rhs[i], tp.valueType(v.Lhs[i]), "assignment", where)
					}
				}
			case *ast.ValueSpec:
				for i, nm := range v.Names {
					if i < len(v.Values) && v.Type != nil {
						if obj := tp.info.Defs[nm]; obj != nil {
							check(v.Values[i], obj.Type(), "declaration", where)
						}
					}
				}
			case *ast.ReturnStmt:
				if sig != nil && len(v.Results) == sig.Results().Len() {
					for i, r := range v.Results {
						check(r, sig.Results().At(i).Type(), "return", where)
					}
				}
			case *ast.CompositeLit:
				T := tp.valueType(v)
				if T == nil {
					break
				}
				switch u := T.Underlying().(type) {
				case *types.Struct:
					for i, el := range v.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							if k, ok := kv.Key.(*ast.Ident); ok {
								if f, ok := tp.info.Uses[k].(*types.Var); ok {
									check(kv.Value, f.Type(), "field", where)
								}
							}
						} else if i < u.NumFields() {
							check(el, u.Field(i).Type(), "field", where)
						}
					}
				case *types.Slice:
					for _, el := range v.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							el = kv.Value
						}
						check(el, u.Elem(), "element", where)
					}
				case *types.Array:
					for _, el := range v.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							el = kv.Value
						}
						check(el, u.Elem(), "element", where)
					}
				case *types.Map:
					for _, el := range v.Elts {
						if kv, ok := el.(*ast.KeyValueExpr); ok {
							check(kv.Key, u.Key(), "map key", where)
							check(kv.Value, u.Elem(), "map value", where)
						}
					}
				}
			case *ast.SendStmt:
				if T := tp.valueType(v.Chan); T != nil {
					if ch, ok := T.Underlying().(*types.Chan); ok {
						check(v.Value, ch.Elem(), "send", where)
					}
				}
			}
			return true
		})
	}
	tp.eachFunc(func(name string, fd *ast.FuncDecl, body ast.Node) {
		var sig *types.Signature
		if fd != nil {
			if fn, ok := tp.info.Defs[fd.Name].(*types.Func); ok {
				sig = fn.Type().(*types.Signature)
			}
		}
		walk(body, sig, name)
	})
	return bad, slots
}

// runBot's per-message goroutine uses no part of the bot identity: refresh on
// the main loop may replace every field of it while a message is answered.
func TestRunBotGoroutinesReadNoBotIdentityField(t *testing.T) {
	tp := loadTypedTelegram(t)
	bad := tp.goroutineRule()

	// Vacuity: runBot, its identity variable, and the AI goroutine are where
	// the pin says they are.
	var runBot *ast.FuncDecl
	for obj, fd := range tp.decls {
		if _, ok := obj.(*types.Func); ok && fd.Recv == nil && fd.Name.Name == "runBot" {
			runBot = fd
		}
	}
	if runBot == nil || runBot.Body == nil {
		t.Fatal("package telegram declares no runBot — the loop the pin guards is gone (re-anchor it)")
	}
	idents := map[string]bool{}
	for id, obj := range tp.info.Defs {
		if v, ok := obj.(*types.Var); ok && !v.IsField() && id.Pos() >= runBot.Body.Pos() && id.Pos() < runBot.Body.End() && tp.isIdentity(v.Type()) {
			idents[id.Name] = true
		}
	}
	aiGoroutines := 0
	ast.Inspect(runBot.Body, func(n ast.Node) bool {
		g, ok := n.(*ast.GoStmt)
		if !ok {
			return true
		}
		ast.Inspect(g.Call.Fun, func(m ast.Node) bool {
			if se, ok := m.(*ast.SelectorExpr); ok {
				if fn, ok := tp.info.Uses[se.Sel].(*types.Func); ok && fn.Name() == "Run" && fn.Pkg() != nil && fn.Pkg().Path() == "nofx/telegram/agent" {
					aiGoroutines++
				}
			}
			return true
		})
		return true
	})

	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("runBot's per-message goroutine can reach the bot identity, which refresh() on the main loop reassigns (data race; M3 made the re-mint routine) — capture what it needs on the main loop and pass THAT in:\n  %s", strings.Join(bad, "\n  "))
	}
	if len(idents) == 0 {
		t.Errorf("runBot declares no variable of type %s — the identity the pin guards is gone (re-anchor it)", tp.typeString(tp.identNamed))
	}
	if aiGoroutines == 0 {
		t.Errorf("runBot has no go statement calling (*agent.Manager).Run — the AI goroutine the pin guards is gone (re-anchor it)")
	}
}

// The closures botIdentity's methods build — and every method value bound to
// a receiver that holds the identity, anywhere — read no field of it: refresh
// hands its LLM factory to agent.NewManager, and the manager calls it on the
// per-message goroutine while the next refresh rewrites b.userID.
func TestBotIdentityClosuresReadNoReceiverField(t *testing.T) {
	tp := loadTypedTelegram(t)
	bad := tp.closureRule()
	methods, handoffs := 0, 0
	tp.eachFunc(func(name string, fd *ast.FuncDecl, body ast.Node) {
		if !tp.isIdentityMethod(fd) {
			return
		}
		methods++
		ast.Inspect(body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if se, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
					if fn, ok := tp.info.Uses[se.Sel].(*types.Func); ok && fn.Name() == "NewManager" && fn.Pkg() != nil && fn.Pkg().Path() == "nofx/telegram/agent" {
						handoffs++
					}
				}
			}
			return true
		})
	})
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("a closure the bot hands out reads the identity, which the next refresh() rewrites on the main loop while the per-message goroutine runs the closure (data race) — close over locals:\n  %s", strings.Join(bad, "\n  "))
	}
	if methods == 0 || handoffs == 0 {
		t.Errorf("botIdentity has %d methods, %d calls to agent.NewManager among them — the LLM-factory hand-off the pin guards is gone (re-anchor it)", methods, handoffs)
	}
}

// No package-level variable holds the identity, no value holding it is
// converted to an interface or unsafe.Pointer, no pointer into it is formed,
// and no generic is instantiated with it: each is a road to its fields from
// any goroutine that the two tests above cannot see.
func TestBotIdentityEscapesNoOtherWay(t *testing.T) {
	tp := loadTypedTelegram(t)
	bad, slots := tp.escapeRule()
	if len(bad) > 0 {
		sort.Strings(bad)
		t.Errorf("the bot identity escapes to where any goroutine can reach its fields (refresh() rewrites them on the main loop):\n  %s", strings.Join(bad, "\n  "))
	}
	if slots == 0 {
		t.Errorf("the interface-conversion walk checked 0 interface slots in package telegram — it walked nothing (re-anchor it)")
	}
}

// synthPrelude is the synthetic proof's package: a botIdentity shaped like
// the real one (plus the value, array and slice fields the pointer rules
// need), holders of it, and helpers — all of it GREEN on its own. Each case
// adds one file.
const synthPrelude = `package synth

type manager struct{ n int }

func (m *manager) Run() int { return m.n }

type vfVal struct{ n int }

func (v *vfVal) bump()   { v.n++ }
func (v *vfVal) leak()   { vfLeak = v }
func (v vfVal) get() int { return v.n }

var vfLeak *vfVal

type vfBox2 struct{ n int }

type botIdentity struct {
	userID string
	agents *manager
	arr    [2]*manager
	sl     []*manager
	val    vfVal
	bx     *vfBox2
	vfVal
}

func (b *botIdentity) refresh() bool {
	st, userID := b.agents, b.userID
	b.agents = &manager{}
	f := func() string { _ = st; return userID }
	return f() != ""
}

type vfHolder struct{ p *botIdentity }

func (h vfHolder) cur() *manager { return h.p.agents }
func (h vfHolder) outer()        { _ = h.cur() }
func (h vfHolder) touch()        { _ = h.p.agents }

type vfW struct{ vfHolder }
type vfW2 struct{ *vfHolder }
type vfPtrs struct{ a **manager }

func vfGet(h vfHolder) *botIdentity             { return h.p }
func vfAgentsOf(h vfHolder) *manager            { return h.p.agents }
func vfFirstAgents(ids []*botIdentity) *manager { return ids[0].agents }
func vfGo[T any](v T, f func(T))                { go f(v) }
func vfUse(id *botIdentity)                     { _ = id.agents }

type vfBox[T any] struct{ v T }

func (b vfBox[T]) spawn(f func(T)) { go f(b.v) }

func runBot() {
	ident := &botIdentity{}
	ident.refresh()
	agents := ident.agents
	go func(agents *manager, userID string) { _ = agents.Run(); _ = userID }(agents, ident.userID)
}
`

// The rules above, run over a synthetic package: every road the verifiers
// found (and the ones this repair added) is RED — by the rule that names it —
// and every shape the pins allow is GREEN. The production tests run the same
// rule functions; this proves the functions, not a copy of them.
func TestBotIdentityPinRulesCatchEveryRoad(t *testing.T) {
	loadTypedTelegram(t) // the go command's export data, for the imports below
	run := func(src string) ([]string, error) {
		fset := token.NewFileSet()
		pre, err := parser.ParseFile(fset, "prelude.go", synthPrelude, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		files := []*ast.File{pre}
		if src != "" {
			f, err := parser.ParseFile(fset, "case.go", "package synth\n\n"+src+"\n", parser.SkipObjectResolution)
			if err != nil {
				return nil, err
			}
			files = append(files, f)
		}
		tp, err := checkTyped(fset, files, "nofx/telegram/synth", exportImporter(fset, telegramExports))
		if err != nil {
			return nil, err
		}
		bad := tp.goroutineRule()
		bad = append(bad, tp.closureRule()...)
		esc, _ := tp.escapeRule()
		return append(bad, esc...), nil
	}

	roads := []struct{ name, src, want string }{
		// The dispatch's shapes (verifier ha2 defect 2).
		{"ME1 a closure reads a field", `func c(ident *botIdentity) { go func() { _ = ident.agents.Run() }() }`, "ident.agents — through"},
		{"ME2 an alias", `func c(ident *botIdentity) { id := ident; go func() { _ = id.agents }() }`, "id.agents — through"},
		{"ME3 an identity method's closure reads the receiver", `func (b *botIdentity) f() func() string { return func() string { return b.userID } }`, "b.userID — through"},
		{"ME4 a method value bound to the identity", `func (b *botIdentity) f() func() bool { return b.refresh }`, "b.refresh — a method value bound to"},
		{"V2 a holder's field", `func c(ident *botIdentity) { h := vfHolder{ident}; go func() { _ = h.p.agents }() }`, "h.p.agents — through"},
		{"V3 a slice element", `func c(ident *botIdentity) { ids := []*botIdentity{ident}; go func() { _ = ids[0].agents }() }`, "ids[0].agents — through"},
		{"V4 a map element", `func c(ident *botIdentity) { m := map[int]*botIdentity{0: ident}; go func() { _ = m[0].agents }() }`, "m[0].agents — through"},
		{"V5a a package func returns it", `func c(ident *botIdentity) { h := vfHolder{ident}; go func() { _ = vfGet(h).agents }() }`, "vfGet(h).agents — through"},
		{"V5b a closure returns it", `func c(ident *botIdentity) { get := func() *botIdentity { return ident }; go func() { _ = get().agents }() }`, "get().agents — through"},
		{"V6a a method value as a go argument", `func c(ident *botIdentity) { go func(f func() bool) { f() }(ident.refresh) }`, "ident.refresh — a method value bound to"},
		{"V7 an identity method's closure through a local copy", `func (b *botIdentity) f() func() string { bb := b; return func() string { return bb.userID } }`, "bb.userID — through"},
		{"go on an identity method", `func c(ident *botIdentity) { go ident.refresh() }`, "go ident.refresh — binds a method to a *botIdentity"},
		{"the identity handed to a goroutine", `func c(ident *botIdentity) { go vfUse(ident) }`, "ident — a *botIdentity handed to a goroutine"},
		{"the identity converted to an interface", `func c(ident *botIdentity) { var x any = ident; _ = x }`, "converted to any"},
		{"the identity converted to unsafe.Pointer", "import \"unsafe\"\n\nfunc c(ident *botIdentity) unsafe.Pointer { return unsafe.Pointer(ident) }", "converted to unsafe.Pointer"},
		{"a package-level variable holding it", `var vfKept []*botIdentity`, "var vfKept []*botIdentity — a package-level variable"},
		// Verifier vf-hc defect 1, P1: a pointer into the identity.
		{"E1a a field pointer, captured", `func c(ident *botIdentity) { ap := &ident.agents; go func() { _ = *ap }() }`, "&ident.agents — a pointer into the identity"},
		{"E1b a field pointer as a go argument", `func c(ident *botIdentity) { go func(ap **manager) { _ = *ap }(&ident.agents) }`, "&ident.agents — a pointer into the identity"},
		{"E1c an identity method's closure through a field pointer", `func (b *botIdentity) f() func() string { up := &b.userID; return func() string { return *up } }`, "&b.userID — a pointer into the identity"},
		{"a method returns a field pointer", `func (b *botIdentity) agentsPtr() **manager { return &b.agents }`, "&b.agents — a pointer into the identity"},
		{"a struct of field pointers", `func c(ident *botIdentity) { ps := vfPtrs{&ident.agents}; go func() { _ = *ps.a }() }`, "&ident.agents — a pointer into the identity"},
		{"an array element's address", `func c(ident *botIdentity) { p := &ident.arr[1]; go func() { _ = *p }() }`, "&ident.arr[1] — a pointer into the identity"},
		{"a field address through the dereferenced identity", `func c(ident *botIdentity) { p := &(*ident).userID; go func() { _ = *p }() }`, "&(*ident).userID — a pointer into the identity"},
		{"a slice of an array field", `func c(ident *botIdentity) { s := ident.arr[:]; go func() { _ = s[0] }() }`, "ident.arr[:] — a slice of an array inside the identity"},
		{"implicit &: a method value on a value field", `func c(ident *botIdentity) { f := ident.val.bump; go f() }`, "ident.val.bump — (*nofx/telegram/synth.vfVal).bump has a pointer receiver"},
		{"implicit &: go on a value field's pointer method", `func c(ident *botIdentity) { go ident.val.bump() }`, "ident.val.bump — (*nofx/telegram/synth.vfVal).bump has a pointer receiver"},
		{"implicit &: a synchronous call that keeps it", `func c(ident *botIdentity) { ident.val.leak(); go func() { _ = vfLeak.n }() }`, "ident.val.leak — (*nofx/telegram/synth.vfVal).leak has a pointer receiver"},
		{"implicit &: promoted through an embedded value", `func c(ident *botIdentity) { ident.leak(); go func() { _ = vfLeak.n }() }`, "ident.leak — (*nofx/telegram/synth.vfVal).leak has a pointer receiver"},
		// P2: a closure captures a holder.
		{"E2a a holder's method in a closure", `func c(ident *botIdentity) { h := vfHolder{ident}; go func() { _ = h.cur() }() }`, "h — a closure captures a vfHolder"},
		{"E2b a holder to a helper in a closure", `func c(ident *botIdentity) { h := vfHolder{ident}; go func() { _ = vfAgentsOf(h) }() }`, "h — a closure captures a vfHolder"},
		{"E2d a slice to a helper in a closure", `func c(ident *botIdentity) { ids := []*botIdentity{ident}; go func() { _ = vfFirstAgents(ids) }() }`, "ids — a closure captures a []*botIdentity"},
		{"a pointer to a holder, captured", `func c(ident *botIdentity) { h := vfHolder{ident}; hp := &h; go func() { _ = hp.cur() }() }`, "hp — a closure captures a *vfHolder"},
		{"a channel of holders, captured", `func c(ident *botIdentity) { hc := make(chan vfHolder, 1); hc <- vfHolder{ident}; go func() { _ = (<-hc).cur() }() }`, "hc — a closure captures a chan vfHolder"},
		{"an embedded pointer holder, captured", `func c(ident *botIdentity) { w := vfW2{&vfHolder{ident}}; go func() { _ = w.cur() }() }`, "w — a closure captures a vfW2"},
		{"an identity method's closure captures a holder", `func (b *botIdentity) f() func() *manager { h := vfHolder{b}; return func() *manager { return h.cur() } }`, "h — a closure captures a vfHolder"},
		// P3: a method value, or go, on a holder.
		{"E2c an identity method hands out a holder's method value", `func (b *botIdentity) f() func() *manager { h := vfHolder{b}; return h.cur }`, "h.cur — a method value bound to a vfHolder"},
		{"E3 go on a holder's method, two levels down", `func c(ident *botIdentity) { h := vfHolder{ident}; go h.outer() }`, "go h.outer — binds a method to a vfHolder"},
		{"E4a a holder's method value, then go", `func c(ident *botIdentity) { h := vfHolder{ident}; f := h.touch; go f() }`, "h.touch — a method value bound to a vfHolder"},
		{"E4b a holder's method value to time.AfterFunc", "import \"time\"\n\nfunc c(ident *botIdentity) { time.AfterFunc(0, vfHolder{ident}.touch) }", "vfHolder{…}.touch — a method value bound to a vfHolder"},
		{"go on a call that returns a holder's method value", "func c(ident *botIdentity) { go vfMk(vfHolder{ident})() }\n\nfunc vfMk(h vfHolder) func() { return h.touch }", "h.touch — a method value bound to a vfHolder"},
		{"go on a method promoted through an embedded holder", `func c(ident *botIdentity) { go vfW{vfHolder{ident}}.outer() }`, "binds a method to a vfW"},
		// P4: a generic instantiated with it.
		{"E5 a generic spawner, inferred", `func c(ident *botIdentity) { vfGo(ident, vfUse) }`, "vfGo[…*botIdentity…] — a generic instantiated"},
		{"a generic spawner, explicit", `func c(ident *botIdentity) { vfGo[*botIdentity](ident, vfUse) }`, "vfGo[…*botIdentity…] — a generic instantiated"},
		{"a generic type's method spawns", `func c(ident *botIdentity) { vfBox[*botIdentity]{v: ident}.spawn(vfUse) }`, "vfBox[…*botIdentity…] — a generic instantiated"},
		{"an atomic.Pointer of the identity", "import \"sync/atomic\"\n\nfunc c(ident *botIdentity) *atomic.Pointer[botIdentity] {\n\tvar ap atomic.Pointer[botIdentity]\n\tap.Store(ident)\n\treturn &ap\n}", "Pointer[…botIdentity…] — a generic instantiated"},
	}
	allowed := []struct{ name, src string }{
		{"the prelude alone: holders and helpers declared, a capture on the main loop", ``},
		{"a field read into a local, then handed in", `func c(ident *botIdentity) { agents := ident.agents; go func(a *manager) { _ = a.Run() }(agents) }`},
		{"a field read as a go argument", `func c(ident *botIdentity) { go func(a *manager, u string) { _, _ = a, u }(ident.agents, ident.userID) }`},
		{"go on a field's own method", `func c(ident *botIdentity) { go ident.agents.Run() }`},
		{"a synchronous identity method call", `func c(ident *botIdentity) bool { return ident.refresh() }`},
		{"a pointer into a field's POINTEE", `func c(ident *botIdentity) { p := &ident.bx.n; go func() { _ = *p }() }`},
		{"a slice field's backing array", `func c(ident *botIdentity) { s := ident.sl[:]; q := &ident.sl[0]; go func() { _, _ = s, q }() }`},
		{"a value-receiver method value on a value field (a copy)", `func c(ident *botIdentity) { g := ident.val.get; go func() { _ = g() }() }`},
		{"a generic over a non-identity", `func c(ident *botIdentity) { vfGo(ident.userID, func(string) {}) }`},
	}

	for _, r := range roads {
		bad, err := run(r.src)
		if err != nil {
			t.Errorf("road %q does not type-check — the proof proves nothing for it: %v", r.name, err)
			continue
		}
		hit := false
		for _, b := range bad {
			if strings.Contains(b, r.want) {
				hit = true
			}
		}
		if !hit {
			t.Errorf("road %q is not caught by the rule that names %q — the pins would pass it:\n  got: %s", r.name, r.want, strings.Join(bad, "\n       "))
		}
	}
	for _, a := range allowed {
		bad, err := run(a.src)
		if err != nil {
			t.Errorf("allowed shape %q does not type-check: %v", a.name, err)
			continue
		}
		if len(bad) > 0 {
			t.Errorf("allowed shape %q is flagged — the rules over-reach:\n  %s", a.name, strings.Join(bad, "\n  "))
		}
	}
}
