package kernel

import (
	"encoding/json"
	"fmt"
)

// LevelScoreCapture travels only inside Go. It never changes prompt JSON or a
// comparator. Values are assigned beside the calculation that actually ran.
type LevelScoreCapture struct {
	Role                 *LevelRole `json:"role"`
	Family               string     `json:"family"`
	Freshness            string     `json:"freshness"`
	ZoneBase             *float64   `json:"zone_base"`
	ReversalMultiplier   *float64   `json:"reversal_multiplier"`
	Evidence             *float64   `json:"evidence"`
	SizeMultiplier       *float64   `json:"size_multiplier"`
	FreshMultiplier      *float64   `json:"fresh_multiplier"`
	ConfluenceRaw        *int       `json:"confluence_raw"`
	ConfluenceCapped     *float64   `json:"confluence_capped"`
	ConfluenceMultiplier *float64   `json:"confluence_multiplier"`
	TFMultiplier         *float64   `json:"tf_multiplier"`
	HTFMultiplier        *float64   `json:"htf_multiplier"`
	Score                *float64   `json:"score"`
	Grade                *string    `json:"grade"`
	Overrides            []string   `json:"overrides"`
	Exclusion            *string    `json:"exclusion"`
}

func scoreValue[T any](v T) *T { return &v }

func researchLevels(levels []DetectedLevel) []DetectedLevel {
	out := append([]DetectedLevel(nil), levels...)
	for i := range out {
		if out[i].Research == nil {
			out[i].Research = &LevelScoreCapture{Family: levelFamily(out[i].Kind), Overrides: []string{}}
		}
	}
	return out
}

func researchCut(l DetectedLevel, stage string) {
	if l.Research == nil {
		return
	}
	reason := fmt.Sprintf("%s %.2f [%s]: %s", l.Kind, l.Price, l.Label, stage)
	l.Research.Exclusion = &reason
}

// Called at actual truncation, before slicing drops the losing rows.
func researchCap(rows []ScoredLevel, cap int, stage string) {
	for i := cap; i < len(rows); i++ {
		researchCut(rows[i].DetectedLevel, fmt.Sprintf("%s position=%d cap=%d score=%g grade=%s", stage, i+1, cap, rows[i].Score, rows[i].Grade))
	}
}

func researchGrade(l DetectedLevel, before, after, rule string) {
	if l.Research != nil && before != after {
		l.Research.Overrides = append(l.Research.Overrides, rule+": "+before+" -> "+after)
	}
}

func researchComponents(c *LevelScoreCapture) string {
	if c == nil {
		return "null"
	}
	b, err := json.Marshal(c)
	if err != nil {
		return "null"
	}
	return string(b)
}

// Observe reordering at the existing seating call, including promotions and
// demotions across the cap. It invokes the original function exactly once.
func researchSeat(name string, rows []ScoredLevel, cap int, seat func([]ScoredLevel, int) []ScoredLevel) []ScoredLevel {
	before := make(map[*LevelScoreCapture]int, len(rows))
	for i, l := range rows {
		if l.Research != nil {
			before[l.Research] = i
		}
	}
	result := seat(rows, cap)
	for i, l := range result {
		if l.Research != nil {
			if old, ok := before[l.Research]; ok && old != i {
				l.Research.Overrides = append(l.Research.Overrides, fmt.Sprintf("%s position %d -> %d cap=%d", name, old+1, i+1, cap))
			}
		}
	}
	return result
}
