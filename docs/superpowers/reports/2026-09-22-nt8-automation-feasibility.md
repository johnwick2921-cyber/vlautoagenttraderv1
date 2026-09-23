# W-ONE-BUTTON M1: NT8 automation feasibility (throwaway spike, read-only pass)

**Lane:** Claude-101. **Branch:** `feat/one-button-updates` (claim `60d1ac24`). **Base:** origin/dev `0960a6ac` ("docs: record verified Planner chart frontend deployment").

**Dispatch:** CTO full dispatch msg `1790131309474-27377-000001` and rulings msg `1790131568827-56264-000001` (R2: assume no isolated install; M1 tonight = read-only facts plus a read-only UIA dump of the live NT8).

**Spec:** the owner's plan "One-button partner updates", relayed verbatim in msg `1790130922327-86971-000001`. It is not a tracked file, so `git log -1 -- <spec>` = n/a.

**Status (L14): BLOCKED on environment for steps 2–8. Step 1 is PROVEN.**
- There is no isolated SIM NT8 install, and the live NT8 is off-limits for anything mutating. Section 8 states once exactly what the owner must provision.
- M2 starts now in parallel, per R2.
- **Supported NT8 versions: none driven.** Step 1 read UI properties on 8.1.8.1 and drove nothing.

**Evidence tiers** (L9): **[A]** means run or read, with the command or file:line. **[B]** is inferred. **[C]** is speculation or background knowledge. Facts came from a read-only workflow: five investigators, a synthesis and a completeness critic. The critic's corrections are folded in and marked "(corrected)".

---

## 1. Verdict per M1 step

| Step (CTO dispatch §3 M1) | Verdict | Evidence / reason |
|---|---|---|
| 1. Read-only UIA tree dump of the live NT8 | **PROVEN** | See §6. `tools/nt8-spike/uia-tree-8.1.8.1.txt`: 191 nodes across 2 top-level windows (Control Center and Chart), 1.3 s. Property reads only. The guard showed NT8 pid 40804, start `2026-09-22T00:27:39.868`, and nofx MainPID 75597 with starttime ticks 35389515 **unchanged** before and after both passes (22:09:00 and 22:09:30 CT). [A] |
| 2. Open the NinjaScript Editor via UIA | **BLOCKED** (no isolated install) | `ControlCenterMenuItemNew` exists and supports ExpandCollapse. Its children are collapsed and cannot be listed without Expand, which is a mutating call and was not made. [A] |
| 3. Compile via the Editor's Compile control (InvokePattern) | **BLOCKED** | The Editor is not open on the live NT8, so its AutomationIds are unknown. |
| 4. Read "0 errors" from the output pane | **BLOCKED** | NT8 writes **nothing** about compiles to its log or trace files (§3), so the Editor grid is the only place errors show. |
| 5. Deliberate error → detect → restore → clean | **BLOCKED** | |
| 6. Graceful close via UIA | **BLOCKED** | `ConfirmWindowClose=true` in Config.xml [A]. The dialog is unverified. |
| 7. Relaunch, then hello with build_id | **BLOCKED**, and **leans BLOCKED even with an isolated install** | Every launch stops at the NT8 **platform login** window (§4). |
| 8. Restore the previous AddOn, then the old build_id on the wire | **BLOCKED** | See §5: today's build_id cannot tell builds apart. |

Per the dispatch, **none of steps 3–8 has FAILED, so the full-automation goal has not been declared blocked.** Findings F2 and F4 below make that outcome likely unless the owner rules on them.

---

## 2. Findings that change the plan (for CTO review and owner ruling)

Each finding lists the fail-closed default I am taking under R4.

