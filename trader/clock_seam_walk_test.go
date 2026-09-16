package trader

// CLEANUP BATCH 2, B2 — THE SEAM LINT WALKS CALLEES (class 113 closed).
//
// clock_seam_lint_test.go certifies NAMES: for each row of clock-seams.list the
// …At variant exists and the wall-clock entry is a one-line delegate. That said
// nothing about the chain beneath: `maybeManageArmedOrders:maybeManageArmedOrdersAt`
// was registered and green while `runArmedPlacement`, called one line below the
// clock it was handed, read time.Now() itself. Green meant "this entry point
// delegates", not "this path is seamed".
//
// This file is the receiver-aware go/ast pass the lint's own OWED note asked
// for. ONE graph — the package's FuncDecls keyed by (receiver type, name) — and
// three queries against it:
//
//	(a) beneath every …At variant, transitively through callees resolved on the
//	    SAME receiver type and package-level functions, no time.Now();
//	(b) no test file calls a registered wall-clock entry on that receiver type;
//	(c) every registered wall-clock entry has ≥1 production caller (A29) —
//	    a seam row with no caller is a dead wrapper the list keeps trusting.
//
// RECEIVER RESOLUTION, STATED (this is what the textual version could not do):
//	- `recv.m(…)` where recv is the enclosing method's receiver → (R, m).
//	- `f(…)` bare → package function f.
//	- `x.m(…)` where m names a method on EXACTLY ONE receiver type in the
//	  package → that method. `Save` and `observe` are NOT unique in their
//	  packages (gorm's db.Save, other observers) and so are resolved only when
//	  x's declaration in the enclosing function names the receiver type
//	  (`x := &R{…}`, `x := R{…}`, `var x *R`, `x := newR(…)` / `NewR(…)`).
//	- anything else is a BOUNDARY: counted and reported, never silently passed
//	  and never called an error. The count is on the record so the walk's
//	  reach is a number, not a claim.
//
// No go/types: the module does not require x/tools and type-checking the
// trader package from source on every run costs more than the check earns.
// The boundary count is the honest price of that choice.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type seamPkg struct {
	fset    *token.FileSet
	files   map[string]*ast.File                // path → file (production only)
	tests   map[string]*ast.File                // path → file (tests only)
	methods map[string]map[string]*ast.FuncDecl // recvType → name → decl ("" = package funcs)
	byName  map[string][]string                 // method name → receiver types that declare it
}

func loadSeamPkg(t *testing.T, dir string) *seamPkg {
	t.Helper()
	p := &seamPkg{fset: token.NewFileSet(), files: map[string]*ast.File{}, tests: map[string]*ast.File{},
		methods: map[string]map[string]*ast.FuncDecl{}, byName: map[string][]string{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		af, err := parser.ParseFile(p.fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		if strings.HasSuffix(e.Name(), "_test.go") {
			p.tests[path] = af
			continue
		}
		p.files[path] = af
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			r := recvTypeOf(fd)
			if p.methods[r] == nil {
				p.methods[r] = map[string]*ast.FuncDecl{}
			}
			p.methods[r][fd.Name.Name] = fd
			if r != "" {
				p.byName[fd.Name.Name] = append(p.byName[fd.Name.Name], r)
			}
		}
	}
	return p
}

func recvTypeOf(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return ""
	}
	switch x := fd.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		if id, ok := x.X.(*ast.Ident); ok {
			return id.Name
		}
	case *ast.Ident:
		return x.Name
	}
	return ""
}

func recvNameOf(fd *ast.FuncDecl) string {
	if fd.Recv == nil || len(fd.Recv.List) == 0 || len(fd.Recv.List[0].Names) == 0 {
		return ""
	}
	return fd.Recv.List[0].Names[0].Name
}

