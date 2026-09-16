package kernel

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Exercise the production scorer -> stripped DetectedLevel -> recorder seam.
// The expected propensity is the scorer's own result, not a second formula.
func TestStageACandidateRecorderKeepsComputedEvidence(t *testing.T) {
	levels := []DetectedLevel{
		{Kind: KindPDH, Price: 10010, Lo: 10010, Hi: 10010, Label: "PDH"},
		{Kind: KindPDL, Price: 9990, Lo: 9990, Hi: 9990, Label: "PDL"},
		{Kind: KindPWH, Price: 10060, Lo: 10060, Hi: 10060, Label: "PWH"},
	}
	seated, pool := ScoreLevelsMinGradeFull(levels, 10000, 100, nil, 1, 1.5, "")
	if len(pool) < 2 || len(seated) != 1 {
		t.Fatalf("fixture needs one seated and at least one scored cut: seated=%d pool=%d", len(seated), len(pool))
	}
	raw := make([]DetectedLevel, 0, len(pool))
	expected := make(map[string]ScoredLevel)
	for _, row := range pool {
		raw = append(raw, row.DetectedLevel)
		expected[row.Label] = row
	}
	rows := BuildCandidatePool(raw, seated, 10000, 100, 1.5, 1)
	cut := 0
	for _, row := range rows {
		t.Run(row.Label, func(t *testing.T) {
			want := expected[row.Label]
			if row.Score != want.Score || row.Grade != want.Grade {
				t.Errorf("captured propensity was discarded: score=%v grade=%q; scorer supplied score=%v grade=%q", row.Score, row.Grade, want.Score, want.Grade)
			}
			var components map[string]any
			if err := json.Unmarshal([]byte(row.Components), &components); err != nil || len(components) == 0 {
				t.Errorf("real scorer components missing: %q (decode=%v)", row.Components, err)
			}
			if !row.Seated {
				cut++
				if !strings.Contains(row.CutReason, fmt.Sprintf("%.2f", row.Price)) {
					t.Errorf("cut reason does not identify this candidate's actual exclusion: %q", row.CutReason)
				}
			}
		})
	}
	if cut == 0 {
		t.Fatal("fixture did not exercise a cut row")
	}
}
