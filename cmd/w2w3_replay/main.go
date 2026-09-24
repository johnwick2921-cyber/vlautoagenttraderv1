// Command w2w3_replay replays the W-EXEC-TRUTH W2 (A1–A5) and W3
// (entry-policy/zone) write-time rules over historical plan rows, READ-ONLY.
//
// It consumes NDJSON produced by sqlite3 -readonly (one json object per row):
//
//	{"rowid":N,"plan_id":"...","version":1,"session":"NY","created_at":"...",
//	 "read_clock_ms":N,"publish_clock_ms":N,"doc":"{...}"}
//
// It emits one NDJSON line per refusal (and one summary line), so every claim
// in the report can name the row ids it rests on (sample-id law).
//
// Every predicate is a PRODUCTION call site:
//   - A1: kernel.EvaluateAuthoredInvalidationAt(sc, nil, zero) — with a nil
//     tape the verdict is AuthoredUnknownGrammar exactly when the sentence is
//     outside the grammar (the tape is never consulted on that path).
//   - A2: kernel.AuthoredBornGroups(read, publish) — the closed-5m-group count
//     between read and publication. The BREACH judgment needs bars that are
//     not stored with the plan; the report says so instead of inventing one.
//   - A3/A4: kernel.CheckScenarioWriteTruth with the seated map rebuilt from
//     the doc's stored IdentityLevels — the same projection the write site
//     froze into the doc.
//   - A5: kernel.ResolveConfirm — a time_hold that resolves from the env
//     authoring default is the missing-stored-duration refusal.
//   - W3: kernel.EffectiveArmPolicy / ArmableConditionFor / EntryPolicyLegal /
//     ArmZoneVerdict over the stored arms and legs. zone width cap = 0
//     (uncapped; ResolveZoneMaxPts reads a strategy config this replay does
//     not have) — stated, not silent.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"nofx/kernel"
)

type row struct {
	RowID          int64  `json:"rowid"`
	PlanID         string `json:"plan_id"`
	Version        int    `json:"version"`
	Session        string `json:"session"`
	CreatedAt      string `json:"created_at"`
	ReadClockMs    int64  `json:"read_clock_ms"`
	PublishClockMs int64  `json:"publish_clock_ms"`
	Doc            string `json:"doc"`
}

type out struct {
	RowID    int64  `json:"rowid"`
	PlanID   string `json:"plan_id"`
	Session  string `json:"session"`
	Rule     string `json:"rule"`
	Scenario string `json:"scenario,omitempty"`
	Issue    string `json:"issue,omitempty"`
	Detail   string `json:"detail,omitempty"`
	Groups   int    `json:"groups,omitempty"`
}

var enc = json.NewEncoder(os.Stdout)

func emit(o out) { _ = enc.Encode(o) }

func main() {
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 4<<20), 4<<20)
	n := 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var r row
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			fmt.Fprintf(os.Stderr, "bad row: %v\n", err)
			continue
		}
		n++
		replay(r)
	}
	if err := sc.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
	}
	emit(out{PlanID: "SUMMARY", Rule: "rows_replayed", Detail: fmt.Sprintf("%d", n)})
}

func replay(r row) {
	d, err := kernel.ParsePlanDocCapped(r.Doc, 12, 5) // H4/H5: the owner raised the caps to 12 levels / 5 scenarios
	if err != nil {
		emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "parse_fail", Detail: err.Error()})
		return
	}
	// ── A1 — invalidation grammar (write-time refusal) ────────────────────
	for _, sc := range d.Scenarios {
		v := kernel.EvaluateAuthoredInvalidationAt(sc, nil, time.UnixMilli(0))
		if !v.Known && v.Unknown == kernel.AuthoredUnknownGrammar {
			emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "A1_grammar", Scenario: sc.ID, Detail: fmt.Sprintf("invalid %q is outside the grammar", sc.Invalid)})
		}
	}
	// ── A2 — publication-time re-validation (clock half; breach needs bars) ──
	if r.ReadClockMs > 0 && r.PublishClockMs > 0 {
		groups := kernel.AuthoredBornGroups(time.UnixMilli(r.ReadClockMs), time.UnixMilli(r.PublishClockMs))
		if len(groups) > 0 {
			emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "A2_regroups", Groups: len(groups), Detail: "closed 5m group(s) between read and publish — the write site would re-judge each (breach judgment needs bars: UNREPLAYED)"})
		}
	}
	// ── A3 + A4 — identity = price, obstacle chain ───────────────────────
	var seated []kernel.MapCandidate
	for _, l := range d.IdentityLevels {
		seated = append(seated, kernel.MapCandidate{ID: l.ID, Identity: l, Price: l.Price})
	}
	v := kernel.CheckScenarioWriteTruth(d, seated, nil, 0.25)
	for _, is := range v.Issues {
		emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "W2_" + is.Class, Scenario: is.Scenario, Detail: is.Text})
	}
	if !v.IdentityChecked {
		emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "A3_unchecked", Detail: "no stored identity_levels — the frozen map was UNKNOWN (nil)"})
	}
	// ── A5 — stored durations (time_hold hold_min) ───────────────────────
	for _, sc := range d.Scenarios {
		if sc.Confirm == nil || !strings.EqualFold(strings.TrimSpace(sc.Confirm.Rule), "time_hold") {
			continue
		}
		if res := kernel.ResolveConfirm(*sc.Confirm); res.Source == kernel.ConfirmSourceAuthoringDefault {
			emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "A5_hold_min", Scenario: sc.ID, Detail: res.Why})
		}
	}
	// ── W3 — entry policy / zone ─────────────────────────────────────────
	for _, sc := range d.Scenarios {
		arm := sc.Arm
		if arm == nil {
			continue
		}
		policy := kernel.EffectiveArmPolicy(arm, nil)
		if policy == "" {
			policy = kernel.EntryPolicyDefaultLegacy
		}
		if policy == kernel.EntryPolicyMarketInZone {
			legs := arm.Legs
			if len(legs) == 0 {
				legs = []kernel.PlanArmLeg{{Entry: arm.Entry, Stop: arm.Stop, Target: arm.Target}}
			}
			for _, leg := range legs {
				zv := kernel.ArmZoneVerdict(sc, leg.Entry, leg.Stop, leg.Target, sc.Direction, 0.25, 0)
				if zv.Code != "" {
					emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "W3_zone_" + zv.Code, Scenario: sc.ID, Detail: zv.Code})
				}
			}
		}
		// Legacy arms never reach the policy branch (byte-identical pre-W3).
		if policy != kernel.EntryPolicyDefaultLegacy && !kernel.ArmableConditionFor(sc.Condition, policy) {
			emit(out{RowID: r.RowID, PlanID: r.PlanID, Session: r.Session, Rule: "W3_not_armable", Scenario: sc.ID, Detail: fmt.Sprintf("condition %q is not armable under policy %q", sc.Condition, policy)})
		}
	}
}
