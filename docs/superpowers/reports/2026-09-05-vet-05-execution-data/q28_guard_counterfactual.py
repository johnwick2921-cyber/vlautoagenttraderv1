import sqlite3
c = sqlite3.connect("file:/home/hoang/nofx/data/data.db?mode=ro", uri=True)
arms = c.execute("""SELECT id, scenario, side, entry_px, stop_px, target_px, strftime('%s',updated_at)*1000 cancel_ms
  FROM armed_orders WHERE state_reason LIKE 'level accepted through%' ORDER BY id""").fetchall()
TICK=0.25
def sim(bars, side, fill, stop, tgt):
    """first-touch on 1m bars after fill bar; returns (outcome, pnl_pts, minutes). Same-bar both-touch = 'ambiguous'."""
    for i,b in enumerate(bars):
        o,h,l,cl = b[1],b[2],b[3],b[4]
        if side=='short':
            s_hit = h >= stop; t_hit = l <= tgt
        else:
            s_hit = l <= stop; t_hit = h >= tgt
        if s_hit and t_hit: return ('ambiguous', None, i)
        if s_hit: return ('stop', (fill-stop) if side=='short' else (stop-fill), i)
        if t_hit: return ('target', (fill-tgt) if side=='short' else (tgt-fill), i)
    last = bars[-1][4]
    return ('open@120m', (fill-last) if side=='short' else (last-fill), len(bars))
print("arm | side | A) limit-at-entry: fill_min outcome pnl_pts | B) bounded-market at close@cancel+1t: fill outcome pnl_pts | R(pts)")
totA=totB=0; nA=nB=0
for id_, sc, side, e, s, t, cms in arms:
    side=side.lower()
    bars = c.execute("""SELECT open_time_ms,o,h,l,c FROM bars WHERE tf='1m' AND symbol='MNQ'
        AND open_time_ms >= ? - 60000 AND open_time_ms < ? + 120*60000 ORDER BY open_time_ms""", (cms, cms)).fetchall()
    pre=[b for b in bars if b[0] <= cms]; p0 = pre[-1][4]
    post=[b for b in bars if b[0] > cms]
    # A) resting limit at entry: fills on first bar touching entry
    fa=None
    for i,b in enumerate(post):
        if (b[2] >= e) if side=='short' else (b[3] <= e): fa=i; break
    R = abs(e-s)
    if fa is None: A=('never filled',0,None)
    else: A=sim(post[fa+1:], side, e, s, t)
    # B) bounded market at cancel: fill = close@cancel one tick adverse
    fb = p0 - TICK if side=='short' else p0 + TICK
    B=sim(post, side, fb, s, t)
    print(f"{id_:>3} {sc} {side:5s} | A: {'—' if fa is None else str(fa)+'m'} {A[0]:10s} {('%+.2f'%A[1]) if A[1] is not None else 'n/a':>8} | B: fill {fb:.2f} {B[0]:10s} {('%+.2f'%B[1]) if B[1] is not None else 'n/a':>8} | R={R:.2f}")
    if A[1] is not None: totA+=A[1]; nA+=1
    if B[1] is not None: totB+=B[1]; nB+=1
print(f"TOTAL A (limit at entry, n={nA}): {totA:+.2f} pts = ${totA*2:+.0f} | TOTAL B (bounded market, n={nB}): {totB:+.2f} pts = ${totB*2:+.0f}")
