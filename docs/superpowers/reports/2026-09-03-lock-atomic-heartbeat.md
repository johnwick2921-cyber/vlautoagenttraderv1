# LOCK — ATOMIC CREATE + HEARTBEAT (checklist class 70)

**Branch:** `fix/lock-atomic-heartbeat`, built ON TOP of a peer lane's `ec2dd8f7`
(see §6 — two sessions were dispatched the same wave and both wrote it)
**Checklist:** entry **70** — number assigned AT MERGE (A16); highest occupied on dev was
**69**, nofx-b3's "reported wired, called by nobody", merged in that same boot.
**Scope:** ops surface only. No Go file, no binary, no cutover, no boot.

---

## 1. The defect, stated once

`~/nofx-main.lock` was **one flat file, written with `>`, carrying a pid, read with
`kill -0`**. On 2026-09-03 it failed in three distinct directions. Not one bug three times —
three different failure modes of the same design, in one day, on one machine.

| # | mode | what happened | who caught it |
|---|---|---|---|
| a | **dead pid, live owner** | agents wrote `pid=$$`; every tool call is a fresh shell, so the pid was dead within a second while its owner worked on | nofx-ed, who found an unexpired lock whose `kill -0` failed and correctly did **not** clear it |
| b | **live pid, silently replaced** | `>` truncates; a second acquirer clobbered an active cutover's lock with no error and no trace | nofx-b3, three minutes after I overwrote their lock |
| c | **stale pid after resume** | a session resumed under a new pid and wrote a lock naming its own former, now-dead process | nofx-b3, on themselves, by running `kill -0` against their own note before trusting it |

**Not one of the three was caught by the lock.** Every one was caught by a peer asking a
question the file could not answer. That is the finding, and it is why this is a class and
not a bug report.

### The fault underneath all three

A pid answers *"does some process exist"*. That was never the question. The question is
*"is the owner still working"* — and **only the owner can answer that**. Every pid-based
scheme is an attempt to infer an answer from a proxy that does not know it.

## 2. The fix

`deploy/nofx-lock.sh`. Two properties do the work:

**Atomic create.** `mkdir ~/nofx-main.lock.d` succeeds exactly once and fails if the
directory exists. Mode (b) is not *detected*, it is **unrepresentable** — a second acquire
cannot overwrite, only refuse, and it names the holder and task while refusing.

**Heartbeat, written by the holder.** `heartbeat=<ISO>`, rewritten every ~2 min as the
holder works. Stop working and it goes stale on its own. There is **no pid field at all**,
so (a) and (c) have nothing to record wrongly.

```
acquire <session> <task> [minutes]    atomic; refuses and names the holder
heartbeat <session>                   one beat; owner-scoped
release <session>                     owner-scoped
status                                free · held (fresh, Ns ago) · held (STALE)
check                                 rc 0 free · 1 held-fresh · 2 held-stale
with-heartbeat <session> -- <cmd>     beats for the lifetime of <cmd>, and no longer
```

### Three design choices worth defending

**STALE IS NOT DEAD, and the surface never says "dead".** A heartbeat older than 5 min
means the owner has not checked in. They may be inside a long build, or resumed, or gone.
The status line says so in those words and prints the corroboration list — ask the session
by name, watch whether HEAD moves, look for a build in flight. The script **never clears a
lock on its own**, at any age. A23's corroboration rule is not softened by having a
timestamp; the timestamp only makes the question sharper.

**The holder beats — there is no daemon.** A background beater that outlived its session
would report a dead owner as live, which is mode (a) rebuilt in a new costume. So
`with-heartbeat` spawns a beater bound to exactly one command and kills it when that
command returns. A pin asserts the beater does not outlive its job.

**Expiry is recorded but is not liveness.** It is the holder's own declaration of how long
they expect to need the tree. Liveness is the heartbeat. An unexpired lock with a stale
heartbeat is precisely the 0C case from the A2 amendment, and it now reads that way on its
face instead of requiring an agent to reason it out.

