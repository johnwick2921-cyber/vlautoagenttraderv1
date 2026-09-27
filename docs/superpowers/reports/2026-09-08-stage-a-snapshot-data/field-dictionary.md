# Schema 1 field dictionary

All payload fields below are nullable. NULL means not captured for this **event**, with a per-field explanation in `missing`. A value on another linked event is not silently substituted. Numeric zero and computed empty arrays are preserved. Raw source documents retain their own source schema; top-level NULL guarantees do not retrospectively certify every nested source default.

Common columns: `id` is archive sequence; `schema_version` is the written schema; `writer_revision` is the writing binary revision; `object` and `event` name the record; `snapshot_id` links the planner read where supplied. `observation_ms` is the source event clock; `receipt_ms` is the local observation/admission clock; `publication_ms` is the successful plan publication clock; `permission_ms` is the existing evaluation clock. Missing source clocks remain NULL. `captured_ms` is the separate DB write clock. CT strings are derived at export. `null_fields` counts registered top-level payload NULLs plus missing clocks.

A later candidate receipt does not establish its historical first availability. An order receipt does not establish acceptance; a book price without role linkage does not establish entry. No experiment may treat missing evidence as zero.

## market

| Field | Meaning; NULL when not captured |
|---|---|
| `root_symbol` | Canonical source symbol |
| `contract` | Received resolved contract |
| `contract_basis` | Evidence and limits of contract attribution |
| `feed` | Data transport/provider name |
| `source_timezone` | Source timestamp convention |
| `timeframe` | Received or detected interval |
| `source_stamp_ms` | Original source timestamp |
| `bar_open_ms` | Canonical bar opening timestamp |
| `bar_close_ms` | Source bar closing timestamp |
| `open` | Source opening price |
| `high` | Source high price |
| `low` | Source low price |
| `close` | Source closing price |
| `volume` | Source volume, including an explicitly supplied zero |
| `finalized` | Bar close has occurred at receipt, not a guarantee against later correction |
| `forming` | Bar close remains after receipt |
| `correction` | Comparison against a previously captured complete version |
| `previous_observation` | Prior version used for comparison |
| `missing_intervals` | Authoritative missing-interval evidence; not currently supplied |
| `bid` | Source bid; not currently supplied |
| `ask` | Source ask; not currently supplied |
| `spread` | Source spread; not currently supplied |
| `price_scale` | Certified historical merge/adjustment policy; currently unavailable |
| `roll_information` | Per-bar roll metadata; currently unavailable |
| `source_build_id` | Received AddOn build identity |

## candidate

| Field | Meaning; NULL when not captured |
|---|---|
| `stable_id` | Deterministic candidate identity hash |
| `identity_basis` | Identity inputs and unknown-formation alias limitation |
| `root_symbol` | Canonical source symbol |
| `raw_origin` | Original detected-level value with capture pointer excluded |
| `family` | Scorer family |
| `price` | Candidate reference price |
| `lo` | Detected lower bound |
| `hi` | Detected upper bound |
| `formation_ms` | Detected formation birth; source zero remains unknown |
| `availability_ms` | Receipt of this candidate observation, not historical first availability |
| `prior_episodes` | Existing detector episode result and its parameters |
| `zone_pattern` | Detected zone pattern |
| `timeframe` | Received or detected interval |
| `freshness_at_read` | Scorer freshness classification |
| `confluence_raw` | Family count actually used before cap |
| `confluence_capped` | Confluence count after the scorer cap |
| `raw_score_components` | Captured terms at actual computation, including nullable unused terms |
| `capped_score_components` | Captured capped confluence and resulting grade |
| `overrides` | Observed grade/seating changes |
| `final_score` | Scorer result; never recomputed by archive |
| `grade` | Captured final or explicit owner grade |
| `rank` | Final seated position, not pure score rank |
| `selection_outcome` | Observed seated or cut state |
| `exclusion_reason` | Actual per-candidate cut cause |
| `role` | Computed target/obstacle/invalidation role |
| `legacy_row_id` | Historical candidate-row link; not currently available |

## plan

