# Section 7 complete evidence

Base b4376246. Captured with SQLite mode=ro and query_only; GETs only. No trading mutations.

Run copies from the authorized scratch directory /home/hoang/nofx-analysis/vet-07-complete-0905, not in a runtime checkout. Scripts have explicit paths to this worktree and production read-only DB. Copy these script/template files into that scratch directory first.

```text
python3 /home/hoang/nofx-analysis/vet-07-complete-0905/capture.py
PYTHONDONTWRITEBYTECODE=1 /home/hoang/nofx-analysis/vet-07-0905/tokvenv/bin/python /home/hoang/nofx-analysis/vet-07-complete-0905/measure.py
python3 /home/hoang/nofx-analysis/vet-07-complete-0905/audit_evidence.py
python3 /home/hoang/nofx-analysis/vet-07-complete-0905/finalize.py
```

The existing tokenizer environment is read only. It provides tiktoken o200k_base and cl100k_base, neither claimed as DeepSeek billing. The snapshot is intentionally dated; future DB runs can change evidence. Prompt measurements/map can be reproduced offline from the committed payloads. The finalize step adds read-only joined plan evidence and publishes the report from preserved templates.

planner-132-actual.txt is exact stored bytes decoded as UTF-8. planner-132-current-contract-replay.txt is a labelled textual two-site update, not a provider call or reconstructed full current runtime. prompt-boundaries.csv is exhaustive character provenance; static claims are instruction, not certified facts. constraint-map.md preserves all constraints; policy-cuts.md proposes changes separately. appendix-rewrite.txt and executor-flow.md are review artifacts only.

Old ../2026-09-05-vet-07-prompts-data outputs are superseded as primary evidence. Eligible trades are58, not65. No claim of model behavioral equivalence or current-strict profitability. The parent owns merge/deployment; keep this worktree until integration.

Raw prompt, source, exact-cut quotations and CSV artifacts intentionally preserve trailing whitespace/newlines (including standard CSV CRLF). Whitespace lint is applied to report prose and scripts; normalization of evidence bytes would invalidate exact-payload claims. Run validate_artifacts.py for the final content and scoped whitespace checks.
