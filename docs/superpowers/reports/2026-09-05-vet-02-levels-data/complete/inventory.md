| Kind | Definition and window · source | Grade/seat terms | Pool seats/candidates: rate [Wilson95] | First-exposure forward: touched/exposures; H/B/A/censored; hold & break [Wilson95]; ambiguous share [Wilson95] |
|---|---|---|---|---|
| PDH | prior CT calendar-day high; >=900 closed 1m bars; ring2000; `kernel/levels_multiday.go:146` | L1,T,H | 15/15=100.0% [79.6%,100.0%] | touch 1/1=100.0% [20.7%,100.0%]; 1/0/0/0; H 1/1=100.0% [20.7%,100.0%]; B 0/1=0.0% [0.0%,79.3%]; A 0/1=0.0% [0.0%,79.3%] |
| PDL | same bucket low; `kernel/levels_multiday.go:154` | L1,T,H | 8/8=100.0% [67.6%,100.0%] | touch 0/2=0.0% [0.0%,65.8%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| PDC | same bucket final close; `kernel/levels_multiday.go:159` | L1,T,H | 15/15=100.0% [79.6%,100.0%] | touch 1/1=100.0% [20.7%,100.0%]; 1/0/0/0; H 1/1=100.0% [20.7%,100.0%]; B 0/1=0.0% [0.0%,79.3%]; A 0/1=0.0% [0.0%,79.3%] |
| RTH-H | prior calendar-day NY high; 08:30–14:45 CT registry; `kernel/levels_multiday.go:161` | L1,T | 13/15=86.7% [62.1%,96.3%] | touch 0/1=0.0% [0.0%,79.3%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| RTH-L | same NY-window low; `kernel/levels_multiday.go:164` | L1,T | 11/11=100.0% [74.1%,100.0%] | touch 0/1=0.0% [0.0%,79.3%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| ONH | current CME-day AS+London high, develops until08:30; `kernel/levels_multiday.go:183` | L.85,T | 15/15=100.0% [79.6%,100.0%] | touch 1/3=33.3% [6.1%,79.2%]; 1/0/0/0; H 1/1=100.0% [20.7%,100.0%]; B 0/1=0.0% [0.0%,79.3%]; A 0/1=0.0% [0.0%,79.3%] |
| ONL | same overnight low; `kernel/levels_multiday.go:188` | L.85,T | 15/15=100.0% [79.6%,100.0%] | touch 0/1=0.0% [0.0%,79.3%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| AS-H | current AS 17:00–02:00 high; `kernel/levels_multiday.go:171` | L.70 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| AS-L | same AS low; `kernel/levels_multiday.go:174` | L.70 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| LDN-H | current London02:00–08:30 high; `kernel/levels_multiday.go:177` | L.70 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| LDN-L | same London low; `kernel/levels_multiday.go:180` | L.70 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| PWH | prior CT week high; guard4320 bars impossible on2000 ring; `kernel/levels_multiday.go:198` | L1,H,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| PWL | same prior-week low; `kernel/levels_multiday.go:202` | L1,H,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| PMH | prior month high; guard10080 bars; `kernel/levels_multiday.go:205` | L1,H,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| PML | same month low; `kernel/levels_multiday.go:209` | L1,H,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| OR-H | first5min high; DEVELOPING emitted before08:35; `kernel/levels_intraday.go:143` | L.70,T | 8/8=100.0% [67.6%,100.0%] | touch 2/2=100.0% [34.2%,100.0%]; 1/1/0/0; H 1/2=50.0% [9.5%,90.5%]; B 1/2=50.0% [9.5%,90.5%]; A 0/2=0.0% [0.0%,65.8%] |
| OR-L | same first5min low; `kernel/levels_intraday.go:166` | L.70,T | 8/8=100.0% [67.6%,100.0%] | touch 0/1=0.0% [0.0%,79.3%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| IB-H | first60min high plus1.5x/2x extensions; DEVELOPING before09:30; `kernel/levels_intraday.go:172` | L.70 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| IB-L | same low and lower extensions; `kernel/levels_intraday.go:179` | L.70 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| VWAP | 17:00 CME anchor; >=2 closed bars, typical-price volume weighting; ±1σ same kind; `kernel/levels_volume.go:35` | L.90,V | 15/15=100.0% [79.6%,100.0%] | touch 5/14=35.7% [16.3%,61.2%]; 1/4/0/0; H 1/5=20.0% [3.6%,62.4%]; B 4/5=80.0% [37.6%,96.4%]; A 0/5=0.0% [0.0%,43.4%] |
| VWAP±2σ | same developing VWAP±2σ; `kernel/levels_volume.go:58` | L.85,V | 4/5=80.0% [37.6%,96.4%] | touch 3/5=60.0% [23.1%,88.2%]; 0/0/3/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A 3/3=100.0% [43.9%,100.0%] |
| eVWAP | last15:00 CT anchor; evolving; `kernel/levels_volume.go:97` | L.85,V | 2/2=100.0% [34.2%,100.0%] | touch 0/2=0.0% [0.0%,65.8%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| pdVWAP | previous CME24h bucket; >=2bars; `kernel/levels_volume.go:318` | L.85,V | 0/1=0.0% [0.0%,79.3%] | touch 0/1=0.0% [0.0%,79.3%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| POC | prior CME-day120-bin CLOSE-bin volume proxy max; `kernel/levels_volume.go:162` | L.90,V | 1/1=100.0% [20.7%,100.0%] | No eligible forward observation; UNMEASURABLE |
| VAH | upper boundary70% proxy value area; `kernel/levels_volume.go:206` | L.80,V,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| VAL | lower boundary70% proxy value area; `kernel/levels_volume.go:206` | L.80,V,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| nPOC | untouched prior POC; up to10sessions, ring+durable extras; `kernel/levels_volume.go:257` | L.85,V,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| SETT | prior24h final available1m close; NOT verified official settlement; `kernel/levels_volume.go:342` | L.80,V,t | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| MID-O | current overnight midpoint; `kernel/levels_volume.go:366` | L.60,V | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| RN | 100/50/25 multiples inside proximity band; `kernel/levels_intraday.go:21` | L.55 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| GAP | unfilled1m gap edge >=ATR14 multiplier1; `kernel/levels_intraday.go:57` | L.55 | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| EQH | k2 pivot clusters3ticks; HTF tolerance max3ticks,.15TFATR; `kernel/levels_zones.go:34` | L.70,H if tagged | 0/4=0.0% [0.0%,49.0%] | touch 1/2=50.0% [9.5%,90.5%]; 0/1/0/0; H 0/1=0.0% [0.0%,79.3%]; B 1/1=100.0% [20.7%,100.0%]; A 0/1=0.0% [0.0%,79.3%] |
| EQL | same pivot-low clusters; `kernel/levels_zones.go:34` | L.70,H if tagged | 1/2=50.0% [9.5%,90.5%] | touch 2/2=100.0% [34.2%,100.0%]; 0/1/0/1; H 0/1=0.0% [0.0%,79.3%]; B 1/1=100.0% [20.7%,100.0%]; A 0/2=0.0% [0.0%,65.8%] |
| SWG-H | recent5m/15m fractal swings; k/minmove; lookbacks144/96;3perTF/side; `kernel/levels_swing.go:38` | L.85,V | 6/6=100.0% [61.0%,100.0%] | touch 0/3=0.0% [0.0%,56.1%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| SWG-L | same swing lows; `kernel/levels_swing.go:38` | L.85,V | 5/7=71.4% [35.9%,91.8%] | touch 1/4=25.0% [4.6%,69.9%]; 0/1/0/0; H 0/1=0.0% [0.0%,79.3%]; B 1/1=100.0% [20.7%,100.0%]; A 0/1=0.0% [0.0%,79.3%] |
| SUPPLY | base<=6bars, bodies<=.5ATR, departure>=1.5ATR;1m and configured HTF; `kernel/levels_zones.go:103` | Z | 5/23=21.7% [9.7%,41.9%] | touch 1/5=20.0% [3.6%,62.4%]; 0/1/0/0; H 0/1=0.0% [0.0%,79.3%]; B 1/1=100.0% [20.7%,100.0%]; A 0/1=0.0% [0.0%,79.3%] |
| DEMAND | same demand base; confirmed departure birth; `kernel/levels_zones.go:147` | Z | 22/59=37.3% [26.1%,50.0%] | touch 2/9=22.2% [6.3%,54.7%]; 0/0/2/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A 2/2=100.0% [34.2%,100.0%] |
| FVG | 3bar imbalance;1m floor max2ticks,2pt; HTF calls gap floor=TFATR; `kernel/levels_zones.go:213` | Z | 2/8=25.0% [7.1%,59.1%] | touch 0/1=0.0% [0.0%,79.3%]; 0/0/0/0; H UNMEASURABLE n=0; B UNMEASURABLE n=0; A UNMEASURABLE n=0 |
| IFVG | inverse filled FVG, same detector; excluded separate HTFzone section; `kernel/levels_zones.go:260` | Z | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
| OB | opposing candle within8bars before displacement;1m/configuredHTF; `kernel/levels_zones.go:296` | Z_OB | 8/117=6.8% [3.5%,12.9%] | touch 2/21=9.5% [2.7%,28.9%]; 1/0/1/0; H 1/1=100.0% [20.7%,100.0%]; B 0/1=0.0% [0.0%,79.3%]; A 1/2=50.0% [9.5%,90.5%] |
| OWNER | active manually-set sticky level; appended after cap; `trader/auto_trader_planner.go:2185` | A fixed | UNMEASURABLE n=0 | No eligible forward observation; UNMEASURABLE |
