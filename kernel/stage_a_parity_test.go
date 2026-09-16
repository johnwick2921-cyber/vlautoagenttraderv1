package kernel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func stageAParityLevels() []DetectedLevel {
	kinds := []LevelKind{KindPDH, KindPDL, KindPWH, KindONH, KindNPOC, KindSupply, KindDemand, KindFVG, KindOB, KindVWAP, KindRound, KindEQH}
	levels := make([]DetectedLevel, 72)
	for i := range levels {
		p := 30000 + float64(i-36)*6.25
		levels[i] = DetectedLevel{Kind: kinds[i%len(kinds)], Price: p, Lo: p - 2, Hi: p + 2, Label: fmt.Sprintf("fixture-%02d", i), OriginDate: "2026-09-07", HTF: i%3 == 0, TF: []string{"1m", "15m", "1h", "4h"}[i%4], ZonePattern: []string{"reversal", "continuation"}[i%2]}
	}
	return levels
}
func TestStageAScoreParityLegacy(t *testing.T) {
	var results []any
	for _, grade := range []string{"", "A", "B", "C"} {
		for _, cap := range []int{1, 4, 12, 50} {
			for _, fresh := range []string{"fresh", "B", "tested", "consumed"} {
				seated, pool := ScoreLevelsMinGradeFull(stageAParityLevels(), 30000, 300, func(DetectedLevel) string { return fresh }, cap, 1.5, grade)
				results = append(results, struct{ Seated, Pool []ScoredLevel }{seated, pool})
			}
		}
	}
	data, err := json.Marshal(results)
	if err != nil {
		t.Fatal(err)
	}
	if out := os.Getenv("STAGE_A_PARITY_OUT"); out != "" {
		if err = os.WriteFile(out, data, 0600); err != nil {
			t.Fatal(err)
		}
		return
	}
	expected, err := os.ReadFile(filepath.Join("testdata", "stage_a_score_legacy.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, expected) {
		t.Fatal("scorer JSON differs from pre-wave source across 64 fixtures; trading output changed")
	}
}
func BenchmarkStageAScoreCapture(b *testing.B) {
	levels := stageAParityLevels()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ScoreLevelsMinGradeFull(levels, 30000, 300, nil, 12, 1.5, "B")
	}
}
