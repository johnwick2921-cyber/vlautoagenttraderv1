// Command planner_replay replays the planner's WRITE-TIME rules over the
// current build's stored plan versions and the 09-24 rejected-prompt rows,
// READ-ONLY (WAVE PLANNER LANE B, B2).
//
// Inputs (from a COPY of data.db, mode=ro):
//   - every plan version whose created_at is >= --since (default
//     2026-09-23T23:50:00Z, the current build's boot), re-run through the
//     SAME production predicates the write site calls:
//     kernel.ArmSpecValid (the legality table — A3),
//     kernel.CheckScenarioWriteTruth with the seated map rebuilt from the
//     doc's stored IdentityLevels (A3/A4/A5),
//     kernel.EvaluateAuthoredInvalidationAt (A1 grammar),
//     kernel.EffectiveArmPolicy + kernel.ArmZoneVerdict +
//     kernel.ArmableConditionFor (W3 entry policy / zone feasibility).
//     zone_max_pts = 0 (uncapped; ResolveZoneMaxPts reads a strategy config
//     this replay does not have) — STATED, not silent, like w2w3_replay.
//   - every planner_rejected_prompts row 339..370: classified into the wave's
//     item classes by the stored reject_reason, printed with row ids.
//
// NOT replayed, and stated as such:
//   - the AI RESPONSE is not stored anywhere in the store
//     (planner_rejected_prompts carries prompt_text + facts, no response
//     column) — so only the plans are replayed;
//   - the geometry rr/net feasibility verdicts (trader.structural_geometry,
//     the A2 rows 342/343/347/368/369/370) need the live ATR5m and the
//     structural stop policy, neither of which the stored rows carry — the
//     rows are listed with their stored reason, marked replay-unavailable.
//
// Output: one NDJSON line per refusal (rule, scenario, detail, ids) and a
// SUMMARY block that names every claim's row ids (sample-id law). The table
// is the wave's acceptance evidence; Lane A runs the same command at its
// head, where the validator deltas appear.
//
// Usage:
//
//	go run ./cmd/planner_replay --db /path/to/copy.db [--since RFC3339]
package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"nofx/kernel"
	"nofx/logger"
)

