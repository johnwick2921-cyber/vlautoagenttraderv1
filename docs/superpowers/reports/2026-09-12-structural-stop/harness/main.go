package main

// Measurement-only replay. data.go and detect.go are byte-identical to Backtest 1;
// detectors and the map's non-transitive merging are called, never reimplemented.
import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"nofx/kernel"
	"nofx/market"
	"os"
	"time"
)

func unixMsTime(ms int64) time.Time { return time.UnixMilli(ms) }

type event struct {
	ID                                             string
	Read                                           int
	Day, Session, Contract, Era, Side, Outcome, TF string
	Families                                       int
	Time                                           int64
	Entry, Lo, Hi, ATR, Penetration                float64
	ReversalBars                                   int
	Zone                                           kernel.LevelZone
	Zones                                          []kernel.LevelZone
	Bars                                           []market.Kline
}

func main() {
	dbPath := flag.String("db", "data/structural-stop/backtest.db", "verified backup only")
	outPath := flag.String("out", "data/structural-stop/events.jsonl", "event cache")
	limit := flag.Int("limit", 0, "measurement smoke limit, 0 all")
	sweep := flag.Bool("sweep", false, "replay corrected fills from event cache using production geometry")
	flag.Parse()
	if *sweep {
		runGeometrySweep(*outPath)
		return
	}
	db, err := openCopy(*dbPath)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	d, err := loadBars(db, []string{"1m", "15m", "1h", "4h", "1d"})
	if err != nil {
		panic(err)
	}
	f, err := os.Create(*outPath)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 1<<20)
	defer w.Flush()
	enc := json.NewEncoder(w)
	first := unixMsTime(d.merged1m[0].openMs).In(kernel.CTLocation())
	last := unixMsTime(d.merged1m[len(d.merged1m)-1].openMs).In(kernel.CTLocation())
	day := time.Date(first.Year(), first.Month(), first.Day(), 0, 0, 0, 0, kernel.CTLocation())
	idx, reads, n := 0, 0, 0
	for ; !day.After(last); day = day.AddDate(0, 0, 1) {
		for _, session := range []string{"LONDON", "NY", "ASIA"} {
			r, ok := buildRead(d, db, idx, day, session)
			idx++
			if !ok {
				continue
			}
			reads++
			m := kernel.BuildLevelZones(r.Raw, r.Price, r.ATR5m, r.Inputs, kernel.ResolveZoneOptions(maxLevelsReplay), r.ReadTime)
			bars := r.SessionBars
			if r.PrevBar != nil {
				bars = append([]market.Kline{*r.PrevBar}, bars...)
			}
			for zi, z := range m.Zones {
				if !z.Shortlisted || z.Lo == nil || z.Hi == nil || *z.Hi <= *z.Lo || z.Anchor <= 0 {
					continue
				}
				j := -1
				for k := 1; k < len(bars); k++ {
					if bars[k].Low <= *z.Hi && bars[k].High >= *z.Lo && !(bars[k-1].Low <= *z.Hi && bars[k-1].High >= *z.Lo) {
						j = k
						break
					}
				}
				if j < 1 {
					continue
				}
				e := event{ID: fmt.Sprintf("read-%06d-zone-%04d", r.IDX, zi), Read: r.IDX, Day: r.Day, Session: session, Contract: r.Contract, Era: "in_sample", Side: "long", Outcome: "ambiguous_horizon", Time: bars[j].OpenTime, Entry: z.Anchor, Lo: *z.Lo, Hi: *z.Hi, ATR: r.ATR5m, Families: len(z.Families), Zone: z, Zones: m.Zones, Bars: bars[j:]}
				if r.ReadTime.Format("2006-01-02") >= "2025-09-12" {
					e.Era = "held_out"
				}
				if bars[j-1].Close < e.Lo {
					e.Side = "short"
				}
				if len(z.Sources) > 0 {
					e.TF = z.Sources[0].TF
				}
				// Original H12 close-based hold/break cohort, but penetration INCLUDES
				// the touch bar. OHLC gives an upper bound when the extreme precedes touch.
				for k := j; k < len(bars) && k-j <= 12; k++ {
					p := e.Lo - bars[k].Low
					if e.Side == "short" {
						p = bars[k].High - e.Hi
					}
					e.Penetration = math.Max(e.Penetration, p)
					if k == j {
						continue
					}
					if bars[k].Close > e.Hi {
						e.Outcome = "hold"
						if e.Side == "short" {
							e.Outcome = "break"
						}
						e.ReversalBars = k - j
						break
					}
					if bars[k].Close < e.Lo {
						e.Outcome = "break"
						if e.Side == "short" {
							e.Outcome = "hold"
						}
						e.ReversalBars = k - j
						break
					}
				}
				if err := enc.Encode(e); err != nil {
					panic(err)
				}
				n++
			}
			if reads%250 == 0 {
				fmt.Printf("reads=%d events=%d day=%s\n", reads, n, r.Day)
			}
			if *limit > 0 && reads >= *limit {
				fmt.Printf("SMOKE reads=%d events=%d\n", reads, n)
				return
			}
		}
	}
	fmt.Printf("DONE reads=%d events=%d\n", reads, n)
}