## 2b. Succession — `reclaim`, added by owner ruling after the first merge

The heartbeat makes an abandoned lock *visible*; it does not hand it over. `reclaim` does,
and only on the record:

```
nofx-lock reclaim <new> <stale> "<what you checked>"
```

- **Refused while the heartbeat is fresh**, without exception. A reclaim that can take a
  live lock is replacement with better manners — the exact failure the atomic create
  removed. The refusal quotes the age and tells you to wait or ask the holder to release.
- **You must name the holder you are taking over.** Naming the wrong session is refused, so
  a misread of `status` cannot become a silent seizure.
- **You must state the corroboration.** Empty is refused. The message names what counts:
  HEAD not moving, no build in flight, the session not answering.
- **It appends to `history`** — who took over, from whom, the heartbeat age, and the reason.
  `status` prints it, and `release` prints it once more before the directory goes, because
  the failure this exists for is invisible succession and a chain that vanishes silently
  would defeat the point.
- **rc 3, not acquire's 0.** A script can distinguish "took a free lock" from "inherited an
  abandoned one", so a lane can refuse to inherit. Suggested by nofx-b3 and adopted.

One peer suggestion was **not** taken: that reclaim "must refuse while the recorded pid is
alive". There is no recorded pid any more, and adding one back to gate reclaim would
reintroduce the class this wave removed. Staleness of the heartbeat is the whole test.

## 3. Tests — `deploy/nofx-lock-test.sh`, 56 assertions

| pins | first run |
|---|---|
| free lock reads free; rc 0 | RED |
| **second acquire FAILS**, refuses, names holder AND task | RED |
| **no `pid=` and no `PID` anywhere in the lock**; session/task/acquired/expiry/heartbeat all present | RED |
| fresh heartbeat reads held, never stale; `check` rc 1 | RED |
| **stale heartbeat says "stale", never "dead"** (case-insensitive), still names the holder, demands corroboration; `check` rc 2 | RED |
| beating clears stale; a foreign session may not beat | RED |
| release is owner-scoped; frees the lock; directory gone | RED |
| `with-heartbeat` refreshes a stale lock and its beater does not outlive the command | RED |
| source pin: the script cannot express `kill -0`, `pgrep`, `$$` | RED |

`pass=36 fail=0`.

### Three defects in my own tests, found by running them

1. **Three source pins passed vacuously** against a `nofx-lock.sh` that did not exist yet —
   a green that proves nothing, which is the exact failure these pins exist to catch. The
   harness now aborts if the script is missing.
2. **The stale pin was case-sensitive** and missed `STALE`. Both the stale and the
   never-says-dead assertions are case-insensitive now; the "dead" pin especially, since a
   future `Dead` would have slipped through.
3. **The source pin was reading the comment block**, which legitimately explains what
   `kill -0` used to do, so it failed on its own documentation. It strips comments now and
   reads code. I mutation-tested that fix — inserting a real `kill -0 1` into a code path
   makes the pin fail (`pass=35 fail=1`), so the strip did not hollow it out.

## 4. Surfaces updated

- `docs/superpowers/AUDIT-CHECKLIST.md` — class **70** (all three modes in ONE entry, with
  the fix and the probe); **class 13's probe** and **PART 3 step 1** both named
  `~/nofx-main.lock (owner/PID/expiry)` and now name the new shape and the heartbeat duty.
- `docs/superpowers/plans/2026-09-02-tree-guard-spec.md` — its expected-dirty suppression
  rule was literally `if ~/nofx-main.lock exists AND its pid is alive (kill -0)`. It now
  calls `nofx-lock.sh check` and distinguishes held-fresh (INFO) from held-stale (WARN,
  naming the session and the age) from no-lock (ALARM, the 08:46 signature). Its fixture
  list changed with it. The guard is still SPEC ONLY — this wave did not build it.
- **No deploy script referenced the lock**; it was a hand-written convention, which is part
  of why it drifted. `deploy/nofx-lock.sh` is now the one implementation.
