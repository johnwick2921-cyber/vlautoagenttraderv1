package kernel

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"nofx/market"
)

// W-EXEC-TRUTH W2 (confirm resolver) — the byte-identity pin for every rule
// the resolver does NOT change. RenderConfirmLines — the production executor
// prompt renderer (engine_analysis.go RenderConfirmLines call) — is run over
// the 2026-09-08 confirmation-truth replay corpus (272 plan rows, 4,554
// decisions, 19,011 1m bars; provenance in that report's manifest) at every
// decision snapshot, with the tape cut exactly as replay.go.txt cut it. The
// golden was written at the wave's BASE (f2ac79eb) before any resolver code
// existed; after the wave the continuation branch reads the STORED rule, and
// every corpus breakdown row stores 1x5m_close (rows 163/181/183/273), so the
// whole render must stay byte-identical at the default env.
//
// Regenerate ONLY with a quoted reason: CONFIRM_REPLAY_WRITE_GOLDEN=1.
const confirmReplayGolden = "testdata/confirm_resolver/replay_render.golden.gz"

func confirmReplayRender(t *testing.T) []string {
	t.Helper()
	for _, k := range []string{"BD_MIN_CLOSES", "ACCEPT_HOLD_MIN", "STALE_CONFIRM_ATR", "MSS_MIN_DISP_ATR"} {
		t.Setenv(k, "") // the corpus settings are {} — every knob at its default
	}
	var corpus struct {
		Plans []struct {
			RowID   int     `json:"rowid"`
			PlanID  string  `json:"plan_id"`
			Version int     `json:"version"`
			Birth   int64   `json:"birth_ms"`
			Doc     PlanDoc `json:"doc"`
		} `json:"plans"`
		Bars []struct {
			T             int64 `json:"open_time_ms"`
			O, H, L, C, V float64
		} `json:"bars"`
		Decisions []struct {
			ID      int    `json:"id"`
			PlanID  string `json:"plan_id"`
			Version int    `json:"version"`
			Now     int64  `json:"snapshot_ms"`
		} `json:"decisions"`
	}
	b, err := os.ReadFile("../docs/superpowers/reports/2026-09-08-confirmation-truth-data/replay-corpus.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &corpus); err != nil {
		t.Fatal(err)
	}
	type planKey struct {
		id string
		v  int
	}
	plans := map[planKey]int{}
	for i, p := range corpus.Plans {
		plans[planKey{p.PlanID, p.Version}] = i
	}
	bars := make([]market.Kline, 0, len(corpus.Bars))
	for _, x := range corpus.Bars {
		bars = append(bars, market.Kline{OpenTime: x.T, CloseTime: x.T + 59999, Open: x.O, High: x.H, Low: x.L, Close: x.C, Volume: x.V})
	}
	var out []string
	for _, d := range corpus.Decisions {
		if d.Now == 0 {
			continue
		}
		pi, ok := plans[planKey{d.PlanID, d.Version}]
		if !ok {
			t.Fatalf("decision %d has no plan", d.ID)
		}
		p := corpus.Plans[pi]
		if d.Now < p.Birth {
			continue
		}
		end := sort.Search(len(bars), func(i int) bool { return bars[i].CloseTime >= d.Now })
		if end == 0 || d.Now > bars[len(bars)-1].CloseTime+120000 {
			continue
		}
		start := end - AISVPBarCount
		if start < 0 {
			start = 0
		}
		tape := bars[start:end]
		if d.Now-tape[len(tape)-1].CloseTime > 120000 || p.Birth < bars[0].OpenTime {
			continue
		}
		r := RenderConfirmLines(p.Doc, tape, p.Birth, d.Now, tape[len(tape)-1].Close, StaleConfirmATR5m(tape))
		if r == "" {
			continue
		}
		out = append(out, fmt.Sprintf("## decision=%d row=%d now=%d\n%s", d.ID, p.RowID, d.Now, r))
	}
	return out
}

func TestConfirmResolverReplayRenderByteIdentical(t *testing.T) {
	got := confirmReplayRender(t)
	if len(got) == 0 {
		t.Fatal("replay corpus rendered nothing — the harness is broken, not the renderer")
	}
	blob := strings.Join(got, "")
	if os.Getenv("CONFIRM_REPLAY_WRITE_GOLDEN") == "1" {
		var buf bytes.Buffer
		zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
		_, _ = zw.Write([]byte(blob))
		_ = zw.Close()
		if err := os.MkdirAll("testdata/confirm_resolver", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(confirmReplayGolden, buf.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %s: %d rendered decisions, %d bytes", confirmReplayGolden, len(got), len(blob))
		return
	}
	f, err := os.Open(confirmReplayGolden)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	want, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) == blob {
		return
	}
	// Name every changed line (before → after) so a diff can be quoted.
	wl, gl := strings.Split(string(want), "\n"), strings.Split(blob, "\n")
	n := len(wl)
	if len(gl) > n {
		n = len(gl)
	}
	shown := 0
	for i := 0; i < n && shown < 20; i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			t.Errorf("line %d\n  before: %s\n  after:  %s", i+1, w, g)
			shown++
		}
	}
	t.Fatalf("replay render is NOT byte-identical (%d vs %d lines)", len(wl), len(gl))
}
