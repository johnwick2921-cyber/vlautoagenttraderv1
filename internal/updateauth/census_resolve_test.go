package updateauth

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// ── M3 census-repair verify #3: rule 6's admission is checked against the
// compiler's resolver, over every declaration form ────────────────────────
//
// Rule 6 admits the loaded key into three callees by NAME: clear (the
// builtin), <import name>.VerifyMAC, and <a LoadAdmin result>
// .PasswordStillBound. The census said it "can only over-report"; probes P1
// (a receiver type parameter named clear) and P2 (the import name shadowed
// before a LoadAdmin binding) proved that false — each trusted name had a
// declaration form the census did not read, and each admitted a real mint.
//
// So the claim is no longer prose. This matrix writes one file per (trusted
// name × declaration form) — every site go/types declares an identifier
// (go1.25.13 go/types, every check.declare / declareTypeParam site: short
// var decl — plain, if-init, switch-init, select receive —, local var,
// const, alias and defined type, range key and value, type-switch symbol,
// parameter, named result, function type parameter, receiver name, receiver
// type parameter — plain and behind parens/pointer in a multi-parameter
// receiver —, a function literal's parameter and named result, and for clear
// the package block: var, const, type, func) — and asks
// go/types, with a stub importer, what the admitted call's name RESOLVES to:
//   - every shadow cell must resolve AWAY from the trusted object (the cell is
//     a genuine shadow — the matrix cannot pass vacuously) and the census must
//     refuse the key use;
//   - every baseline cell must resolve TO the trusted object (the oracle
//     works) and the census must admit it (no false refusal on the real shape).
// Labels are not in the matrix: a label lives in its own namespace and can
// shadow nothing here. An import can shadow neither name either: a second
// import of updateauth is refused by rule 2 and any other package under the
// same name does not compile, and `import clear "…"` makes clear(key) a
// compile error ("use of package clear not in selector").

// resolveTrust is the oracle: for the one admitted call in files (clear(key),
// <x>.VerifyMAC(key, …) or <adm>.PasswordStillBound(key, …)), whether go/types
// resolves its trusted name to the trusted object — the universe's builtin
// clear; the PkgName of nofx/internal/updateauth; a variable DEFINED by `:=`
// from a call of <the updateauth PkgName>.LoadAdmin.
func resolveTrust(t *testing.T, files map[string]string) (kind string, trusted bool) {
	t.Helper()
	fset := token.NewFileSet()
	var parsed []*ast.File
	var names []string
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		f, err := parser.ParseFile(fset, n, files[n], 0)
		if err != nil {
			t.Fatalf("%s does not parse: %v", n, err)
		}
		parsed = append(parsed, f)
	}
	stub := types.NewPackage(resolveModule+"/internal/updateauth", "updateauth")
	stub.MarkComplete()
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}, Defs: map[*ast.Ident]types.Object{}}
	conf := types.Config{
		Importer: importerFunc(func(path string) (*types.Package, error) {
			if path == stub.Path() {
				return stub, nil
			}
			return types.NewPackage(path, path[strings.LastIndex(path, "/")+1:]), nil
		}),
		Error: func(error) {}, // the stub exports nothing; resolution still runs
	}
	_, _ = conf.Check("api", fset, parsed, info)

	isUpdateAuth := func(e ast.Expr) bool {
		id, ok := e.(*ast.Ident)
		if !ok {
			return false
		}
		pn, ok := info.Uses[id].(*types.PkgName)
		return ok && pn.Imported() == stub
	}
	parent := map[ast.Node]ast.Node{}
	for _, f := range parsed {
		var stack []ast.Node
		ast.Inspect(f, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			if len(stack) > 0 {
				parent[n] = stack[len(stack)-1]
			}
			stack = append(stack, n)
			return true
		})
	}
	found := 0
	for _, f := range parsed {
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			if a, ok := call.Args[0].(*ast.Ident); !ok || a.Name != "key" {
				return true
			}
			switch fn := call.Fun.(type) {
			case *ast.Ident:
				if fn.Name == "clear" {
					found++
					kind, trusted = "clear", info.Uses[fn] == types.Universe.Lookup("clear")
				}
			case *ast.SelectorExpr:
				switch fn.Sel.Name {
				case "VerifyMAC":
					found++
					kind, trusted = "VerifyMAC", isUpdateAuth(fn.X)
				case "PasswordStillBound":
					found++
					kind, trusted = "PasswordStillBound", false
					x, ok := fn.X.(*ast.Ident)
					if !ok {
						return true
					}
					v, ok := info.Uses[x].(*types.Var)
					if !ok {
						return true
					}
					for def, obj := range info.Defs {
						if obj != v {
							continue
						}
						as, ok := parent[def].(*ast.AssignStmt)
						if !ok || as.Tok != token.DEFINE || len(as.Rhs) != 1 {
							break
						}
						c, ok := as.Rhs[0].(*ast.CallExpr)
						if !ok {
							break
						}
						s, ok := c.Fun.(*ast.SelectorExpr)
						trusted = ok && s.Sel.Name == "LoadAdmin" && isUpdateAuth(s.X)
					}
				}
			}
			return true
		})
	}
	if found != 1 {
		t.Fatalf("oracle: want exactly one admitted-shape call on the key, found %d", found)
	}
	return kind, trusted
}