// declaredTypeOf resolves a local identifier's static type NAME inside fn by
// its declaration: `x := &R{`, `x := R{`, `var x *R`, `var x R`, `x := newR(`,
// `x := NewR(`, or a func parameter `x *R` / `x R`. "" when unresolved.
func declaredTypeOf(fn *ast.FuncDecl, ident string) string {
	if fn == nil {
		return ""
	}
	if fn.Recv != nil && recvNameOf(fn) == ident {
		return recvTypeOf(fn)
	}
	typeName := func(e ast.Expr) string {
		switch x := e.(type) {
		case *ast.StarExpr:
			if id, ok := x.X.(*ast.Ident); ok {
				return id.Name
			}
		case *ast.Ident:
			return x.Name
		}
		return ""
	}
	for _, f := range fn.Type.Params.List {
		for _, n := range f.Names {
			if n.Name == ident {
				return typeName(f.Type)
			}
		}
	}
	found := ""
	if fn.Body == nil {
		return ""
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if found != "" {
			return false
		}
		switch s := n.(type) {
		case *ast.AssignStmt:
			for i, l := range s.Lhs {
				id, ok := l.(*ast.Ident)
				if !ok || id.Name != ident || i >= len(s.Rhs) {
					continue
				}
				switch r := s.Rhs[i].(type) {
				case *ast.UnaryExpr: // &R{…}
					if cl, ok := r.X.(*ast.CompositeLit); ok {
						found = typeName(cl.Type)
					}
				case *ast.CompositeLit: // R{…}
					found = typeName(r.Type)
				case *ast.CallExpr: // newR(…) / NewR(…)
					if fid, ok := r.Fun.(*ast.Ident); ok {
						nm := strings.TrimPrefix(strings.TrimPrefix(fid.Name, "new"), "New")
						if nm != "" && nm != fid.Name {
							found = nm
						}
					}
				}
			}
		case *ast.ValueSpec:
			for _, n := range s.Names {
				if n.Name == ident && s.Type != nil {
					found = typeName(s.Type)
				}
			}
		}
		return true
	})
	return found
}

// resolveCallee names the FuncDecl a call reaches, or "" plus a boundary reason.
func (p *seamPkg) resolveCallee(enclosing *ast.FuncDecl, call *ast.CallExpr) (recv, name, boundary string) {
	switch f := call.Fun.(type) {
	case *ast.Ident:
		if _, ok := p.methods[""][f.Name]; ok {
			return "", f.Name, ""
		}
		return "", "", "" // builtin / imported / closure — not a package path
	case *ast.SelectorExpr:
		x, ok := f.X.(*ast.Ident)
		if !ok {
			return "", "", "receiver is an expression: " + f.Sel.Name
		}
		if x.Name == "time" && f.Sel.Name == "Now" {
			return "time", "Now", ""
		}
		if tn := declaredTypeOf(enclosing, x.Name); tn != "" {
			if _, ok := p.methods[tn][f.Sel.Name]; ok {
				return tn, f.Sel.Name, ""
			}
			return "", "", "" // a field or a method on a type outside this package
		}
		if owners := p.byName[f.Sel.Name]; len(owners) == 1 {
			return owners[0], f.Sel.Name, ""
		} else if len(owners) > 1 {
			return "", "", "ambiguous method name " + f.Sel.Name + " on " + strings.Join(owners, "/") + " via unresolved receiver " + x.Name
		}
	}
	return "", "", ""
}

type walkResult struct {
	wallClock  []string // file:line of time.Now() beneath the entry that feeds a VERDICT
	stamps     int      // time.Now() reads that only stamp a record/log — not a rule
	boundaries []string
	visited    int
}

