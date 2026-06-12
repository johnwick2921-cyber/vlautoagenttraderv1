# ADR-002: ICT/SMC concepts operationalized as rules + confluence scoring

## Status

Accepted (2026-05-22). Encoded in `kernel/engine_prompt_futures.go` (the `futures` prompt variant) and the validation layer in `kernel/engine_position.go`.

## Context

We need a decision-making framework for NQ/MNQ futures that is **codeable, auditable, and reproducible**. The bot must justify every entry, and a human reviewer must be able to look at a decision row in `data/data.db` and answer "why did the bot enter here?" weeks later.

ICT/SMC (Inner Circle Trader / Smart Money Concepts) is the trader-community vocabulary the user is already fluent in: Order Blocks, Fair Value Gaps (FVG), liquidity sweeps, Market Structure Shifts (MSS), session-based bias. The framework has well-defined chart constructs — they can be turned into hard rules instead of "feel".

Three styles were considered:

1. **Pure ML signal** — train a model on historical NQ data, output a probability of upmove.
2. **Pure rule-based** — encode entry triggers as Go logic, no AI in the loop.
3. **AI + deterministic rules** — AI generates reasoning + a candidate decision; deterministic code validates the decision against rules and a confluence score.

Constraints:

- Every entry needs an audit trail readable by a human (regulatory mindset, not regulatory requirement).
- The decision must be reproducible: given the same inputs, the same constraints must trip.
- The user wants to tune the prompt itself, not retrain a model.

## Decision

**ICT/SMC concepts are operationalized as the `futures` prompt variant** in `kernel/engine_prompt_futures.go`. The prompt instructs the AI to evaluate the chart in ICT/SMC terms (Order Blocks, FVG, liquidity, MSS, session bias) and emit a structured JSON decision with `reasoning` that cites the concepts it found.

**Deterministic code does the gatekeeping**, not the AI:

- The decision JSON shape is fixed (`{action, symbol, entry, stop_loss, take_profit, reasoning, leverage, confidence}`) — see ADR-004 cross-reference and `CLAUDE.md`.
- `kernel/engine_position.go` validates SL < entry (long) / SL > entry (short), R/R ≥ 1.5, position-size bounds, and the action enum.
- Session gating, contract-roll gating, risk limits, and stale-data gating run BEFORE the AI call in `engine_analysis.go` — see ADR-006 (CME calendar).

**Confluence is scored deterministically** via the indicator set the prompt is asked to consider (ATR / EMA / MACD line / RSI / Bollinger / Donchian — see `market/data_indicators.go`). The AI does not invent numbers; it interprets them.

## Consequences

**Positive**

- Every decision row in `data/data.db` is auditable end-to-end: prompt + indicator snapshot + AI reasoning + final validated decision.
- Backtests have semantic meaning — replaying the same bar sequence reconstructs the same gates and validations.
- The user can iterate the prompt (`kernel/engine_prompt_futures.go`) without retraining anything.
- Adding a new gate is local: add a function, wire it into `engine_analysis.go`. No coupling to model weights.

**Negative**

- Discretionary "feel" is sacrificed. If a setup looks great to a human but doesn't hit the JSON shape + validator, the bot will not take it.
- ICT/SMC vocabulary is fuzzy — the AI's interpretation of "Order Block" can drift. Mitigated by deterministic R/R and SL/TP validation, which catches anything actually unsound.
- Prompt drift between Claude / DeepSeek / OpenAI is possible. The golden-test harness in `kernel/engine_prompt_golden_test.go` (Plan 5 Task 27) freezes the prompt structure.

**Neutral**

- The architecture is provider-agnostic. Switching the LLM (DeepSeek ↔ Claude ↔ GPT) does not change the gating logic — only the language of the reasoning.

## Alternatives Considered

- **Pure ML signal (rejected)** — No decision audit trail beyond "the model said so." Cannot explain a single trade to the user weeks later. Cannot patch a class of bad trades without retraining.
- **Pure rule-based (rejected)** — Too rigid for the inherently pattern-driven nature of ICT/SMC. The user explicitly wanted AI in the loop for context interpretation (session bias, multi-timeframe alignment).
- **Discretionary AI (rejected)** — Letting the AI emit free-form trade ideas without a fixed JSON contract makes auditing impossible and breaks every downstream consumer (DB schema, web UI, validation layer). This is enforced in `CLAUDE.md` under "Decision JSON shape — do NOT break."

See also ADR-006 (CME calendar — the session gate that runs before any AI inference).
