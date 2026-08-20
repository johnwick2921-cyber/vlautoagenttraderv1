// P4.1/P4.4 — Day-Plan API client (mirrors the /api/plan/* Go handlers).
// All GETs are silent (self-fetched by the PlanCard via SWR); errors resolve to a
// safe empty/absent value so the card renders its no-plan / error state instead of
// toasting. Consumes plan_final from the server — the UI never merges overlays.

import { httpClient } from '../httpClient'
import { API_BASE } from './helpers'

// ── plan document (mirrors kernel.PlanDoc JSON) ──
export interface PlanBias {
  direction: string // long | short | neutral
  conviction: string // high | medium | low
  flip_condition: string
}

export interface PlanLevel {
  price: number
  label: string // provenance chip: PDH, ONH, nPOC·Tue, RN, EQH…
  grade: string // A | B | C
  instruction: string
}

export interface PlanScenario {
  id: string // S1, S2, S3
  trigger: string
  condition: string // reclaim | hold | sweep_reclaim | reject | acceptance | breakout_retest
  direction: string // long | short
  target_chain: number[]
  invalid: string
  quality: string // A+ | A | B
}

export interface PlanDoc {
  reasoning: string
  bias: PlanBias
  levels: PlanLevel[]
  scenarios: PlanScenario[]
  no_trade: string[]
  death_condition: string
  day_type?: string
}

// ── live per-level facts from the P0.4 evaluator (one array, three renderers) ──
export interface PlanLevelFact {
  price: number
  label: string
  grade: string
  instruction: string
  distance: number // signed distance in points (backend-computed)
  sweep: boolean
  closes_beyond: number
  accept_have: number
  accept_need: number
  still_valid: boolean
  // Owner-overlay fields (populated once P5 overlays land; absent pre-★2). The
  // card renders 👤 / 📝 / an S-tag only when these are present.
  origin?: 'AI' | 'OWNER'
  note?: string
  scenario_id?: string
}

// Backend-owned scenario status (the state machine ships in the executor phase;
// absent for now → the card renders every scenario as 'armed', the plan-born
// initial state). The UI never computes trading state itself.
export type ScenarioStatusValue =
  | 'armed'
  | 'waiting'
  | 'triggered'
  | 'invalidated'
  | 'expired'

// ── GET /api/plan/today ──
export interface PlanToday {
  found: boolean
  trade_date: string
  session: string
  night: boolean
  mode: string // advisory | direction | strict
  /** UI-verification (2026-08-18) — a planner read is in flight for this chain. */
  reading?: boolean
  version?: number
  overlay_count?: number
  lifecycle?: string // active | expired | died | superseded
  model_id?: string
  doc?: PlanDoc
  level_facts?: PlanLevelFact[]
  price?: number
  replans_left?: number
  warming?: string // "n/10" while the SVP is uncalibrated; "" once warm
  /** P2 — regime fields unavailable when this plan was written (of 7). */
  dark_regime_count?: number
  /** P2 — true when too much of the regime map was dark to fully trust the plan. */
  degraded?: boolean
  // Per-scenario live status keyed by scenario id (executor-phase; absent now).
  scenario_status?: Record<string, ScenarioStatusValue>
  // A1/A4: verdict basis ("machine"|"heuristic") + scenarios with no anchor
  scenario_meta?: { basis?: Record<string, string>; unevaluable?: string[] }
  /** W15.B — the acceptance rule the executor evaluates these levels with. */
  acceptance_rule?: string
  /** W15.B — which session is LIVE right now, regardless of the tab requested. */
  active_session?: string
  /** W15.B — true when the payload IS the live session (false = viewing a sibling). */
  is_active?: boolean
  /** W15.B — sessions THIS strategy runs, resolved by the same gate the bot uses. */
  runnable_sessions?: string[]
  /** The RESOLVED re-plan cap (config, never a literal). */
  replan_cap?: number
  /** ITEM 4 — owner edits a re-plan could not re-anchor onto this version. */
  uncarried_edits?: UncarriedEdit[]
  /** ITEM 15 — true when ?version= served a superseded version, not the latest. */
  historical?: boolean
  /** ITEM 15 — the newest stored version, so the card can offer the way back. */
  latest_version?: number
  /** ITEM 15 — why this version was written (session open / death condition / …). */
  trigger_reason?: string
  created_at?: string
}

