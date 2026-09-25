package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"nofx/kernel"
	"nofx/store"
	"nofx/trader"
	nt "nofx/trader/ninjatrader"
	"os"
)

type sweepRow struct {
	ID, Era, Session, Day, Cell, Fill, Reason, ConfiguredAdmission string
	Filled                                                         bool
	NetUpper                                                       float64
	Entry, Stop, Target, RiskUSD, Net                              float64
	Time, ExitTime                                                 int64
	Exit                                                           string
	FillBarAmbiguous                                               bool
}

func runGeometrySweep(path string) {
	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	out, err := os.Create("data/structural-stop/sweep.jsonl")
	if err != nil {
		panic(err)
	}
	defer out.Close()
	w := bufio.NewWriterSize(out, 1<<20)
	defer w.Flush()
	enc := json.NewEncoder(w)
	dec := json.NewDecoder(bufio.NewReaderSize(f, 1<<20))
	n := 0
	for {
		var e event
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		var readIndex, zoneIndex int
		if _, err := fmt.Sscanf(e.ID, "read-%06d-zone-%04d", &readIndex, &zoneIndex); err != nil {
			panic(err)
		}
		for _, cell := range []struct {
			name   string
			buffer float64
		}{{"legacy_corrected", 0}, {"p50_p75_tick_floor", .25}, {"p90", 1.25}, {"p95", 4.5}} {
			entry := nt.RoundToTick(e.Entry, .25)
			stop, target, reason, admission := 0.0, 0.0, "", ""
			if cell.buffer == 0 {
				others := []kernel.PlanLevel{}
				for _, z := range e.Zones {
					if z.Anchor != e.Entry && z.Lo != nil {
						others = append(others, kernel.PlanLevel{Price: z.Anchor})
					}
					if z.Lo == nil {
						continue
					}
					a := z.Anchor
					if e.Side == "long" && a > entry && (target == 0 || a < target) {
						target = a
					}
					if e.Side == "short" && a < entry && (target == 0 || a > target) {
						target = a
					}
				}
				authored := e.Lo - .5
				if e.Side == "short" {
					authored = e.Hi + .5
				}
				stop = composeArmStop(e.Side, entry, authored, e.ATR, .25, others, 1.5, 2, 3).Stop
				stop = nt.RoundToTick(stop, .25)
				target = nt.RoundToTick(target, .25)
			} else {
				p := store.StructuralStopPolicy{BufferPoints: cell.buffer, BufferKnown: true, CostPoints: 2, CostKnown: true, MinRR: 2, BufferSource: "C5 training grid", Calibration: store.StructuralBufferCalibration}
				g := trader.ComposeFrozenLevelFadeGeometry(e.Zones, zoneIndex, e.Side, entry, p, e.ATR, .25, 2)
				if g.Stop != nil {
					stop = *g.Stop
				}
				if g.Target != nil {
					target = *g.Target
				}
				reason, admission = g.Reason, g.Reason
				// Owner has requested a configurable cap but supplied no value. This
				// separately labeled diagnostic tests GEOMETRY before that cap. It does
				// not treat absent risk permission as actual admission.
				if reason == "risk_cap_missing" {
					reason = ""
				}
			}
			for _, fill := range []string{"A_touch", "B_through", "C_adverse_tick"} {
				row := sweepRow{ID: e.ID, Era: e.Era, Session: e.Session, Day: e.Day, Cell: cell.name, Fill: fill, Entry: entry, Stop: stop, Target: target, Reason: reason, ConfiguredAdmission: admission, Time: e.Time, RiskUSD: (math.Abs(entry-stop) + 2) * 2}
				if reason == "" {
					simulateCorrected(e, &row)
				}
				if err := enc.Encode(row); err != nil {
					panic(err)
				}
			}
		}
		n++
		if n%1000 == 0 {
			fmt.Printf("swept events=%d\n", n)
		}
	}
	fmt.Printf("SWEEP DONE events=%d cells=4 fills=3; structural admission still requires owner cap\n", n)
}

// First-touch minute is the common opportunity window (same B window as the
// original harness); unfilled opportunities are not assumed filled later.
// Minute OHLC cannot resolve ordering or queue position: stop-first bounds,
// and fill-bar targets need a closing price beyond target to prove post-fill
// reach. Results are conservative proxies, not reconstructed tick executions.
func simulateCorrected(e event, r *sweepRow) {
	if len(e.Bars) == 0 {
		r.Reason = "no_tape"
		return
	}
	long := e.Side == "long"
	b := e.Bars[0]
	through := 0.0
	if r.Fill == "B_through" {
		through = .25
	}
	filled := b.Low <= r.Entry-through
	if !long {
		filled = b.High >= r.Entry+through
	}
	if !filled {
		r.Reason = "unfilled"
		return
	}
	r.Filled = true
	actualEntry := r.Entry
	if r.Fill == "C_adverse_tick" {
		actualEntry += .25
		if !long {
			actualEntry = r.Entry - .25
		}
	}
	r.Entry = actualEntry
	r.RiskUSD = (math.Abs(actualEntry-r.Stop) + 2) * 2
	exit := e.Bars[len(e.Bars)-1].Close
	upperPossible := false
	r.Exit = "flat"
	r.ExitTime = e.Bars[len(e.Bars)-1].OpenTime
	for i, b := range e.Bars {
		stopHit, targetHit := b.Low <= r.Stop, r.Target > 0 && b.High >= r.Target+through
		if !long {
			stopHit = b.High >= r.Stop
			targetHit = r.Target > 0 && b.Low <= r.Target-through
		}
		if stopHit && targetHit {
			r.FillBarAmbiguous = true
			upperPossible = true
		}
		if stopHit {
			exit = r.Stop
			if long && b.Open < exit {
				exit = b.Open
			}
			if !long && b.Open > exit {
				exit = b.Open
			}
			r.Exit = "stop"
			r.ExitTime = b.OpenTime
			break
		}
		if i == 0 && targetHit {
			proved := b.Close >= r.Target+through
			if !long {
				proved = b.Close <= r.Target-through
			}
			if !proved {
				r.FillBarAmbiguous = true
				upperPossible = true
				continue
			}
		}
		if targetHit {
			exit = r.Target
			r.Exit = "target"
			r.ExitTime = b.OpenTime
			break
		}
	}
	gross := exit - actualEntry
	if !long {
		gross = actualEntry - exit
	}
	r.Net = gross - 2
	r.NetUpper = r.Net
	if upperPossible && r.Target > 0 {
		upper := r.Target - actualEntry - 2
		if !long {
			upper = actualEntry - r.Target - 2
		}
		r.NetUpper = math.Max(r.Net, upper)
	}
}
