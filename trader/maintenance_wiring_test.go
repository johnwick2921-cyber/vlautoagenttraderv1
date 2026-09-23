package trader

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

// W-ONE-BUTTON M2 site 4 — THE WIRING IS THE THING UNDER TEST (canon 53).
// A permit nobody installs is a hold that does not exist. NewAutoTrader is
// the only production AutoTrader constructor (manager/trader_manager.go) and
// cannot be called from a test here (its NT8 branch binds the live port
// 36974), so this pin parses it: inside NewAutoTrader's *TCPTrader block, the
// permit and the queue hold-check are installed from the process-wide gate.
func TestNewAutoTraderWiresTheMaintenancePermitAndQueueCheck(t *testing.T) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "auto_trader.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var fn *ast.FuncDecl
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "NewAutoTrader" && fd.Recv == nil {
			fn = fd
		}
	}
	if fn == nil {
		t.Fatal("NewAutoTrader not found in auto_trader.go")
	}
	found := map[string]string{}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		// M2.1: SetDroppedEntrySink(owner, fn) — the owner must be the trader id.
		if sel.Sel.Name == "SetDroppedEntrySink" && len(call.Args) == 2 {
			if o, ok := call.Args[0].(*ast.SelectorExpr); ok {
				if x, ok := o.X.(*ast.Ident); ok {
					found["SetDroppedEntrySink.owner"] = x.Name + "." + o.Sel.Name
				}
			}
			call = &ast.CallExpr{Fun: call.Fun, Args: call.Args[1:]}
		}
		if len(call.Args) != 1 {
			return true
		}
		switch sel.Sel.Name {
		case "SetEntryPermit", "SetEntryHoldCheck", "SetMaintenanceSource", "SetDroppedEntrySink":
			switch a := call.Args[0].(type) {
			case *ast.Ident:
				found[sel.Sel.Name] = a.Name
			case *ast.SelectorExpr:
				if x, ok := a.X.(*ast.Ident); ok {
					found[sel.Sel.Name] = x.Name + "." + a.Sel.Name
				}
			case *ast.FuncLit:
				found[sel.Sel.Name] = "funclit"
			}
		}
		return true
	})
	if found["SetEntryPermit"] != "MaintenanceEntryPermit" {
		t.Fatalf("NewAutoTrader must call nt.SetEntryPermit(MaintenanceEntryPermit); found %q", found["SetEntryPermit"])
	}
	if found["SetEntryHoldCheck"] != "maintenanceQueueHeld" {
		t.Fatalf("NewAutoTrader must call nt.SetEntryHoldCheck(maintenanceQueueHeld); found %q", found["SetEntryHoldCheck"])
	}
	// Site 7: the wire learns the hold from the same process-wide gate.
	if found["SetMaintenanceSource"] != "maintenanceWireState" {
		t.Fatalf("NewAutoTrader must call nt.SetMaintenanceSource(maintenanceWireState); found %q", found["SetMaintenanceSource"])
	}
	// M-2: this trader settles its own entries the hold dropped from the queue.
	if found["SetDroppedEntrySink"] != "at.onMaintenanceDroppedEntry" {
		t.Fatalf("NewAutoTrader must call nt.SetDroppedEntrySink(at.id, at.onMaintenanceDroppedEntry); found %q", found["SetDroppedEntrySink"])
	}
	if found["SetDroppedEntrySink.owner"] != "at.id" {
		t.Fatalf("the drop sink must be keyed by the trader id (reload replaces, never leaks); found owner %q", found["SetDroppedEntrySink.owner"])
	}
}

// maintenanceWireState is what the AddOn is told: absent → not held; held →
// the job id; a corrupt file → held with no job (fail-closed).
func TestMaintenanceWireStateFollowsTheHoldFile(t *testing.T) {
	dir := withMaintenanceDir(t)
	if held, job := maintenanceWireState(); held || job != "" {
		t.Fatalf("absent → (false, \"\"), got (%v, %q)", held, job)
	}
	setHold(t, dir, "job-wire")
	if held, job := maintenanceWireState(); !held || job != "job-wire" {
		t.Fatalf("held → (true, job-wire), got (%v, %q)", held, job)
	}
	if err := writeRaw(dir, "{not json"); err != nil {
		t.Fatal(err)
	}
	if held, job := maintenanceWireState(); !held || job != "" {
		t.Fatalf("corrupt → (true, \"\"), got (%v, %q)", held, job)
	}
}

// maintenanceQueueHeld is exactly MaintenanceHeld's boolean.
func TestMaintenanceQueueHeldFollowsTheHoldFile(t *testing.T) {
	dir := withMaintenanceDir(t)
	if maintenanceQueueHeld() {
		t.Fatal("absent → not held")
	}
	setHold(t, dir, "job-q")
	if !maintenanceQueueHeld() {
		t.Fatal("held file → queue held")
	}
}