// ── GET /api/plan/versions — every stored version of ONE session's plan ──
export interface PlanVersionItem {
  version: number
  lifecycle: string
  trigger_reason: string
  created_at: string
  model_id: string
  degraded?: boolean
  is_latest: boolean
  level_count?: number
  scenario_count?: number
  bias?: string
  day_type?: string
  death_condition?: string
  /** the version that replaced this one (absent on the latest) */
  superseded_by?: number
  /** WHY it stopped being the plan — the successor's trigger_reason */
  death_reason?: string
  /** plain-language change list vs the version that replaced it */
  diff_vs_next?: string[]
}
export interface PlanVersionsResponse {
  trade_date: string
  session: string
  latest_version: number
  versions: PlanVersionItem[]
}

// ITEM 4 — an owner edit that could not carry into a new plan version.
export interface UncarriedEdit {
  op: string
  path: string
  reason: string
  summary: string
}

// ── GET /api/plan/history ──
export interface PlanHistoryItem {
  trade_date: string
  session: string
  version: number
  lifecycle: string
  model_id: string
  trigger_reason: string
}

// ── GET /api/plan/alerts ──
export type AlertLevel = 'P0' | 'P1' | 'P2'
export interface PlanAlert {
  id: number
  trader_id: string
  level: AlertLevel
  event_id: string
  kind: string
  title: string
  body: string
  acked: boolean
  created_at: number // unix seconds
}
export interface PlanAlertsResponse {
  alerts: PlanAlert[]
  unacked: number
}

// ITEM 3 — the owner's manual re-read.
export interface RereadGate {
  allowed: boolean
  reason?: string
  session?: string
  replans_left: number
  replan_cap: number
  version: number
}

// P6 — the owner reset: abandon the chain, restore the budget, fresh plan.
export interface ResetGate {
  allowed: boolean
  reason?: string
  note?: string
  session?: string
  version: number
  replan_cap: number
}

const enc = encodeURIComponent

