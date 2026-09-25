package trader

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W5 (builder C2) — PICTURE'S OWN SEND PATH IS RETIRED ───────
//
// Picture used to own a MARKET entry: the evaluator's seam fired a send
// through its own broker method, a trade that could happen behind a card
// reading "No plan" (D27). Since W5 the seam is the Day Plan hand-off and the
// ONLY door from a Picture opportunity to the wire is the shared armed
// executor, which places it as a market_in_zone LIMIT. These pins keep the
// retired path from coming back by any route:
//
//   - the seam is bound to the hand-off (no other production binding);
//   - the broker adapter has no market-entry method left to call;
//   - no Picture production file calls ANY broker entry method — its orders
//     exist only as Day Plan scenarios the executor places;
//   - the retired file is gone.
//
// The wire half — every Picture frame is a 1-lot LIMIT — is pinned at the
// executor's call site by TestPicturePathWritesOnlyLimitFrames.

// pictureMarketEntryName matches any broker method that would send a market
// entry (GetMarketPrice is a read, not an entry).
var pictureMarketEntryName = regexp.MustCompile(`(?i)market.*entry|entry.*market`)

// pictureRetiredIdent matches the retired send's own names in a Picture
// production file. (kernel.EntryPolicyMarketInZone is the LIMIT policy the
// executor places a Picture scenario with — not a market entry — so the ident
// rule is narrower than the method rule.)
var pictureRetiredIdent = regexp.MustCompile(`(?i)^(picturehtf(send|contractsize)|marketentry.*)$`)

// brokerEntryMethods is every broker call that puts an ENTRY on the wire.
var brokerEntryMethods = map[string]bool{
	"OpenLong": true, "OpenShort": true, "PlaceLimitEntry": true, "PlaceStopEntry": true,
	"SendSignal": true, "DebugPlaceTestTrade": true, "placeEntry": true,
}

func TestPictureHasNoMarketEntryPath(t *testing.T) {
	// (1) The seam is the hand-off.
	if reflect.ValueOf(pictureHtfSubmitSeam).Pointer() != reflect.ValueOf(pictureHtfHandOffSeam).Pointer() {
		t.Fatal("the production pictureHtfSubmitSeam must be the Day Plan hand-off — a Picture opportunity reaches the wire only as a plan scenario")
	}
	// (2) The broker adapter carries no market-entry method.
	typ := reflect.TypeOf(&ntTrader.TCPTrader{})
	for i := 0; i < typ.NumMethod(); i++ {
		if name := typ.Method(i).Name; pictureMarketEntryName.MatchString(name) {
			t.Fatalf("TCPTrader.%s is a market-entry method — Picture's retired send path must not come back", name)
		}
	}
	// (3) No Picture production file reaches a broker entry method.
	files, err := filepath.Glob("picture_*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("fixture: no Picture production files found (%v)", err)
	}
	scanned := 0
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		scanned++
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, f, nil, 0)
		if perr != nil {
			t.Fatalf("%s: %v", f, perr)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.SelectorExpr:
				if brokerEntryMethods[x.Sel.Name] || pictureRetiredIdent.MatchString(x.Sel.Name) {
					t.Errorf("%s: %s reaches the broker entry method %s — Picture's orders exist only as Day Plan scenarios the armed executor places",
						f, fset.Position(x.Pos()), x.Sel.Name)
				}
			case *ast.Ident:
				if pictureRetiredIdent.MatchString(x.Name) {
					t.Errorf("%s: %s names %s — the market-entry send is retired", f, fset.Position(x.Pos()), x.Name)
				}
			}
			return true
		})
	}
	if scanned < 5 {
		t.Fatalf("fixture: only %d Picture production files scanned — the glob no longer finds the path", scanned)
	}
	// (4) The retired send file is gone.
	if _, err := os.Stat("picture_htf_send.go"); !os.IsNotExist(err) {
		t.Fatalf("picture_htf_send.go must stay deleted (stat err %v)", err)
	}
}

// The 📷 boot line (moved here from the retired send's test file; the line
// itself is 103's, picture_htf_live.go).
func TestPictureHtfBootLineNamesEverything(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PictureHtf: &store.PictureHtfConfig{Enabled: true}}})
	line := at.pictureHtfBootLine()
	for _, want := range []string{"picture-htf:", "mode=on", "rule=v1", "SIM-only", "final+emitted_at", "addon=not proven", ntwire.MinAddonBuildPictureHtf} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line must name %q, got %q", want, line)
		}
	}
	off, _ := resetTrader(t, store.StrategyConfig{})
	if l := off.pictureHtfBootLine(); !strings.Contains(l, "mode=off") {
		t.Fatalf("disabled mode must read off: %q", l)
	}
}

// pictureOnGrid5M is pictureBars5M(true) on the MNQ tick grid: the same climb
// and the same strict 98.50 swing, every close a tick price. A hand-off judges
// its zone after inward rounding to the tick, so the off-grid closes of the
// geometry fixtures (…+0.03) make a one-price zone EMPTY and the hand-off
// refuses; a scenario the executor can place needs grid prices. Newest close
// 101.50 → entry 101.50, stop 98.25, target 110 → 2.62R at the far edge.
func pictureOnGrid5M() []market.Kline {
	base := t4h0 + 39*3600*1000 + 40*60*1000 // 16:40Z
	out := make([]market.Kline, 0, 28)
	for i := 0; i < 28; i++ {
		c := math.Round((99.3+float64(i)*0.08)*4) / 4
		lo := c - 0.5
		if i == 22 {
			lo = 98.5
		}
		out = append(out, mkBar(base+int64(i)*5*60*1000, 5*60*1000, c, c+0.5, lo, c))
	}
	return out
}
