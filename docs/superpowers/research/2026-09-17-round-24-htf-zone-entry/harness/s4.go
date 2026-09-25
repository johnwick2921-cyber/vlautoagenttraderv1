//go:build r24harness

package main

// s4.go — S4 MEASUREMENT GATE (CTO dispatch 2026-09-16 22:58Z): Q-A freshness
// grading under BOTH modes (1m-touch vs own-TF/S2), Q-B trend rows (S1 port),
// Q-C/Q-D joins happen offline from the emitted jsonl.
//
// Q-A semantics: for every HTF level-scan's ordinal-1 episode, count "tests"
// before the anchor: bars of the grading timeframe whose range traded into
// [Lo,Hi] at or after the level's origin. Grade: 0 fresh · 1 tested-1 ·
// 2 tested-2 · >=3 stale — the S2 display vocabulary (kernel/
// levels_fresh_by_tf.go @ cba7478c, fix/levels-fresh-by-tf). The 1m-touch
// grade is the SAME rule on 1m bars; today's live ladder maps it to
// A/B/C/done. Origin resolution is S2's levelOriginTime verbatim: OriginDate
// (YYYY-MM-DD) else FormedAtMs; unknown origin → 0 tests (fresh).

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// s1HTFFreshTF is S2's htfFreshTFSet verbatim (kernel/levels_fresh_by_tf.go).
func s1HTFFreshTF(tf string) bool {
	switch strings.ToLower(strings.TrimSpace(tf)) {
	case "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w":
		return true
	}
	return false
}

type s4State struct {
	qaw *bufio.Writer
	tw  *bufio.Writer
}

func newS4State(outDir string) (*s4State, error) {
	qaFile, err := os.Create(filepath.Join(outDir, "qa.jsonl"))
	if err != nil {
		return nil, err
	}
	trFile, err := os.Create(filepath.Join(outDir, "trends.jsonl"))
	if err != nil {
		qaFile.Close()
		return nil, err
	}
	return &s4State{
		qaw: bufio.NewWriterSize(qaFile, 1<<20),
		tw:  bufio.NewWriterSize(trFile, 1<<20),
	}, nil
}

func (s *s4State) close() error {
	if err := s.qaw.Flush(); err != nil {
		return err
	}
	if err := s.tw.Flush(); err != nil {
		return err
	}
	return nil
}

// trendRow is one read's S1 structure state (trend only — what Q-B joins on).
type trendRow struct {
	Day      string `json:"day"`
	Session  string `json:"session"`
	Contract string `json:"contract"`
	ReadAtMs int64  `json:"read_at_ms"`
	DTrend   string `json:"d_trend"`
	DSwings  int    `json:"d_swings"`
	H4Trend  string `json:"h4_trend"`
	H4Swings int    `json:"h4_swings"`
	H1Trend  string `json:"h1_trend"`
	H1Swings int    `json:"h1_swings"`
	DCBars   int    `json:"d_closed"`
	H4CBars  int    `json:"h4_closed"`
	H1CBars  int    `json:"h1_closed"`
}

// emitTrends writes the S1 port's trend for D/4h/1h at this read. Bars come
// from the same store copy the planner reads (ContractAt-scoped per read).
func (s *s4State) emitTrends(bd *barDB, r *readSnapshot) error {
	now := r.ReadTime.UnixMilli()
	tr := trendRow{Day: r.Day, Session: r.Session, Contract: r.Contract, ReadAtMs: now}
	tr.DTrend, tr.DSwings, tr.DCBars = trendFor(bd, r.Contract, "1d", now)
	tr.H4Trend, tr.H4Swings, tr.H4CBars = trendFor(bd, r.Contract, "4h", now)
	tr.H1Trend, tr.H1Swings, tr.H1CBars = trendFor(bd, r.Contract, "1h", now)
	line, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	s.tw.Write(line)
	s.tw.WriteByte('\n')
	return nil
}

// trendFor computes S1's trend for one TF from the store bars (the port's
// entry point). tf is "1d"|"4h"|"1h" (the loaded store labels).
func trendFor(bd *barDB, contract, tf string, nowMs int64) (string, int, int) {
	rows := bd.byKey[seriesKey{contract, tf}]
	if len(rows) == 0 {
		return "range", 0, 0
	}
	bars := toKline(rows, tf)
	ivMin := int(tfMillis[tf] / 60_000)
	return s1Trend(bars, ivMin, nowMs)
}