// wallClockVerdictReads returns the positions of time.Now()/time.Since() reads
// in fd whose value feeds a VERDICT, and counts the rest as stamps.
//
// A STAMP is a wall-clock read that only records when something happened —
// a struct-literal field, a field assignment, an argument to a log/record
// call. A VERDICT read is one whose value is compared, subtracted, or used
// to pick an hour/weekday: it changes what the rule DECIDES with the hour of
// the day, which is the whole of class 60. The dataflow is local to the
// function: a read assigned to `v` counts if `v` (or v.UnixMilli() etc.)
// later appears in a comparison, in .Before/.After/.Sub/.Equal, in a
// time.Since, or is handed to a callee on the walk graph as its clock — that
// last one is exactly runArmedPlacement's shape: the caller was handed a
// clock and passed the WALL clock down instead.
func (p *seamPkg) wallClockVerdictReads(fd *ast.FuncDecl) (verdicts []token.Pos, stamps int) {
	if fd.Body == nil {
		return nil, 0
	}
	isNow := func(e ast.Expr) bool {
		call, ok := e.(*ast.CallExpr)
		if !ok {
			return false
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return false
		}
		x, ok := sel.X.(*ast.Ident)
		return ok && x.Name == "time" && (sel.Sel.Name == "Now" || sel.Sel.Name == "Since")
	}
	// unwrap time.Now().UnixMilli() / .Unix() / .In(...) etc. to the Now call
	unwrapNow := func(e ast.Expr) ast.Expr {
		for {
			call, ok := e.(*ast.CallExpr)
			if !ok {
				return nil
			}
			if isNow(call) {
				return call
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return nil
			}
			e = sel.X
		}
	}
	verdictMethod := map[string]bool{"Before": true, "After": true, "Sub": true, "Equal": true, "Hour": true, "Minute": true, "Weekday": true, "Day": true, "YearDay": true, "Truncate": true, "Compare": true}
	compare := map[token.Token]bool{token.LSS: true, token.GTR: true, token.LEQ: true, token.GEQ: true, token.EQL: true, token.NEQ: true}
	// pass 1: identifiers assigned from a wall-clock read, and direct verdict uses
	derived := map[string]token.Pos{}
	verdictPos := map[token.Pos]bool{}
	allNow := map[token.Pos]bool{}
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if isNow(x) {
				allNow[x.Pos()] = true
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Since" {
					verdictPos[x.Pos()] = true // time.Since is a duration for comparing
				}
			}
			// x.Method(...) where x is a wall-clock read → verdict method?
			if sel, ok := x.Fun.(*ast.SelectorExpr); ok && verdictMethod[sel.Sel.Name] {
				if nw := unwrapNow(sel.X); nw != nil {
					verdictPos[nw.Pos()] = true
				}
			}
		case *ast.AssignStmt:
			for i, l := range x.Lhs {
				id, ok := l.(*ast.Ident)
				if !ok || i >= len(x.Rhs) {
					continue
				}
				if nw := unwrapNow(x.Rhs[i]); nw != nil {
					derived[id.Name] = nw.Pos()
				}
			}
		case *ast.BinaryExpr:
			if compare[x.Op] || x.Op == token.SUB {
				for _, side := range []ast.Expr{x.X, x.Y} {
					if nw := unwrapNow(side); nw != nil {
						verdictPos[nw.Pos()] = true
					}
				}
			}
		}
		return true
	})
	// pass 2: derived identifiers used as verdicts or handed to a graph callee
	if len(derived) > 0 {
		usesIdent := func(e ast.Expr) string {
			for {
				switch x := e.(type) {
				case *ast.Ident:
					if _, ok := derived[x.Name]; ok {
						return x.Name
					}
					return ""
				case *ast.CallExpr:
					sel, ok := x.Fun.(*ast.SelectorExpr)
					if !ok {
						return ""
					}
					e = sel.X
				case *ast.SelectorExpr:
					e = x.X
				default:
					return ""
				}
			}
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BinaryExpr:
				if compare[x.Op] || x.Op == token.SUB {
					for _, side := range []ast.Expr{x.X, x.Y} {
						if v := usesIdent(side); v != "" {
							verdictPos[derived[v]] = true
						}
					}
				}
			case *ast.CallExpr:
				if sel, ok := x.Fun.(*ast.SelectorExpr); ok {
					if verdictMethod[sel.Sel.Name] {
						if v := usesIdent(sel.X); v != "" {
							verdictPos[derived[v]] = true
						}
						for _, a := range x.Args {
							if v := usesIdent(a); v != "" && (sel.Sel.Name == "Before" || sel.Sel.Name == "After" || sel.Sel.Name == "Sub" || sel.Sel.Name == "Equal") {
								verdictPos[derived[v]] = true
							}
						}
					}
					if xid, ok := sel.X.(*ast.Ident); ok && xid.Name == "time" && sel.Sel.Name == "Since" {
						for _, a := range x.Args {
							if v := usesIdent(a); v != "" {
								verdictPos[derived[v]] = true
							}
						}
					}
				}
				// handed down as a clock to a callee on the graph
				if r, nm, _ := p.resolveCallee(fd, x); nm != "" && r != "time" {
					for _, a := range x.Args {
						if id, ok := a.(*ast.Ident); ok {
							if pos, ok := derived[id.Name]; ok {
								verdictPos[pos] = true
							}
						}
					}
				}
			}
			return true
		})
	}
	// ELAPSED TIMERS ARE NOT VERDICTS: `start := time.Now(); … time.Since(start)`
	// measures a real wait against itself (the ForceReset exemption in
	// clock-seams.list, generalised). Both ends are the wall clock; feeding it a
	// fixture clock would measure nothing. Drop the Since and the start it reads.
	ast.Inspect(fd.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		x, ok := sel.X.(*ast.Ident)
		if !ok || x.Name != "time" || sel.Sel.Name != "Since" || len(call.Args) != 1 {
			return true
		}
		if id, ok := call.Args[0].(*ast.Ident); ok {
			if startPos, ok := derived[id.Name]; ok {
				delete(verdictPos, call.Pos())
				delete(verdictPos, startPos)
			}
		}
		return true
	})
	for pos := range allNow {
		if verdictPos[pos] {
			verdicts = append(verdicts, pos)
		} else {
			stamps++
		}
	}
	sort.Slice(verdicts, func(i, j int) bool { return verdicts[i] < verdicts[j] })
	return verdicts, stamps
}

