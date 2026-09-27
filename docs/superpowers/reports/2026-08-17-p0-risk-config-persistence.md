# NO P0 — the gates were running the owner's values all along; my audit query read the wrong path

**Root cause, one sentence:** `risk_control` is nested under `ai_config` (exactly where the hand-rolled codec writes it), my audit query read the **top level**, found nothing, defaulted the miss to `{}`, and I reported the gates as running R:R 1.0 / confidence 50 / cap 10 / hold-lock OFF — when they were running **3.0 / 65 / 2 / ON** the entire time.

## Phase 1 — what is actually true

**Live config** (`data/data.db`, read-only), both running strategies:
`ai_config.risk_control` = `min_risk_reward_ratio: 3` · `min_confidence: 65` · `max_contracts_per_order: 2` · `max_contracts_enabled: true` · `hold_discipline: true` · `breakeven_enabled: true` · `guardrails_enabled: false`.

**Effective gate values**, proven by loading the real stored JSON through the production codec and reading what the gate reads after `ClampLimits()`:

```
AFTER Unmarshal:  minRR=3.00 minConf=65 maxContracts=2 holdLock=true breakeven=true
AFTER ClampLimits: minRR=3.00 minConf=65 maxContracts=2
AFTER round-trip:  minRR=3.00 minConf=65 maxContracts=2 holdLock=true
guardrails master: false   (untouched — the owner's dated decision)
```

**Why "empty" (item 2): (e) it lives elsewhere and my query was wrong.** Not (a)–(d). The codec carries `risk_control` inside `ai_config` on **both** halves — `store/strategy.go:763-769` (Marshal) and `:806-811` (Unmarshal). `hold_discipline` maps correctly to `HoldDisciplineEnabled` (`strategy.go:1233`). Nothing is dropped, stripped, or overwritten.

**Three-way divergence check (item 3) — NO DIVERGENCE.** For the newest stored decision of the day-plan trader:

| Source | R:R | Confidence |
|---|---|---|
| DB (`ai_config.risk_control`) | 3 | 65 |
| **PROMPT** (the stored `system_prompt` the AI actually received) | *"reward must be at least **3.00x** the risk"* | *"Min confidence to open: **65**"* · *"Confidence ≥ 65 required"* |
| GATE (effective, post-clamp) | 3.00 | 65 |

All three agree. **The AI is not being lied to.**

**UI source (item 4):** the same one. The Studio reads the strategy row's `ai_config.risk_control`; there is no second source. The 65 the owner has been seeing was correct — it was my report that disagreed with reality.

**History (item 5):** no regression commit. The values have persisted correctly; `git log` shows no change to the codec's risk path.

**Blast radius (item 6)** — all 9 strategies, correct paths:

| | R:R | conf | cap | hold | guardrails | day_plan |
|---|---|---|---|---|---|---|
| **a5b7662e** (hoang, running) | **3** | **65** | **2** | **true** | false | on |
| **70695b25** (15m, running) | **3** | **65** | **2** | **true** | false | on |
| 4104ca0a / 82b3d482 / a79ba71f / 08539e82 | 3 | 70–75 | – | – | – | on |
| 0362594e | 4 | 80 | – | – | – | on |
| *(blank id)* / 578ac8f6 | **0** | **0** | – | – | – | on |

Everything **PERSISTS**. The two `0/0` rows are not running; before this train they would have started at 1.0/50.

**Live impact (item 7): zero.** 2114 decisions in 7 days for the day-plan trader; the 500 most recent carry **no** `risk_check_error`, and journald shows no R:R or confidence refusals. The gates ran at 3.0/65 throughout, so there is no population of trades that "should have been refused".

## Phase 2 — what I actually changed

**The one real defect** (`7e5932fc`): `ClampLimits` treated *unset* and *explicitly low* identically — a zero was raised only to the **range floor** (R:R 1.0 / confidence 50), the loosest setting available, and the per-order cap fell back to **10**. So "never configured" was the most permissive configuration. Unset now resolves to the researched **3.0 / 65 / 2**; an explicit 1.5 still yields 1.5; out-of-range still clamps. **No effect on the live system** (both strategies set all three explicitly) — it closes the hole for the two `0/0` rows.

*Deliberately not changed:* the hold-lock default stays OFF. It is **not** futures-gated, so flipping it globally would alter crypto behavior and break the byte-identical invariant. Both live strategies set it `true` explicitly. Guardrails master untouched.

**Owner values applied (item 10)** via `cmd/dayplan-sessions` (`2c15f000`) — through the store layer, never raw SQL, dry-run by default, and it **refuses to loosen** without an explicit flag. Backup taken and integrity-checked first (`~/nofx-backups/pre-session-grade-fix/`, `quick_check = ok`).

| Setting | Before | After |
|---|---|---|
| R:R · confidence · cap · hold-lock | 3.0 · 65 · 2 · ON | **unchanged — already correct** |
| ASIA min_grade | B | **A** |
| ASIA max_trades | 3 | **1** |
| LONDON min_grade | B | **A** |
| NY, guardrails master | B/3, OFF | **untouched** (spec: "NY normal"; master is the owner's decision) |

Re-run reports `0 changes` — idempotent. Takes effect at the next trader reload, i.e. the owner's deploy.

**The regression test that ends this class** (`b4b28895`, `kernel/risk_config_truth_test.go`, 4/4 PASS): ① every one of 19 risk fields survives **both** codec halves ② `risk_control` lives under `ai_config` — *the test that would have caught my error* ③ unset never widens risk ④ **the rendered prompt's stated thresholds must equal the values the gate enforces**, checked at two different configured levels by scraping the real futures prompt.

**Prior reports corrected** (`a8c308ee`): the false headline is retracted at the top of both the CTO verification and checklist run 1, with the affected rows restated. An agent had reported `cap=2` correctly and I "corrected" it to 10 — that reversal is undone too.

## Exit bar

`go build` · `go vet` · `go test ./...` · **`-race` clean** (kernel/store/trader/api) · `tsc` · **goldens byte-identical** vs `909d3a48` · vitest 190/191 (the one failure and the `e2e/gate.spec.ts` collection error are the same pre-existing pair).

`TestFuturesOrderQuantity` needed updating: its sizing cases passed `maxFuturesContracts` as "a cap high enough not to interfere", silently coupling them to the default's value. Each case now carries its own cap. Sizing math unchanged.

**Item 13 (Playwright) NOT DONE:** the sandbox is unauthenticated and `httpClient` redirects to `/login` on 401, so a UI round-trip of a risk value cannot be driven headlessly. Unchanged from the last two runs; it stays owner-to-verify.

**Operational note worth keeping:** my first dry-run said "already correct" because I snapshotted with `cp data/data.db` — which omits the 3.5 MB `-wal` and silently loses every recent commit. Use `sqlite3 ".backup"` for any live snapshot, as `deploy/nofx-db-backup.sh` already does.

## Deploy (the only manual step)

```bash
cd /home/hoang/nofx && git pull
go build -o nofx-bin . && echo BUILD OK
git rev-parse HEAD > deploy/RELEASE     # MANDATORY — else the boot assertion refuses trading
sudo systemctl restart nofx
journalctl -u nofx --since '2 min ago' | grep 'BOOT INTEGRITY'
cd web && npm run build && cd ..        # then hard-reload
```

The restart is what makes the new ASIA/LONDON grades live (config is cached at trader-load).
