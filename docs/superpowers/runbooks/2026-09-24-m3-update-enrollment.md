# M3 update enrollment: runbook (W-ONE-BUTTON M3, 2026-09-24)

**Lane:** Claude-101 · **Branch:** `feat/one-button-m3-update-authz` · **Dispatch:** CTO `1790208803193` (WAVE 2, M3) and the M3 block of the master dispatch `1790131309474`.

**What this is:** the owner's procedure for binding this installation's update administrator (**enroll**), re-binding it after a password change (**enroll --replace**), and taking the binding away again (**un-enroll**). It also covers what a refusal looks like and where its cause is logged.

**Owner-attended, on the box, as the bot's own user.** No API creates or resets the enrollment. The `/updates` gate READS `admin.json` and `device.key` on every `/api/updates*` request, read-only (that is why enrolling needs no restart). Nothing here restarts the bot or touches trading.

## What M3 does and does not do

- **OFF until enrolled.** With nothing enrolled, all five `/api/updates*` routes answer `403 {"error":"forbidden"}`. Nothing else in the app changes.
- **Enrolled, M3 still installs nothing.**
  - The manifest verifier is a stub that refuses every release (`422 {"error":"release not verified"}`).
  - The Updates page's install button stays disabled ("install authorization under review") until the CTO's adversarial review of M3 closes.
  - Enrolling now only proves the identity half end to end.
- **Two factors per install** (from M4 on):
  - the enrolled admin's signed-in session (**identity**);
  - an HMAC computed on this box from `device.key` by the attended `updater-bootstrap authorize` (**possession**).
  - The key never leaves the box and is never printed. Nothing on the API side can compute a MAC.

## Where the files live

The data dir is resolved by `internal/installpath` exactly as the bot resolves it: the bot's WorkingDirectory plus the database path in its `.env`.

| File | Mode | Written by |
|---|---|---|
| `<data>/updater/` | 0700, owned by the bot's user | `enroll` (or the maintenance hold) |
| `<data>/updater/admin.json` | 0600 | `updater-bootstrap enroll` only. Holds user id, email, enrolled-at, and a binding to the account's current password hash |
| `<data>/updater/device.key` | 0600 | `updater-bootstrap enroll` only. 32 random bytes |
| `<data>/updater/seen_job_ids.json` | 0600 | the install route only. Holds the job ids already used (single use) |
| `<data>/updater/hold.json` | 0600 | the maintenance hold, **not** enrollment. Leave it alone here |

The loaders refuse a symlink, a mode looser than 0600, a file owned by another user, and a degenerate key. They fail closed: 403 on every route.

## Enroll

**Preconditions.** The CLI checks each one and refuses with a reason:

- You are the bot's user, not root. A root-owned `data/updater` locks the bot out.
- You are in a real terminal. It is attended, so piped stdin is refused.
- `--install-dir` (default: the current directory) is the installation the bot runs from, and its database exists. A relative path is made absolute first.
- **Your shell's `DB_PATH` must not divert it.** If your shell exports `DB_PATH` and it resolves to a different database file than the installation's own `.env` (or the default `data/data.db` when the `.env` sets none), the CLI refuses before any prompt and names both values and both files. `unset DB_PATH` and re-run. Any spelling of the same file proceeds.
- `<email>` is **exactly** the app account's email: the same predicate login uses.

**Steps:**

1. From a checkout at the running binary's revision (the source only compiles the CLI; `--install-dir` points it at the live installation), run:
   ```
   go run ./cmd/updater-bootstrap --install-dir <the bot's WorkingDirectory> enroll <email>
   ```
2. Before the prompt, the CLI prints what it will act on: `installation:`, `bot database:`, `data dir:` and `DB_PATH from:` (which file or default the path came from). Check them. Then type exactly `ENROLL <email>`. Anything else writes nothing.
3. Expect `enrolled: user_id=<first 8>… dir=<data>/updater (both enrollment files 0600; the key is never printed)`.
4. Check the modes. **Never** `cat`, copy or paste `device.key`.
   ```
   stat -c '%a %U %n' <data>/updater <data>/updater/admin.json <data>/updater/device.key
   ```
   Expect `700`, `600` and `600`, all owned by the bot's user.
