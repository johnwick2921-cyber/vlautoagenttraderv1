import math
def wilson(k,n,z=1.96):
    if n==0: return (0,0,0)
    p=k/n; d=1+z*z/n; c=p+z*z/(2*n); h=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))
    return p,(c-h)/d,(c+h)/d
rows=[
("arm-legs authored in window that ever confirmed (touch/close) while their version lived",18,63),
("arm-legs filled / authored (Run A, replay_out.txt)",3,63),
("arm-legs filled / confirmed (Run A)",3,18),
("live ledger arms filled / rows, ids 29-37",1,9),
("live ledger arms filled / rows ids 5-37 excl test seams",8,28),
("live ledger arms marketable-cancelled ids 23-37 (Part D's window)",5,15),
("live ledger arms marketable-cancelled ids 5-37 excl test seams",6,28),
("live ledger arms marketable-cancelled ids 29-37",3,9),
("counterfactual reclaim stop-entries passing R:R>=2 on target_chain[0] (Run B, q35)",4,24),
("counterfactual reclaim stop-entries passing R:R>=2 on target_chain[-1] (q35)",12,24),
("counterfactual reclaim stop-entries filled (Run B)",0,24),
("decision-path open intents executed 09-02..09-04",4,24),
("decision-path open intents refused by strict 09-03",13,13),
("RTH-L break / (hold+break) all-time touch_outcomes",82,124),
("RTH-L break / (hold+break) LONDON window 09-02..09-04",18,26),
("win rate era usable rows (wins/(wins+losses))",21,63),
("MFE>=2R era rows with recoverable stop",13,55),
("long scenarios arm-enabled 09-03",2,28),
("short scenarios arm-enabled 09-03",8,20),
("long scenarios arm-enabled 09-02",9,64),
("short scenarios arm-enabled 09-02",27,43),
("09-03 decisions proposing open_long / decisions",0,596),
("stopped within 9 pts of the turn (two-day audit) 3/5",3,5),
("stop-entry submissions on 09-04 with Stop price=0 at NT8 (q37)",21,21),
]
for name,k,n in rows:
    p,lo,hi=wilson(k,n)
    print(f"{name}: {k}/{n} = {p:.3f} Wilson95 [{lo:.3f}, {hi:.3f}]")