**F1. The NT8 instance is not SIM-only, and the gate as specified cannot see most of what an NT8 restart affects.** [A]
- `Config.xml` has 7 connections, 4 with `ConnectOnStartup=true`. 3 of those are non-SIM, one of them a prop-firm FCM.
- 29 vendor assemblies load, including an Apex trade copier (`CopierLog.txt` exists).
- The log shows `Chart Trader submitting order without strategy` ×336 since 09-02.
- A Chart window with Chart Trader is open right now (§6).
- An NT8 compile or restart disconnects and reconnects every one of those accounts. The M2 aggregate gate, as specified, covers VL traders' accounts only.
- Partner machines will likely look similar [B].
- **Default:** the installation gate treats **every account the AddOn can see** as in scope. Any position or working order on any of them blocks, and NT8-affecting steps require explicit coverage of all connected accounts. If the AddOn cannot enumerate an account, that account is "unknown coverage" and blocks. Whether a flat but connected prop account may be disconnected by an update is an **owner ruling**. Until then, every NT8 step is blocked while a non-SIM connection is up.

**F2. Every NT8 launch goes through the platform login window, so unattended relaunch is unproven and leans BLOCKED.** [A]
- The trace starts at `LoginInternal`.
- Recorded failures: "Client app is prohibited" (08-27 ×2, 09-17 ×2, delaying start by 6 min); "Too many incorrect login attempts" with a CAPTCHA and a 30 s then 49 s penalty (09-06, where the process never got past login); "Incorrect username or password" (09-07).
- One automated relaunch sat at the Welcome window for more than 9 minutes.
- No stored auto-login exists: nothing in `Config.xml`, the only registry value under `HKCU\Software\NinjaTrader, LLC\NinjaTrader 8` is `Info`, and Credential Manager has 0 matching targets. [A; "no auto-login" is B]
- **Default:** the helper never enters credentials. A login window after relaunch means the job reports **BLOCKED: human login required**, with maintenance still held.

**F3. Canon correction: F5 hot-reloads the AddOn inside the running NT8.** [A/B]
- **After F5:** within about 10–30 ms the NT8 log shows the pair `VLTraderTCPClient: AddOn Terminated` → `AddOn Active`, with **no Session Break** between them. On 09-22 the pair came 443 ms after the DLL write at 00:27:10.428.
- **Inside one process** (trigger is [B]): `research_facts.source_build_id` changes from `2026-09-07-h1` (ids ≤102825383, last at 00:27:00.380) to `2026-09-20-p1` (from id 102825384, 00:27:06.768). The next NT8 process started only at 00:27:39. [A]
- **Copy + restart without F5 keeps the old DLL:** on 09-06 and 09-07, Go kept reporting `build_id=2026-09-03-f12 expected=2026-09-05-g2 match=NO` across two restarts until F5 at 09-07 21:42. [A]
- **Consequences:**
  - The CLAUDE.md line "AddOns do NOT hot-reload" is wrong.
  - **The flat gate must run before the compile, not only before the restart.**
  - A "compile without NT8 restart" path would avoid F2 entirely. The plan says restart, and I am **not** substituting silently.
- **Question for the owner via CTO:** is an in-process reload followed by build verification an acceptable NT8 update path, with full restart kept as an attended fallback?

**F4. "Reuse the existing AddOn build receipts" cannot prove the restarted AddOn as the code stands.** [A]
- `VL_BUILD_ID` is a hand-set constant (`ninjascript/VLTraderTCPClient.cs:55`, `"2026-09-20-p1"`) and has a Go twin (`provider/ninjatrader/order_snapshot.go:214`).
- The id `h1` covered at least 8 distinct source states: seven unbumped commits in the h1 era.
- `farSideBuild` (`provider/ninjatrader/tcp_server.go:85`) is process-global and is never cleared by `closeConn` (`:2442-2449`).
- There is no connection epoch and no process epoch.
- A Go restart yields a fresh hello from the **same** AddOn instance.
- **To prove "the restarted AddOn", a frame needs three things none of today's frames carry:**
  1. It arrived on a connection accepted after the old NT8 process was seen to exit.
  2. It carries the sending NT8 process identity (PID + StartTime).
  3. It carries a **content-derived** build identity (source hash + compiled assembly MVID).