func main() {
	// The replay output is NDJSON; the kernel's INFO telemetry (scenario
	// economics PASS lines) would flood it. Errors still print.
	_ = logger.InitWithSimpleConfig("error")
	dbPath := flag.String("db", "", "path to a COPY of data.db (opened read-only)")
	sinceS := flag.String("since", "2026-09-23T23:50:00Z", "only plans created at/after this instant (RFC3339, UTC)")
	base := flag.String("base", "be8679ad", "the dev base this head builds on (printed, READ only)")
	head := flag.String("head", "", "this head (default: read from git)")
	flag.Parse()

	if *dbPath == "" {
		fmt.Fprintln(os.Stderr, "PLANNER REPLAY: --db is required (a COPY of data.db)")
		os.Exit(2)
	}
	since, err := time.Parse(time.RFC3339, *sinceS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bad --since: %v\n", err)
		os.Exit(2)
	}

	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  PLANNER WRITE-TIME REPLAY (WAVE PLANNER B2) — READ-ONLY")
	fmt.Println("  no ledger writes · no AI calls · no wire frames")
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Printf("  db      : %s (read-only)\n", *dbPath)
	fmt.Printf("  since   : %s (the current build's boot, UTC)\n", since.UTC().Format(time.RFC3339))
	fmt.Printf("  base    : %s (validator surface identical at head by the lane's report)\n", *base)
	fmt.Printf("  head    : %s\n", *head)
	fmt.Println("  NOTE    : zone_max_pts = 0 (uncapped) — the strategy config is not in the replay.")
	fmt.Println("  NOTE    : AI responses are NOT stored — plans only. Geometry rr/net needs live ATR + stop policy — listed, not re-judged.")
	fmt.Println()

	db, err := sql.Open("sqlite", "file:"+*dbPath+"?mode=ro&_pragma=busy_timeout(5000)")
	if err != nil {
		fmt.Fprintf(os.Stderr, "open db: %v\n", err)
		os.Exit(2)
	}
	defer db.Close()

	// ── Part 1: the rejected-prompt rows 339..370 ─────────────────────────
	rejected := loadRejected(db)
	classOf := classifyRejected(rejected)
	printRejectedTable(rejected, classOf)

	// ── Part 2: every plan version since the boot ─────────────────────────
	rows := loadPlans(db)
	refusals := map[string]int{}
	itemRows := map[string][]string{}
	replayed := 0
	for _, r := range rows {
		at, perr := parseStoredTime(r.CreatedAt)
		if perr != nil || at.Before(since) {
			continue
		}
		replayed++
		d, perr := kernel.ParsePlanDocCapped(r.Doc, 12, 5)
		if perr != nil {
			emit(r, "parse_fail", "", perr.Error())
			refusals["parse_fail"]++
			itemRows["parse"] = append(itemRows["parse"], fmt.Sprintf("plan rowid %d", r.RowID))
			continue
		}
		refs := replayPlan(r, d)
		for _, e := range refs {
			emit(r, e.Rule, e.Scenario, e.Detail)
			refusals[e.Rule]++
			itemRows[e.Item] = append(itemRows[e.Item], fmt.Sprintf("plan rowid %d", r.RowID))
		}
		if len(refs) == 0 {
			emit(r, "pass", "", fmt.Sprintf("%d scenario(s) clear every replayed predicate", len(d.Scenarios)))
		}
	}
	// fold the rejected rows into the same per-item table
	for id, item := range classOf {
		itemRows[item] = append(itemRows[item], fmt.Sprintf("rejected id %d", id))
	}
	fmt.Printf("\nREPLAYED plan versions (>= %s): %d\n", since.UTC().Format(time.RFC3339), replayed)
	fmt.Println("REFUSALS by rule:")
	for _, k := range sortedKeys(refusals) {
		fmt.Printf("  %-28s %d\n", k, refusals[k])
	}
	fmt.Println("\n════════════ BEFORE/AFTER TABLE BY ITEM (row ids named) ════════════")
	for _, item := range sortedKeys(itemRows) {
		sort.Strings(itemRows[item])
		fmt.Printf("  %-22s n=%d ids: %s\n", item, len(itemRows[item]), strings.Join(itemRows[item], " "))
	}
	fmt.Println("════════════════════════════════════════════════════════════")
	fmt.Println("  LABEL: this table is the acceptance evidence. At THIS head the")
	fmt.Println("  validator predicates are the lane-B surface (kernel/ untouched by")
	fmt.Println("  B1 — executor only); Lane A runs the same command at its head and")
	fmt.Println("  its validator deltas show up here.")
	fmt.Println("════════════════════════════════════════════════════════════")
}

type planRow struct {
	RowID     int64
	PlanID    string
	Version   int
	Session   string
	Lifecycle string
	Doc       string
	CreatedAt string
}

type rejectedRow struct {
	ID           int64
	Session      string
	Attempt      int
	RejectReason string
	CreatedAt    string
}

func loadRejected(db *sql.DB) []rejectedRow {
	rows, err := db.Query(`SELECT id, session, attempt, reject_reason, created_at
		FROM planner_rejected_prompts WHERE id BETWEEN 339 AND 370 ORDER BY id`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "rejected query: %v\n", err)
		os.Exit(2)
	}
	defer rows.Close()
	var out []rejectedRow
	for rows.Next() {
		var r rejectedRow
		if err := rows.Scan(&r.ID, &r.Session, &r.Attempt, &r.RejectReason, &r.CreatedAt); err != nil {
			fmt.Fprintf(os.Stderr, "rejected scan: %v\n", err)
			os.Exit(2)
		}
		out = append(out, r)
	}
	return out
}

