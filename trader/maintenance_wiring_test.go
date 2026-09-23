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
		if !ok || len(call.Args) != 1 {
			return true
		}
		switch sel.Sel.Name {
		case "SetEntryPermit", "SetEntryHoldCheck":
			switch a := call.Args[0].(type) {
			case *ast.Ident:
				found[sel.Sel.Name] = a.Name
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
