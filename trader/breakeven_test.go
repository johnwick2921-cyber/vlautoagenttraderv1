package trader

import (
	"testing"

	"nofx/store"
)

func bePtr(b bool) *bool { return &b }

func TestBreakevenTrigger(t *testing.T) {
	on, off := bePtr(true), bePtr(false)
	cases := []struct {
		name      string
		rc        store.RiskControlConfig
		side      string
		entry     float64
		mark      float64
		wantFire  bool
		wantPtsGE float64 // sanity: pts should be >= this
	}{
		{"disabled (nil) never fires", store.RiskControlConfig{}, "long", 30352, 30450, false, 0},
		{"disabled (false) never fires", store.RiskControlConfig{BreakevenEnabled: off}, "long", 30352, 30450, false, 0},
		{"long +98 >= default 50 → fire", store.RiskControlConfig{BreakevenEnabled: on}, "long", 30352, 30450, true, 90},
		{"long +40 < default 50 → no", store.RiskControlConfig{BreakevenEnabled: on}, "long", 30352, 30392, false, 0},
		{"long exactly +50 → fire", store.RiskControlConfig{BreakevenEnabled: on}, "long", 30352, 30402, true, 50},
		{"short +60 >= 50 → fire (mirror)", store.RiskControlConfig{BreakevenEnabled: on}, "short", 30400, 30340, true, 60},
		{"short losing → no", store.RiskControlConfig{BreakevenEnabled: on}, "short", 30400, 30460, false, 0},
		{"custom trigger 20, long +25 → fire", store.RiskControlConfig{BreakevenEnabled: on, BreakevenTriggerPoints: 20}, "long", 30352, 30377, true, 25},
		{"custom trigger 100, long +25 → no", store.RiskControlConfig{BreakevenEnabled: on, BreakevenTriggerPoints: 100}, "long", 30352, 30377, false, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fire, pts := breakevenTrigger(c.rc, c.side, c.entry, c.mark)
			if fire != c.wantFire {
				t.Fatalf("fire=%v want %v (pts=%.1f)", fire, c.wantFire, pts)
			}
			if fire && pts < c.wantPtsGE {
				t.Fatalf("pts=%.1f want >= %.1f", pts, c.wantPtsGE)
			}
		})
	}
}
