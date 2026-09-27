# ADVERSARIAL REFUTATION — A2 "Killzone weighting" non-conformance

Verifier pass at deployed rev 70af663d (worktree files byte-identical to 70af663d:
`git diff --stat 70af663d..HEAD -- kernel/planner_prompt.go kernel/no_trade_band.go
kernel/session_registry.go kernel/adherence.go` = empty).

## VERDICT: REFUTED (as a conformance NO). A different, larger defect is confirmed in its place.

### What holds [A]
- file:line `kernel/planner_prompt.go:653-656` — correct (code comment there calls it **A4**, not A2; A2 in code is the sweep-chain block at :625).
- resolved string — reproduced independently by rendering the builder, not by reading a default:
  `go test ./kernel/ -run TestSamplePlannerPrompt -v` →
  `NY AM 07:30–10:00 CT is the primary window; 09:00–10:00 CT is the premium FVG window`
  `ETtoCT` = fixed −60 min (`kernel/no_trade_band.go:243-249`, via `HHMM` :75).

### What fails
1. **LABEL [I] is contradicted by the peer's own citation.** `2026-08-25-1h-timeframe-research-wave.md:99`
   (7ce2f772) states "NY Open 8:30–11 AM ET" — the prompt renders exactly that window.
   Per the legend that is **[R]**. The census's [I] footnote is narrower than they quoted:
   `2026-09-02-belief-census.md:120` — "The premium-FVG-window and Monday/Thursday conviction
   lines (A2/A3) carry no citation anywhere in docs/". Only the **premium** half is [I].
   Split label: primary=[R] (:99, contradicted by :91 = "NY 07:00-09:00/10:00 ET"), premium=[I].
   Not cited by the peer: `2026-08-30-knob-census.md:144` (741bfc2a) labels the same pair **[O] advisory**.
   Nearest premium-window origin found: `docs/superpowers/plans/2026-05-22-...md:3844`
   ("TTrades AM Silver Bullet, 10:00-11:00 ET") — in a paragraph saying it *cannot be validated*.
2. **CONFORMS=NO rests on an invented research value.** The research value is 8:30–11 AM ET;
   resolved is 07:30–10:00 CT ≡ 08:30–11:00 ET. Printed beside each other they are IDENTICAL.
   The peer substituted a *session boundary* (NY 08:30→14:45 CT) for the research value.
   → primary window CONFORMS = **yes**.
3. **EFFECT citation truncated.** `2026-08-18-re-audit.md:54` (995319a6) reads in full:
   "**There is NO killzone gate** (killzones feed adherence facts only)". Killzones DO carry a
   live effect: `kernel/adherence.go:76` steps the grade down one letter on `!InKillzone`.
4. **Caller count wrong in kind.** `planner_prompt.go:654` is the rule's own print site, not a caller.
   Sole production caller of `BuildPlannerPrompt`: **`trader/auto_trader_planner.go:907`**
   (30 other hits are `_test.go`). And a SECOND production surface was missed entirely.

### The real defect [A] — TWO NY-AM killzones, one hour apart, both live
| surface | window | source | effect |
|---|---|---|---|
| prompt advisory | **07:30–10:00 CT** | `kernel/planner_prompt.go:654-655` | advisory |
| machine grader | **08:30–11:00 CT** (`ny_am`) | live DB `system_config.session_registry` (RESOLVED, not the `session_registry.go:110-113` default) | grade step-down |

Grader chain: `session_registry.go:252 InKillzone` → `kernel/adherence.go:120 SessionWindowFacts`
→ `kernel/adherence.go:76` → `trader/auto_trader_clock.go:787`.

Measured (n=587 `trader_positions`, 2026-05-31 → 2026-09-03):
07:30–08:30 CT **n=27** · 08:30–10:00 CT n=51 · 10:00–11:00 CT n=27. Graded rows n=70 (21 IN a
registry killzone, 49 OUT). Two rows — **588** (2026-09-02 07:41:05 CT) and **558**
(2026-08-25 07:56:41 CT) — were cited+matched with `plan_band=''` (base **A**) and stored **B**:
one step-down, and neither is in lunch or a first-5m window, so the cause is the killzone penalty.
Both sat INSIDE the prompt's advertised primary window and OUTSIDE the window the grader scores.

The peer's premise ("that hour is unreachable, it is LONDON") is also wrong on the tape: 27 entries
exist there. Going forward LONDON is `enabled=false` in the live registry — so the hour is shut by
the London disable, not by the NY window they cited.
