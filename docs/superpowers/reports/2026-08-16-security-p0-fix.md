# VL — P0 SECURITY FIX (acceptance-gate v2, finding #1)

**LINE 1: FIXED AND VERIFIED — the unauthenticated account-takeover chain is dead (reset-password 410 · reset-account 401 · register inherits nothing · API now loopback-only), plus two extra holes found and closed en route: a public RSA decryption oracle and plaintext API keys written to the on-disk logs. 5 commits, full suite green, nothing deployed yet — the owner still has to rebuild+restart.**

## The chain, before → after (live, isolated instance of the fixed binary)

| Route | Before | After | Was |
|---|---|---|---|
| `POST /api/reset-password` | worked (404 only for unknown email) | **410** | reset ANY account from an email alone |
| `POST /api/reset-account` | **public + destructive** | **401** | deleted every user/trader/strategy |
| `POST /api/crypto/decrypt` | 200 | **401** | RSA decryption oracle, server's long-term key |
| bind | `*:8080` (all interfaces) | **`127.0.0.1:8232`** | LAN-reachable on WSL2 mirrored networking |
| register + orphan adoption | inherited victim's keys | **inherits nothing** (live-proven) | broker/wallet credential handover |

Matrices: `2026-08-16-security-p0-artifacts/unauth-{before,after-final}.txt`.

## What changed

- **S1 `81e54fc9`** — `Server.Start` built `":port"` (0.0.0.0) while logging `http://localhost`. Now `config.APIServerHost` (default `127.0.0.1`) → `net.JoinHostPort`; `API_SERVER_HOST` overrides and warns when off-loopback. The NT8 TCP wire was **already** loopback (`tcp_server.go:31`, confirmed live) — no change.
- **S2 `821c895a`** — `reset-password` → permanent **410** (no mail path exists in the repo, so no safe version of it does either; authenticated users use `PUT /api/user/password`). `reset-account` → JWT group **+** `ALLOW_ACCOUNT_RESET=1` **+** `{"confirm":"RESET-ALL-DATA"}`; every refusal logs IP + user id.
- **S3 (same commit)** — deleted `adoptOrphanRecords`, which re-homed every orphaned `ai_models`/`exchanges` row to whoever registered next. *Honest note: S3 landed inside the S2 commit — same file, and I staged the whole file. Not a separate commit as the dispatch asked.*
- **S4 `8dcc4521`** — `/api/crypto/decrypt` moved to the JWT group and refused unless `TRANSPORT_ENCRYPTION` is on. The frontend never called it (`decryptSensitiveData` has zero callers). Its 5-minute replay window was no barrier: the check is skipped when `ts == 0`, which the caller sets.
- **S5 `92969990`** — both config-update handlers ended in `logger.Infof(…"%+v", req.Models/req.Exchanges)`, writing **plaintext provider keys, exchange secret keys, passphrases and wallet private keys** into `data/nofx_*.log` (mode 0644, kept forever). Full DeepSeek keys were recovered from **three** existing log files. Now masked via `MaskSensitiveString`. `api/utils.go` already had `SanitizeModelConfigForLog`/`SanitizeExchangeConfigForLog` — written, unit-tested, and never wired to a call site.
- **`0d97a672`** — `API_SERVER_HOST` + `ALLOW_ACCOUNT_RESET` documented in `.env.example`.

## Tests — `api/security_p0_test.go`, 11 cases, all green

reset-password refuses every body shape · reset-account refuses without the env flag, refuses wrong/empty/lowercase confirm tokens, still works when fully authorized · **register inherits zero credential rows** (seeded orphans stay with the ghost user) · register still closed after the first user · crypto-decrypt refused while transport encryption is off · two wiring lints (public registration must not return; protected must stay) · the log-leak lint + masking primitive.

**One test caught itself being wrong.** The log-leak lint's first pattern had a stray leading quote, so it matched only this file's own prose and passed while the leak was live. Found by deliberately reintroducing the leak and watching it *not* fail; the committed version fails on reintroduction and passes on the fix. Verified in both directions.

## Still open — reported, not fixed (all now loopback-only, which is the mitigation)

1. **HIGH `GET /api/equity-history`** — unauthenticated full equity/balance/P&L curve for any `trader_id`, and it **defaults to the first loaded trader** when the parameter is omitted.
2. **MED `POST /api/equity-history-batch`** — unauthenticated, drives up to 20 × `GetBalance()`/`GetPositions()` live broker calls using the owner's credentials.
3. **MED `GET /api/traders/:id/public-config`, `/api/traders`, `/api/competition`, `/api/top-traders`** — trader identity + performance without auth. These back the "public leaderboard" product feature (whose UI page was removed), so switching them off is a product decision, not a bug fix.
4. **MED `GET /metrics`** — Prometheus at root, unauthenticated (runtime metrics only; no trader labels).
5. **MED** — no rate limiting anywhere: `POST /api/login` allows unlimited credential guessing.
6. **MED** — `CORS: *` hardcoded; no security headers (HSTS/CSP/X-Frame-Options); plain HTTP.
7. **LOW `POST /api/wallet/generate|validate`** — unauthenticated key generation and an outbound RPC call on a caller-supplied private key.

## Owner handoff

```
cd /home/hoang/nofx
git pull
go build -o nofx-bin .
sudo systemctl restart nofx
go version -m ./nofx-bin | grep vcs      # expect vcs.modified=false, revision=<new HEAD>
ss -lntp | grep 8080                     # expect 127.0.0.1:8080 — NOT *:8080
```

**Then, please:** the old log files still contain plaintext keys. Rotate the DeepSeek API key and prune `data/nofx_*.log` — the code fix stops new leaks, it cannot unwrite the old ones.

**If you reach the UI from another machine**, that will now fail by design — set `API_SERVER_HOST=0.0.0.0` in `.env` *only* behind a firewall/reverse proxy, and know it re-exposes everything in the "still open" list.

## Session notes

- STEP 0 passed (HEAD `205b1753`, clean tree, one session). **The one-session precondition later broke**: at 23:28 a second session ran a deploy (`cp … nofx-bin; kill -9`) and systemd relaunched the bot as PID 1344028. The binary on disk was **unchanged** (`298d75b0`, mtime 19:08:45) — the swap did not land, so the same pre-fix build came back up. No amend/rebase was used here; nothing was lost.
- Verification used an **isolated instance** of the fixed binary on a spare port with a temp DB. The production bot was never restarted, and no destructive request was ever sent to it — `reset-account` was proven public by reading the route table and by a harmless `reset-password` probe, never by firing it.
