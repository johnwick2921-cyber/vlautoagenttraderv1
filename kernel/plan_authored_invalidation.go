package kernel

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"nofx/market"
)

// Deliberately a small complete grammar, not a price mined from prose. Compound,
// sequential ("back below", "after"), MSS and subjective rules stay UNKNOWN.
var authoredCloseRule = regexp.MustCompile(`(?i)^(?:(?:a|one) )?(?:(1|2)x)?5m(?:_close| closes?)?\s*(above|below|>|<)\s*(\d+(?:\.\d+)?)(?:\s+\((?:SWG-[HL]·(?:1|5|15|30)m|OR-[HL]|ON[HL]|PD[HL]|VWAP[+−-][12]σ|POC)\))?(?:\s+(?:invalidates|kills|cancels|voids|negates|aborts) (?:the |this )?(?:setup|short|long|rejection|fade|hold|hold thesis|breakout|reclaim))?\.?$`)

type AuthoredInvalidationVerdict struct {
	ScenarioID  string
	Known       bool
	Invalidated bool
	Anchor      float64
	Price       float64
	At          time.Time
	Reason      string
}

// EvaluateAuthoredInvalidationAt tests only the most recently completed rule
// window at authoring. Every constituent minute is required; incomplete,
// missing, duplicate or non-finite tape never becomes a refusal. This does NOT
// alter EvaluateScenario, confirmation timing, or any EntryGate verdict.
func EvaluateAuthoredInvalidationAt(sc PlanScenario, bars []market.Kline, now time.Time) AuthoredInvalidationVerdict {
	out := AuthoredInvalidationVerdict{ScenarioID: sc.ID, Reason: "UNKNOWN: authored invalidation is outside the supported explicit 1/2 × 5m close grammar"}
	m := authoredCloseRule.FindStringSubmatch(strings.TrimSpace(sc.Invalid))
	if m == nil {
		return out
	}
	ref, err := strconv.ParseFloat(m[3], 64)
	if err != nil || ref <= 0 || math.IsInf(ref, 0) || math.IsNaN(ref) {
		return out
	}
	need := 1
	if m[1] == "2" {
		need = 2
	}
	end := now.UnixMilli() / 300000 * 300000
	start := end - int64(need)*300000
	minutes := make(map[int64]float64, need*5)
	for _, b := range bars {
		if b.OpenTime < start || b.OpenTime >= end {
			continue
		}
		if b.OpenTime%60000 != 0 || b.CloseTime < b.OpenTime+59999 || b.CloseTime > b.OpenTime+60000 || b.CloseTime > now.UnixMilli() || b.Close <= 0 || math.IsNaN(b.Close) || math.IsInf(b.Close, 0) {
			out.Reason = "UNKNOWN: malformed or incomplete minute tape"
			return out
		}
		if _, exists := minutes[b.OpenTime]; exists {
			out.Reason = "UNKNOWN: duplicate minute tape"
			return out
		}
		minutes[b.OpenTime] = b.Close
	}
	for ms := start; ms < end; ms += 60000 {
		if _, ok := minutes[ms]; !ok {
			out.Reason = "UNKNOWN: missing completed minute tape for authored close window"
			return out
		}
	}
	out.Known = true
	out.Anchor = ref
	out.At = time.UnixMilli(end)
	out.Price = minutes[end-60000]
	out.Invalidated = true
	above := m[2] == "above" || m[2] == ">"
	for boundary := start + 300000; boundary <= end; boundary += 300000 {
		close := minutes[boundary-60000]
		if (above && close <= ref) || (!above && close >= ref) {
			out.Invalidated = false
		}
	}
	out.Reason = fmt.Sprintf("authored condition %q: %d completed 5m close(s), last %.2f at %s, threshold %.2f, invalidated=%t", sc.Invalid, need, out.Price, FormatCT(out.At), ref, out.Invalidated)
	return out
}