// qaRec is one Q-A row: an HTF level-scan's ordinal-1 episode with both
// freshness grades computed at its anchor.
type qaRec struct {
	Day        string  `json:"day"`
	Session    string  `json:"session"`
	Kind       string  `json:"kind"`
	TF         string  `json:"tf"`
	Price      float64 `json:"price"`
	Lo         float64 `json:"lo"`
	Hi         float64 `json:"hi"`
	HasOrigin  bool    `json:"has_origin"`
	OriginMs   int64   `json:"origin_ms"`
	AnchorMs   int64   `json:"anchor_ms"`
	Tests1m    int     `json:"tests_1m"`
	Grade1m    string  `json:"grade_1m"`
	TestsTF    int     `json:"tests_tf"` // -1 = own-TF bars not loaded (NOT MEASURED)
	GradeTF    string  `json:"grade_tf,omitempty"`
	Entry      string  `json:"entry"`
	Outcome    string  `json:"outcome"`
	OpenedAtMs int64   `json:"opened_at_ms"`
}

// emitQA computes and writes the Q-A row for one HTF level-scan.
func (s *s4State) emitQA(bd *barDB, r *readSnapshot, l kernel.DetectedLevel, tf string, e kernel.TouchOutcome) error {
	lo, hi := l.Lo, l.Hi
	if hi < lo {
		lo, hi = hi, lo
	}
	origin, has := s4Origin(l)
	anchor := e.OpenedAtMs

	row := qaRec{
		Day: r.Day, Session: r.Session,
		Kind: string(l.Kind), TF: tf,
		Price: l.Price, Lo: l.Lo, Hi: l.Hi,
		HasOrigin: has, OriginMs: origin, AnchorMs: anchor,
		TestsTF: -1,
		Entry:   e.Entry, Outcome: e.Outcome, OpenedAtMs: e.OpenedAtMs,
	}
	row.Tests1m = countTests(bd, r.Contract, "1m", origin, anchor, lo, hi)
	row.Grade1m = gradeByTests(row.Tests1m)
	if _, ok := bd.byKey[seriesKey{r.Contract, tf}]; ok {
		row.TestsTF = countTests(bd, r.Contract, tf, origin, anchor, lo, hi)
		row.GradeTF = gradeByTests(row.TestsTF)
	}
	line, err := json.Marshal(row)
	if err != nil {
		return err
	}
	s.qaw.Write(line)
	s.qaw.WriteByte('\n')
	return nil
}

// s4Origin is S2's levelOriginTime verbatim: OriginDate first, then
// FormedAtMs; neither → has=false (0 tests, fresh).
func s4Origin(l kernel.DetectedLevel) (ms int64, has bool) {
	if d := strings.TrimSpace(l.OriginDate); d != "" {
		if t, err := time.Parse("2006-01-02", d); err == nil {
			return t.UnixMilli(), true
		}
	}
	if l.FormedAtMs > 0 {
		return l.FormedAtMs, true
	}
	return 0, false
}

// gradeByTests maps a test count onto the S2 display vocabulary.
func gradeByTests(n int) string {
	switch n {
	case 0:
		return "fresh"
	case 1:
		return "tested-1"
	case 2:
		return "tested-2"
	default:
		return "stale"
	}
}

// countTests counts bars of (contract, tf) with originMs <= openMs < anchorMs
// whose range traded into [lo,hi] — S2's LevelFreshnessByTF test condition
// verbatim (b.Low <= hi && b.High >= lo). Bars are sorted by openMs.
func countTests(bd *barDB, contract, tf string, originMs, anchorMs int64, lo, hi float64) int {
	rows := bd.byKey[seriesKey{contract, tf}]
	if len(rows) == 0 || anchorMs <= originMs {
		return 0
	}
	start := sort.Search(len(rows), func(i int) bool { return rows[i].openMs >= originMs })
	end := sort.Search(len(rows), func(i int) bool { return rows[i].openMs >= anchorMs })
	if end > len(rows) {
		end = len(rows)
	}
	tests := 0
	for _, b := range rows[start:end] {
		if b.l <= hi && b.h >= lo {
			tests++
		}
	}
	return tests
}

var _ = fmt.Sprintf
var _ = market.Kline{}
