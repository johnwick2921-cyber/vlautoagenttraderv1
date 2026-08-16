// P5.2 — the edit sheet (bottom-sheet). Tap a level row → edit it (an RFC-6902
// overlay replace/remove on the plan doc, armored server-side). The ＋ Add mode
// creates a STICKY OWNER LEVEL (note + scenario tag ride to the planner) that
// appears at the next read with 👤. createPortal + own Esc/focus (no shared
// modal helper exists). Tokens only (vl design system).

import { useEffect, useRef, useState } from 'react'
import { createPortal } from 'react-dom'
import { toast } from 'sonner'
import type { Language } from '../../i18n/translations'
import { tp } from '../../i18n/plan-translations'
import { api } from '../../lib/api'
import type { PlanLevelFact, RealignChange } from '../../lib/api/plan'
import { LEVEL_TYPES, INSTRUCTION_VERBS, GRADES } from './vocab'
import { fmtPrice } from './levelState'

interface Props {
  open: boolean
  traderId: string
  symbol: string
  language: Language
  // editing an existing card level: its fact + index in the plan doc; omitted = Add.
  level?: PlanLevelFact
  levelIndex?: number
  scenarioIds?: string[]
  onClose: () => void
  onSaved: (change?: RealignChange) => void // parent re-fetches + W13 re-align
}

function ChipRow({
  items,
  value,
  onChange,
}: {
  items: { value: string; label: string }[]
  value: string
  onChange: (v: string) => void
}) {
  return (
    <div className="flex flex-wrap gap-1.5">
      {items.map((it) => {
        const on = value === it.value
        return (
          <button
            key={it.value}
            onClick={() => onChange(it.value)}
            className="text-[11px] px-2.5 py-1"
            style={{
              borderRadius: 8,
              border: `1px solid ${on ? 'var(--vl-gold-line)' : 'var(--vl-hair)'}`,
              background: on ? 'var(--vl-gold-dim)' : 'transparent',
              color: on ? 'var(--vl-ivory)' : 'var(--vl-faint)',
            }}
          >
            {it.label}
          </button>
        )
      })}
    </div>
  )
}

