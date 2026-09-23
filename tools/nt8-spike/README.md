# tools/nt8-spike — W-ONE-BUTTON M1 (throwaway feasibility spike)

Report: `docs/superpowers/reports/2026-09-22-nt8-automation-feasibility.md`.

## What is here

| File | What it is |
|---|---|
| `uia-dump.ps1` | READ-ONLY UI Automation dump of every top-level window of one NT8 process. Property reads only (`Current.*`, `GetSupportedPatterns`); never Invoke / SetFocus / Expand / Select / Scroll. Does not descend DataGrid/DataItem/List/Table/Tree/Document/Edit subtrees or Custom classes matching `Grid|Account|Position|Order|Execution|Log`; keeps names only for interactive UI labels (others: length only); masks 4+ digit runs. Pure ASCII (Windows PowerShell 5.1 reads BOM-less .ps1 as ANSI). |
| `uia-tree-8.1.8.1.txt` | Its output against the live NT8 8.1.8.1 (pid 40804), 2026-09-22 22:09:30 CT, with the before/after identity guard unchanged. |

## How it was run (the guard is part of the procedure)

```bash
guard() { local p=$(systemctl show -p MainPID --value nofx); echo "nofx MainPID=$p ticks=$(awk '{print $22}' /proc/$p/stat) | $(powershell.exe -NoProfile -Command "\$p=Get-Process NinjaTrader; 'NT8 pid=' + \$p.Id + ' start=' + \$p.StartTime.ToString('o')" | tr -d '\r')"; }
B=$(guard)
timeout 180 powershell.exe -NoProfile -ExecutionPolicy Bypass -File "$(wslpath -w tools/nt8-spike/uia-dump.ps1)" -TargetPid <pid> > out.raw
A=$(guard); [ "$B" = "$A" ] || echo "GUARD CHANGED - STOP"
```

## Rules for anything added here later

- Mutating spikes (steps 2-8: open Editor, compile, close, relaunch, restore) run ONLY inside the isolated VM described in the report section 8 — never against the live NT8, never on the owner's host desktop.
- Every spike records the live-host guard before/after; any change fails the run.
- Scope every UIA search to the target window's subtree (the live Chart window exposes Chart Trader Buy/Sell/Close buttons with InvokePattern).
- No SendKeys, no screen coordinates, no `keybd_event` (F5 means ReloadNinjaScript outside the Editor).
- Never write account names, connection names, credentials or tokens into this public repo.
