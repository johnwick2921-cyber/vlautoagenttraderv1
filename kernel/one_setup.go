package kernel

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// ── ONE SETUP (dispatch 102, 2026-09-10) — THE PREDICATE ─────────────────────
//
// "One setup you understand completely beats five you half-do." The book arms
// ONE play — the fade (`reject`) — at the BEST level near price, and only on a
// permitted day. Three verdicts, all three must hold, and a declined scenario
// names all three (A9). This file is PURE: `now` arrives as an argument, the
// map arrives as an argument, the permission arrives already evaluated (A28,
// class 60). It reads no store, no clock and no globals.
//
// What this predicate is NOT (A31): it never touches the map, the seat race,
// the merge, the planner or what the planner is shown (E0 pins that); it never
// cancels anything; it never reaches the wire. It decides AUTHORIZATION at one
// call site on the arm seam, and nothing else.
//
// The fade is an OWNER-RULED [O] selection — the less-contradicted side of
// round 17, not a proven one. The follow side is recorded, never armed
// (follow_plan.go).

// OneSetupConfig carries the RESOLVED knobs (A11). Enabled defaults ON and
// MinGrade to B at the resolver (store.DayPlanConfig), never here.
type OneSetupConfig struct {
	Enabled  bool
	MinGrade string // "A+" | "A" | "B" | "C"
}

// OneSetupDefaultMinGrade is the [O] default the resolver applies when the
// strategy carries no one_setup_min_grade.
const OneSetupDefaultMinGrade = "B"

// Level-reference bases. candidate_id is 105's identity resolution;
// price_proximity is the dispatch's fallback ("else ScenarioNearest with
// basis=price_proximity"); anything else is unresolved and NEVER resolves.
const (
	LevelBasisCandidateID    = "candidate_id"
	LevelBasisPriceProximity = "price_proximity"
)

// OneSetupLevelRef is how the CALLER resolved the scenario's level: by
// level_id (Basis candidate_id, ID set), by the evaluator's anchor price
// (Basis price_proximity), or not at all (Basis "unresolved:…" /
// "legacy:no_level_id" with no price).
type OneSetupLevelRef struct {
	Price float64
	ID    *string
	Basis string
}

// OneSetupLevelFacts is the map as it stands at `now`: the merged candidates
// (W3's BuildMapCandidates), the reference price, the reachability half-width
// and the scenario's own level reference.
type OneSetupLevelFacts struct {
	Price      float64
	BandPts    float64
	Candidates []MapCandidate
	Scenario   OneSetupLevelRef
}

// OneSetupVerdict is the three-legged answer. Level/Play/Permission are "ok"
// or a reason; Allowed is true only when all three are "ok" (or the switch is
// OFF, in which case Reason says so and the seam behaves as today).
type OneSetupVerdict struct {
	Allowed     bool          `json:"allowed"`
	Level       string        `json:"level"`
	Play        string        `json:"play"`
	Permission  string        `json:"permission"`
	Reason      string        `json:"reason"`
	Best        *MapCandidate `json:"-"`
	BestPrice   float64       `json:"best_price,omitempty"`
	BestNames   string        `json:"best_names,omitempty"`
	BestGrade   string        `json:"best_grade,omitempty"`
	EvaluatedMs int64         `json:"evaluated_ms"`
}

// OneSetupPlay is the one condition the book arms.
const OneSetupPlay = "reject"

