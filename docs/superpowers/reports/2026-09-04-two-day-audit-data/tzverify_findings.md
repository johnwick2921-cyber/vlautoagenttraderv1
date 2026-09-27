# Adversarial verification — "plans.created_at UTC vs plan_lifecycle_log.at CT"
Verified 2026-09-04 (CT). Read-only, DB opened mode=ro. Worktree /home/hoang/nofx-2day04
HEAD 8579df0a (dfbfa660 is an ancestor; the diff is audit CSVs only, no Go changes).

## Verdict: PLAUSIBLE — observation reproduces exactly, scope + reasoning are wrong.

## 1. The observation reproduces (with the n they omitted)
    sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
      "SELECT substr(created_at,-6) off, COUNT(*) n FROM plans GROUP BY 1;"
      -> +00:00  n=243
    sqlite3 ... "SELECT substr(at,-6) off, COUNT(*) n FROM plan_lifecycle_log GROUP BY 1;"
      -> -05:00  n=2      <-- the whole table is 2 rows
Both are genuine instants (cross-checked: plan v2 created 18:47:53 CT, dormant 19:25:00 CT,
v3 authored 19:28:36 CT — coherent only under the labelled offsets). Nothing is mislabelled.
Their substr(...,20) also captures fractional seconds, not just the offset.

## 2. It is NOT a table property. It is a WRITER property.
    store/gorm.go:26   NowFunc: func() time.Time { return time.Now().UTC() }   -> UTC
    store/plan.go:59   CreatedAt time.Time `gorm:"...autoCreateTime"`          -> UTC (n=243)
    store/plan.go:306  At: time.Now()   (explicit, LOCAL)                      -> CT  (n=2)
GORM's NowFunc fills CreatedAt/UpdatedAt ONLY when the field is zero; any caller that
assigns time.Now() itself wins and writes local CT.

## 3. The claim's framing is contradicted inside its own subsystem
plan_overlays is the THIRD plans-subsystem table. Its single created_at column holds BOTH:
    2026-08-16:NY v2 ov1  2026-08-16 00:12:57.848033595-05:00
    2026-08-17:NY v1 ov1  2026-08-17 14:52:46.24319788 +00:00   (n=2, one each)

## 4. The real defect the claim misses: mixing INSIDE one column / one row
    armed_orders.created_at   n=37   +00:00 n=6 (ids 1-4,15,16) / -05:00 n=31 (ids 5-14,17-37)
    ab_confirm_log.updated_at n=194  +00:00 n=188 / -05:00 n=6
    armed_orders: 28 of 37 rows carry DIFFERENT offsets in created_at vs updated_at:
      ids 5,6,7,8,9,10,11,12,14,17,18,19,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37
Cause (both verified): trader/armed_executor.go:202,692 `now := time.Now()` (LOCAL) is
assigned to CreatedAt/UpdatedAt on the insert/upsert path (armed_executor.go:349,518,712;
store/armed_orders.go:233), while every state transition uses
.Model(&ArmedOrderDB{}).Where("id = ?", id).Updates(...) (store/armed_orders.go:265,273,
280,296,322) which lets GORM stamp updated_at through NowFunc -> UTC.

### 4a. Offset-blind duration overstates arm lifetime by exactly 300 min
    id  created_at(raw)              updated_at(raw)              naive_min  true_min
    32  09-02 10:30:30-05:00         09-02 19:45:01+00:00              554       254
    33  09-02 14:10:29-05:00         09-02 19:10:29+00:00              300         0
    34  09-02 22:15:01-05:00         09-03 03:15:01+00:00              299         0
    35  09-03 09:02:54-05:00         09-03 15:28:29+00:00              385        85
    36  09-03 09:20:47-05:00         09-03 14:20:47+00:00              300         0
    37  09-03 11:58:33-05:00         09-03 17:15:01+00:00              316        16
(ids 32-37 are exactly the two-day audit window.)

### 4b. A REAL lexical ORDER BY inversion exists today
store/armed_orders.go:316 ListFilled uses Order("updated_at DESC") on the mixed column.
    lexical order : 18(13:17:51Z) 17(13:15:51Z) 20(18:09:28Z) 21(17:25:29Z)
    true order    : 20 21 18 17
CAVEAT [A]: 17/18 are 'cancelled' and 20/21 are 'filled', and ListFilled filters
state='filled', so this inversion does NOT currently reach that reader. The hazard is
LATENT, not firing. plans.created_at is uniformly +00:00 (n=243) so plan.go:416/429
Order("created_at DESC") is safe today.

## 5. The named pair has no consumer
LifecycleLog (store/plan.go:542) orders by `id`, not by `at`. grep finds NO query that
joins or compares plans.created_at to plan_lifecycle_log.at. As stated, the peer's pair
is the least harmful instance of the family.