export const planApi = {
  // Active plan (overlay-resolved) + live scenario facts. Returns null on any
  // failure so the card falls back to its error/no-plan state (never throws).
  async getPlanToday(
    traderId: string,
    symbol?: string,
    silent = true,
    session?: string,
    version?: number
  ): Promise<PlanToday | null> {
    const q = symbol ? `&symbol=${enc(symbol)}` : ''
    // W15.B — an explicit session makes the card's tabs real (they used to be
    // pure highlighting). Omitted → the live session, i.e. unchanged.
    const sq = session ? `&session=${enc(session)}` : ''
    // ITEM 15 — an explicit version serves that HISTORICAL plan; omitted = latest.
    const vq = version ? `&version=${version}` : ''
    const res = await httpClient.request<PlanToday>(
      `${API_BASE}/plan/today?trader_id=${enc(traderId)}${q}${sq}${vq}`,
      { silent }
    )
    return res.success && res.data ? res.data : null
  },

  // Every stored version of one session's plan, oldest first, with each
  // version's death reason and a plain-language diff vs its successor.
  async getPlanVersions(
    traderId: string,
    session: string,
    tradeDate?: string,
    silent = true
  ): Promise<PlanVersionsResponse | null> {
    const dq = tradeDate ? `&trade_date=${enc(tradeDate)}` : ''
    const res = await httpClient.request<PlanVersionsResponse>(
      `${API_BASE}/plan/versions?trader_id=${enc(traderId)}&session=${enc(session)}${dq}`,
      { silent }
    )
    return res.success && res.data ? res.data : null
  },

  async getPlanHistory(
    traderId: string,
    silent = true
  ): Promise<PlanHistoryItem[]> {
    const res = await httpClient.request<{ history: PlanHistoryItem[] }>(
      `${API_BASE}/plan/history?trader_id=${enc(traderId)}`,
      { silent }
    )
    return res.success && res.data?.history ? res.data.history : []
  },

  // May the owner force a fresh planner read right now, and what does it cost?
  // ITEM 5a — hide one alert from the feed (soft-delete; the row survives).
  async dismissAlert(
    traderId: string,
    alertId: number
  ): Promise<{ ok: boolean; needsAck?: boolean; error?: string }> {
    const res = await httpClient.request<{ dismissed: boolean }>(
      `${API_BASE}/plan/alert-dismiss`,
      {
        method: 'POST',
        data: { trader_id: traderId, alert_id: alertId },
        silent: true,
      }
    )
    if (res.success && res.data?.dismissed) return { ok: true }
    const msg = res.message ?? ''
    return { ok: false, needsAck: /acknowledg/i.test(msg), error: msg }
  },

  // ITEM 5b — clear every ACKNOWLEDGED alert, leaving unacked ones in place.
  async clearReadAlerts(
    traderId: string
  ): Promise<{ ok: boolean; cleared: number }> {
    const res = await httpClient.request<{ cleared: number }>(
      `${API_BASE}/plan/alert-clear-read`,
      { method: 'POST', data: { trader_id: traderId }, silent: true }
    )
    return { ok: !!res.success, cleared: res.data?.cleared ?? 0 }
  },

  async getRereadGate(
    traderId: string,
    silent = true
  ): Promise<RereadGate | null> {
    const res = await httpClient.request<RereadGate>(
      `${API_BASE}/plan/reread?trader_id=${enc(traderId)}`,
      { silent }
    )
    return res.success && res.data ? res.data : null
  },

  // SPENDS one re-plan. The server re-checks eligibility, so a stale gate on the
  // client cannot buy an extra read.
  async forceReread(
    traderId: string
  ): Promise<{ ok: boolean; error?: string; gate?: RereadGate }> {
    const res = await httpClient.request<{ ok: boolean; gate: RereadGate }>(
      `${API_BASE}/plan/reread`,
      {
        method: 'POST',
        data: { trader_id: traderId },
        silent: true,
        timeoutMs: 320_000,
      }
    )
    if (res.success && res.data) return { ok: true, gate: res.data.gate }
    return { ok: false, error: res.message || 'reread refused' }
  },

  // P6 — may the owner reset this session's plan chain right now?
  async getResetGate(
    traderId: string,
    silent = true
  ): Promise<ResetGate | null> {
    const res = await httpClient.request<ResetGate>(
      `${API_BASE}/plan/reset?trader_id=${enc(traderId)}`,
      { silent }
    )
    return res.success && res.data ? res.data : null
  },

  // ABANDONS the current chain, restores the full re-plan budget, and runs a
  // fresh read (trigger_reason "owner reset"). The server re-checks eligibility.
  async forceReset(
    traderId: string
  ): Promise<{ ok: boolean; error?: string; note?: string; gate?: ResetGate }> {
    const res = await httpClient.request<{ ok: boolean; gate: ResetGate }>(
      `${API_BASE}/plan/reset`,
      // The reset runs a SYNCHRONOUS planner read server-side (60-300s) — the
      // 30s axios default was the stuck-dialog trigger (reset-dialog hotfix).
      {
        method: 'POST',
        data: { trader_id: traderId },
        silent: true,
        timeoutMs: 320_000,
      }
    )
    if (res.success && res.data) {
      return { ok: true, gate: res.data.gate, note: res.data.gate?.note }
    }
    return { ok: false, error: res.message || 'reset refused' }
  },

  async getPlanAlerts(
    traderId: string,
    silent = true
  ): Promise<PlanAlertsResponse> {
    const res = await httpClient.request<PlanAlertsResponse>(
      `${API_BASE}/plan/alerts?trader_id=${enc(traderId)}`,
      { silent }
    )
    return res.success && res.data ? res.data : { alerts: [], unacked: 0 }
  },

  async ackPlanAlert(traderId: string, alertId: number): Promise<boolean> {
    const res = await httpClient.request<{ acked: boolean }>(
      `${API_BASE}/plan/alert-ack`,
      {
        method: 'POST',
        data: { trader_id: traderId, alert_id: alertId },
      }
    )
    return res.success && !!res.data?.acked
  },

  // ── P5.1 overlay editing ──
  // Post an RFC-6902 overlay. Returns {ok, error?} — non-silent so armor/conflict
  // rejections (409/422) surface their message for the sheet to show inline.
  async postOverlay(
    traderId: string,
    patch: PatchOp[],
    origin: 'owner' | 'planner-revised' = 'owner',
    symbol = 'MNQ'
  ): Promise<{ ok: boolean; error?: string; overlay_version?: number }> {
    const res = await httpClient.request<{ overlay_version: number }>(
      `${API_BASE}/plan/overlay`,
      {
        method: 'POST',
        data: {
          trader_id: traderId,
          symbol,
          patch: JSON.stringify(patch),
          origin,
        },
        silent: true,
      }
    )
    if (res.success)
      return { ok: true, overlay_version: res.data?.overlay_version }
    return { ok: false, error: res.message }
  },

  async addOwnerLevel(
    traderId: string,
    level: {
      price: number
      label?: string
      note?: string
      scenario_tag?: string
    },
    symbol = 'MNQ'
  ): Promise<{ ok: boolean; error?: string; id?: number }> {
    const res = await httpClient.request<{ id: number }>(
      `${API_BASE}/plan/owner-level`,
      {
        method: 'POST',
        data: { trader_id: traderId, symbol, ...level },
        silent: true,
      }
    )
    if (res.success) return { ok: true, id: res.data?.id }
    return { ok: false, error: res.message }
  },

  async deleteOwnerLevel(traderId: string, id: number): Promise<boolean> {
    const res = await httpClient.request<{ deleted: boolean }>(
      `${API_BASE}/plan/owner-level/delete`,
      { method: 'POST', data: { trader_id: traderId, id }, silent: true }
    )
    return res.success && !!res.data?.deleted
  },

  // ── P5.4 Ask-Planner ──
  async askPlanner(
    traderId: string,
    question: string,
    symbol = 'MNQ'
  ): Promise<{ ok: boolean; error?: string; data?: AskPlannerResponse }> {
    const res = await httpClient.request<AskPlannerResponse>(
      `${API_BASE}/plan/ask`,
      {
        method: 'POST',
        data: { trader_id: traderId, symbol, question },
        silent: true,
        // The planner's backend AI budget is 300s — the 30s instance default
        // aborted every slow ask and latched the panel (stuck-send bug).
        timeoutMs: 320_000,
      }
    )
    if (res.success && res.data) return { ok: true, data: res.data }
    return { ok: false, error: res.message }
  },

  async getPlanThread(
    traderId: string,
    planId?: string,
    silent = true
  ): Promise<{ thread: PlanQAMessage[]; kpi: SycophancyKPI }> {
    const q = planId ? `&plan_id=${enc(planId)}` : ''
    const res = await httpClient.request<{
      thread: PlanQAMessage[]
      kpi: SycophancyKPI
    }>(`${API_BASE}/plan/ask?trader_id=${enc(traderId)}${q}`, { silent })
    if (res.success && res.data) return res.data
    return { thread: [], kpi: EMPTY_KPI }
  },

  // W13 — ask the planner to re-examine the whole plan after an owner edit.
  // Never throws and never mutates: the response is a PROPOSAL the owner applies.
  async realignPlan(
    traderId: string,
    change: RealignChange,
    symbol = 'MNQ',
    manual = false
  ): Promise<RealignResponse> {
    const res = await httpClient.request<RealignResponse>(
      `${API_BASE}/plan/realign`,
      {
        method: 'POST',
        data: { trader_id: traderId, symbol, manual, change },
        silent: true,
        timeoutMs: 320_000,
      }
    )
    if (!res.success || !res.data) {
      return { status: 'failed', reason: res.message || 'request failed' }
    }
    return res.data
  },

  // W16/R2 — declining a proposal is a RECORDED decision, not local UI state.
  // The plan is untouched; the row keeps the KPI series honest about rejections.
  async declineAsk(
    traderId: string,
    qaId: number
  ): Promise<{ ok: boolean; error?: string }> {
    const res = await httpClient.request<{ declined: boolean }>(
      `${API_BASE}/plan/ask/decline`,
      {
        method: 'POST',
        data: { trader_id: traderId, qa_id: qaId },
        silent: true,
      }
    )
    if (res.success && res.data?.declined) return { ok: true }
    return { ok: false, error: res.message }
  },

  // W16/R3 — the gate-block tally (in-memory, per CME session-day). It has been
  // served since B6 with no frontend consumer at all.
  async getGateBlocks(): Promise<{
    session_day_utc?: string
    summary?: string
    by_trader?: Record<string, Record<string, number>>
  } | null> {
    const res = await httpClient.request<{
      session_day_utc?: string
      summary?: string
      by_trader?: Record<string, Record<string, number>>
    }>(`${API_BASE}/risk/gate-blocks`, { silent: true })
    return res.success && res.data ? res.data : null
  },

  async applyAsk(
    traderId: string,
    qaId: number,
    symbol = 'MNQ'
  ): Promise<{ ok: boolean; error?: string }> {
    const res = await httpClient.request<{ applied: boolean }>(
      `${API_BASE}/plan/ask/apply`,
      {
        method: 'POST',
        data: { trader_id: traderId, symbol, qa_id: qaId },
        silent: true,
      }
    )
    if (res.success && res.data?.applied) return { ok: true }
    return { ok: false, error: res.message }
  },

  // ── P5.5 / P5.6 read models ──
  async getPlanTrades(
    traderId: string,
    silent = true
  ): Promise<PlanTradesResponse> {
    const res = await httpClient.request<PlanTradesResponse>(
      `${API_BASE}/plan/trades?trader_id=${enc(traderId)}`,
      { silent }
    )
    return res.success && res.data
      ? res.data
      : { trades: [], summary: { counts: {}, total: 0, gpa: 0 } }
  },

  async getPlanStats(
    traderId: string,
    silent = true
  ): Promise<PlanStatsResponse> {
    const res = await httpClient.request<PlanStatsResponse>(
      `${API_BASE}/plan/stats?trader_id=${enc(traderId)}`,
      { silent }
    )
    return res.success && res.data
      ? res.data
      : { weekly: null, progress: [], target_n: 1565, alpha: 0.00625 }
  },
}