export function EditSheet({
  open,
  traderId,
  symbol,
  language,
  level,
  levelIndex,
  scenarioIds = [],
  onClose,
  onSaved,
}: Props) {
  const isEdit = level != null && typeof levelIndex === 'number'
  const [price, setPrice] = useState('')
  const [type, setType] = useState('Level')
  const [instruction, setInstruction] = useState('watch_reclaim')
  const [grade, setGrade] = useState('B')
  const [note, setNote] = useState('')
  const [scenarioTag, setScenarioTag] = useState('')
  const [busy, setBusy] = useState(false)
  const firstRef = useRef<HTMLInputElement>(null)

  // seed from the tapped level; reset on open.
  useEffect(() => {
    if (!open) return
    setPrice(level ? fmtPrice(level.price) : '')
    setGrade(level?.grade || 'B')
    setInstruction(level?.instruction || 'watch_reclaim')
    setType('Level')
    setNote(level?.note || '')
    setScenarioTag(level?.scenario_id || '')
    setBusy(false)
    setTimeout(() => firstRef.current?.focus(), 30)
  }, [open, level])

  // Esc closes.
  useEffect(() => {
    if (!open) return
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [open, onClose])

  if (!open) return null

  const priceNum = parseFloat(price)
  const validPrice = !Number.isNaN(priceNum) && priceNum > 0

  const save = async () => {
    if (!validPrice || busy) return
    setBusy(true)
    if (isEdit) {
      // overlay replace on the plan doc level.
      const res = await api.postOverlay(
        traderId,
        [
          {
            op: 'replace',
            path: `/levels/${levelIndex}`,
            value: { price: priceNum, label: level!.label, grade, instruction },
          },
        ],
        'owner',
        symbol
      )
      setBusy(false)
      if (!res.ok) {
        toast.error(tp('saveFailed', language), { description: res.error })
        return
      }
      toast.success(tp('overlayApplied', language))
    } else {
      // Add = sticky owner level (appears at the next read with 👤).
      const label = type === 'My level' ? '👤' : type
      const res = await api.addOwnerLevel(
        traderId,
        { price: priceNum, label, note, scenario_tag: scenarioTag },
        symbol
      )
      setBusy(false)
      if (!res.ok) {
        toast.error(tp('saveFailed', language), { description: res.error })
        return
      }
      toast.success(tp('overlayApplied', language))
    }
    // W13 — hand the planner exactly what changed (its NOTE included).
    onSaved({
      kind: isEdit ? 'edit-level' : 'add-level',
      summary: `${isEdit ? 'changed' : 'added'} ${type || 'level'} @ ${priceNum}`,
      price: priceNum,
      label: isEdit ? level?.label : type,
      grade,
      instruction,
      note,
      scenario_tag: scenarioTag,
    })
    onClose()
  }

  const del = async () => {
    if (!isEdit || busy) return
    setBusy(true)
    const res = await api.postOverlay(
      traderId,
      [{ op: 'remove', path: `/levels/${levelIndex}` }],
      'owner',
      symbol
    )
    setBusy(false)
    if (!res.ok) {
      toast.error(tp('saveFailed', language), { description: res.error })
      return
    }
    toast.success(tp('overlayApplied', language))
    onSaved({
      kind: 'delete-level',
      summary: `deleted ${level?.label ?? 'level'} @ ${level?.price ?? ''}`,
      price: level?.price,
      label: level?.label,
    })
    onClose()
  }

  const sheet = (
    <div
      className="fixed inset-0 z-[60] flex items-end sm:items-center justify-center"
      style={{ background: 'rgba(0,0,0,0.6)' }}
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-label={
          isEdit ? tp('editLevel', language) : tp('addLevel', language)
        }
        className="vl-sheet-in w-full sm:max-w-[430px]"
        style={{
          background: 'var(--vl-card-2)',
          border: '1px solid var(--vl-gold-line)',
          borderRadius: '16px 16px 14px 14px',
          overflow: 'hidden',
          fontFamily: 'var(--vl-font-ui)',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <div
          className="mx-auto mt-2 mb-1"
          style={{
            width: 36,
            height: 4,
            borderRadius: 2,
            background: 'var(--vl-faint)',
            opacity: 0.5,
          }}
        />
        <div
          className="flex items-center gap-2 px-4 py-2"
          style={{ borderBottom: '1px solid var(--vl-hair)' }}
        >
          <span
            style={{
              fontFamily: 'var(--vl-font-display)',
              fontSize: 18,
              fontWeight: 600,
              flex: 1,
              color: 'var(--vl-ivory)',
            }}
          >
            {isEdit ? tp('editLevel', language) : tp('addLevel', language)}
          </span>
          <span
            className="text-[9px] font-bold"
            style={{
              color: 'var(--vl-gold)',
              border: '1px solid var(--vl-gold-line)',
              background: 'var(--vl-gold-dim)',
              borderRadius: 5,
              padding: '3px 7px',
            }}
          >
            👤 OWNER
          </span>
        </div>

        <div
          className="px-4 py-2 flex flex-col gap-3"
          style={{ maxHeight: '70vh', overflowY: 'auto' }}
        >
          <label className="flex flex-col gap-1">
            <span
              className="text-[9.5px] font-semibold tracking-wider uppercase"
              style={{ color: 'var(--vl-faint)' }}
            >
              {tp('priceRange', language)}
            </span>
            <input
              ref={firstRef}
              value={price}
              onChange={(e) => setPrice(e.target.value)}
              inputMode="decimal"
              className="vl-num text-[15px] px-3 py-2"
              style={{
                background: 'rgba(201,162,75,0.05)',
                border: '1px solid var(--vl-gold-line)',
                borderRadius: 8,
                color: 'var(--vl-ivory)',
              }}
            />
          </label>

          {!isEdit && (
            <div className="flex flex-col gap-1">
              <span
                className="text-[9.5px] font-semibold tracking-wider uppercase"
                style={{ color: 'var(--vl-faint)' }}
              >
                {tp('levelType', language)}
              </span>
              <ChipRow
                items={LEVEL_TYPES.map((t) => ({
                  value: t.value,
                  label: tp(t.key, language),
                }))}
                value={type}
                onChange={setType}
              />
            </div>
          )}

          <div className="flex flex-col gap-1">
            <span
              className="text-[9.5px] font-semibold tracking-wider uppercase"
              style={{ color: 'var(--vl-faint)' }}
            >
              {tp('instructionLabel', language)}
            </span>
            <ChipRow
              items={INSTRUCTION_VERBS.map((v) => ({
                value: v.value,
                label: tp(v.key, language),
              }))}
              value={instruction}
              onChange={setInstruction}
            />
          </div>

          <div className="flex flex-col gap-1">
            <span
              className="text-[9.5px] font-semibold tracking-wider uppercase"
              style={{ color: 'var(--vl-faint)' }}
            >
              {tp('gradeLabel', language)}
            </span>
            <ChipRow
              items={GRADES.map((g) => ({ value: g, label: g }))}
              value={grade}
              onChange={setGrade}
            />
          </div>

          <label className="flex flex-col gap-1">
            <span
              className="text-[9.5px] font-semibold tracking-wider uppercase"
              style={{ color: 'var(--vl-faint)' }}
            >
              {tp('noteLabel', language)}
            </span>
            <input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="…"
              className="text-[12px] px-3 py-2"
              style={{
                background: 'rgba(201,162,75,0.05)',
                border: '1px solid var(--vl-gold-line)',
                borderRadius: 8,
                color: 'var(--vl-muted)',
              }}
            />
          </label>

          {!isEdit && scenarioIds.length > 0 && (
            <div className="flex flex-col gap-1">
              <span
                className="text-[9.5px] font-semibold tracking-wider uppercase"
                style={{ color: 'var(--vl-faint)' }}
              >
                {tp('scenarioTag', language)}
              </span>
              <ChipRow
                items={[
                  ...scenarioIds.map((s) => ({ value: s, label: s })),
                  { value: 'new', label: tp('newPlay', language) },
                ]}
                value={scenarioTag}
                onChange={setScenarioTag}
              />
            </div>
          )}

          <div
            className="text-[10.5px] px-3 py-2"
            style={{
              color: 'var(--vl-short)',
              border: '1px solid rgba(224,108,108,0.25)',
              background: 'rgba(224,108,108,0.06)',
              borderRadius: 9,
            }}
          >
            {tp('armorWarn', language)}
          </div>
        </div>

        <div
          className="flex gap-2 px-4 py-3"
          style={{ borderTop: '1px solid var(--vl-hair)' }}
        >
          {isEdit && (
            <button
              onClick={del}
              disabled={busy}
              className="text-[12px] font-semibold px-4 py-2.5"
              style={{
                border: '1px solid rgba(224,108,108,0.3)',
                color: 'var(--vl-short)',
                borderRadius: 9,
              }}
            >
              {tp('deleteLevel', language)}
            </button>
          )}
          <button
            onClick={onClose}
            className="text-[12px] px-3 py-2.5"
            style={{
              color: 'var(--vl-muted)',
              border: '1px solid var(--vl-hair)',
              borderRadius: 9,
            }}
          >
            {tp('cancel', language)}
          </button>
          <button
            onClick={save}
            disabled={!validPrice || busy}
            className="flex-1 text-[12px] font-semibold px-4 py-2.5"
            style={{
              background:
                validPrice && !busy ? 'var(--vl-gold)' : 'var(--vl-faint)',
              color: 'var(--vl-ink)',
              borderRadius: 9,
              opacity: busy ? 0.6 : 1,
            }}
          >
            {tp('save', language)}
          </button>
        </div>
      </div>
    </div>
  )

  return createPortal(sheet, document.body)
}
