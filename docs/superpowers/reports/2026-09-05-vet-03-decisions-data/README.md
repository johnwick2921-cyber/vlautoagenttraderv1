# Section 3 evidence status

`complete/` is the primary evidence for the renewed full Section 3 audit on branch `docs/vet-03-0905-complete`.

All earlier scripts, outputs, API captures and the `revise/` directory are retained solely as historical audit material. Their old 65-row cuts, trigger-touch definitions, inferred initial risk, fixed-R target estimates and gate-savings recommendations are superseded. Do not use them as current results. The replacement report explains each withdrawal and provides corrected 58-trade and 47-decision-trade exports.

The complete pass opens SQLite with `mode=ro` and `PRAGMA query_only=ON`; it never invokes `cmd/gate-jwt` or imports a production store initializer. Its two scripts and their outputs are hashed in `complete/manifest.json`.

The two `complete/executor_*_system.txt` files preserve the exact database strings, including three trailing-space lines each. These six raw-evidence whitespace findings are intentional; generated CSV/source excerpts use clean LF formatting. Do not trim the captured prompt bytes and continue to describe them as exact.