- **Default:** additive, omitempty hello fields plus a per-connection Go record, with verification bound to that record. These go into the M2 AddOn change, alongside the `maintenance` frames, because that PR already touches the protocol. The plan says "not new protocol work", so this is flagged for ruling rather than done silently. Keep the ISO-date prefix of `VL_BUILD_ID`, because `FarSideProven` is a bytewise `>=` (`provider/ninjatrader/tcp_framing.go:274-276`).

**F5. NT8 logs nothing about compiles: no start, no success, no failure, no CS####.** [A]
- Zero `CS[0-9]{4}` matches in any log or trace file, including across the known 09-12 CS1503 failure.
- Detection must use artifacts: DLL/pdb/xml hash and mtime, the `Terminated→Active` pair, and the build received on a new epoch.
- **Compile duration, corrected:** two compiles on 09-22 took ≤7.95 s and ≤9.5 s (n=2).
- **Corrected:** the csproj rewrite is **not** a per-compile marker. The second compile left the csproj mtime at 00:26:53.015.
- **Corrected:** "the DLL hash is unchanged on failure" was never observed [C]. The failure signature ("no reload pair while broken") rests on about 2 events [B].

**F6. A second NT8 on the same Windows host is not isolated.** [A/B]
- The AddOn dials the constant `127.0.0.1:36974` (`VLTraderTCPClient.cs:37`). WSL networking is mirrored, and live ATI binds `0.0.0.0:36973`.
- A test AddOn on the same host would dial the **live** bot and could take its single client slot whenever the live NT8 reconnects [B].
- **Only a VM with its own loopback isolates** (§8).
- A VM NT8 logged in with the **same** NinjaTrader/Tradovate user might invalidate the live session: the live instance runs an `AuthenticatedUser.RunSchedulerAsync` token refresh [A trace / C effect].

