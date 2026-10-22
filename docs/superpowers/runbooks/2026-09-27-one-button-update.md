names rewritten to vl on 2026-09-30 (VL rename)
# One-button update: operator runbook (UPDATER-USABLE-V1, 2026-09-27)

**What this is:** the owner-side procedure for the one-button update — from a
signed release to the bot running it, using only this box. The enrollment and
the `authorize` mechanics live in
`docs/superpowers/runbooks/2026-09-24-m3-update-enrollment.md`; this file is the
whole loop, first-time setup through recovery.

**Status note (reads the truth, not a hope):** while
`INSTALL_AUTHZ_UNDER_REVIEW` is `true` in `web/src/lib/api/updates.ts` (the CTO
flips it in a separate commit after his adversarial pass), the Updates page
shows "install authorization under review" and no install can be started from
the page, whatever the server answers. Everything below still works end to end
on the CLI; the button lights up the moment that constant flips and the server
answers `install_enabled: true` + `worker_listening: true`.

## 1. First-time setup (once)

Run ONE script on the partner machine, then every release is one button:

```
bash deploy/partner-first-time-setup.sh <your exact account email>
```

The script is idempotent and attended, no sudo. It prints each step and
REFUSES with a reason on any failure:

1. checks prerequisites — `go`, `git`, `curl`, `systemctl --user` + linger,
   the bot install dir, and that `deploy/release_allowed_signers` holds the
   PARTNER key (never the owner repo's);
2. runs the attended `vl-updater-bootstrap --install-dir <bot> enroll <email>`
   (writes the enrollment files + the worker token into
   `~/.config/vl-updater/env`, mode 0600 — the token is never printed);
3. writes `VL_RELEASE_DIR` (absolute, OUTSIDE the install) beside the token
   line, preserving every other line byte-identically;
4. sets `VL_UPDATER=1` in the bot's `.env` (backed up first, other lines kept),
   then you restart the bot to read it;
5. runs `deploy/install-updater-worker.sh` at the CURRENT partner main sha
   (builds `~/bin/vl-updater` from that exact clean commit in a throwaway
   clone) and starts the worker: `systemctl --user start vl-updater`;
6. finishes by checking that `GET /api/updates` answers
   `install_enabled: true` + `worker_listening: true`, printing **PASS** or
   **FAIL** (auth is required — export `VL_ADMIN_TOKEN` with your logged-in
   session Bearer token, or open Settings → Updates in a browser).

`--dry-run` prints every action and changes nothing.

Then **each release = press Update** in Settings → Updates. The AddOn is built
and placed as a compiled DLL automatically (step 7) — the only manual NT8 step
is the ONE-TIME migration F5 if this box previously used the copy-`.cs` method.
No token refresh, no per-release CLI: the worker token is long-lived.

**The token clock:** `VL_CUTOVER_TOKEN` is a long-lived machine token
(`auth.WorkerTokenTTL` = 10 years), written ONCE by `enroll` — **not** a
24-hour gate-jwt. The earlier "refresh before each window" text was stale; the
installer and the worker read the same `~/.config/vl-updater/env` file. An
`authorize` grant is valid **5 minutes**, single use.

## 2. Every-release loop

1. **Tag + approve.** The release is cut from an APPROVED tag only
   (`release.yml` refuses anything else). Approve the `release` Environment run.
   The workflow proves the tree clean, builds the bot, the web assets, and the
   two updater binaries (`updater/vl-updater`,
   `updater/vl-updater-bootstrap`), stages the allow-list, scans for secrets,
   and signs the manifest.
2. **Download to the local inbox:** `gh release download <release_id> --dir <VL_RELEASE_INBOX>`
   — `vl-updater fetch` reads `<inbox>/<release_id>.tar.gz`; there is no
   network fetch.
3. **Fetch (attended, verifies):** `vl-updater fetch <release_id>`. It checks
   the signature and every file against the INSTALL's
   `deploy/release_allowed_signers`, materializes the release under
   `VL_RELEASE_DIR/<source_sha>`, and writes the verdict into
   `data/updater/verdicts/`. No verdict, no install (`422 release not
   verified`).
4. **Authorize (attended, single use):**
   `go run ./cmd/vl-updater-bootstrap --install-dir <bot folder> authorize <release_id>`,
   type `AUTHORIZE <release_id>`. It prints ONE JSON line —
   `{release_id, job_id, expires_at, hmac}` — valid 5 minutes, single use.
5. **Paste + Update now.** Paste that line into the Updates page box and press
   **Update now**. The exact parsed body is POSTed; the page shows the 202
   `job_id` and then the job's states/timestamps/blockers from the worker's own
   job file, and offers the receipt download when the API does. Any refusal
   (400/403/409/422/503) shows the server's own text, and the code is spent
   either way — authorize a new one.
6. **Watch.** The job runs its states until it parks at `nt8_updated`; the
   maintenance hold covers the bot meanwhile.
7. **The AddOn is built and placed automatically** — no manual `cp`, no
   per-release F5. The worker compiles the release's `ninjascript/*.cs` into
   `VL.AddOn.dll` with `build-addon.ps1` (NT8's bundled Roslyn, no .NET SDK —
   it FAILS FAST unless `-Src`, `-Out` AND `-CustomDir` are all passed) and
   places it under `Documents\NinjaTrader 8\bin\Custom\`. The old copy-`.cs`-
   then-F5 path is retired.
   **One-time migration only** for a box that previously used copy+F5: the job
   refuses with `One-time step: open NinjaTrader → NinjaScript Editor → press F5
   once, then click Update again.` It has already MOVEd the old `VL*.cs` out of
   `bin\Custom\AddOns\` (never deleted), so that one F5 recompiles
   `NinjaTrader.Custom.dll` WITHOUT the old VL classes — otherwise NT8 would
   load the AddOn twice. After that single F5, re-run the Update; every later
   release is DLL-only. (Windows can redirect `Documents` into OneDrive; the
   detector accepts either `Documents\NinjaTrader 8\bin\Custom` or
   `OneDrive\...\Documents\NinjaTrader 8\bin\Custom`, and refuses if it finds
   more than one — set `VL_NT8_DOCS_DIR` to pin the folder.)
8. **Resume (attended):** `vl-updater resume <job>` — a TERMINAL only: it
   refuses when stdin is not a terminal (a pipe or a script cannot resume),
   prints the job's state and blocker, and makes you type the job id back.
9. **The RELEASE marker commit.** After the cutover, the deploy lane writes the
   post-update `deploy/RELEASE` marker commit per the boot procedure — that is
   a deploy-lane step, never yours from the Updates page.

### The NT8-closed flow (preferred — UPDATER-NT8-CLOSED, owner ruling 22:1x CT 09-27)

Close NT8 FIRST, then run the loop above. With NT8 closed nothing can trade
(every account is NT8 SIM and Sim101 executes INSIDE NT8), so the updater
PROCEEDS instead of waiting for an AddOn that cannot answer:

- The drain takes the **nt8_absent** path — taken only when ALL hold: the
  AddOn link has been down **≥60 s continuously** (measured from the bot's
  per-connection record, never from a stale ack), the hold is present, the
  entry barrier is drained, nothing is queued or in flight, no planner read
  is claimed, the ledger has no placed/pending/working arm or picture row,
  and every bound trading account is SIM-tradeable. The receipt records the
  path and each leg's evidence; the job file says `nt8_absent (link down
  since <ts>)`.
- **When the closed-NT8 path is reachable** (the link-down start is MEASURED,
  never assumed — `ConnectionRecord.LinkDownSince`): (1) NT8 was connected to
  THIS bot process and then closed — the start is the disconnect; or (2) the bot
  booted with NT8 already closed and no AddOn has connected since its listener
  came up (after a bot or WSL restart with NT8 off) — the start is the moment the
  bot began listening, so the path is eligible after 60 s of bot uptime. Before
  this was written (2) was unreachable: the never-connected bot had no
  disconnect stamp, preflight fell to the connected-world legs and refused
  "trader_cutover … api_positions: NT8 account positions unknown" (job de4cf900,
  2026-10-04 07:27) until NT8 was opened. If the link has been down less than
  60 s the job is refused at preflight with that blocker: wait, then re-run.
  With NT8 OPEN instead, take the normal path: flat → the job parks at
  `nt8_updated` → the AddOn DLL was already built and placed by the worker
  (step 7) → `vl-updater resume` (step 8). The only F5 is the ONE-TIME
  migration when a box previously used copy+F5 (the job says so).
- If NT8 reconnects DURING the job, the normal ack/census legs apply again
  from that moment — the absent verdict is revoked, never grandfathered.
- The NT8 step with NT8 absent: **no AddOn change → nt8_skipped** (as
  today); **an AddOn change → the Go side builds + places the DLL and the job
  records "AddOn owed at next NT8 start"** — it does NOT park (NT8 is closed;
  the DLL will load on the next NT8 start). `boot_verified` skips the
  AddOn-ack wait in this mode and records that it did.
- The simple owner flow: **close NT8 → authorize → paste → Update now →
  wait for complete → open NT8** (the DLL is already in place; no F5).
- With NT8 OPEN, live (non-SIM) connections may stay connected: the census
  admits them when **every account is flat** — the census proves
  `positions=0` and `working=0` across ALL accounts, live ones included
  (owner ruling 2026-09-28; the updater never disconnects anything). Any
  open position or working order on ANY account — live or SIM — still
  refuses, as does any connection in a transitional state. The owner flow
  is the same open or closed: **just flat**, then update.

## 3. Recovery

- **`recovery_needed`:** `vl-updater recovery <job>` prints the manual steps
  for that job. Do them in order, then resume.
- **Rollback:** restore the prior binary per the boot procedure's rollback
  (the `vl-bin.old.*` the cutover kept), then RELEASE + restart — the
  deploy-lane runbook owns this; do not improvise it here.
- **Token expiry:** a stale `VL_CUTOVER_TOKEN` shows up as 401s from the app;
  re-mint it (step 1.5) and restart the worker. An expired authorize grant is
  a `403` — authorize a new one (the old one stays spent).

## 4. Warnings (named, not implied)

- **NEVER put `VL_RELEASE_DIR` in the bot's `.env`.** It belongs ONLY in
  `~/.config/vl-updater/env`, the worker's own file. The bot's environment
  must not see it; the worker refuses a release root inside the install anyway.
- **Never hand-edit `deploy/release_allowed_signers`.** The CTO adds your key.
- **No privilege escalation, no sudo, ever.** The worker runs as your own user
  from `~/bin`; a root-owned `data/updater` locks the bot out.
- **The worker never runs an update step by itself unattended:** `fetch`,
  `authorize`, the F5 and `resume` are all attended by you, in that order.
- **Loopback-direct only.** The Updates page works when you open the bot at
  `127.0.0.1`/`localhost` **on port 8080**, signed in as the enrolled account —
  not through a proxy, a tunnel, the LAN name, or the `:3000` dev server
  (the page says so plainly instead of a bare refusal; the origin check
  itself is unchanged).
