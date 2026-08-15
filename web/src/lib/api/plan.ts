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
  version?: number
  overlay_count?: number
  lifecycle?: string // active | expired | died | superseded
  model_id?: string
  doc?: PlanDoc
  level_facts?: PlanLevelFact[]
  price?: number
  replans_left?: number
  warming?: string // "n/10" while the SVP is uncalibrated; "" once warm
  // Per-scenario live status keyed by scenario id (executor-phase; absent now).
  scenario_status?: Record<string, ScenarioStatusValue>
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

const enc = encodeURIComponent

export const planApi = {
  // Active plan (overlay-resolved) + live scenario facts. Returns null on any
  // failure so the card falls back to its error/no-plan state (never throws).
  async getPlanToday(
    traderId: string,
    symbol?: string,
    silent = true
  ): Promise<PlanToday | null> {
    const q = symbol ? `&symbol=${enc(symbol)}` : ''
    const res = await httpClient.request<PlanToday>(
      `${API_BASE}/plan/today?trader_id=${enc(traderId)}${q}`,
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
}
