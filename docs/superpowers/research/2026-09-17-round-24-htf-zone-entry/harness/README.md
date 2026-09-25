# Round 24 harness — build note

These files are the Round 23 harness (`docs/superpowers/research/2026-09-16-round-23/harness`
@ `a488fd59`, as carried by `d15db077`) with ONE change, in `eval.go`: the episode writer emits
`price/lo/hi/label/polarity/delta/read_at_ms/contract`, and every read writes its raw HTF zone
universe to `zones.jsonl` (`emitZones`, `zonePolarity`).

They are tagged `//go:build r24harness` because they target the kernel API of `d15db077`
(`AssembleResearchLevels` changed on dev after it). The binary that produced `out-r24/` was
built from a `git archive d15db077` tree with this `eval.go` dropped in — the SAME kernel the
Round 23 episode set was built with, so the level universe is identical by construction:

```
git archive d15db077 | tar -x -C $T
cp harness/eval.go $T/docs/superpowers/research/2026-09-16-round-23/harness/eval.go
(cd $T && go build -o r24harness ./docs/superpowers/research/2026-09-16-round-23/harness)
```

Binary sha256 (first 16) and the exact command are on the first line of `out-r24/harness.log`.
