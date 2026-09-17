package kernel

import "strings"

// ── S3 (2026-09-16, amended by the CTO 02:23Z) — THE RELATION FIELDS ─────────
//
// Every scenario's relation to the STRUCTURE table is VALIDATOR-COMPUTED.
// recorded, never inferred, never model-authored (canon 35/53). The model's own
// claim, if any, is kept separately (relation_claimed) and never trusted.
// Counter-trend is a restriction-with-hint, NOT a block — the owner decides
// blocks after the S4 measurement is final.

const (
	// RelationWithTrend — the scenario trades WITH the structure trend.
	RelationWithTrend = "with-trend"
	// RelationCounterTrend — the scenario trades AGAINST the structure trend.
	RelationCounterTrend = "counter-trend"
	// RelationRange — the structure is range, or the TF state is unknown.
	RelationRange = "range"
)

// relationValues is the enum the prompt names and the schema accepts.
var relationValues = map[string]bool{
	RelationWithTrend:    true,
	RelationCounterTrend: true,
	RelationRange:        true,
}

// RelationForTrend computes the validator's relation for one scenario direction
// against one structure-trend state. Anything that is not long/short up/down
// resolves to range — never a guess.
func RelationForTrend(trend, direction string) string {
	switch strings.ToLower(strings.TrimSpace(trend)) {
	case "up":
		if strings.ToLower(strings.TrimSpace(direction)) == "long" {
			return RelationWithTrend
		}
		if strings.ToLower(strings.TrimSpace(direction)) == "short" {
			return RelationCounterTrend
		}
	case "down":
		if strings.ToLower(strings.TrimSpace(direction)) == "short" {
			return RelationWithTrend
		}
		if strings.ToLower(strings.TrimSpace(direction)) == "long" {
			return RelationCounterTrend
		}
	}
	return RelationRange
}

// StampScenarioRelations writes relation_d / relation_4h from the doc's
// structure table. Absent structure or a missing TF leaves the field EMPTY
// (absent ≠ fabricated). Any model-supplied relation_d / relation_4h value is
// moved to relation_claimed (if that is empty) and then overwritten.
func StampScenarioRelations(d *PlanDoc) {
	if d == nil || d.Structure == nil || d.Structure.TFs == nil {
		return
	}
	for i := range d.Scenarios {
		claimed := strings.TrimSpace(d.Scenarios[i].RelationClaimed)
		if claimed == "" {
			if v := strings.TrimSpace(d.Scenarios[i].RelationD); v != "" {
				claimed = v
			} else if v := strings.TrimSpace(d.Scenarios[i].Relation4h); v != "" {
				claimed = v
			}
			d.Scenarios[i].RelationClaimed = claimed
		}
		d.Scenarios[i].RelationD = ""
		d.Scenarios[i].Relation4h = ""
		if tf, ok := d.Structure.TFs["D"]; ok && strings.TrimSpace(tf.Trend) != "" {
			d.Scenarios[i].RelationD = RelationForTrend(tf.Trend, d.Scenarios[i].Direction)
		}
		if tf, ok := d.Structure.TFs["4h"]; ok && strings.TrimSpace(tf.Trend) != "" {
			d.Scenarios[i].Relation4h = RelationForTrend(tf.Trend, d.Scenarios[i].Direction)
		}
	}
}

// CounterTrendRelationHint is the restriction-with-hint the prompt and the
// repair block carry: the vocabulary the author is judged by, never a block.
const CounterTrendRelationHint = "STRUCTURE LAW: the structure table names the D and 4h trend — a scenario trading AGAINST it must carry the validator-stamped relation (counter-trend); the validator stamps relation_d / relation_4h itself, the model never writes them."

// RepairStructureRelationLaw is the repair-prompt excerpt for S3(c).
const RepairStructureRelationLaw = "STRUCTURE RELATION LAW: relation_d and relation_4h are VALIDATOR-STAMPED from the structure table (with-trend | counter-trend | range) — do NOT write them; your own claim may go in relation_claimed and is kept but never trusted. A counter-trend scenario is flagged, never blocked."

// StructureTrend4h reads the structure table's 4h trend, empty when the table
// or the TF is absent (absent ≠ fabricated).
func StructureTrend4h(s *StructureMap) string {
	if s == nil || s.TFs == nil {
		return ""
	}
	if tf, ok := s.TFs["4h"]; ok {
		return tf.Trend
	}
	return ""
}
