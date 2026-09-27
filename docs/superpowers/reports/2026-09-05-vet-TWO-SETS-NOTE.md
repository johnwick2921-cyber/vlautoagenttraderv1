# 2026-09-05 veteran review — TWO SETS OF SECTION REPORTS, AND WHICH FILE IS WHICH

**Owner ruling, 2026-09-05 18:2x CT: keep both. Nothing was deleted.**

Two independent lanes answered the same ten-section veteran-review dispatch on the same day.
Both sets are on `dev`. This note exists because for about two hours only one of them was
visible at the canonical paths, and a reader had no way to tell that the other existed.

## What each filename holds

| Path | Lane | What it is |
|---|---|---|
| `2026-09-05-vet-NN-<slug>.md` | `vet-NN-0905` (nine lanes, branches `docs/vet-NN-0905`) | Section report **with an adversarial VERIFICATION RECORD folded in** — three verifier lenses (numbers re-derived from the store, code citations + checklist cross-check, research labels + trader logic), every finding listed with its disposition (ACCEPTED-FIXED / REJECTED) and evidence. |
| `2026-09-05-vet-NN-<slug>-complete.md` | `codex-vet-09-0905` / `codex-vet-09-complete-0905` | The same section re-answered by the section-09 lane, with its own evidence directories and a requirement-coverage table. |
| `2026-09-05-vet-09-top-ten.md` | `codex-vet-09-0905` | Section 09 (top ten + rulings page). **Only one lane wrote section 09**; there is no second version. |
| `2026-09-05-vet-audit-index.md` | codex lane | That lane's own index. |

Evidence directories (`-data/`, `-complete-data/`) were never moved and are shared where both
lanes wrote into the same directory.

## How this happened (provenance, not blame)

Every commit in this repo carries the identical git author, so the author field cannot answer
"which lane wrote this" (PROVENANCE, CLAUDE.md). The branch and the claim commit can:

- Sections 01–08 and 10 were claimed at 09:55–10:00 CT on branches `docs/vet-NN-0905`, each with
  a conforming claim commit naming its session (`deploy/nofx-claim.sh check` passes on all nine).
- Section 09 was claimed at 15:47:56 CT (`codex-vet-09-0905`), and a second wave at 17:27:04 CT
  (`codex-vet-09-complete-0905`).
- At 15:48 CT the section-09 lane merged all nine section branches into its own branch and
  fast-forwarded `dev`, which is how every report first reached `dev`.
- Between 17:27 and 18:19 CT that lane rewrote all nine section reports **in place** with its own
  "complete" versions. That is what removed the verification records from the canonical paths.
- Two of the nine (07, 08) had not yet re-merged their verified versions. Their merge was stopped
  deliberately rather than allowed to overwrite the other lane's newer file.

**The step-0 claim protocol did its job and still did not prevent this.** A branch name on origin
claims a *branch*; it does not claim the *report path* inside it. Two lanes can hold different
claims and still collide on one file. If a rule is wanted from this day, it is that a wave which
rewrites a file another live claim owns must say so in its commit message and leave the prior
version at a sibling path — which is what this note restores after the fact.

## Reading order, and one caveat

Read the `-complete.md` set for the section-09 lane's own chain of evidence. Read the
plain-named set for the numbers that survived an adversarial pass. Where they disagree, the
verification record in the plain-named file shows which specific figures were challenged, by
what query, and whether the challenge was accepted.

**Caveat:** `2026-09-05-vet-09-top-ten.md` cites the section reports by `path:line`. Those line
numbers were written against the `-complete` versions, so they now resolve against
`-complete.md`, not the plain-named file at the same path.