type importerFunc func(string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// resolveModule is the synthetic module's path (mintBase writes "module nofx").
const resolveModule = "nofx"

// resolvePrelude declares, once per matrix file, the stand-ins a shadow binds.
const resolvePrelude = `package api

import "nofx/internal/updateauth"

type sinkFn func([]byte)

type bytesT []byte

type fakeNS struct{}

func (fakeNS) VerifyMAC(k []byte, r, j string, e int64, h string) bool { return true }

func (fakeNS) LoadAdmin(d string) (fakeAdmin, error) { return fakeAdmin{}, nil }

type fakeAdmin struct{}

func (fakeAdmin) PasswordStillBound(k []byte, h string) bool { return true }

type box[P any] struct{}

type box2[K comparable, P any] struct{}
`

// resolveName is one trusted name: what shadows it, and the admitted call
// that trusts it.
type resolveName struct {
	id       string // matrix label
	name     string // the identifier a shadow re-binds
	valType  string // a defined type whose VALUE can shadow it
	value    string // a value of valType
	typeName string // a type an alias declaration can bind it to
	pre      string // statements before the shadow (after the key load)
	use      string // the admitted call (and any binding it trusts), inside the shadow's scope
}

var resolveNames = []resolveName{
	{id: "builtin clear", name: "clear", valType: "sinkFn", value: "sinkFn(nil)", typeName: "bytesT",
		use: "clear(key)"},
	{id: "import name at VerifyMAC", name: "updateauth", valType: "fakeNS", value: "fakeNS{}", typeName: "fakeNS",
		use: `_ = updateauth.VerifyMAC(key, "r", "j", 0, "h")`},
	{id: "import name at LoadAdmin", name: "updateauth", valType: "fakeNS", value: "fakeNS{}", typeName: "fakeNS",
		use: "adm, _ := updateauth.LoadAdmin(d)\n\tadm.PasswordStillBound(key, \"h\")"},
	{id: "LoadAdmin binding", name: "adm", valType: "fakeAdmin", value: "fakeAdmin{}", typeName: "fakeAdmin",
		pre: "adm, _ := updateauth.LoadAdmin(d)\n\t_ = adm", use: "adm.PasswordStillBound(key, \"h\")"},
}

// resolveSig is where a signature-site shadow goes; zero fields are absent.
type resolveSig struct{ recv, typeParams, params, results string }

// resolveFile is the one function a matrix cell is: the key loaded first,
// then n.pre, then open + n.use + close.
func resolveFile(n resolveName, sig resolveSig, open, close string) string {
	body := "\tkey, _ := updateauth.LoadDeviceKey(d)\n"
	if n.pre != "" {
		body += "\t" + n.pre + "\n"
	}
	if open != "" {
		body += "\t" + open + "\n"
	}
	body += "\t" + n.use + "\n"
	if close != "" {
		body += "\t" + close + "\n"
	}
	if sig.results != "" {
		body += "\treturn\n"
	}
	return resolvePrelude + "\nfunc " + sig.recv + "m" + sig.typeParams + "(d string" + sig.params + ")" + sig.results + " {\n" + body + "}\n"
}

// resolveForms: one per declaration site go/types has (see the header).
var resolveForms = []struct {
	id   string
	file func(n resolveName) string
}{
	{"short var decl", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "{\n\t"+n.name+" := "+n.value+"\n\t_ = "+n.name, "}")
	}},
	{"if-init short var decl", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "if "+n.name+" := ("+n.value+"); true {\n\t_ = "+n.name, "}")
	}},
	{"switch-init short var decl", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "switch "+n.name+" := ("+n.value+"); {\n\tdefault:\n\t_ = "+n.name, "}")
	}},
	{"local var", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "{\n\tvar "+n.name+" "+n.valType+"\n\t_ = "+n.name, "}")
	}},
	{"local const", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "{\n\tconst "+n.name+" = 1", "}")
	}},
	{"local type alias", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "{\n\ttype "+n.name+" = "+n.typeName, "}")
	}},
	{"local defined type", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "{\n\ttype "+n.name+" "+n.typeName, "}")
	}},
	{"range key", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "for "+n.name+" := range (chan "+n.valType+")(nil) {\n\t_ = "+n.name, "}")
	}},
	{"range value", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "for _, "+n.name+" := range []"+n.valType+"{} {\n\t_ = "+n.name, "}")
	}},
	{"type-switch symbol", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "switch "+n.name+" := any("+n.value+").(type) {\n\tdefault:\n\t_ = "+n.name, "}")
	}},
	{"select receive", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "select {\n\tcase "+n.name+" := <-(chan "+n.valType+")(nil):\n\t_ = "+n.name, "}")
	}},
	{"func literal parameter", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "func("+n.name+" "+n.valType+") {", "}("+n.value+")")
	}},
	{"func literal named result", func(n resolveName) string {
		return resolveFile(n, resolveSig{}, "_ = func() ("+n.name+" "+n.valType+") {", "return\n\t}")
	}},
	{"parameter", func(n resolveName) string {
		return resolveFile(n, resolveSig{params: ", " + n.name + " " + n.valType}, "", "")
	}},
	{"named result", func(n resolveName) string {
		return resolveFile(n, resolveSig{results: " (" + n.name + " " + n.valType + ")"}, "", "")
	}},
	{"function type parameter", func(n resolveName) string {
		return resolveFile(n, resolveSig{typeParams: "[" + n.name + " any]"}, "", "")
	}},
	{"receiver name", func(n resolveName) string {
		return resolveFile(n, resolveSig{recv: "(" + n.name + " " + n.valType + ") "}, "", "")
	}},
	{"receiver type parameter", func(n resolveName) string {
		return resolveFile(n, resolveSig{recv: "(box[" + n.name + "]) "}, "", "")
	}},
	{"receiver type parameter, (*T[K, P])", func(n resolveName) string {
		return resolveFile(n, resolveSig{recv: "(r (*box2[int, " + n.name + "])) "}, "", "")
	}},
}

