// Offline necessary-condition replay. No store.New, database, broker or engine.
// Runs pinned kernel functions over closed retained minute prefixes. Checkpoints
// are measurable alternatives, NOT claims about historical callbacks or fills.
package main
import("encoding/json";"encoding/csv";"os";"fmt";"strconv";"sort";"strings";"math";"nofx/kernel";"nofx/market")
type Life struct { Time int64 `json:"time_ms"`; Date string `json:"date"`; Session string `json:"session"`; Version int `json:"version"`; Event string `json:"event"` }
type Plan struct { PlanID string `json:"plan_id"`; Version int `json:"version"`; Start int64 `json:"start"`; End int64 `json:"end"`; Born int64 `json:"born"`; Doc kernel.PlanDoc `json:"doc"` }
func must(e error){if e!=nil{panic(e)}}
func main(){
 root:=os.Args[1]; bb,e:=os.ReadFile(root+"/plans.json");must(e);var pp []Plan;must(json.Unmarshal(bb,&pp));lb,e:=os.ReadFile(root+"/life_events.json");must(e);var life []Life;must(json.Unmarshal(lb,&life))
 f,e:=os.Open(root+"/bars.csv");must(e);rr,e:=csv.NewReader(f).ReadAll();must(e); var bars []market.Kline
 for _,r:=range rr[1:]{t,_:=strconv.ParseInt(r[3],10,64);v:=make([]float64,5);for i:=0;i<5;i++{v[i],_=strconv.ParseFloat(r[4+i],64)};bars=append(bars,market.Kline{OpenTime:t,CloseTime:t+59999,Open:v[0],High:v[1],Low:v[2],Close:v[3],Volume:v[4]})}
 t10:=int64(1788447600000);j10:=sort.Search(len(bars),func(i int)bool{return bars[i].CloseTime>=t10});i10:=j10-2000;if i10<0{i10=0};a10:=market.ExportCalculateATR(kernel.AcceptanceBars(bars[i10:j10],"2x5m"),14);aout,_:=json.MarshalIndent(map[string]interface{}{"time_ms":t10,"bars":j10-i10,"atr5m":a10,"floor":1.5*a10,"source":"trader/entry_gate.go:332"},"","  ");must(os.WriteFile(root+"/q2_atr.json",aout,0644))
 out,e:=os.Create(root+"/replay_checkpoints.csv");must(e);w:=csv.NewWriter(out);defer w.Flush();w.Write([]string{"opportunity","leg","time_ms","price","atr5m","confirm","invalidated","stop_composed","rr","kind","actual_guard","corrected_guard","status","bar_open_ms"})
 static,e:=os.Create(root+"/replay_static.csv");must(e);sw:=csv.NewWriter(static);defer sw.Flush();sw.Write([]string{"opportunity","leg","status","reason","kind","entry","authored_stop","target"})
 for _,p:=range pp {for _,sc:=range p.Doc.Scenarios{
  key:=fmt.Sprintf("%s/v%d/%s",p.PlanID,p.Version,sc.ID)
  if sc.Arm==nil||!sc.Arm.Enabled {sw.Write([]string{key,"","disabled","stored arm is absent or disabled",kernel.ArmKindFor(sc.Condition),"","",""});continue}
  legs:=sc.Arm.Legs;if len(legs)==0{legs=[]kernel.PlanArmLeg{{Entry:sc.Arm.Entry,Stop:sc.Arm.Stop,Target:sc.Arm.Target,WaitConfirm:sc.Arm.WaitConfirm,Kind:kernel.ArmKindFor(sc.Condition)}}}
  for li,leg:=range legs {
   kind:=kernel.ArmKindFor(sc.Condition);status,reason:="checkpoint_evaluated",""
   if len(legs)>2{status,reason="refused","split_leg_capacity"} else if kernel.IsConditionShadowed(sc.Condition,nil,nil,""){status,reason="shadowed","condition_shadowed"} else if e:=kernel.ArmSpecValid(sc);e!=nil{status,reason="refused",e.Error()} else if e:=kernel.ArmKindMismatch(sc.Condition,leg.Kind);e!=nil{status,reason="refused",e.Error()} else if kind==""{status,reason="refused","no arm kind"}
   sw.Write([]string{key,strconv.Itoa(li),status,reason,kind,fmt.Sprint(leg.Entry),fmt.Sprint(leg.Stop),fmt.Sprint(leg.Target)})
   if status!="checkpoint_evaluated"{continue}
   for t:=((p.Start+59999)/60000)*60000;t<p.End;t+=60000{
    j:=sort.Search(len(bars),func(i int)bool{return bars[i].CloseTime>=t});if j==0||bars[j-1].CloseTime<t-60000{continue};i:=j-2000;if i<0{i=0};b:=bars[i:j];price:=b[len(b)-1].Close
    atr:=market.ExportCalculateATR(kernel.AcceptanceBars(b,"2x5m"),14)
    conf:=true;if leg.WaitConfirm{c:=sc.Confirm2;if len(sc.Arm.Legs)==0{c=sc.Confirm};conf=c!=nil;if c!=nil{conf=kernel.EvaluateConfirm(*c,b,p.Born,t).Met}}
    // dATR only changes activation proximity; invalidated branch precedes it.
    ev:=kernel.EvaluateScenario(sc,p.Doc.Levels,kernel.BarsSince(b,p.Born),price,0,kernel.ActivationWindowK,"5m_close",true,t)
    invalid:=ev.HasAnchor&&ev.Status==kernel.ScenarioInvalidated
    stop:=composeArmStop(sc.Direction,leg.Entry,leg.Stop,atr,.25,p.Doc.Levels,1.5,2,3).Stop
    risk,reward:=leg.Entry-stop,leg.Target-leg.Entry;if sc.Direction=="short"{risk,reward=stop-leg.Entry,leg.Entry-leg.Target};rrv:=0.;if risk>0{rrv=reward/risk}
    aGuard,cGuard:="outside_band","outside_band";trigger:=leg.Entry
    if kind=="stop_entry"{trigger-=.5;if sc.Direction=="long"{trigger=leg.Entry+.5};aGuard="send_stop_price_zero";cGuard="send_valid_stop";if limitMarketableWrongSide(price,trigger,sc.Direction){aGuard="cancel_wrong_side"};if (sc.Direction=="long"&&price>=trigger)||(sc.Direction=="short"&&price<=trigger){cGuard="cancel_already_through"}}else if limitMarketableWrongSide(price,trigger,sc.Direction){aGuard,cGuard="cancel_marketable","cancel_marketable"}else if math.Abs(price-trigger)<=25{aGuard,cGuard="send_limit","send_limit"}
    st:="passes_known_local_checks";if !conf{st="waiting_confirm"}else if rrv+1e-9<2{st="rr_refused"}else if risk+1e-9<1.5*atr{st="min_sl_refused"}else if invalid{st="invalidation_refused"}
    for _,le:=range life { if strings.HasPrefix(p.PlanID,le.Date+":"+le.Session+":") && le.Version==p.Version && le.Time<=t && le.Time>=p.Born && le.Event=="DORMANT" {st="recorded_dormant_requires_fresh_authorization"} }
    w.Write([]string{key,strconv.Itoa(li),fmt.Sprint(t),fmt.Sprint(price),fmt.Sprint(atr),fmt.Sprint(conf),fmt.Sprint(invalid),fmt.Sprint(stop),fmt.Sprint(rrv),kind,aGuard,cGuard,st,fmt.Sprint(b[len(b)-1].OpenTime)})
   }
  }
 }}
}

