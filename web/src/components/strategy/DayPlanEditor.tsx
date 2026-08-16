// P4.5 — the Strategy Studio "Day Plan" block (futures-visible, per-strategy).
// Same visual grammar as RiskControlEditor: config in, onChange writes the whole
// slice back up, save→hot-reload persists it (no new persistence mechanics).
// Additive + defaults-off: absent day_plan leaves a strategy byte-identical, and
// plan_enabled=false is the master switch. Killzones are shown as ACTIVE windows.

import { useState } from 'react'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import type {
  DayPlanConfig,
  DayPlanSessionOverride,
} from '../../types/strategy'
import {
  SESSION_BANDS,
  londonDSTWarning,
  type SessionName,
} from '../plan/sessionConfig'

interface Props {
  config?: DayPlanConfig
  onChange: (config: DayPlanConfig) => void
  disabled?: boolean
  language: Language
}

// Mirror of Go DefaultDayPlanConfig() — enabling the master switch materializes
// this whole block so a first-time save writes the spec defaults.
const DEFAULT_DAY_PLAN: DayPlanConfig = {
  plan_enabled: false,
  plan_mode: 'advisory',
  planner_timeframes: ['D', '4h', '1h', '15m'],
  proximity_filter_atr: 1.5,
  max_levels: 8,
  scenario_cap: 3,
  acceptance_rule: '2x5m',
  replan_cap: 2,
  sessions_enabled: ['NY'],
  approval_required: false,
  evening_digest: true,
  last_entry_ct: '13:00',
  eod_flat_ct: '14:45',
  realign_cap: 5,
}

// NY's window end — the EOD flat must not sit AFTER it, or the session gate
// would call NY open after the day was already flattened (the P3 drift band).
const NY_WINDOW_END_CT = '14:45'

const ALL_SESSIONS: SessionName[] = ['NY', 'ASIA', 'LONDON']

// The structure-summary timeframes the planner may read (mockup: Daily…5m).
const PLANNER_TFS = ['D', '4h', '1h', '15m', '5m']

// ── primitives ──
function Toggle({
  on,
  onChange,
  disabled,
  testId,
  ariaLabel,
}: {
  on: boolean
  onChange: (v: boolean) => void
  disabled?: boolean
  /** stable hook for tests/Playwright */
  testId?: string
  /** accessible name — a bare role=switch announces as "switch" with no label */
  ariaLabel?: string
}) {
  return (
    <button
      role="switch"
      aria-checked={on}
      data-testid={testId}
      aria-label={ariaLabel}
      disabled={disabled}
      onClick={() => !disabled && onChange(!on)}
      style={{
        width: 40,
        height: 22,
        borderRadius: 999,
        background: on ? 'var(--vl-long)' : 'var(--vl-faint)',
        opacity: disabled ? 0.5 : 1,
        position: 'relative',
        transition: 'background 150ms',
      }}
    >
      <span
        style={{
          position: 'absolute',
          top: 2,
          left: on ? 20 : 2,
          width: 18,
          height: 18,
          borderRadius: 999,
          background: 'var(--vl-ivory)',
          transition: 'left 150ms',
        }}
      />
    </button>
  )
}

function Segmented({
  options,
  value,
  onChange,
  disabled,
}: {
  options: { key: string; label: string }[]
  value?: string
  onChange: (v: string) => void
  disabled?: boolean
}) {
  return (
    <div
      className="inline-flex"
      style={{
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-chip)',
        overflow: 'hidden',
      }}
    >
      {options.map((o) => {
        const active = value === o.key
        return (
          <button
            key={o.key}
            disabled={disabled}
            onClick={() => !disabled && onChange(o.key)}
            className="text-[10px] font-semibold uppercase tracking-wide px-2 py-1"
            style={{
              background: active ? 'var(--vl-gold-dim)' : 'transparent',
              color: active ? 'var(--vl-gold)' : 'var(--vl-muted)',
              opacity: disabled ? 0.5 : 1,
            }}
          >
            {o.label}
          </button>
        )
      })}
    </div>
  )
}