// walkBeneath BFS-walks callees of (recv, name), collecting time.Now() sites.
func (p *seamPkg) walkBeneath(recv, name string) walkResult {
	var res walkResult
	seen := map[string]bool{}
	queue := [][2]string{{recv, name}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		key := cur[0] + "." + cur[1]
		if seen[key] {
			continue
		}
		seen[key] = true
		fd := p.methods[cur[0]][cur[1]]
		if fd == nil || fd.Body == nil {
			continue
		}
		res.visited++
		verdicts, stamps := p.wallClockVerdictReads(fd)
		res.stamps += stamps
		for _, v := range verdicts {
			pos := p.fset.Position(v)
			res.wallClock = append(res.wallClock, fmt.Sprintf("%s:%d in %s", filepath.Base(pos.Filename), pos.Line, fd.Name.Name))
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			r, nm, b := p.resolveCallee(fd, call)
			if b != "" {
				res.boundaries = append(res.boundaries, fmt.Sprintf("%s (in %s)", b, fd.Name.Name))
				return true
			}
			if r == "time" {
				return true
			}
			if nm != "" {
				queue = append(queue, [2]string{r, nm})
			}
			return true
		})
	}
	sort.Strings(res.wallClock)
	return res
}

// callersOf counts production call sites of (recv, name) across the module,
// receiver-aware where the receiver resolves, name-unique otherwise.
func callersOf(t *testing.T, root string, recv, name string) (sites []string) {
	t.Helper()
	fset := token.NewFileSet()
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			b := info.Name()
			if b == "node_modules" || b == ".git" || b == "web" || (strings.HasPrefix(b, ".") && b != "." && b != "..") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		af, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			return nil
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			if fd.Name.Name == name && recvTypeOf(fd) == recv {
				continue // the declaration itself
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch f := call.Fun.(type) {
				case *ast.Ident:
					if recv == "" && f.Name == name {
						rel, _ := filepath.Rel(root, path)
						sites = append(sites, fmt.Sprintf("%s:%d", rel, fset.Position(call.Pos()).Line))
					}
				case *ast.SelectorExpr:
					if f.Sel.Name != name {
						return true
					}
					// GENEROUS BY DESIGN: this query answers "is there NO caller",
					// so only a receiver that resolves to a DIFFERENT type excludes a
					// site. A package-qualified call (pkg.Func), a field receiver
					// (r.sink.Save) and an unresolved identifier all count.
					x, ok := f.X.(*ast.Ident)
					if !ok {
						rel, _ := filepath.Rel(root, path)
						sites = append(sites, fmt.Sprintf("%s:%d", rel, fset.Position(call.Pos()).Line))
						return true
					}
					tn := declaredTypeOf(fd, x.Name)
					if tn == "" || strings.EqualFold(tn, recv) {
						rel, _ := filepath.Rel(root, path)
						sites = append(sites, fmt.Sprintf("%s:%d", rel, fset.Position(call.Pos()).Line))
					}
				}
				return true
			})
		}
		return nil
	})
	sort.Strings(sites)
	return sites
}