5. **No restart.** The gate reads the enrollment on every request. Open Settings → Updates in a browser on this box at `http://127.0.0.1:8080` or `http://localhost:8080` (the Go-served UI). It should read enrolled. `GET /api/updates` answers `{"enrolled":true,"manifest_verifier":"stub","install_enabled":false}`.

A plain `enroll` on an enrolled box refuses: "already enrolled — re-run with --replace".

## After a password change: enroll --replace

A password change (Settings → Account, which now requires the **current** password) does four things:

- ends every session signed in before it, on every page; sign in again with the new password;
- ends the Telegram bot's token, which the bot re-mints on its next `/start` or AI message and logs `Bot: token re-minted for <id> — the previous one would be refused (credential change or expiry)`. That line is INFO, so it goes to stdout/journald and `data/nofx_<boot date>.log`, not to `log_events`;
- **un-enrolls Updates**: `admin.json`'s binding was computed from the old password hash, so every `/updates` route answers 403 ("password changed since enrollment (re-enroll with --replace)");
- nothing else.

To restore: first confirm you can sign in with the new password, then run the same command as above with `enroll --replace <email>` and type `ENROLL <email>`. `--replace` rotates `device.key`, so every authorization printed before it is dead.

**If `--replace` dies halfway**, the new key is left beside the old `admin.json`. That pair is **not** a working enrollment: every route answers 403 until a `--replace` completes. Re-run it. This is pinned by `TestEnrollCommentTruthACrashBetweenTheTwoRenames`.

## Locked out of the app account

Password reset by email is disabled (`POST /api/reset-password` answers 410), so the fix is one statement on the box. Back up `data/data.db` first; it is the live database.

```sql
UPDATE users SET password_hash='NEW_BCRYPT_HASH', updated_at=CURRENT_TIMESTAMP WHERE email='YOUR_ACCOUNT_EMAIL';
```

Replace `NEW_BCRYPT_HASH` with a bcrypt hash of the new password and `YOUR_ACCOUNT_EMAIL` with the account's exact email; keep the quotes. The 410 body serves this same statement with the same placeholders, readable as-is in raw `curl` output.

- **Set both columns in ONE statement.** `users.updated_at` is the credential epoch (`auth.CredentialEpoch`). Moving it is what signs out every session issued before the reset, including a stolen one. A hash-only UPDATE leaves every one of those sessions valid until it expires. This is pinned by executing the served advice against a temp SQLite store (`TestResetPasswordAdviceRetiresPreResetSessionsOnSQLite`).
- **No restart.** The bot reads the users row on each request.
- **Afterwards**, sign in with the new password. Updates is now un-enrolled (the password binding changed), so run `enroll --replace` as above. If the row is the FIRST account (the one the Telegram bot acts for), the bot re-mints its own token on its next message.

## Authorize one install (M4 onwards; inert in M3)

```
go run ./cmd/updater-bootstrap --install-dir <the bot's WorkingDirectory> authorize <release_id>
```

Type exactly `AUTHORIZE <release_id>`. It prints one JSON authorization, `{release_id, job_id, expires_at, hmac}`, **valid 5 minutes, single use**. A second use answers `409`.

In M3 this path is deliberately inert: the stub verifier refuses every release (`422`), and the page's install button is disabled.

## A refusal, and where its cause is

Every audit line names the **socket peer**. The `🔒 [updates]`, `🔒 [credentials]`, `🔒 [auth]` and `blocked …` lines, and gin's access log, ignore `X-Forwarded-For` and `X-Real-IP`: the router trusts no proxy. Behind a reverse proxy every line would name the proxy; this box has none.

**Signed in but refused right after a password change?** If the `🔒 [auth]` line says `credential epoch is Ns in the future — clock stepped back; sign-in refused until then`, the box's clock moved backwards after the change. Sign-in works again once the clock passes that moment. The bound is the size of the step, so fix the clock or wait N seconds.

