// guardedCall — the shared never-latch pattern for long-await UI actions
// (hotfix/reset-dialog-stuck, 2026-08-19; extracted from the #55 Ask-Planner
// fix per the twin-path rule).
//
// The bug class it kills: `setBusy(true); await api.x(); setBusy(false)` — any
// throw (the 30s axios default vs a 60–300s planner-backed backend call, a
// network blip) skips the reset, latches `busy`, freezes the dialog, and only
// F5 recovers. Worse: the frozen dialog invites a second click, and the
// backend call usually LANDED — the owner reset log showed same-pair double
// executions minutes apart.
//
// Contract: guardedCall NEVER throws. Call sites do
//   if (busy) return            // double-click dedupe
//   setBusy(true)
//   try { const g = await guardedCall(() => api.x(...)); ... } finally { setBusy(false) }
// and treat g.ok===false as "toast + recover the dialog".

export type GuardedResult<T> =
  | { ok: true; value: T }
  | { ok: false; error: string }

export async function guardedCall<T>(
  call: () => Promise<T>
): Promise<GuardedResult<T>> {
  try {
    return { ok: true, value: await call() }
  } catch (e) {
    const raw = e instanceof Error ? e.message : String(e)
    return { ok: false, error: friendlyLongActionError(raw) }
  }
}

// friendlyLongActionError — timeouts/network blips on long planner-backed
// actions very often mean "the server is still working and the action LANDED";
// say so instead of a bare "Network error".
export function friendlyLongActionError(raw: string): string {
  if (/timeout|network/i.test(raw)) {
    return `The server did not answer in time — long planner actions can take minutes and usually STILL LAND. The view will refresh; check before retrying. (${raw})`
  }
  return raw
}