function NumberField({
  value,
  min,
  max,
  step = 1,
  onChange,
  disabled,
}: {
  value?: number
  min: number
  max: number
  step?: number
  onChange: (v: number) => void
  disabled?: boolean
}) {
  return (
    <input
      type="number"
      value={value ?? ''}
      min={min}
      max={max}
      step={step}
      disabled={disabled}
      onChange={(e) => {
        const n = parseFloat(e.target.value)
        if (!Number.isNaN(n)) onChange(Math.min(max, Math.max(min, n)))
      }}
      className="vl-num text-[12px] w-16 px-1.5 py-0.5"
      style={{
        background: 'var(--vl-card-2)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-chip)',
        color: 'var(--vl-ivory)',
        opacity: disabled ? 0.5 : 1,
      }}
    />
  )
}

/** HH:MM (America/Chicago) input for the day-trader clock times. */
function TimeField({
  value,
  onChange,
  disabled,
  testId,
}: {
  value?: string
  onChange: (v: string) => void
  disabled?: boolean
  testId?: string
}) {
  return (
    <input
      type="time"
      value={value ?? ''}
      disabled={disabled}
      data-testid={testId}
      onChange={(e) => onChange(e.target.value)}
      className="vl-num text-[12px] px-1.5 py-0.5"
      style={{
        background: 'var(--vl-card-2)',
        border: '1px solid var(--vl-hair)',
        borderRadius: 'var(--vl-radius-chip)',
        color: 'var(--vl-ivory)',
        opacity: disabled ? 0.5 : 1,
      }}
    />
  )
}

function FieldRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-3 py-1.5">
      <span
        className="text-[11px]"
        style={{ color: 'var(--vl-muted)', fontFamily: 'var(--vl-font-ui)' }}
      >
        {label}
      </span>
      <div className="flex items-center gap-2">{children}</div>
    </div>
  )
}

const MODE_OPTS = (language: Language) => [
  { key: 'advisory', label: tp('modeAdvisory', language) },
  { key: 'direction', label: tp('modeDirection', language) },
  { key: 'strict', label: tp('modeStrict', language) },
]