Every gate refusal is the **same** response: `403 {"error":"forbidden"}`. The cause is logged server-side as a category only, never the token, the MAC or the key:

```
🔒 [updates] refused <METHOD> "<path>": <category>
```

It is WARN, so it appears in journald, `data/nofx_<boot date>.log` **and** `log_events`. The categories, in the order they are checked:

| Category | Meaning / what to do |
|---|---|
| `data dir unconfigured` | the bot has no data dir; nothing to do from the page |
| `peer not loopback`, `peer unparseable` | the request did not come from this box. Updates are loopback-DIRECT only |
| `host not a loopback name` | open the UI as `127.0.0.1`/`localhost`, not the LAN name or IP |
| `forwarded request (…)`, `x-forwarded-*` | a proxy or tunnel relayed it. Not supported: go direct |
| `update header missing or wrong` | the client did not send `X-NOFX-Update: 1`. The Updates page sends it on its `/updates*` calls |
| `cross-origin`, `cross-site fetch` | another origin made the request. Refused by design |
| the JWT-secret categories | the bot runs on a JWT secret this public repo publishes, or one shorter than 32 bytes. Set a private `JWT_SECRET` (`openssl rand -base64 64`) |
| `authorization missing` / `malformed` / `token revoked` / `token invalid` | sign in again |
| `machine token` | the Telegram bot's or the gate-jwt tool's token. They can never reach Updates |
| `not enrolled`, `enrollment unreadable`, `device key unreadable` | enroll; or fix the file modes and owner (see the table above) |
| `not the enrolled admin`, `admin user row absent or changed` | signed in as another account, or the account's email changed. Re-enroll with `--replace` |
| `password changed since enrollment (re-enroll with --replace)` | see "After a password change" |
| `token older than the user row` | the session predates the account's last change. Sign in again |

## Un-enroll

As the bot's user, remove **only** the two enrollment files. To see the exact `<data>` directory without writing anything, run `go run ./cmd/updater-bootstrap --install-dir <the bot's WorkingDirectory> authorize x`, read its `data dir:` line, and answer the prompt with anything other than the confirmation ("confirmation did not match — nothing written"). Every route returns to `403` "not enrolled". No restart is needed.

```
rm <data>/updater/admin.json <data>/updater/device.key
```

Keep `hold.json`, which is the maintenance hold. `seen_job_ids.json` may stay: its job ids are dead with the key. It must NOT be deleted on its own: once the installation is enrolled, a MISSING seen store reads as corrupt (never empty) and every install is refused `403` until `enroll --replace` re-creates it (red-team red-3 #4).

## Known limits (named, not implied)

- **Loopback-direct only.** A relay that adds no forwarding header is indistinguishable from a local client. Do not put a proxy or tunnel in front of `/api/updates`.
- **WSL2 mirrored networking makes the whole machine the transport boundary.** This box needs mirrored mode so the bot can reach NT8. In that mode a process on the WINDOWS host arrives at the bot as a loopback peer. "Loopback" therefore means "anything on this physical machine", not "this Linux user". The factors that still stand are the enrolled admin's session (JWT), the `X-NOFX-Update` header, and the HMAC that only `updater-bootstrap authorize` can compute from `device.key`.
- **Logout is process-lifetime.** The logout blacklist lives in the bot's memory. A bot restart forgets it, so a session logged out before the restart is accepted again until its expiry + 60 s. A password change is the durable way to end sessions: it moves the credential epoch, which is stored.
- **Vite dev server.** Under `npm run dev` (`:3000`, proxy `changeOrigin: true`), the POSTs (`/check`, `/install`) read `cross-origin`. Production is same-origin: Go serves `web/dist`.
- **The receipt link** is a plain `<a href>`, which cannot carry the header or the bearer token. In M3 every job id is `404`; M4/M5 must fetch receipts through the client.
- **The census is a belt, not the boundary.** The worker shares the key file's UID. The boundary is the file mode, the attended enrollment and, later, the isolated host (M1 §8). "Nothing API-side mints a MAC" is enforced by a syntactic census with its named limits, never proven.