| Field | Meaning; NULL when not captured |
|---|---|
| `input_snapshot_id` | Authoring read identity |
| `input_snapshot` | Actual PlannerInput value |
| `prompt` | Exact submitted outer-attempt prompt |
| `system_prompt` | Actual system prompt |
| `prompt_hash` | SHA-256 of submitted prompt |
| `prompt_version` | Separate prompt-version identity; not currently supplied |
| `model` | Selected model identity |
| `model_config` | Actual reasoning mode/effort/token-cap request settings |
| `config_version` | Available indicator AI configuration fingerprint, not complete client config |
| `config` | Complete credential-free configuration snapshot; currently unavailable |
| `attempt` | Outer authoring attempt number |
| `attempt_mode` | Author/repair/resend label |
| `attempt_started_ms` | Command-call start clock |
| `attempt_ended_ms` | Provider-return clock |
| `duration_ms` | Measured outer provider-call elapsed duration |
| `rejection_reason` | Actual provider/parser/validation error when present |
| `raw_output` | Exact outer provider response |
| `accepted_output` | Final accepted stored document, including scenario economics |
| `normalization` | Raw-before/final-after pair without replaying normalization |
| `plan_id` | Stored plan identity |
| `plan_version` | Stored accepted plan version |
| `tokens_in` | Actual provider input usage; not currently exposed |
| `tokens_out` | Actual provider output usage; not currently exposed |

## scenario

| Field | Meaning; NULL when not captured |
|---|---|
| `plan_id` | Stored plan identity |
| `plan_version` | Stored accepted plan version |
| `scenario_id` | Authored scenario identity |
| `ordered_predicates` | Authored confirm then confirm2, preserving order |
| `initial_risk` | Absolute composed entry-stop distance at placement |
| `risk_basis` | Units and source of composed risk |
| `target_path` | Authored target chain or composed objective, distinguished by event |
| `predicate_timestamps` | Dedicated predicate-clock projection; currently NULL, existing timestamps remain in raw revalidation metadata |
| `invalidation` | Authored rule or observed invalidation with basis |
| `expiry` | Observed expiry verdict and reason; absent explicit expiry remains unknown |
| `revalidation` | Existing metadata/verdict, never re-evaluated by recorder |
| `reason` | Existing verdict/frame/close reason; h1 omitted reason remains unknown |
| `permission_status` | Returned EntryGate result only, never scenario activation |
| `arm_id` | Existing arm row identity |
| `authored_geometry` | Authored arm declaration, not an accepted order |

## exec

| Field | Meaning; NULL when not captured |
|---|---|
| `root_symbol` | Canonical source symbol |
| `contract` | Received resolved contract |
| `signal_id` | Wire signal linkage |
| `order_id` | Received broker order identity |
| `parent_id` | Parent-order linkage; not supplied by current capture |
| `order_type` | Source order type |
| `order_semantics` | Present broker prices/type or command semantics with basis |
| `side` | Source action/direction |
| `oco_id` | Received OCO group identity |
| `tif` | Received time-in-force |
| `intended_entry` | Arm row intended entry |
| `composed_entry` | Actual command entry parameters |
| `accepted_entry` | Role-linked accepted entry; currently NULL pending reliable entry attribution |
| `attainable_entry` | Stored position entry with explicit basis; unclassified fills stay in fills |
| `entry_basis` | Evidence for entry field |
| `exit_price` | Received or stored exit price |
| `exit_basis` | Evidence for exit field |
| `fills` | Sanitized received fill facts, without guessing entry versus exit |
| `simulation_assumption` | Explicit experiment assumption; Stage A chooses none |
| `costs` | Measured broker cost; source fee default zero is not treated as measured |
| `size` | Source command/order/position quantity |
| `common_horizon` | Experiment horizon; not selected by Stage A |
| `ambiguity` | Explicit missing role/linkage or other ambiguity |
| `timeout` | Existing placement-timeout observation |
| `broker_frame` | Sanitized received frame, not implied validation/acceptance |
| `source_build_id` | Received AddOn build identity |
| `reason` | Existing verdict/frame/close reason; h1 omitted reason remains unknown |
| `pnl_corrected` | Corrected outcome only; absent is UNRESOLVED |
| `outcome_exclusion` | Pre-era/test/UNRESOLVABLE/UNRESOLVED reason |
| `position_id` | Stored or received position identity |
| `plan_id` | Stored plan identity |
| `plan_version` | Stored accepted plan version |
| `scenario_id` | Authored scenario identity |