**F7. A possible route with no compile step (for the owner's consideration, not adopted).**
- NT8 loads every vendor DLL in `bin\Custom` at startup without compiling [A]. There are 29, and `Apex.AAA` appears in no config or csproj [A].
- Building the VL AddOn offline into its own DLL would reduce an update to "stop NT8, copy the DLL, start". That removes UIA compile automation, and the DLL's MVID/hash becomes the build identity.
- Unknowns [C]:
  - whether NT8 accepts a DLL that was not exported through NT8;
  - duplicate types during the one-time transition;
  - whether vendor DLLs hot-reload.
- It still needs a restart, so F2 still applies.

**F8. F5 means different things in different windows.** [A]
- `Compile='F5'` in the NinjaScript Editor; ReloadNinjaScript in Market Analyzer, SuperDOM and Charts. A Chart window is open right now.
- SendKeys-based compile is unsafe. The dispatch already forbids it; this is the evidence for why.

**F9. NT8 has its own updater.** [A]
- `ControlCenterMenuItemHelpUpdates` ("Update") exists in the Control Center menu.
- There are 4 `NTAutoUpdateLog_*` and 3 `NTInstall_*` files in the user-data folder.
- An NT8 version change between releases would force a full recompile against new stock `@`-files [B].
- **Default:** the helper refuses unless `NinjaTrader.exe` FileVersion is in the manifest's tested range.

**F10. An unrelated compiled file has no source of record.** [A]
- `Strategies\vltrader.cs` (the legacy CSV strategy) is compiled into the same assembly. It has **never been in git**: `git log --all -- '*vltrader.cs'` is empty.
- An error there blocks every release compile, and the repo cannot restore it.
- **Default:** the helper hashes it, backs it up, and never modifies it. Its errors are reported, not suppressed.

---

## 3. Compile and restart behaviour (NT8's own logs)

Log paths: `C:\Users\hoang\Documents\NinjaTrader 8\{log,trace}`.

| Question | Finding | Tier |
|---|---|---|
| Compile markers in log or trace | None. The only hits are the `Compile='F5'` hotkey line and CLR stack frames. | A |
| Success evidence | New DLL/pdb/xml mtime and hash, the `Terminated→Active` pair, and a hello on the new build. On 09-22: DLL 00:27:10.428 → pair 00:27:10.871/.890 → `hello handshake OK` 00:27:10.968. | A |
| Failure evidence | Only "no reload pair" (about 2 events, 09-11/12). The DLL-on-failure behaviour and what a restart loads after a failed compile are unobserved. | B / C |
| Compile scope | All **301** `<Compile>` items build one assembly: 291 `@`-stock, 4 NT infrastructure, 5 release-owned `AddOns\VL*.cs`, and `Strategies\vltrader.cs`. An error anywhere blocks everything. | A |
| Startup compile | **Never.** Restarts with changed sources and no F5 kept the old DLL (09-06 06:51, 09-06 17:06; also 09-08, 09-14, 09-17, 09-19). `Copying custom assemblies` is followed about 6 ms later by `Loading …Custom.dll`. | A |
| Copying `.cs` while NT8 runs | No lock and no auto-compile. On 09-22 the files were written at 00:20:56 and nothing followed until F5 at 00:27:00. Whether an *open Editor* auto-compiles on an external change is unknown. | A / C |
| Start sequence (09-22) | Process start 00:27:39.868 → `LoginInternal … mode='Simulation'` 00:27:52.482 → `Copying custom assemblies` 00:27:55.494 → `Auto connecting` ×4 00:27:57.396 → `AddOn Active` 00:27:58.036 → `hello handshake OK (protocol v3)` 00:27:58.080 → `Restoring workspace 'Easy Open'` 00:27:58.199. About 19 s from start to hello, including about 12.6 s of login. | A |
| Reliable process-epoch markers | Get-Process PID and StartTime; trace `Session End` / `Session Start`; the AddOn's `AddOn Active`. **Not** the log's `Session Break`: 61-byte files also appear during shutdown. | A |
| Unclean exits | Windows Update (event 1074) and crashes (event 41) end the trace with no `Session End`. **NT8 never relaunches itself.** | A |
| Workspace on relaunch | `Easy Open.xml` was saved at close. It had `<NTWindows />` at 00:27:24, but a Chart window has been opened since (§6), so the next close saves it. | A |
| Each reload re-pulls history | After the 00:27:00 reload, 44 `RequestBars` calls in 6 s, daily back to 1950 (`trace.20260922.00001.txt:485-528`). Repeated compiles multiply data-provider load. | A |
| Stale state after an in-process reload | The AddOn's `State.Terminated` branch cancels its token, unhooks the account/order/position/static connection handlers, disposes BarsRequests and history pulls, and closes the socket (`VLTraderTCPClient.cs:300-313`). Runtime residue is untested. | A (code) / B |

---

## 4. Environment facts

| Fact | Tier |
|---|---|
| NT8 is **8.1.8.1** at `C:\Program Files\NinjaTrader 8\bin\NinjaTrader.exe`, with a single uninstall-registry entry (InstallDate 20260731). | A |
| User-data directory is `C:\Users\hoang\Documents\NinjaTrader 8`, not OneDrive-redirected (User Shell Folders `Personal` = GetFolderPath). | A |
| Running process: PID 40804, started 2026-09-22 00:27:39.868, SessionId 1, parent explorer.exe (launched by hand or a shortcut [B]), owner hoang. | A |
| Release-owned: 5 `AddOns\VL*.cs`, byte-identical to `ninjascript/*.cs` at `0960a6ac` (md5 and sha256). `AddOns` also holds 4 `*.cs.bak-20260911-*` files; they are not compiled because of their extension. | A |
| Compiled assembly `bin\Custom\NinjaTrader.Custom.dll`: 1,371,136 bytes, mtime 2026-09-22 00:27:10.428. The csproj is SDK-style, net48, LangVersion 13, and regenerated by NT8 at compile time. | A |
| Rollback source set: `~/nofx-backups/addon/20260922-002056/` holds all 5 `.cs` files. **(Corrected)** Their CRLF-stripped sha256 values equal the **`28ac9192`** state (09-12, reports `h1`). `38585854` changed only the protocol doc. **No DLL backup exists** for h1 or p1. | A |
| Toolchain: Windows PowerShell **5.1.26100.9444** only (no pwsh 7), execution policy RemoteSigned. .NET SDK 9.0.307 on the Windows side (not on the WSL PATH). MSBuild from VS BuildTools. NT8's own Roslyn. The framework `csc` v4 cannot build LangVersion 13. | A |
| Isolation vehicles: Windows 11 Pro 25H2, build 26200. Windows Sandbox and Hyper-V are disabled; HypervisorPlatform, VirtualMachinePlatform and WSL are enabled. One usable profile. D:\ has about 1725 GB free. | A |
| WSL 2.6.3.0, networking mirrored, systemd=true. `powershell.exe` from an interactive WSL shell runs in SessionId 1, UserInteractive, not elevated, and UIA loads. **Untested from `nofx.service`**, which has no `WSL_INTEROP` and no `/mnt/c` on PATH. | A / unknown |
| Helper hosting pattern already on this machine: an At-logon Interactive scheduled task (`\Claude\ClaudeCoworkWatchdog`). A Windows service runs in session 0 and cannot drive UIA. | A / C |

---

## 5. Received-build verification

- **Where build_id is sent:** hello (`VLTraderTCPClient.cs:1503-1513`), order_snapshot (`:1656-1662`) and heartbeat (`:2714-2717`, first beat after 30 s). Protocol doc: `ninjascript/vltrader_tcp_PROTOCOL.md:439-459`, `git log -1` = `38585854 2026-09-20 01:47:02 W-PICTURE-HTF (6/6)`. [A]
- **The capability floor is a bytewise string compare.** `FarSideProven(buildID, minBuild) = buildID != "" && buildID >= minBuild`. [A]
- **What passes wrongly today:**
  - After an NT8 restart, `FarSideBuildID()` returns the previous DLL's id until a new frame overwrites it. [A]
  - Restoring a pre-F12 build (heartbeat-only id) would leave the newer id cached for about 30 s. A pre-E7 build would leave it for the life of the process. [B]
  - The 🔌 boot line is one-shot per process (the `bootSweepDone` latch) and was seen stuck at `build_id=none` on 09-17. [A]
- **(Corrected) What does work:** for the actual M1 pair (p1 → h1 restore), the id changes on the restored instance's first hello. It is a usable but insufficient signal: it cannot tie the frame to a restart epoch or tell h1 source states apart. [A/B]
- **Design direction (F4, [C], to be built in M2's AddOn change if ruled):**
  - hello fields: `nt8_pid`, `nt8_start_ms`, `assembly_mvid`, `source_hash` (release-stamped over the 5 files), and `activation_nonce`.
  - Go per-connection record: accept sequence, monotonic accept time, remote port, and the hello fields.
  - The verifier binds to that record, never to `FarSideBuildID()`.

---

## 6. Step 1: UIA dump of the live NT8 (read-only)

**How it was run:**
- Script: `tools/nt8-spike/uia-dump.ps1`. Output: `tools/nt8-spike/uia-tree-8.1.8.1.txt`.
- Announced to CTO (msg `1790132896723-19287-000004`).
- Run at 22:09:00 CT (pass 1) and 22:09:30 CT (pass 2), outside 08:30–15:30.

**Rules the script follows:**
- Property reads only (`Current.*`, `GetSupportedPatterns`). No Invoke, SetFocus, Expand/Collapse, Select or Scroll.
- Subtrees of type DataGrid/DataItem/List/Table/Tree/Document/Edit are **not descended**. Neither are Custom classes matching `Grid|Account|Position|Order|Execution|Log`.
- Names are recorded **only** for interactive UI labels (Window, Menu/MenuItem, Button, TabItem, ToolBar, Header, CheckBox, RadioButton, Hyperlink, TitleBar). Every other element records its name length only.
- Any run of 4+ digits is masked.
- Result: no account or connection name is in the file, and the leak check (every Text/Custom/Pane/Group/StatusBar/ComboBox element with a literal name) returned nothing. [A]

**Pass 1** skipped the whole Control Center, because its top-level element is UIA ControlType **Custom** (`aid='ControlCenter'`, class `ControlCenter`). That is a finding for any walker. **Pass 2** descends into it. [A]

**AutomationIds found (8.1.8.1)** [A]:
- **Control Center menu (`PART_Menu`):**
  - `ControlCenterMenuItemNew` ("New", ExpandCollapse), the route to the NinjaScript Editor; its children are collapsed and not listed;
  - `ControlCenterMenuItemAdmin`, `…Tools`, `…Workspaces`, `…Connections`, `…Help`, `ControlCenterMenuItemChat` (Invoke);
  - `ControlCenterMenuItemHelpUpdates` ("Update", Invoke), NT8's own updater (F9).
- **Control Center tabs:** `OrdersGridTabItem`, `ExecutionsGridTabItem`, `StrategiesGridTabItem`, `PositionsGridTabItem`, `AccountsGridTabItem` (grid `AccountsGrid` → `grdAccount`, skipped), `LogGridTabItem`, `MessagesGridTabItem` ("Messages - 4").
- **Window chrome:** `NTWindowButtonMinimize`, `NTWindowButtonMaximize`, `NTWindowButtonClose` ("Close", Invoke). The exit route for step 6 would be `NTWindowButtonClose` or `File → Exit`. The confirmation dialog (`ConfirmWindowClose=true`) is unverified.
- **An open Chart window** (`aid='ChartWindow'`, "Chart - MNQ 12-26") with Chart Trader. `ChartTraderControlAccountSelector` (name empty, value not read) and Quick Buy/Sell/Reverse/Close buttons, all with InvokePattern. **These are live order buttons.** Any future helper must scope its UIA search to the Editor window's subtree, never search from the desktop root by Name.
- The NinjaScript Editor was **not open**, so its Compile control and error-grid AutomationIds are still unknown. Finding them needs the isolated install (step 2).

**Guard:** before and after both passes, nofx MainPID 75597 with starttime ticks 35389515, and NT8 pid 40804 with start `2026-09-22T00:27:39.8680177-05:00`, were identical. [A]

**Tooling note:** Windows PowerShell 5.1 reads a UTF-8 `.ps1` without BOM as ANSI. The first attempt failed to **parse** because of a non-ASCII dash, so nothing ran and the guard was unchanged. The script is now pure ASCII. [A]

---

## 7. Service, restart and CWD-relative resources (feeds M4 and CTO correction C2)

- **Unit:** `/etc/systemd/system/nofx.service`, generated by `install-autostart.sh`: Type=simple, User=hoang, `WorkingDirectory=/home/hoang/nofx`, `ExecStart=/home/hoang/nofx/nofx-bin`, Restart=on-failure, RestartSec=5, StartLimitIntervalSec=0. No Environment or EnvironmentFile. [A]
- **Process identity without sudo:**
  - `systemctl show -p MainPID --value nofx` = 75597.
  - `/proc/75597/stat` field 22 = 35389515 ticks (CLK_TCK 100), within 30 ms of systemd's monotonic start.
  - `readlink /proc/75597/exe` is readable.
  - `go version -m /proc/75597/exe` gives `a2bac00d…`, modified=false.
  - **Wall-clock start times disagree by 151 s** (a WSL clock step), so restart detection must use (pid, starttime ticks), never wall clock. [A]
- **CWD-relative census** (52 in-module packages, 612 non-test files) [A]:
  - **Release-owned:**
    - `nofx-bin`;
    - `deploy/RELEASE` (`kernel/boot_integrity.go:90`, read once at boot);
    - `web/dist` (`api/ui_serving.go:35`). This one is resolved **per request** by `http.Dir`, so a swap goes live immediately, and it was hot-swapped at 16:05:13 on 09-22 with no record.
  - **Mutable, must stay at the installation:**
    - `.env`, read at boot **and written** by `api/handler_onboarding.go:241-246,339`;
    - `data/nofx_<date>.log`, created before `.env` loads;
    - `data/data.db`. `main.go:71-76` runs MkdirAll, so **a wrong working directory silently creates an empty DB and boots "healthy"**;
    - `data/data.db.research.db`;
    - `data/clock-guard-state.json`, read relative but written with an absolute path.
  - **Ambiguous, needs an owner ruling in M4:** `calendar_static_t1.json`. It is git-tracked, but its comment calls it both "repo-shipped template" and "owner-editable".
  - **Rule:** never symlink the working directory itself. The process would pin the release inode, and `data/` and `.env` would resolve inside the release. [B]
- **Second UI surface:** `nofx-web.service` runs `npm run dev` from the deploy tree's `web/`. No release layout versions it. [A]
- **Rollback today:**
  - `~/nofx-backups/cutover-auto-rollback-v3.sh` covers `nofx-bin` and RELEASE (plus a bars key), with a 90 s `BOOT INTEGRITY OK` watch and `/api/health` check. It does not restore `web/dist`, HEAD or the DB. It refuses when `nofx-bin.old.$LIVE` already exists, and 67 `nofx-bin.*` files (4.6 G) sit in the tree.
  - `RESTORE.md` and `leveltruth-cutover.sh` use `pgrep -f nofx-bin`, which also matches `go version -m nofx-bin`.
  - Journald for uid 1000 keeps about 4 h. Boot proof survives only in `data/nofx_<boot-date>.log`. [A]

---

## 8. What the isolated SIM install must provide (owner side, stated once)

1. **A Windows VM with its own loopback.**
   - A second user on this host is **not** isolated (F6).
   - Options:
     - Hyper-V, which needs admin and a **reboot** of this host, killing the live NT8 and the bot, so it must be scheduled in a flat window;
     - a third-party hypervisor on the already-enabled HypervisorPlatform [C: may avoid the reboot];
     - a second physical machine or a cloud Windows VM.
   - Windows Sandbox is not persistent, so it would need a fresh install and login every run.
   - Disk is available (D:\ about 1725 GB).
2. **NT8 8.1.8.1 exactly**, and no NT8 auto-update inside the VM.
3. **A NinjaTrader platform login for the VM**, entered by the owner. The owner must also answer:
   - (a) whether the license allows a second concurrent instance;
   - (b) whether the VM must use a **different** NinjaTrader/Tradovate user so the live session's token refresh is never disturbed (F6). Default: a different user.
4. **Connections: Sim101 only.** No broker or prop connections, nothing ConnectOnStartup except SIM, ATI disabled or on another port, and no vendor DLLs (no Apex).
5. **Market data for bars:** Playback or a simulated feed, because a SIM-only VM has no real-time data [C]. The verifier needs a bar frame on the new epoch, not only a hello.
6. **`bin\Custom`:** stock NT8 plus the 5 `VL*.cs` at `0960a6ac`, plus **one deliberately unrelated dummy indicator** so error attribution for non-release files can be tested.
7. **`%USERPROFILE%\NofxTrader\account.txt`** inside the VM naming the SIM account (`VLTraderTCPClient.cs:411-417`).
8. **A Go probe listener on the VM's `127.0.0.1:36974`**, built by me (spike S1). The VM must be able to run it: either a WSL instance or the Windows Go toolchain.
9. **An interactive, logged-on desktop session** with PowerShell 5.1 and UIA. VM checkpoints so each run resets. A channel from this host to trigger spikes and fetch receipts.
10. **Written owner authorization** that the helper may run UIA, compile, close and relaunch **only inside the VM NT8**. The live PID stays out of scope.

---

## 9. Spike harness design (runs only on the §8 VM)

**Global guard:** record the live NT8 (PID + StartTime) and the live bot (MainPID + starttime ticks) before and after every spike. Any change fails the run.

| Spike | What it proves | Pass criterion |
|---|---|---|
| S0 isolation | No VM → live contact | From the VM, `127.0.0.1:36974` refuses when the probe is down; the live bot log shows no `client connected` or `rejecting concurrent client` during the run; the live guard is unchanged. |
| S1 probe listener | A hello/heartbeat sink on the VM that logs an accept sequence, monotonic accept time, remote port and hello fields, and accepts concurrent clients so duplicate AddOn instances show up | The installed AddOn's hello is received. |
| S2 compile success (steps 2–4) | UIA: `ControlCenterMenuItemNew` → Editor → Compile (Invoke). Read "0 errors" from the Editor pane. | DLL hash changes; `Terminated→Active` pair; a new accept sequence with the new spike stamp. All within 3× the measured compile time (≈8–10 s, so 30 s), and exactly one AddOn connection afterwards. |
| S3 compile failure (step 5) | (a) an error in a release file; (b) an error in the dummy file; (c) both. A shadow `dotnet build` of a copy for parity. | Failure declared within the timeout, with per-file error text read from the Editor grid. (b) is reported as non-release and not suppressed. If shadow parity fails, the shadow build may only advise. |
| S4 full restart (steps 6–7) | UIA close, an explicit answer to the confirm dialog, `Start-Process`, login | New PID and StartTime; hello on an accept after the old exit; no human input. **At most 2–3 attempts per window** (CAPTCHA risk). A login that needs a human is **BLOCKED**, not a retry. |
| S5 epoch binding | Verifier plus the F4 hello fields | Proven only for a frame on a post-exit accept whose PID/StartTime equals Get-Process and whose stamp is the expected one. Negatives: cached id after close; probe restart only; F5 with no restart; restore of a build that omits build_id. **All must stay unproven.** |
| S6 restore (step 8) | Snapshot the 5 `.cs` plus DLL/pdb/xml before install. R1: restore the DLL with NT8 stopped. R2: restore sources, recompile, restart. Probe: replace the DLL while NT8 runs. | The received identity equals the snapshot on a new epoch, and source and binary agree. |
| S7 session lock | Lock and unlock; sample `quser`, LogonUI and WTS flags; try a harmless UIA read | At least one signal differs deterministically. Record whether UIA fails while locked. |

---

## 10. Open unknowns

- Whether a UIA Invoke of the Editor's Compile control works, and whether it can run without bringing the window to the foreground.
- What a failed compile does to the DLL, pdb and csproj, and what a restart then loads.
- Whether an open Editor auto-compiles on an external change, possibly partway through a 5-file copy.
- Whether an external DLL replacement is allowed while NT8 runs. The module list cannot answer this: IL-only assemblies are not listed.
- The `ConfirmWindowClose` dialog, and the steady 5.1 s pause between `Finalizing addon` and `Shutting down`.
- What NT8 does at startup when `NinjaTrader.Custom.dll` is missing.
- Unattended login (F2); license terms for a VM; whether a same-user concurrent login is safe.
- Whether `nofx.service` or a `--user` unit can reach WSL interop.
- A verified lock-detection signal.
- Whether any of the 291 `@`-files were hand-edited, since their contents were not compared with a pristine set.
- Provenance of the a2bac00d cutover at 00:48 and of the dirty `store/armed_orders.go` (preserved per owner ruling).

---

## 11. What I did NOT do (L14)

- No write to `/home/hoang/nofx`. No write under `/mnt/c`.
- No compile, F5, restart, close, Invoke, SetFocus, Expand or keystroke against the live NT8.
- No signal to any process, no lock acquire, no RELEASE edit, no merge.
- No DB writes. Every sqlite read was `-readonly`.
- Nothing claimed as "supported" beyond "8.1.8.1: UI properties read only".
- Account and connection names are not reproduced anywhere in this report or the dump.
- The Picture HTF verification paste was not acted on (R1).