// OneSetupAllowsAt is the predicate. Every leg is evaluated INDEPENDENTLY and
// every leg is reported — the verdict is a list, never the first miss.
func OneSetupAllowsAt(now time.Time, sc PlanScenario, level OneSetupLevelFacts, perm FadeVerdict, cfg OneSetupConfig) OneSetupVerdict {
	v := OneSetupVerdict{EvaluatedMs: now.UnixMilli()}
	if !cfg.Enabled {
		v.Allowed, v.Level, v.Play, v.Permission, v.Reason = true, "off", "off", "off", "one_setup=off"
		return v
	}

	// (1) LEVEL — the scenario's level IS the top-ranked merged candidate
	// within the reachability band: grade first (≥ min), distance second.
	best, hasBest := OneSetupBestCandidate(level.Candidates, level.Price, level.BandPts, cfg.MinGrade)
	if hasBest {
		b := best
		v.Best = &b
		v.BestPrice, v.BestNames, v.BestGrade = b.Price, b.NamesLine(), b.Grade
	}
	ref := level.Scenario
	switch {
	case ref.Basis != LevelBasisCandidateID && ref.Basis != LevelBasisPriceProximity:
		// unresolved:* or legacy with no price — NULL never resolves (A24).
		basis := ref.Basis
		if basis == "" {
			basis = "unresolved:no_basis"
		}
		v.Level = "level_unresolved:" + basis
	case ref.Price <= 0:
		v.Level = "level_unresolved:no_price"
	case !hasBest:
		v.Level = "level_no_candidate"
	case oneSetupSameLevel(ref, best):
		v.Level = "ok"
	default:
		v.Level = fmt.Sprintf("level_not_best:%s@%s(%s)", best.NamesLine(), trimFloat(best.Price), best.Grade)
	}

	// (2) PLAY — condition == reject.
	cond := strings.ToLower(strings.TrimSpace(sc.Condition))
	if cond == OneSetupPlay {
		v.Play = "ok"
	} else {
		v.Play = "play_not_reject:" + cond
	}

	// (3) PERMISSION — fade_permitted == true at now. false → excluded with its
	// names; not evaluated → not_evaluated. NULL never permits.
	switch {
	case !perm.Evaluated:
		v.Permission = "not_evaluated"
	case perm.Permitted:
		v.Permission = "ok"
	default:
		names := make([]string, 0, len(perm.Exclusions))
		for _, ex := range perm.Exclusions {
			names = append(names, ex.Name)
		}
		v.Permission = "day_excluded(" + strings.Join(names, ",") + ")"
	}

	v.Allowed = v.Level == "ok" && v.Play == "ok" && v.Permission == "ok"
	if v.Allowed {
		v.Reason = "allowed"
	} else {
		v.Reason = fmt.Sprintf("level=%s play=%s permission=%s", v.Level, v.Play, v.Permission)
	}
	return v
}

// oneSetupSameLevel: the scenario's reference and the best candidate are the
// same reference — by identity when both carry one (the candidate's own id or
// any id merged into it), else by price within the map's own merge width.
func oneSetupSameLevel(ref OneSetupLevelRef, c MapCandidate) bool {
	if ref.ID != nil && *ref.ID != "" {
		if c.ID != nil && *c.ID == *ref.ID {
			return true
		}
		for _, sid := range c.Identity.SourceIDs {
			if sid == *ref.ID {
				return true
			}
		}
	}
	return math.Abs(ref.Price-c.Price) <= clusterToleranceFor(c.Price)
}

// OneSetupBestCandidate ranks the merged map: inside the band, grade ≥ min,
// grade first, distance second, price as the deterministic tiebreak. Any tf,
// any kind. A projection is never the best — it is never an entry (W3 D5).
// ok=false when nothing inside the band reaches the minimum grade.
func OneSetupBestCandidate(cands []MapCandidate, price, bandPts float64, minGrade string) (MapCandidate, bool) {
	minRank := qualityRank(minGrade)
	elig := make([]MapCandidate, 0, len(cands))
	for _, c := range cands {
		if c.Projection || c.Price <= 0 {
			continue
		}
		if bandPts > 0 && math.Abs(c.Price-price) > bandPts {
			continue
		}
		if qualityRank(c.Grade) < minRank {
			continue
		}
		elig = append(elig, c)
	}
	if len(elig) == 0 {
		return MapCandidate{}, false
	}
	sort.SliceStable(elig, func(i, j int) bool {
		gi, gj := qualityRank(elig[i].Grade), qualityRank(elig[j].Grade)
		if gi != gj {
			return gi > gj
		}
		di, dj := math.Abs(elig[i].Price-price), math.Abs(elig[j].Price-price)
		if di != dj {
			return di < dj
		}
		return elig[i].Price < elig[j].Price
	})
	return elig[0], true
}

// OneSetupOrder is D4's iteration order: ALLOWED scenarios first, ranked by
// quality (A+ > A > B > C) then doc order; everything else follows in doc
// order. Nothing is dropped. With no allowed set (switch OFF) the input is
// returned as-is, so the seam iterates exactly as it does today (E2).
func OneSetupOrder(scs []PlanScenario, allowed map[string]bool) []PlanScenario {
	if len(allowed) == 0 {
		return scs
	}
	idx := make([]int, len(scs))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		ia, ib := allowed[scs[idx[a]].ID], allowed[scs[idx[b]].ID]
		if ia != ib {
			return ia
		}
		if !ia {
			return idx[a] < idx[b]
		}
		qa, qb := qualityRank(scs[idx[a]].Quality), qualityRank(scs[idx[b]].Quality)
		if qa != qb {
			return qa > qb
		}
		return idx[a] < idx[b]
	})
	out := make([]PlanScenario, len(scs))
	for i, k := range idx {
		out[i] = scs[k]
	}
	return out
}