// testCallsOf finds calls of (recv, name) in the package's test files whose
// receiver RESOLVES to recv, plus the unresolved ones as boundaries.
func (p *seamPkg) testCallsOf(recv, name string) (resolved, unresolved []string) {
	for path, af := range p.tests {
		if filepath.Base(path) == "clock_seam_walk_test.go" || filepath.Base(path) == "clock_seam_lint_test.go" {
			continue // these files name the entry points as data
		}
		for _, d := range af.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != name {
					return true
				}
				x, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				loc := fmt.Sprintf("%s:%d", filepath.Base(path), p.fset.Position(call.Pos()).Line)
				tn := declaredTypeOf(fd, x.Name)
				switch {
				case tn == recv:
					resolved = append(resolved, loc)
				case tn == "" && len(p.byName[name]) == 1 && p.byName[name][0] == recv:
					resolved = append(resolved, loc) // unique name in the package: it can only be ours
				case tn == "":
					unresolved = append(unresolved, loc+" ("+x.Name+"."+name+")")
				}
				return true
			})
		}
	}
	sort.Strings(resolved)
	sort.Strings(unresolved)
	return
}

// ── THE OWED LIST: the walk is a gate from day one ────────────────────────────

type seamOwed struct {
	verdicts     map[string]string // "file:func" → OWED|EXEMPT
	testcalls    map[string]int    // "testfile:entry" → count
	deadwrappers map[string]bool   // "file:Recv.entry"
}

func loadSeamOwed(t *testing.T) seamOwed {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "clock-seams-owed.list"))
	if err != nil {
		t.Fatalf("read clock-seams-owed.list: %v", err)
	}
	o := seamOwed{verdicts: map[string]string{}, testcalls: map[string]int{}, deadwrappers: map[string]bool{}}
	for _, ln := range strings.Split(string(raw), "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		p := strings.SplitN(ln, ":", 5)
		switch p[0] {
		case "verdict":
			if len(p) < 5 || (p[3] != "OWED" && p[3] != "EXEMPT") || strings.TrimSpace(p[4]) == "" {
				t.Fatalf("malformed verdict row %q — want verdict:<file>:<func>:<OWED|EXEMPT>:<reason>", ln)
			}
			o.verdicts[p[1]+":"+p[2]] = p[3]
		case "testcall":
			if len(p) != 4 {
				t.Fatalf("malformed testcall row %q", ln)
			}
			n := 0
			fmt.Sscanf(p[3], "%d", &n)
			o.testcalls[p[1]+":"+p[2]] = n
		case "deadwrapper":
			if len(p) != 3 {
				t.Fatalf("malformed deadwrapper row %q", ln)
			}
			o.deadwrappers[p[1]+":"+p[2]] = true
		default:
			t.Fatalf("unknown row kind %q", ln)
		}
	}
	return o
}

// ── THE THREE QUERIES, against the real list ──────────────────────────────────