func loadPlans(db *sql.DB) []planRow {
	rows, err := db.Query(`SELECT rowid, plan_id, version, session, lifecycle, doc, created_at
		FROM plans WHERE created_at >= '2026-09-23 00:00:00' ORDER BY rowid`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "plans query: %v\n", err)
		os.Exit(2)
	}
	defer rows.Close()
	var out []planRow
	for rows.Next() {
		var r planRow
		if err := rows.Scan(&r.RowID, &r.PlanID, &r.Version, &r.Session, &r.Lifecycle, &r.Doc, &r.CreatedAt); err != nil {
			fmt.Fprintf(os.Stderr, "plans scan: %v\n", err)
			os.Exit(2)
		}
		out = append(out, r)
	}
	return out
}

// parseStoredTime accepts the store's mixed separators and offsets
// ("2026-09-23T01:51:47.9-05:00", "2026-09-24 13:59:26.9+00:00") and returns
// the UTC instant.
func parseStoredTime(s string) (time.Time, error) {
	for _, layout := range []string{
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05-07:00",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable stored time %q", s)
}

type refusal struct {
	Rule     string
	Item     string
	Scenario string
	Detail   string
}

func emit(r planRow, rule, scenario, detail string) {
	fmt.Printf("rowid=%d plan=%s v=%d session=%s lifecycle=%s rule=%s scenario=%s detail=%s\n",
		r.RowID, r.PlanID, r.Version, r.Session, r.Lifecycle, rule, scenario, detail)
}

// replayPlan runs the production write-site predicates over one stored doc.
func replayPlan(r planRow, d *kernel.PlanDoc) []refusal {
	var out []refusal
	add := func(rule, item, sc, detail string) {
		out = append(out, refusal{Rule: rule, Item: item, Scenario: sc, Detail: detail})
	}
	// A3 — the legality table (the SAME ArmSpecValid the write site calls).
	for _, sc := range d.Scenarios {
		if err := kernel.ArmSpecValid(sc); err != nil {
			add("legality", "A3", sc.ID, err.Error())
		}
	}
	// A1 — invalidation grammar.
	for _, sc := range d.Scenarios {
		v := kernel.EvaluateAuthoredInvalidationAt(sc, nil, time.UnixMilli(0))
		if !v.Known && v.Unknown == kernel.AuthoredUnknownGrammar {
			add("A1_grammar", "A1", sc.ID, fmt.Sprintf("invalid %q is outside the grammar", sc.Invalid))
		}
	}
	// A3/A4/A5 — identity = price, obstacle chain, stored durations.
	var seated []kernel.MapCandidate
	for _, l := range d.IdentityLevels {
		seated = append(seated, kernel.MapCandidate{ID: l.ID, Identity: l, Price: l.Price})
	}
	v := kernel.CheckScenarioWriteTruth(d, seated, nil, 0.25)
	for _, is := range v.Issues {
		item := "A3"
		if strings.HasPrefix(is.Class, "obstacle") || strings.Contains(is.Class, "gap") {
			item = "A4"
		}
		if strings.Contains(is.Class, "identity") {
			item = "A5"
		}
		add("W2_"+is.Class, item, is.Scenario, is.Text)
	}
	if !v.IdentityChecked {
		add("A3_unchecked", "A3", "", "no stored identity_levels — the frozen map was UNKNOWN (nil)")
	}
	for _, sc := range d.Scenarios {
		if sc.Confirm == nil || !strings.EqualFold(strings.TrimSpace(sc.Confirm.Rule), "time_hold") {
			continue
		}
		if res := kernel.ResolveConfirm(*sc.Confirm); res.Source == kernel.ConfirmSourceAuthoringDefault {
			add("A5_hold_min", "A5", sc.ID, res.Why)
		}
	}
	// W3 — entry policy / zone feasibility (width uncapped, stated above).
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
					add("W3_zone_"+zv.Code, "A2", sc.ID, zv.Code)
				}
			}
		}
		if policy != kernel.EntryPolicyDefaultLegacy && !kernel.ArmableConditionFor(sc.Condition, policy) {
			add("W3_not_armable", "A3", sc.ID, fmt.Sprintf("condition %q is not armable under policy %q", sc.Condition, policy))
		}
	}
	return out
}

