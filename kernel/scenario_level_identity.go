package kernel

import (
	"math"
	"time"

	"nofx/levelidentity"
)

func identityString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
func identityInt64(n int64) *int64 {
	if n <= 0 {
		return nil
	}
	return &n
}

// Preserve the pre-identity overlay comparison exactly. Recording metadata
// must not turn an unchanged authored level into an overlay change.
func sameTradingPlanLevel(a, b PlanLevel) bool {
	return a.Price == b.Price && a.Label == b.Label && a.Grade == b.Grade && a.Instruction == b.Instruction && a.MachineGrade == b.MachineGrade
}

// WithFormationClose is an OUTPUT seam. Callers supply an actual source close
// (or the explicitly defined VWAP anchor), never alter detection to obtain it.
func WithFormationClose(l DetectedLevel, closeMs int64, lookback int, basis string, now time.Time) DetectedLevel {
	l.FormationLookback = lookback
	l.FormationBasis = basis
	l.FormedCloseMs = nil
	if closeMs > 0 && closeMs <= now.UnixMilli() {
		l.FormedCloseMs = &closeMs
	}
	return l
}

// Completion is evidenced by the actual last bar at the existing session end.
// A missing final bar or a still-developing window leaves formation unknown.
func completedSessionSourceClose(closeMs int64, name string, reg SessionRegistry, now time.Time) int64 {
	s, ok := reg.SessionByName(name)
	if !ok || closeMs <= 0 {
		return 0
	}
	m, ok := parseHHMM(s.WindowEndCT)
	if !ok {
		return 0
	}
	d := time.UnixMilli(closeMs - 1).In(CTLocation())
	end := time.Date(d.Year(), d.Month(), d.Day(), m/60, m%60, 0, 0, CTLocation())
	if now.Before(end) || closeMs < end.UnixMilli()-1 || closeMs > end.UnixMilli() {
		return 0
	}
	return closeMs
}

func captureFormationGroup(levels []DetectedLevel, closeMs int64, lookback int, basis string, now time.Time) []DetectedLevel {
	for i := range levels {
		levels[i] = WithFormationClose(levels[i], closeMs, lookback, basis, now)
	}
	return levels
}

// CaptureIdentityContext records the source series at the assembly boundary.
// baseTF is supplied by the caller that owns the series; it is not inferred
// from a label, a guessed bar interval, or a legacy record.
func CaptureIdentityContext(levels []DetectedLevel, symbol, baseTF string) {
	for i := range levels {
		l := &levels[i]
		l.IdentitySymbol = symbol
		l.FormationTF = l.TF
		if l.FormationTF == "" {
			l.FormationTF = baseTF
		}
	}
}

func IdentityInputs(l PlanLevel) levelidentity.Inputs {
	value := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	return levelidentity.Inputs{Symbol: value(l.Symbol), Kind: value(l.Kind), Lo: l.Lo, Hi: l.Hi, OriginDate: value(l.OriginDate), TF: value(l.TF), FormedCloseMs: l.FormedCloseMs}
}

// CandidateIdentity copies the PRIMARY merged member's original identity;
// weaker merged members never lend their formation to an unknown primary.
func CandidateIdentity(l DetectedLevel) PlanLevel {
	kind := string(l.Kind)
	r := PlanLevel{Price: l.Price, Label: l.Label, Symbol: identityString(l.IdentitySymbol), Kind: &kind, Lo: &l.Lo, Hi: &l.Hi, OriginDate: identityString(l.OriginDate), TF: identityString(l.FormationTF), FormedAtMs: identityInt64(l.FormedAtMs), FormedCloseMs: l.FormedCloseMs}
	if l.FormationLookback > 0 {
		n := l.FormationLookback
		r.LookbackBars = &n
	}
	r.ID, _ = levelidentity.ID(IdentityInputs(r))
	return r
}

// LevelByID is the only read-side identity resolver. Stored IDs are checked
// against all recorded inputs. Duplicate IDs with conflicting prices fail
// unresolved; no nearest-price fallback is hidden in this function.
func LevelByID(id *string, levels []PlanLevel) (PlanLevel, bool) {
	if id == nil || *id == "" {
		return PlanLevel{}, false
	}
	var found PlanLevel
	ok := false
	for _, l := range levels {
		if l.ID == nil || *l.ID != *id {
			continue
		}
		computed, _ := levelidentity.ID(IdentityInputs(l))
		if computed == nil || *computed != *id {
			return PlanLevel{}, false
		}
		if ok && (found.Price != l.Price || found.Label != l.Label) {
			return PlanLevel{}, false
		}
		found = l
		ok = true
	}
	return found, ok
}

type ScenarioIdentity struct {
	LevelID         *string    `json:"level_id"`
	Level           *PlanLevel `json:"level"`
	Basis           string     `json:"basis"`
	EvaluatorAnchor *float64   `json:"evaluator_anchor"`
	Disagreed       bool       `json:"disagreed"`
}