export function DayPlanEditor({ config, onChange, disabled, language }: Props) {
  const cfg = config ?? DEFAULT_DAY_PLAN
  const enabled = cfg.plan_enabled === true
  const [openSession, setOpenSession] = useState<SessionName | null>('NY')

  // An EOD flat set AFTER NY's window end reopens the P3 drift band: the session
  // gate would still call NY open on a day already flattened. Warn, don't clamp —
  // the owner may be running a session whose window genuinely ends later.
  const eodAfterWindow =
    (cfg.eod_flat_ct ?? '14:45').localeCompare(NY_WINDOW_END_CT) > 0

  const update = <K extends keyof DayPlanConfig>(
    key: K,
    value: DayPlanConfig[K]
  ) => {
    if (disabled) return
    onChange({ ...cfg, [key]: value })
  }

  // per-session override helpers (⚪ inherit = field absent, 🔸 override = set)
  const sessionOf = (s: SessionName): DayPlanSessionOverride | undefined =>
    cfg.sessions?.find((x) => x.session === s)
  const setSessionField = <K extends keyof DayPlanSessionOverride>(
    s: SessionName,
    field: K,
    value: DayPlanSessionOverride[K]
  ) => {
    const list = [...(cfg.sessions ?? [])]
    let i = list.findIndex((x) => x.session === s)
    if (i < 0) {
      list.push({ session: s })
      i = list.length - 1
    }
    list[i] = { ...list[i], [field]: value }
    update('sessions', list)
  }
  const clearSessionField = (
    s: SessionName,
    field: keyof DayPlanSessionOverride
  ) => {
    const list = (cfg.sessions ?? []).map((x) =>
      x.session === s ? { ...x, [field]: undefined } : x
    )
    update('sessions', list)
  }

  // Mirrors trader.sessionRunnable: an explicit per-session enable wins; otherwise
  // inherit the registry default AND the sessions_enabled subset (default [NY]).
  const sessionRunnableUI = (s: SessionName): boolean => {
    const ov = cfg.sessions?.find((x) => x.session === s)
    if (ov?.enable !== undefined) return ov.enable
    const band = SESSION_BANDS.find((b) => b.name === s)
    const subset = cfg.sessions_enabled ?? ['NY']
    return !!band?.enabled && subset.some((x) => x.toUpperCase() === s)
  }

  // planner-reads multiselect: toggle a TF in/out, preserving PLANNER_TFS order.
  const toggleTimeframe = (tf: string) => {
    const cur =
      cfg.planner_timeframes ?? DEFAULT_DAY_PLAN.planner_timeframes ?? []
    const next = cur.includes(tf)
      ? cur.filter((t) => t !== tf)
      : PLANNER_TFS.filter((t) => t === tf || cur.includes(t))
    update('planner_timeframes', next)
  }

  const bodyDisabled = disabled || !enabled

  return (
    <div
      className="flex flex-col gap-2"
      style={{ fontFamily: 'var(--vl-font-ui)' }}
    >
      {/* master switch */}
      <FieldRow label={tp('enableDayPlan', language)}>
        <Toggle
          on={enabled}
          onChange={(v) => update('plan_enabled', v)}
          disabled={disabled}
        />
      </FieldRow>

      <div
        style={{ opacity: bodyDisabled ? 0.55 : 1 }}
        className="flex flex-col gap-2"
      >
        {/* planner model (pinned id shown) */}
        <FieldRow label={tp('plannerModel', language)}>
          <input
            type="text"
            value={cfg.planner_model ?? ''}
            placeholder="inherit primary"
            disabled={bodyDisabled}
            onChange={(e) => update('planner_model', e.target.value)}
            className="vl-num text-[11px] w-40 px-1.5 py-0.5 text-right"
            style={{
              background: 'var(--vl-card-2)',
              border: '1px solid var(--vl-hair)',
              borderRadius: 'var(--vl-radius-chip)',
              color: 'var(--vl-ivory)',
            }}
          />
        </FieldRow>

        {/* plan mode segmented */}
        <FieldRow label={tp('planMode', language)}>
          <Segmented
            options={MODE_OPTS(language)}
            value={cfg.plan_mode ?? 'advisory'}
            onChange={(v) => update('plan_mode', v)}
            disabled={bodyDisabled}
          />
        </FieldRow>

        {/* planner reads — an EDITABLE timeframe multiselect (which structure
            TFs the planner summarizes). Applies at the NEXT read, never mid-plan. */}
        <div className="py-1.5">
          <span className="text-[11px]" style={{ color: 'var(--vl-muted)' }}>
            {tp('plannerReads', language)}
          </span>
          <div className="mt-1 flex flex-wrap gap-1">
            {PLANNER_TFS.map((tf) => {
              const on = (
                cfg.planner_timeframes ??
                DEFAULT_DAY_PLAN.planner_timeframes ??
                []
              ).includes(tf)
              return (
                <button
                  key={tf}
                  type="button"
                  role="switch"
                  aria-checked={on}
                  disabled={bodyDisabled}
                  onClick={() => toggleTimeframe(tf)}
                  className="vl-num text-[10px] px-2 py-1"
                  style={{
                    color: on ? 'var(--vl-gold)' : 'var(--vl-faint)',
                    border: `1px solid ${on ? 'var(--vl-gold-line)' : 'var(--vl-hair)'}`,
                    background: on ? 'var(--vl-gold-dim)' : 'transparent',
                    borderRadius: 'var(--vl-radius-chip)',
                    opacity: bodyDisabled ? 0.5 : 1,
                    cursor: bodyDisabled ? 'not-allowed' : 'pointer',
                  }}
                >
                  {tf}
                </button>
              )
            })}
          </div>
        </div>

        {/* one-line regime (read-only, AUTO — auto-computed, not a setting) */}
        <FieldRow label={tp('regime', language)}>
          <span
            className="text-[10px] uppercase cursor-help"
            style={{ color: 'var(--vl-faint)' }}
            title={tp('autoTooltip', language)}
          >
            {tp('autoLabel', language)}
          </span>
        </FieldRow>

        {/* filters */}
        <div
          className="mt-1 pt-2"
          style={{ borderTop: '1px solid var(--vl-hair)' }}
        >
          <span
            className="text-[10px] uppercase tracking-widest"
            style={{ color: 'var(--vl-faint)' }}
          >
            {tp('filters', language)}
          </span>
          <FieldRow label={tp('proximity', language)}>
            <input
              type="range"
              min={0.5}
              max={3.0}
              step={0.1}
              value={cfg.proximity_filter_atr ?? 1.5}
              disabled={bodyDisabled}
              onChange={(e) =>
                update('proximity_filter_atr', parseFloat(e.target.value))
              }
              style={{ width: 90 }}
            />
            <span
              className="vl-num text-[11px]"
              style={{ color: 'var(--vl-muted)' }}
            >
              {(cfg.proximity_filter_atr ?? 1.5).toFixed(1)}×
            </span>
          </FieldRow>
          <FieldRow label={tp('maxLevels', language)}>
            <NumberField
              value={cfg.max_levels ?? 8}
              min={3}
              max={12}
              onChange={(v) => update('max_levels', v)}
              disabled={bodyDisabled}
            />
          </FieldRow>
          <FieldRow label={tp('maxScenarios', language)}>
            <NumberField
              value={cfg.scenario_cap ?? 3}
              min={1}
              max={5}
              onChange={(v) => update('scenario_cap', v)}
              disabled={bodyDisabled}
            />
          </FieldRow>
          <FieldRow label={tp('maxReplans', language)}>
            <NumberField
              value={cfg.replan_cap ?? 2}
              min={0}
              max={4}
              onChange={(v) => update('replan_cap', v)}
              disabled={bodyDisabled}
            />
          </FieldRow>
          <FieldRow label={tp('acceptance', language)}>
            <Segmented
              options={[
                { key: '2x5m', label: '2×5m' },
                { key: '15m-close', label: '15m' },
              ]}
              value={cfg.acceptance_rule ?? '2x5m'}
              onChange={(v) => update('acceptance_rule', v)}
              disabled={bodyDisabled}
            />
          </FieldRow>
          <FieldRow label={tp('approval', language)}>
            <Toggle
              on={cfg.approval_required === true}
              onChange={(v) => update('approval_required', v)}
              disabled={bodyDisabled}
            />
          </FieldRow>
          <FieldRow label={tp('digest', language)}>
            <Toggle
              on={cfg.evening_digest !== false}
              onChange={(v) => update('evening_digest', v)}
              disabled={bodyDisabled}
            />
          </FieldRow>

          {/* W15.B — THE DAY-TRADER CLOCK. Both times were already read by the
              live gates (entryBlockedByLastEntry / enforceEODFlat) but had NO
              control: the only way to change them was a CLI. */}
          <FieldRow label={tp('lastEntry', language)}>
            <TimeField
              value={cfg.last_entry_ct ?? '13:00'}
              onChange={(v) => update('last_entry_ct', v)}
              disabled={bodyDisabled}
              testId="last-entry-ct"
            />
          </FieldRow>
          <FieldRow label={tp('eodFlat', language)}>
            <TimeField
              value={cfg.eod_flat_ct ?? '14:45'}
              onChange={(v) => update('eod_flat_ct', v)}
              disabled={bodyDisabled}
              testId="eod-flat-ct"
            />
          </FieldRow>
          {eodAfterWindow && (
            <div
              className="text-[10px] leading-snug pb-1"
              style={{ color: 'var(--vl-short)' }}
              data-testid="eod-flat-warning"
            >
              ⚠ {tp('eodFlatWarning', language)}
            </div>
          )}
          <FieldRow label={tp('realignCap', language)}>
            <NumberField
              value={cfg.realign_cap ?? 5}
              min={0}
              max={10}
              onChange={(v) => update('realign_cap', v)}
              disabled={bodyDisabled}
            />
          </FieldRow>
        </div>

        {/* sessions accordion */}
        <div
          className="mt-1 pt-2"
          style={{ borderTop: '1px solid var(--vl-hair)' }}
        >
          <span
            className="text-[10px] uppercase tracking-widest"
            style={{ color: 'var(--vl-faint)' }}
          >
            {tp('sessionsHeader', language)}
          </span>
          {ALL_SESSIONS.map((s) => {
            const ov = sessionOf(s)
            const band = SESSION_BANDS.find((b) => b.name === s)
            const isOpen = openSession === s
            const dst = s === 'LONDON' && londonDSTWarning()
            return (
              <div
                key={s}
                className="mt-1"
                style={{
                  border: '1px solid var(--vl-hair)',
                  borderRadius: 'var(--vl-radius-inner)',
                }}
              >
                <div className="w-full flex items-center gap-2 px-2.5 py-1.5">
                  {/* PART A — the per-session ENABLE toggle. Explicit ON/OFF is
                      authoritative for this strategy (🔸override); clearing it
                      returns the row to ⚪inherit (registry + sessions_enabled).
                      ASIA/LONDON inherit OFF, so they stay off until switched on. */}
                  <Toggle
                    testId={`session-enable-${s}`}
                    ariaLabel={`${s} session enabled`}
                    on={sessionRunnableUI(s)}
                    onChange={(on) => setSessionField(s, 'enable', on)}
                    disabled={bodyDisabled}
                  />
                  <button
                    onClick={() => setOpenSession(isOpen ? null : s)}
                    className="flex-1 flex items-center justify-between"
                    aria-expanded={isOpen}
                    disabled={bodyDisabled}
                  >
                    <span
                      className="text-[11px] font-semibold uppercase tracking-wide"
                      style={{ color: 'var(--vl-ivory)' }}
                    >
                      {tp(
                        s === 'ASIA'
                          ? 'sessionAsia'
                          : s === 'LONDON'
                            ? 'sessionLondon'
                            : 'sessionNY',
                        language
                      )}
                    </span>
                    <span className="flex items-center gap-2">
                      {ov?.enable !== undefined && (
                        <span
                          data-testid={`session-enable-chip-${s}`}
                          className="text-[9px]"
                          style={{ color: 'var(--vl-gold)' }}
                          title="explicit override — clear to inherit"
                        >
                          🔸
                        </span>
                      )}
                      <span
                        className="text-[10px]"
                        style={{ color: 'var(--vl-faint)' }}
                      >
                        {isOpen ? '▾' : '▸'}
                      </span>
                    </span>
                  </button>
                </div>
                {isOpen && (
                  <div className="px-2.5 pb-2 flex flex-col gap-1">
                    {/* ACTIVE windows (killzones, spec wording) */}
                    <div
                      className="text-[10px]"
                      style={{ color: 'var(--vl-faint)' }}
                    >
                      {band?.killzones.map((kz) => (
                        <span key={kz.name} className="vl-num mr-2">
                          {String(Math.floor(kz.startMin / 60)).padStart(
                            2,
                            '0'
                          )}
                          :{String(kz.startMin % 60).padStart(2, '0')}–
                          {String(Math.floor(kz.endMin / 60)).padStart(2, '0')}:
                          {String(kz.endMin % 60).padStart(2, '0')}
                        </span>
                      ))}
                      <span
                        className="uppercase"
                        style={{ color: 'var(--vl-gold)' }}
                      >
                        {tp('windows', language)}
                      </span>
                    </div>
                    {dst && (
                      <div
                        className="text-[10px]"
                        style={{ color: 'var(--vl-short)' }}
                      >
                        ⚠ {tp('dstWarning', language)}
                      </div>
                    )}
                    {/* override rows: min_grade · max_trades · plan_mode · replan_cap · acceptance */}
                    <OverrideRow
                      label={tp('minGrade', language)}
                      overridden={ov?.min_grade !== undefined}
                      onToggle={(on) =>
                        on
                          ? setSessionField(s, 'min_grade', 'B')
                          : clearSessionField(s, 'min_grade')
                      }
                      disabled={bodyDisabled}
                      language={language}
                    >
                      <Segmented
                        options={[
                          { key: 'A', label: 'A' },
                          { key: 'B', label: 'B' },
                          { key: 'C', label: 'C' },
                        ]}
                        value={ov?.min_grade}
                        onChange={(v) => setSessionField(s, 'min_grade', v)}
                        disabled={bodyDisabled}
                      />
                    </OverrideRow>
                    <OverrideRow
                      label={tp('maxTrades', language)}
                      overridden={ov?.max_trades !== undefined}
                      onToggle={(on) =>
                        on
                          ? setSessionField(s, 'max_trades', 3)
                          : clearSessionField(s, 'max_trades')
                      }
                      disabled={bodyDisabled}
                      language={language}
                    >
                      <NumberField
                        value={ov?.max_trades}
                        min={0}
                        max={20}
                        onChange={(v) => setSessionField(s, 'max_trades', v)}
                        disabled={bodyDisabled}
                      />
                    </OverrideRow>
                    <OverrideRow
                      label={tp('planMode', language)}
                      overridden={ov?.plan_mode !== undefined}
                      onToggle={(on) =>
                        on
                          ? setSessionField(s, 'plan_mode', 'advisory')
                          : clearSessionField(s, 'plan_mode')
                      }
                      disabled={bodyDisabled}
                      language={language}
                    >
                      <Segmented
                        options={MODE_OPTS(language)}
                        value={ov?.plan_mode}
                        onChange={(v) => setSessionField(s, 'plan_mode', v)}
                        disabled={bodyDisabled}
                      />
                    </OverrideRow>
                    <OverrideRow
                      label={tp('maxReplans', language)}
                      overridden={ov?.replan_cap !== undefined}
                      onToggle={(on) =>
                        on
                          ? setSessionField(s, 'replan_cap', 2)
                          : clearSessionField(s, 'replan_cap')
                      }
                      disabled={bodyDisabled}
                      language={language}
                    >
                      <NumberField
                        value={ov?.replan_cap}
                        min={0}
                        max={4}
                        onChange={(v) => setSessionField(s, 'replan_cap', v)}
                        disabled={bodyDisabled}
                      />
                    </OverrideRow>
                    <OverrideRow
                      label={tp('acceptance', language)}
                      overridden={ov?.acceptance_rule !== undefined}
                      onToggle={(on) =>
                        on
                          ? setSessionField(s, 'acceptance_rule', '2x5m')
                          : clearSessionField(s, 'acceptance_rule')
                      }
                      disabled={bodyDisabled}
                      language={language}
                    >
                      <Segmented
                        options={[
                          { key: '2x5m', label: '2×5m' },
                          { key: '15m-close', label: '15m' },
                        ]}
                        value={ov?.acceptance_rule}
                        onChange={(v) =>
                          setSessionField(s, 'acceptance_rule', v)
                        }
                        disabled={bodyDisabled}
                      />
                    </OverrideRow>
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>
    </div>
  )
}

// ⚪ inherit / 🔸 override chip + (when overridden) the inline control.
function OverrideRow({
  label,
  overridden,
  onToggle,
  disabled,
  language,
  children,
}: {
  label: string
  overridden: boolean
  onToggle: (on: boolean) => void
  disabled?: boolean
  language: Language
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-2 py-1">
      <button
        onClick={() => !disabled && onToggle(!overridden)}
        disabled={disabled}
        className="inline-flex items-center gap-1.5 text-[11px]"
        style={{ color: overridden ? 'var(--vl-gold)' : 'var(--vl-faint)' }}
        title={overridden ? tp('override', language) : tp('inherit', language)}
      >
        <span aria-hidden>{overridden ? '🔸' : '⚪'}</span>
        <span>{label}</span>
      </button>
      {overridden && <div className="flex items-center gap-1">{children}</div>}
    </div>
  )
}