// Verbatim pure source from trader/arm_stop_anchor.go (pinned base).
type StopComposition struct {
	Stop         float64 // the chosen stop
	Authored     float64 // what the planner wrote
	AnchorPrice  float64 // the seated level the stop sits beyond (0 = none)
	AnchorLabel  string  // that level's provenance chip
	AnchorStop   float64 // anchor ± clearance (0 = none)
	ATRFloorStop float64 // entry ∓ mult×ATR5m (0 = no ATR)
	Bound        string  // anchor | atr_floor | authored — which one won
	Unanchored   bool    // no seated level within the dead-zone bound
}

// composeArmStop is the pure stop composition (fixture-tested).
//
//	side      long|short
//	entry     the arm's entry price
//	authored  the planner's stop (never tightened)
//	atr5m     ATR(14) on 5m; ≤0 → the ATR leg is skipped (fail-open)
//	tick      instrument tick size
//	levels    the plan's seated levels
//	mult      MIN_SL_ATR_MULT (resolved)
//	clearTicks the level-clearance leg (MinSLTickClearance)
//	maxAnchorATR the dead-zone bound in ATR units; ≤0 disables anchoring
func composeArmStop(side string, entry, authored, atr5m, tick float64, levels []kernel.PlanLevel, mult float64, clearTicks int, maxAnchorATR float64) StopComposition {
	c := StopComposition{Stop: authored, Authored: authored, Bound: "authored"}
	long := strings.EqualFold(strings.TrimSpace(side), "long")
	if entry <= 0 || authored <= 0 {
		return c // no usable geometry — leave the authored stop untouched
	}
	if tick <= 0 {
		tick = 0.25
	}
	clearance := float64(clearTicks) * tick

	// ATR floor leg.
	if atr5m > 0 && mult > 0 {
		if long {
			c.ATRFloorStop = entry - mult*atr5m
		} else {
			c.ATRFloorStop = entry + mult*atr5m
		}
	}

	// Anchor leg: the NEAREST seated level on the RISK side (below entry for a
	// long, above for a short), within the dead-zone bound.
	if maxAnchorATR > 0 {
		bound := math.MaxFloat64
		if atr5m > 0 {
			bound = maxAnchorATR * atr5m
		}
		best, bestDist, found := 0.0, math.MaxFloat64, false
		for _, l := range levels {
			if l.Price <= 0 {
				continue
			}
			var dist float64
			if long {
				if l.Price >= entry {
					continue // not on the risk side
				}
				dist = entry - l.Price
			} else {
				if l.Price <= entry {
					continue
				}
				dist = l.Price - entry
			}
			if dist > bound {
				continue // dead zone
			}
			if dist < bestDist {
				best, bestDist, found = l.Price, dist, true
				c.AnchorLabel = l.Label
			}
		}
		if found {
			c.AnchorPrice = best
			if long {
				c.AnchorStop = best - clearance
			} else {
				c.AnchorStop = best + clearance
			}
		} else {
			c.Unanchored = true
			c.AnchorLabel = ""
		}
	} else {
		c.Unanchored = true
	}

	// WIDEST WINS. For a long a wider stop is LOWER; for a short, HIGHER.
	pick := func(cand float64, name string) {
		if cand <= 0 {
			return
		}
		if (long && cand < c.Stop) || (!long && cand > c.Stop) {
			c.Stop, c.Bound = cand, name
		}
	}
	pick(c.AnchorStop, "anchor")
	pick(c.ATRFloorStop, "atr_floor")
	return c
}

func limitMarketableWrongSide(price, entry float64, side string) bool {
	if price <= 0 || entry <= 0 {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(side)) {
	case "long":
		return price < entry
	case "short":
		return price > entry
	}
	return false
}

// throughWord is the human word for "the market has traded through" per side.