// ── P5 types ──
export interface PatchOp {
  op: 'add' | 'remove' | 'replace' | 'test'
  path: string
  value?: unknown
  from?: string
}

export type PointClass = 'NEW-INFO' | 'BARE-DISAGREEMENT'
export type Verdict = 'DEFEND' | 'CONCEDE' | 'PROPOSE-MERGE'

export interface AskPlannerReply {
  evidence: string
  point_class: PointClass | ''
  verdict: Verdict | ''
  summary: string
  patch: string // JSON string of RFC-6902 ops ('' when none)
}
export interface AskPlannerResponse {
  qa_id: number
  plan_id: string
  plan_version: number
  reply: AskPlannerReply
}
// W13 — plan re-alignment on owner edit. Always resolves; `status` drives the UI.
export interface RealignChange {
  kind: 'add-level' | 'edit-level' | 'delete-level' | 'bulk-add'
  summary?: string
  price?: number
  label?: string
  grade?: string
  instruction?: string
  note?: string
  scenario_tag?: string
  batch_count?: number
}
export type RealignStatus =
  | 'proposal'
  | 'no-change'
  | 'skipped'
  | 'debounced'
  | 'capped'
  | 'failed'
export interface RealignResponse {
  status: RealignStatus
  qa_id?: number
  plan_id?: string
  plan_version?: number
  would_become?: string
  latency_ms?: number
  cost_usd?: number
  used?: number
  cap?: number
  reason?: string
  reply?: {
    evidence: string
    point_class: string
    verdict: string
    summary: string
    patch: string
  }
}