- **No Guide change.** GUIDE CONTENT LAW covers knobs, plays, chips, gates and defaults of
  the running bot. This is an agent-ops surface; `web/src/guide/content/*` does not mention
  the lock and would be wrong to. Stated explicitly so the omission is a ruling and not an
  oversight.
- Root `CLAUDE.md` is **gitignored** (`.gitignore:8`), so it cannot ride this branch. Its
  MAIN-TREE LOCK LAW and LOCK LIVENESS AMENDMENT are edited in the main tree separately —
  under the new lock, which is also the first live use of it.

## 4b. What changing the lock broke, six minutes later

`ec2dd8f7` and my merge both edited `docs/superpowers/plans/2026-09-02-tree-guard-spec.md`.
The tree-guard wave was implementing that spec's expected-dirty rule at the same time, from
a worktree cut before either edit — so it built the **old** model: `~/nofx-main.lock`, a pid,
`kill -0`.

Under the new lock there is no legacy file. During a cutover that guard would have found
"no live holder", seen a legitimately dirty tree, and **ALARMED at precisely the moment it
exists to be trusted** — running and printing normally the whole time. It would have fired
on the next boot. A guard that cries wolf on every deploy is worse than no guard, because
the next real alarm is the one everybody scrolls past.

Found and fixed by nofx-ed at `ac345a7a`: the lock directory is authoritative, and the
legacy file is surfaced but never honoured for liveness — honouring it would restore the
exact `kill -0` test this wave removed. One of their tests asserted the old contract and was
migrated with its reason rather than deleted.

**The generalisable part:** a spec on `dev` is a moving artifact, and a worktree cut from an
older base silently freezes it. Two waves read and wrote the same file within six minutes
and neither could see the other. The probe costs nothing — `git log -1 -- <spec>` against
your worktree's base before building on it. It is the same family as class 70 itself: a
value read once, at a moment nobody recorded.

## 4c. The class-45 stash, disarmed (class 71)

Recorded here because it was found while checking this wave: `stash@{0}` held the class-45
VS Code revert — 127 insertions, **596 deletions** of shipped safety code — and `git stash`
is per-REPOSITORY, so any of this repo's 56 worktrees could pop it. nofx-47 did, by routine
stash/pop; three files applied CLEANLY and staged, deleting the class-33 boot sweep.

**The annotated tag `class45-found-revert-1203` is what makes a drop lossless** — it
references `6b770196`, so removing the stack entry cannot let gc collect the object. The
human-readable copy at `docs/superpowers/reports/class45-found-revert-1203.patch.txt` is a
convenience, not the guarantee.

I got that wrong once and it is worth keeping: I first committed the copy as `.patch`,
`.gitignore:143` (`*.patch`) silently swallowed it, and I reported both halves as landed
having verified neither. nofx-47 caught it by reading `dev` rather than my report. **A tool
that skips silently and a report that asserts success are the same failure twice** — the
rule this checklist already states as "read the value back out of the artifact".

**The stash entry was dropped 2026-09-03 on explicit owner authorisation**, after a
four-point pre-flight: one entry on the stack, its sha IS `6b770196`, the local and origin
tags both dereference to it, and the evidence file is present on `origin/dev`. Afterwards
`git stash list` is empty and `git cat-file -t 6b770196` still answers `commit` — the
landmine is gone and the evidence is not. Held for the owner rather than taken on a peer's
suggestion, because dropping another lane's evidence is destructive and was never mine to
decide.

## 5. What this does not fix

- **An owner who resumes under a new identity** still has to re-acquire or beat. The
  heartbeat makes that visible within 5 minutes instead of never, but it does not make the
  handover automatic.
- **A holder who beats while doing nothing useful** looks alive. Liveness is not progress.
  Corroboration by HEAD movement is what covers that, and it stays mandatory.