// classifyRejected maps each rejected row 339..370 to the wave's item class by
// the STORED reject_reason text. The mapping is the plan file's own table
// (planner-wave-plan-0924.md §0): A1 no_provenance, A2 write-time feasibility,
// A3 schema/legality, A4 gap reachability, A5 identity/obstacle chain,
// A6 born-dead/flip-met. printRejectedTable self-checks against the cited ids.
func classifyRejected(rows []rejectedRow) map[int64]string {
	out := map[int64]string{}
	for _, r := range rows {
		reason := strings.ToLower(r.RejectReason)
		switch {
		case strings.Contains(reason, "born-dead"), strings.Contains(reason, "born dead"),
			strings.Contains(reason, "flip"), strings.Contains(reason, "already met"):
			out[r.ID] = "A6"
		case strings.Contains(reason, "no_provenance"), strings.Contains(reason, "provenance"):
			out[r.ID] = "A1"
		case strings.Contains(reason, "gap"), strings.Contains(reason, "reach"):
			out[r.ID] = "A4"
		case strings.Contains(reason, "identity"), strings.Contains(reason, "obstacle"):
			out[r.ID] = "A5"
		case strings.HasPrefix(reason, "write-time feasibility"), strings.Contains(reason, "insufficient balance"):
			out[r.ID] = "A2"
		default:
			out[r.ID] = "A3" // schema / legality — everything else the write site refused
		}
	}
	return out
}

// planFileTable is planner-wave-plan-0924.md §0's cited grouping of rows
// 339..370 — the harness self-checks its classification against it instead of
// asserting by hand.
var planFileTable = map[string][]int64{
	"A1": {340, 350},
	"A2": {342, 343, 347, 368, 369, 370},
	"A3": {346, 352, 355, 356, 358, 359, 361, 362, 364, 366},
	"A4": {357, 360, 363},
	"A5": {348, 351, 354, 367},
	"A6": {339, 341, 344, 345, 349, 353, 365},
}

func printRejectedTable(rows []rejectedRow, classOf map[int64]string) {
	fmt.Println("── REJECTED PROMPTS 339..370 (stored reasons, classified) ──")
	byClass := map[string][]int64{}
	byAttempt := map[int]int{}
	for _, r := range rows {
		c := classOf[r.ID]
		byClass[c] = append(byClass[c], r.ID)
		byAttempt[r.Attempt]++
	}
	mismatch := false
	for c, want := range planFileTable {
		got := append([]int64(nil), byClass[c]...)
		sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
		w := append([]int64(nil), want...)
		sort.Slice(w, func(i, j int) bool { return w[i] < w[j] })
		if len(got) != len(w) || fmt.Sprint(got) != fmt.Sprint(w) {
			mismatch = true
			fmt.Printf("  ⚠ MISMATCH vs the plan file's §0 table for %s: got %v want %v\n", c, got, w)
		}
	}
	for _, c := range sortedKeys(byClass) {
		ids := byClass[c]
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		var ss []string
		for _, id := range ids {
			ss = append(ss, fmt.Sprintf("%d", id))
		}
		fmt.Printf("  %-4s n=%d ids: %s\n", c, len(ids), strings.Join(ss, ","))
	}
	if !mismatch {
		fmt.Println("  ✓ the classification matches planner-wave-plan-0924.md §0's cited ids exactly")
	}
	fmt.Printf("  rows read: %d · attempt distribution:", len(rows))
	var ats []int
	for a := range byAttempt {
		ats = append(ats, a)
	}
	sort.Ints(ats)
	for _, a := range ats {
		fmt.Printf(" attempt%d=%d", a, byAttempt[a])
	}
	fmt.Println()
}

func sortedKeys[V any](m map[string]V) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}
