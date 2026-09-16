package kernel

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// ── ONE SETUP E0 — THE MAP STAYS WHOLE ───────────────────────────────────────
//
// D0: nothing the planner is shown changes. This pin has two halves and both
// must hold before E1 exists:
//
//  1. STRUCTURAL — the level, score, seat, merge and planner-render files carry
//     no reference to one-setup at all. The predicate lives at the arm seam;
//     the map path has no input it could read. (Class-113 caveat: a text
//     match, not a call graph — the behavioural half below is the proof.)
//  2. BEHAVIOURAL — the model's level block and the seated table, produced by
//     the production emitter on the same fixture the identity wave pinned,
//     are byte-identical to that pinned golden. The golden was generated
//     before this wave existed, so "ON vs OFF → EMPTY" is the same statement
//     as "this wave's tree == the pinned bytes".
func TestOneSetupMapStaysWhole(t *testing.T) {
	files := []string{"levels_assemble.go", "levels_score.go", "levels_zones.go", "levels_role.go",
		"map_candidates.go", "planner_prompt.go", "plan_render.go", "plan_doc.go", "scenario_level_identity.go"}
	re := regexp.MustCompile(`(?i)one[_ ]?setup`)
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for i, ln := range strings.Split(string(b), "\n") {
			if re.MatchString(ln) {
				t.Fatalf("%s:%d references one-setup — the map, seat race, merge and planner render are NEVER touched by this wave (D0/A31): %s", f, i+1, strings.TrimSpace(ln))
			}
		}
	}

	now := time.Date(2030, 9, 12, 12, 0, 0, 0, CTLocation())
	start := now.Add(-48 * time.Hour)
	bars := make([]market.Kline, 48*60)
	for i := range bars {
		s := start.Add(time.Duration(i) * time.Minute)
		p := 20000 + 60*math.Sin(float64(i)/13) + 20*math.Cos(float64(i)/71)
		bars[i] = market.Kline{OpenTime: s.UnixMilli(), CloseTime: s.Add(time.Minute).UnixMilli() - 1, Open: p - 1, High: p + 3, Low: p - 3, Close: p + 1, Volume: float64(100 + i%31)}
	}
	seated, pool, price, datr, raw := AssembleResearchLevels("identity-parity", bars, DefaultSessionRegistry(), "MNQ", 12, now, 2, "")
	rendered := RenderMapBlock(BuildMapCandidates(seated, price, 20, MapCandidateOpts{}), price)
	data, err := json.MarshalIndent(struct {
		Raw          []DetectedLevel
		Pool, Seated []ScoredLevel
		Price, DATR  float64
		Rendered     string
	}{raw, pool, seated, price, datr, rendered}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	expected, err := os.ReadFile(filepath.Join("testdata", "identity_legacy_output.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(expected) != string(append(data, '\n')) {
		t.Fatal("E0: the model's level block or the seated table differ from the pinned golden — the map was touched")
	}
}