func TestSeamWalk_NoWallClockBeneathAnySeamedEntry(t *testing.T) {
	rules := loadSeamRules(t)
	pkgs := map[string]*seamPkg{}
	boundaries, stampsTotal := 0, 0
	owed := loadSeamOwed(t)
	seenVerdict := map[string]bool{}
	for _, r := range rules {
		dir := filepath.Join("..", filepath.Dir(r.file))
		p := pkgs[dir]
		if p == nil {
			p = loadSeamPkg(t, dir)
			pkgs[dir] = p
		}
		// find the …At variant's receiver
		recv := ""
		for rt, m := range p.methods {
			if _, ok := m[r.at]; ok {
				recv = rt
				break
			}
		}
		if _, ok := p.methods[recv][r.at]; !ok {
			t.Errorf("%s: %s not found in %s", r.file, r.at, dir)
			continue
		}
		res := p.walkBeneath(recv, r.at)
		boundaries += len(res.boundaries)
		stampsTotal += res.stamps
		for _, w := range res.wallClock {
			// "file.go:line in func" → the owed key is "<dir>/<file>:<func>"
			parts := strings.SplitN(w, " in ", 2)
			fileLine, fn := parts[0], parts[1]
			file := filepath.Join(filepath.Dir(r.file), strings.SplitN(fileLine, ":", 2)[0])
			key := file + ":" + fn
			seenVerdict[key] = true
			if _, listed := owed.verdicts[key]; listed {
				continue
			}
			t.Errorf("%s: %s reaches the WALL CLOCK beneath its seam — time.Now() at %s. The …At variant was handed a clock; this callee ignored it (class 113). Fix it, or add `verdict:%s:OWED|EXEMPT:<reason>` to clock-seams-owed.list.", r.file, r.at, w, key)
		}
	}
	for key, kind := range owed.verdicts {
		if !seenVerdict[key] {
			t.Errorf("clock-seams-owed.list row `verdict:%s:%s` no longer reproduces — the walk found no wall-clock verdict there. Delete the row (a fixed debt is not kept as debt).", key, kind)
		}
	}
	t.Logf("walked %d rules; %d receiver boundaries left unresolved (counted, not errors); %d stamp-only wall-clock reads beneath seams (not rules); %d verdict reads, all on the owed list", len(rules), boundaries, stampsTotal, len(seenVerdict))
}

func TestSeamWalk_NoTestCallsAWallClockEntry(t *testing.T) {
	rules := loadSeamRules(t)
	pkgs := map[string]*seamPkg{}
	total, unres := 0, 0
	owed := loadSeamOwed(t)
	seenCalls := map[string]int{}
	for _, r := range rules {
		dir := filepath.Join("..", filepath.Dir(r.file))
		p := pkgs[dir]
		if p == nil {
			p = loadSeamPkg(t, dir)
			pkgs[dir] = p
		}
		recv := ""
		for rt, m := range p.methods {
			if _, ok := m[r.entry]; ok {
				recv = rt
				break
			}
		}
		resolved, unresolved := p.testCallsOf(recv, r.entry)
		total += len(resolved)
		unres += len(unresolved)
		perFile := map[string]int{}
		for _, loc := range resolved {
			perFile[strings.SplitN(loc, ":", 2)[0]]++
		}
		for file, n := range perFile {
			key := file + ":" + r.entry
			seenCalls[key] = n
			if n <= owed.testcalls[key] {
				continue
			}
			t.Errorf("%s calls the wall-clock entry %s.%s() from a test %d time(s) (owed list allows %d) — use %s(…, now) with a clock the test controls (class 60/113).", file, recv, r.entry, n, owed.testcalls[key], r.at)
		}
	}
	for key, n := range owed.testcalls {
		if seenCalls[key] < n {
			t.Errorf("clock-seams-owed.list row `testcall:%s:%d` is stale — the walk now finds %d. Lower the count or delete the row (the ratchet only tightens).", key, n, seenCalls[key])
		}
	}
	t.Logf("test calls of wall-clock entries: %d resolved (all within the owed ratchet), %d unresolved receivers (boundaries)", total, unres)
}

func TestSeamWalk_EveryWallClockEntryHasAProductionCaller(t *testing.T) {
	rules := loadSeamRules(t)
	pkgs := map[string]*seamPkg{}
	owed := loadSeamOwed(t)
	seenDead := map[string]bool{}
	for _, r := range rules {
		dir := filepath.Join("..", filepath.Dir(r.file))
		p := pkgs[dir]
		if p == nil {
			p = loadSeamPkg(t, dir)
			pkgs[dir] = p
		}
		recv := ""
		for rt, m := range p.methods {
			if _, ok := m[r.entry]; ok {
				recv = rt
				break
			}
		}
		sites := callersOf(t, "..", recv, r.entry)
		key := r.file + ":" + recv + "." + r.entry
		if len(sites) == 0 {
			seenDead[key] = true
			if owed.deadwrappers[key] {
				continue
			}
			t.Errorf("%s: %s.%s has NO production caller — a dead wrapper on the seam list (A29). Either wire it, remove the row with its reason, or list it as `deadwrapper:%s`.", r.file, recv, r.entry, key)
		}
	}
	for key := range owed.deadwrappers {
		if !seenDead[key] {
			t.Errorf("clock-seams-owed.list row `deadwrapper:%s` is stale — it has a production caller now. Delete the row.", key)
		}
	}
}

