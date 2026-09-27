# Report — Edit sheet prefill fix (2026-08-18)

1. Classified: not a property mismatch — card row and sheet both read `instruction`
   from the same `PlanLevelFact`, but the sheet rendered it via a closed 6-verb
   `ChipRow`, so AI planner prose never matched a chip → looked blank, and tapping
   any chip rewrote the prose (silent semantic wipe).
2. Field audit: price ✓ · type hidden in edit (by design) · INSTRUCTION broken →
   fixed · grade ✓ · note ✓ seeds from `level.note` (API emits it only for OWNER
   rows) · scenario tag was HIDDEN in edit mode → fixed · owner note/tag edits
   were silently dropped on save → fixed.
3. Fix (FE `EditSheet.tsx`): instruction is now a text input seeded with the
   exact string; verb chips demoted to quick-fill (empty input never rewrites);
   scenario-tag row shows when editing; save carries note/scenario_tag for OWNER.
4. Fix (BE): overlay handler strips note/scenario_tag from the RFC-6902 patch
   (stored overlay stays schema-pure) and writes them through to `owner_levels`
   by symbol+price+label (`OwnerLevelStore.UpdateNoteTag`).
5. Guard: 3 new tests in `P5_door.test.tsx` assert the sheet's `instruction-input`
   equals the card's instruction (AI prose + OWNER) and that the save patch keeps
   note/scenario_tag — card/sheet single-source-of-truth per field. 10/10 pass.
6. Verify: `go build`/`go test ./api/ ./store/`/`go vet` ✓ · `tsc --noEmit` ✓ ·
   `vitest` 247/248 (2 pre-existing, unrelated).
7. Browser (fixture-driven; live card was post-session no-plan): AI level sheet
   showed input `hold / defend for long structure`; OWNER level sheet showed
   instruction `target / fade`, note `my shelf — defend until breakdown`, tag S1.
   Screenshots captured both.
8. Deploy: build 8d5cfa1ff784 → RELEASE stamped → SIGKILL restart → boot
   `🔐 BOOT INTEGRITY OK — rev 8d5cfa1ff784 +dirty … expected 8d5cfa1ff784 · goldens PASS`
   → FE `npm run build` ✓. Bot cycling on NT8 bars, one PID.
