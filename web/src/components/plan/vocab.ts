// P5.2 — the controlled vocabulary for owner level edits. Stored VALUES are the
// canonical English tokens (replay-log consistency, design-system Open Q #4); the
// tp() label keys translate the display only.

import type { PlanKey } from '../../i18n/plan-translations'

export interface VocabItem {
  value: string
  key: PlanKey
}

// Level TYPE (the edit-sheet "TYPE" row).
export const LEVEL_TYPES: VocabItem[] = [
  { value: 'D-zone', key: 'typeDzone' },
  { value: 'S-zone', key: 'typeSzone' },
  { value: 'Level', key: 'typeLevel' },
  { value: 'Liquidity', key: 'typeLiquidity' },
  { value: 'My level', key: 'typeMyLevel' },
]

// INSTRUCTION verb — canonical tokens the executor understands.
export const INSTRUCTION_VERBS: VocabItem[] = [
  { value: 'sweep_reclaim', key: 'instrSweepReclaim' },
  { value: 'watch_reclaim', key: 'instrWatchReclaim' },
  { value: 'fade_first_touch', key: 'instrFadeFirst' },
  { value: 'target_only', key: 'instrTargetOnly' },
  { value: 'magnet', key: 'instrMagnet' },
  { value: 'no_touch', key: 'instrNoTouch' },
]

export const GRADES = ['A', 'B', 'C']