// ── THE FIXTURE: RED when a callee reads the wall clock, GREEN when seamed ────

const seamFixtureBad = `package fx
import "time"
type Box struct{ n int }
func (b *Box) rule() bool { return b.ruleAt(time.Now()) }
func (b *Box) ruleAt(now time.Time) bool { return b.helper(now) }
func (b *Box) helper(now time.Time) bool { return b.deeper() }
func (b *Box) deeper() bool { return time.Now().Hour() > 3 }
func other(b *Box) { _ = b.rule() }
`

const seamFixtureGood = `package fx
import "time"
type Box struct{ n int }
func (b *Box) rule() bool { return b.ruleAt(time.Now()) }
func (b *Box) ruleAt(now time.Time) bool { return b.helper(now) }
func (b *Box) helper(now time.Time) bool { return b.deeperAt(now) }
func (b *Box) deeperAt(now time.Time) bool { b.n = int(time.Now().Unix()); return now.Hour() > 3 } // a STAMP, not a verdict
func other(b *Box) { _ = b.rule() }
`

func TestSeamWalk_FixtureCalleeReadingTheWallClockIsRed(t *testing.T) {
	for _, c := range []struct {
		name string
		src  string
		want int
	}{{"bad", seamFixtureBad, 1}, {"good", seamFixtureGood, 0}} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "fx.go"), []byte(c.src), 0o644); err != nil {
			t.Fatal(err)
		}
		p := loadSeamPkg(t, dir)
		res := p.walkBeneath("Box", "ruleAt")
		if len(res.wallClock) != c.want {
			t.Fatalf("%s fixture: %d wall-clock reads beneath ruleAt, want %d: %v", c.name, len(res.wallClock), c.want, res.wallClock)
		}
		if res.visited < 3 {
			t.Fatalf("%s fixture: the walk visited %d functions — it did not descend", c.name, res.visited)
		}
	}
}

// The receiver resolution is what separates at.Save( from db.Save( — pinned
// on a fixture where the same method name exists on two types.
func TestSeamWalk_ReceiverResolutionSeparatesSameNamedMethods(t *testing.T) {
	dir := t.TempDir()
	src := `package fx
import "time"
type Archive struct{}
func (a *Archive) Save() { a.saveAt(time.Now()) }
func (a *Archive) saveAt(now time.Time) {}
type DB struct{}
func (d *DB) Save() {}
`
	test := `package fx
import "testing"
func TestX(t *testing.T) {
	a := &Archive{}
	a.Save()          // ours: resolved
	d := &DB{}
	d.Save()          // gorm-shaped: NOT ours
	var u interface{ Save() }
	_ = u
	x := mystery()
	x.Save()          // unresolved receiver: a boundary, not an error
}
func mystery() interface{ Save() } { return nil }
`
	if err := os.WriteFile(filepath.Join(dir, "fx.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fx_test.go"), []byte(test), 0o644); err != nil {
		t.Fatal(err)
	}
	p := loadSeamPkg(t, dir)
	resolved, unresolved := p.testCallsOf("Archive", "Save")
	if len(resolved) != 1 || !strings.HasPrefix(resolved[0], "fx_test.go:5") {
		t.Fatalf("resolved = %v, want exactly the a.Save() at line 5", resolved)
	}
	if len(unresolved) != 1 || !strings.Contains(unresolved[0], "x.Save") {
		t.Fatalf("unresolved = %v, want exactly x.Save()", unresolved)
	}
}
