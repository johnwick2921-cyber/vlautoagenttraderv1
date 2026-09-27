package trader

import (
	"encoding/json"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// W-EXEC-TRUTH W2 A4 — a COMPLIANT scripted model for the real-path tests.
// Those tests drive the production read (real level assembly → the MAP block
// in the prompt → the write site) with a canned plan. Under A4 a canned
// obstacle chain that ignores the map is refused, exactly as a live model's
// would be. mapCompliantPlanJSON does what a compliant model does: it reads
// the MAP block the prompt carries and authors S1's first obstacle and
// path_levels from it. It also proves the prompt carries enough to comply.

var w2MapLine = regexp.MustCompile(`^\s+(?:id=\S+\s+)?(\d+(?:\.\d+)?)\s`)

// promptMapPrices returns the seated prices of the prompt's MAP block (nil
// when the prompt has no MAP block — the canned plan is then returned as is).
func promptMapPrices(user string) []float64 {
	i := strings.Index(user, "MAP (merged references")
	if i < 0 {
		return nil
	}
	var out []float64
	for _, line := range strings.Split(user[i:], "\n")[1:] {
		if strings.TrimSpace(line) == "" {
			break
		}
		if strings.Contains(line, "projection:") {
			continue
		}
		m := w2MapLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if p, err := strconv.ParseFloat(m[1], 64); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// mapCompliantPlanJSON rewrites validTraderPlanJSON's S1 (long, entry 15480,
// stop 15470, target 15620) so its first obstacle is the nearest MAP level on
// the path (or the target when none) and every other one is in path_levels.
func mapCompliantPlanJSON(user string) string {
	prices := promptMapPrices(user)
	if prices == nil {
		return validTraderPlanJSON
	}
	const entry, stop, target, tick = 15480.0, 15470.0, 15620.0, 0.25
	var path []float64
	for _, p := range prices {
		r := math.Round(p/tick) * tick
		if r > entry+tick && r < target-tick {
			path = append(path, r)
		}
	}
	sort.Float64s(path)
	var doc map[string]any
	_ = json.Unmarshal([]byte(validTraderPlanJSON), &doc)
	s1 := doc["scenarios"].([]any)[0].(map[string]any)
	e := s1["economics"].(map[string]any)
	first := target
	if len(path) > 0 {
		first = path[0]
	}
	e["first_obstacle"] = map[string]any{"price": first, "level": "map level", "family": "reference", "response": "pass_through"}
	e["r_to_obstacle"] = (first - entry) / (entry - stop)
	if len(path) > 1 {
		var pl []map[string]any
		for _, p := range path[1:] {
			pl = append(pl, map[string]any{"price": p, "level": "map level", "role": "pass_through"})
		}
		e["path_levels"] = pl
	}
	b, _ := json.Marshal(doc)
	return string(b)
}