export interface PlanQAMessage {
  id: number
  role: 'owner' | 'planner'
  content: string
  evidence: string
  point_class: string
  verdict: string
  patch: string
  applied: boolean
  created_at: number
}
export interface SycophancyKPI {
  total: number
  new_info: number
  bare_disagreement: number
  defend: number
  concede: number
  propose_merge: number
  applied: number
  defend_on_bare: number
}
const EMPTY_KPI: SycophancyKPI = {
  total: 0,
  new_info: 0,
  bare_disagreement: 0,
  defend: 0,
  concede: 0,
  propose_merge: 0,
  applied: 0,
  defend_on_bare: 0,
}

export interface PlanTrade {
  symbol: string
  side: string
  entry_price: number
  exit_price: number
  entry_time: number
  exit_time: number
  realized_pnl: number
  mae: number
  mfe: number
  entry_confidence: number
  cited_scenario_id: string
  plan_matched: boolean
  plan_version: number
  adherence_grade: string
  adherence_label: string
}
export interface AdherenceSummaryFE {
  counts: Record<string, number>
  total: number
  gpa: number
}
export interface PlanTradesResponse {
  trades: PlanTrade[]
  summary: AdherenceSummaryFE
}

export type MatchedRandomStatus = 'WARMING' | 'BEATS-RANDOM' | 'NO-EDGE'
export interface TypeVerdict {
  level_type: string
  n: number
  reactions: number
  react_rate: number
  delta_pp: number
  p_value: number
  status: MatchedRandomStatus
  label: string
  target_n: number
}
export interface StatsProgress {
  level_type: string
  n: number
  reactions: number
  target_n: number
  react_rate: number
  warming: boolean
}
export interface PlanStatsResponse {
  weekly: {
    iso_week: string
    computed_at: number
    verdicts: TypeVerdict[]
  } | null
  progress: StatsProgress[]
  target_n: number
  alpha: number
}