- **Nothing here governs editors.** The 2026-09-02 VS Code Save All that reverted 596 lines
  of shipped safety code was a non-agent writer; the lock governs agents and always did.
  That hole is the tree guard's job and the tree guard is still unbuilt.

## 6. Two lanes wrote this, and the branch keeps both

I finished my version and found the branch name already taken on the remote by `ec2dd8f7`,
an independent implementation of the same ruling — same atomic `mkdir`, same session
identity, same 2-min/5-min heartbeat, same STALE-not-DEAD wording. **I did not force-push
over it.** That commit is the base; mine is layered on top, and the shipped script takes the
better half of each:

**`ec2dd8f7` is nofx-47's**, confirmed by them 2026-09-03 after two wrong guesses. Nothing
of their wave was lost: the 36 lines of extra reclaim tests they had locally are covered by
the shipped suite (refused-on-fresh, rc nonzero, holder unchanged after refusal,
corroboration required once stale), which they ran themselves at `bd20be31` — 56 pass, 0
fail. **That it took three rounds of asking to establish is itself the finding.** I first
told nofx-b3 it was theirs. They corrected it with their own timeline — their boot marker
lands at 21:49:36, sixty seconds after `ec2dd8f7`, and at 21:48:12 they were reading a
boot-integrity line; nobody writes 150 lines of bash in that minute. **Every commit in this
repo carries the identical author identity (`johnwick2921-cyber`), so git cannot answer
"which lane wrote this".** Provenance has to come from the branch, the worktree and the
timestamp. Both wrong guesses were ruled out on evidence I verified myself before believing
the denial:
nofx-b3's boot marker lands 60s after `ec2dd8f7` while they were reading a boot-integrity
line, and none of nofx-ed's six branches contains it (`git merge-base --is-ancestor`, all
six), with their own commits bracketing the timestamp at 21:44:33 and 22:01:48. nofx-47 then
confirmed. Their own verdict on the merge: combining rather than forcing was right, and the
owner-scoped heartbeat I added is **a bug fix rather than a refinement** — theirs took no
session, so any lane could beat any lock, which defeats the liveness claim the whole class
rests on.

The sharper version of the problem: a lane's work was merged into `dev` without its author
being told, under a wave another lane was reporting. A24 bans a boot line that claims a
neighbour's work — the same rule has to run in reverse, or credit silently migrates to
whoever merges. PART 3 step 0 (push-empty-at-accept) is the mechanism that stops it
happening again.

| kept from `ec2dd8f7` | why |
|---|---|
| `heartbeat_epoch` beside the ISO stamp | reading age is integer arithmetic, no `date -d` parse at read time |
| meta replaced via temp + `mv`, never edited in place | a reader cannot catch a half-written heartbeat |
| **legacy `~/nofx-main.lock` surfaced in `status`** | the transition's real hazard: a lane still on the old shape is invisible to the new one. I had missed this entirely |
| one `meta` file, and the corroboration wording | theirs was better written than mine |

| kept from mine | why |
|---|---|
| `heartbeat` is **owner-scoped** | theirs let any caller beat any lock — an unauthenticated beat keeps a stranger's abandoned lock looking alive, which is failure (a) rebuilt |
| `check` with rc 0/1/2 | the tree-guard spec's suppression rule needs a scriptable answer |
| `with-heartbeat <session> -- <cmd>` | long builds outrun a 5-min window; the bounded beater is how you cover that without a daemon |
| the 36 pins, class 70, class 13, PART 3, this report | theirs shipped the script and the spec edit, no tests and no checklist entry |
| `mktemp` instead of the shell's own id for the temp name | legitimate use, but it keeps the source pin blunt — the script contains no process id at all |

Their default expiry (45 min) replaced my 90.

**This collision is itself the lesson.** Two lanes independently produced ~250 lines solving
the same ruling within an hour, and the only reason it was caught is that the second push
was rejected as non-fast-forward. Nothing in the dispatch protocol announces "I have started
this wave". A branch name is the closest thing we have to a claim, and it is only checked at
push time — after the work is done.