func TestUpdateAuthKeyFlowAdmissionMatchesTheCompilersResolution(t *testing.T) {
	const rel = "api/handler_updates.go"
	const used = rel + `: the loaded device key "key" is used outside`
	judge := func(t *testing.T, files map[string]string, wantTrusted bool) {
		t.Helper()
		kind, trusted := resolveTrust(t, files)
		if trusted != wantTrusted {
			t.Fatalf("oracle: go/types resolves the %s call's name trusted=%v, want %v — the cell is not what it claims", kind, trusted, wantTrusted)
		}
		off := keyFlowOffendersFor(t, files)
		refused := false
		for _, o := range off {
			refused = refused || strings.HasPrefix(o, used)
		}
		if refused == trusted {
			t.Fatalf("census: go/types resolves the %s call trusted=%v but the census refused=%v:\n%s", kind, trusted, refused, strings.Join(off, "\n"))
		}
	}
	for _, n := range resolveNames {
		n := n
		t.Run(n.id+"/baseline", func(t *testing.T) {
			judge(t, map[string]string{rel: resolveFile(n, resolveSig{}, "", "")}, true)
		})
		for _, form := range resolveForms {
			form := form
			t.Run(n.id+"/"+form.id, func(t *testing.T) {
				judge(t, map[string]string{rel: form.file(n)}, false)
			})
		}
	}
	// the package block, for the one trusted name that lives in the universe
	clear := resolveNames[0]
	for i, decl := range []string{"var clear = sinkFn(nil)", "const clear = 1", "type clear = bytesT", "func clear(b []byte) {}"} {
		decl := decl
		t.Run("builtin clear/package block "+strconv.Itoa(i)+": "+decl, func(t *testing.T) {
			judge(t, map[string]string{
				rel:           resolveFile(clear, resolveSig{}, "", ""),
				"api/util.go": "package api\n\n" + decl + "\n",
			}, false)
		})
	}
}

// The matrix reads the census's own files: the declaration sites it claims
// to cover are named in census_test.go's rule-6 text, so a later edit of that
// text without the matrix (or the reverse) is visible in review.
func TestUpdateAuthResolutionMatrixIsNamedInTheCensus(t *testing.T) {
	src, err := os.ReadFile(filepath.Join(".", "census_test.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "TestUpdateAuthKeyFlowAdmissionMatchesTheCompilersResolution") {
		t.Fatal("census_test.go's rule-6 text must name the resolution matrix that checks its admission claim")
	}
}
