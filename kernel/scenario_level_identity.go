package kernel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"strings"
	"time"

	"nofx/levelidentity"
)

func identityString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// identityValue is the nil-safe inverse of identityString.
func identityValue(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// referenceAnchorKinds (W-GEOMETRY-REFUSAL, 2026-09-18) are the session/anchor
// reference kinds whose SOURCE WINDOW can still be developing at authoring
// time, so their formation close is unknown and the strict identity id is NULL
// (levelidentity.ID requires formed_close_ms). Census [A] on dev b70fc6ca:
// ONH/ONL/AS-H/AS-L/LDN-H/LDN-L/RTH-H/RTH-L (levels_multiday.go fills
// FormedCloseMs only when completedSessionSourceClose evidences completion) +
// OR-H/OR-L + the VWAP family (eVWAP/pdVWAP) whose capture window can be open.
var referenceAnchorKinds = map[string]bool{
	"ONH": true, "ONL": true, "AS-H": true, "AS-L": true,
	"LDN-H": true, "LDN-L": true, "RTH-H": true, "RTH-L": true,
	"OR-H": true, "OR-L": true, "eVWAP": true, "pdVWAP": true, "VWAP": true,
}

// ReferenceAnchorKind reports whether a kind may legitimately reach the map
// without a formation close.
func ReferenceAnchorKind(kind string) bool { return referenceAnchorKinds[kind] }

// ReferenceLevelID derives a STABLE sha id for a reference-anchor level whose
// strict identity inputs are incomplete (formed_close_ms unknown while the
// source window is still developing). The id is a deterministic hash of
// (symbol|kind|lo|hi|origin_date|tf) — the SAME inputs always yield the SAME
// id, so the planner can author it and the executor resolves it against the
// frozen map. The "ref|" prefix keeps it disjoint from strict ids.
func ReferenceLevelID(symbol, kind string, lo, hi float64, originDate, tf string) *string {
	if strings.TrimSpace(kind) == "" || strings.TrimSpace(originDate) == "" || lo <= 0 || hi <= 0 || math.IsNaN(lo) || math.IsInf(lo, 0) || math.IsNaN(hi) || math.IsInf(hi, 0) {
		return nil
	}
	raw := fmt.Sprintf("ref|%s|%s|%g|%g|%s|%s", symbol, kind, lo, hi, originDate, tf)
	h := sha256.Sum256([]byte(raw))
	// The "ref|" prefix rides on the ID STRING so LevelByReferenceID can tell
	// these ids from strict ids at a glance.
	id := "ref|" + hex.EncodeToString(h[:])
	return &id
}

// EnsureReferenceLevelIDs fills NULL candidate ids for reference-anchor kinds
// with the stable reference id (W-GEOMETRY-REFUSAL). Non-anchor kinds with a
// NULL id stay NULL — their identity is genuinely unknown. Call sites gate this
// on day_plan.geometry_reference_levels (OFF = today's map byte-identical).
func EnsureReferenceLevelIDs(cs []MapCandidate) {
	for i := range cs {
		c := &cs[i]
		if c.ID != nil || c.Identity.Kind == nil {
			continue
		}
		if !ReferenceAnchorKind(*c.Identity.Kind) {
			continue
		}
		if c.Identity.Lo == nil || c.Identity.Hi == nil || *c.Identity.Lo <= 0 || *c.Identity.Hi <= 0 {
			continue
		}
		id := ReferenceLevelID(identityValue(c.Identity.Symbol), *c.Identity.Kind, *c.Identity.Lo, *c.Identity.Hi, identityValue(c.Identity.OriginDate), identityValue(c.Identity.TF))
		if id == nil {
			continue
		}
		c.ID = id
		c.Identity.ID = id
	}
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

// LevelByReferenceID resolves a STABLE reference id (the "ref|" ids
// W-GEOMETRY-REFUSAL assigns to reference-anchor levels whose formation close
// is unknown). Strict inputs cannot recompute these ids by construction (the
// strict id REQUIRES formed_close_ms), so the check is id equality plus
// kind/price/label consistency — duplicates with conflicting prices fail
// unresolved, the same discipline as LevelByID.
func LevelByReferenceID(id *string, levels []PlanLevel) (PlanLevel, bool) {
	if id == nil || *id == "" || !strings.HasPrefix(*id, "ref|") {
		return PlanLevel{}, false
	}
	var found PlanLevel
	ok := false
	for _, l := range levels {
		if l.ID == nil || *l.ID != *id {
			continue
		}
		if l.Kind == nil || !ReferenceAnchorKind(*l.Kind) {
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
	} else if l, ok := LevelByReferenceID(sc.LevelID, levels); ok {
		// W-GEOMETRY-REFUSAL (b1): a stable reference id (anchor kind without a
		// formation close) resolves as a candidate — same discipline, no strict
		// recompute possible by construction.
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
	// W2 A3: the ONE projection the write-time identity check also reads.
	doc.IdentityLevels = IdentityLevelsFromCandidates(candidates)
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
			// W-GEOMETRY-REFUSAL (F3): a ref| id resolves through the reference
			// lookup, so episodes link to reference-anchor scenarios too.
			named, ok = LevelByReferenceID(sc.LevelID, doc.IdentityLevels)
		}
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
		if _, ok := LevelByReferenceID(id, doc.IdentityLevels); !ok {
			return nil // W-GEOMETRY-REFUSAL (F3): ref| ids are real identities
		}
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