// ResolveScenarioIdentity records the comparison, never changes the evaluator.
func ResolveScenarioIdentity(sc PlanScenario, levels []PlanLevel, anchor float64, hasAnchor bool) ScenarioIdentity {
	r := ScenarioIdentity{LevelID: sc.LevelID, Basis: "legacy:no_level_id"}
	if hasAnchor {
		r.EvaluatorAnchor = &anchor
	}
	if sc.LevelID == nil || *sc.LevelID == "" {
		return r
	}
	r.Basis = "unresolved:unknown_level_id"
	if l, ok := LevelByID(sc.LevelID, levels); ok {
		r.Level = &l
		r.Basis = "candidate_id"
		r.Disagreed = hasAnchor && math.Abs(l.Price-anchor) > clusterToleranceFor(l.Price)
	}
	return r
}

type IdentityWarnings struct {
	Named, Unnamed, Unresolved int
	Scenarios                  map[string]ScenarioIdentity
}

// StampAuthoredIdentity is WARN-only, for NEW authoring. No error/refusal can
// escape this recording boundary. Legacy reads do not call it.
func StampAuthoredIdentity(doc *PlanDoc, candidates []MapCandidate) IdentityWarnings {
	w := IdentityWarnings{Scenarios: map[string]ScenarioIdentity{}}
	if doc == nil {
		return w
	}
	doc.IdentityLevels = nil
	if candidates != nil {
		doc.IdentityLevels = make([]PlanLevel, 0, len(candidates))
	}
	for _, c := range candidates {
		l := c.Identity
		l.Names = append([]string(nil), c.Names...)
		doc.IdentityLevels = append(doc.IdentityLevels, l)
	}
	for i := range doc.Levels {
		old := doc.Levels[i]
		// Metadata is machine-supplied, never model-supplied.
		doc.Levels[i] = PlanLevel{Price: old.Price, Label: old.Label, Grade: old.Grade, Instruction: old.Instruction, MachineGrade: old.MachineGrade}
		// Only an exact, unique shown price can carry the machine's metadata.
		// This does not name a scenario; only its authored level_id does that.
		var match *PlanLevel
		for j := range doc.IdentityLevels {
			if doc.IdentityLevels[j].Price == doc.Levels[i].Price {
				if match != nil {
					match = nil
					break
				}
				match = &doc.IdentityLevels[j]
			}
		}
		if match != nil {
			old := doc.Levels[i]
			doc.Levels[i] = *match
			doc.Levels[i].Label = old.Label
			doc.Levels[i].Grade = old.Grade
			doc.Levels[i].Instruction = old.Instruction
			doc.Levels[i].MachineGrade = old.MachineGrade
		}
	}
	for i := range doc.Scenarios {
		sc := &doc.Scenarios[i]
		if sc.LevelID != nil && *sc.LevelID == "" {
			sc.LevelID = nil
		}
		anchor, hasAnchor := ScenarioAnchor(*sc, doc.Levels)
		r := ResolveScenarioIdentity(*sc, doc.IdentityLevels, anchor, hasAnchor)
		w.Scenarios[sc.ID] = r
		switch r.Basis {
		case "candidate_id":
			w.Named++
		case "legacy:no_level_id":
			w.Unnamed++
		default:
			w.Unresolved++
		}
	}
	return w
}

// ScenarioIdentities is the shared read projection for cards and desk strips.
// It keeps the trading evaluator's original anchor visible beside the ID.
func ScenarioIdentities(doc *PlanDoc) map[string]ScenarioIdentity {
	out := map[string]ScenarioIdentity{}
	if doc == nil {
		return out
	}
	for _, sc := range doc.Scenarios {
		a, ok := ScenarioAnchor(sc, doc.Levels)
		out[sc.ID] = ResolveScenarioIdentity(sc, doc.IdentityLevels, a, ok)
	}
	return out
}

// EpisodeLevelID names an exact primary or an exactly recorded merged member.
// Membership comes from the map's existing merge, never a later price guess.
func EpisodeLevelID(l DetectedLevel, doc *PlanDoc) *string {
	if doc == nil {
		return nil
	}
	own := CandidateIdentity(l)
	if own.ID == nil {
		return nil
	}
	var found *string
	for _, sc := range doc.Scenarios {
		named, ok := LevelByID(sc.LevelID, doc.IdentityLevels)
		if !ok {
			continue
		}
		match := *named.ID == *own.ID
		for _, memberID := range named.SourceIDs {
			if memberID == *own.ID {
				match = true
			}
		}
		if match {
			if found != nil && *found != *named.ID {
				return nil
			}
			found = named.ID
		}
	}
	return found
}

// EpisodeScenarioByID does not choose between two scenarios naming one level.
// Corroborating ScenarioNearest is a fallback only when the row has no ID.
func EpisodeScenarioByID(id *string, doc *PlanDoc) *string {
	if doc == nil {
		return nil
	}
	if _, ok := LevelByID(id, doc.IdentityLevels); !ok {
		return nil
	}
	var found *string
	for _, sc := range doc.Scenarios {
		if sc.LevelID != nil && *sc.LevelID == *id {
			if found != nil {
				return nil
			}
			v := sc.ID
			found = &v
		}
	}
	return found
}
