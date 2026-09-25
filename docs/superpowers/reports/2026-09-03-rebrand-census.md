# Rebrand Census — every "nofx" identity, mapped and classified (2026-09-03)

**Dispatch:** REBRAND CENSUS · Owner hoang · READ-ONLY. This wave produces a map, not a change.
**Branch:** `docs/rebrand-census-0903` · base `origin/dev` @ `89673ccc` (`feat(1D): fold the expectancy table into a collapsed dropdown`).
**Worktree:** `~/nofx-rebrand` (removed at closeout). **Method:** grep, case-insensitive; SQLite read-only URI `file:/home/hoang/nofx/data/data.db?mode=ro`; main tree read-only (A2b: porcelain-clean at start, 0 dirty).
**Evidence tier:** every count below is **[A]** — re-runnable with the quoted command. Where a value resolves from a variable/default, the source line is quoted (A11). No keys/tokens/secrets printed (A25).

---

## 0 · Headline counts

| Metric | Count | Command |
|---|---|---|
| Total "nofx" occurrences (case-insensitive; covers NOFX/NoFX/nofx) | **14,150** | `grep -rIio nofx . --exclude-dir=.git \| wc -l` |
| Files touched | **953** | `grep -rIil nofx . --exclude-dir=.git \| wc -l` |
| "nofxAI" occurrences | **668** | `grep -rIio nofxai . --exclude-dir=.git \| wc -l` |
| "NOFXi" occurrences | **556** | `grep -rIio nofxi . --exclude-dir=.git \| wc -l` |
| docs/superpowers alone | **10,297** | `grep -rIio nofx docs/superpowers \| wc -l` |
| of which: the two copies of the 2026-08-20 census | **6,464** | `grep -rIio nofx docs/superpowers/reports/2026-08-20-brand-census*.md \| wc -l` |

Per top-level directory (command: `for d in $(find . -maxdepth 1 -type d ! -name .git); do grep -rIio nofx "$d" | wc -l; done`): `.github 56 · agent 159 · api 138 · auth 1 · cmd 40 · config 4 · deploy 148 · discipline 1 · docker 5 · docs 11,405 · expectancy 7 · hook 1 · kernel 242 · logger 2 · manager 5 · market 8 · mcp 28 · nginx 2 · ninjascript 6 · patches 10 · provider 63 · railway 3 · safe 1 · scripts 3 · store 35 · telegram 26 · telemetry 21 · trader 590 · web 842 · .husky/calendar/crypto/internal/security/screenshots/wallet = 0`.

---

## 1 · Surprises (A23 — included, not acted on)

1. **A prior census exists.** `docs/superpowers/reports/2026-08-20-brand-census.md` (3,232 nofx hits) has a **byte-identical duplicate** `docs/superpowers/reports/2026-08-20-brand-census-docs-brand-census.md` (md5 `638de5d847682deefdedc3f3decd7363`, `md5sum` on both files). The prior in-scope count was 3,258 (it excluded `node_modules/`, `dist/`, binaries, `data/`, `screenshots/`, `.playwright-mcp/`, `patches/`). My 14,150 is a superset count; the deltas are mostly docs/superpowers growth (the census counting itself) and the prior exclusions.
2. **The Studio's two most visible surfaces are ALREADY VL.** `web/index.html:8` `<title>VL Trader - AI Trading System</title>`; `web/src/components/common/HeaderBar.tsx:95` renders `<span …>VL</span>`; `web/public/icons/vl.svg` exists (`aria-label="VL"`); `web/public/favicon.svg` contains **no** nofx text (`grep -o 'nofx\|VL' web/public/favicon.svg` → empty). The tab title and header the owner sees today already say VL.
3. **The entire NT8 side is already VL-named.** Windows AddOn folder `/mnt/c/Users/hoang/Documents/NinjaTrader 8/bin/Custom/AddOns/` contains only `VLBarsSubscriptionManager.cs`, `VLContractResolver.cs`, `VLTraderTCPClient.cs`; repo `ninjascript/` is likewise all VL-named. No nofx-named `.cs` anywhere in either. The only Windows-side "nofx" string is a **path NT8 reads**: `%USERPROFILE%\NofxTrader\account.txt` (`ninjascript/VLTraderTCPClient.cs:391,402`); the directory exists (`/mnt/c/Users/hoang/NofxTrader`, [A]).
4. **`nofx-tree-guard` does not exist.** No tree-guard service, timer, or script in `deploy/`, `scripts/`, `.github/`, or `~/.config/systemd/user/` (`grep -rIin 'tree[-_]guard' deploy/ scripts/ .github/` → empty). The dispatch's premise of a `nofx-tree-guard` unit is unmatched today.
5. **`deploy/nofx.service` and `deploy/nofx-web.service` are repo templates, not installed units.** Installed in `~/.config/systemd/user/` are only `nofx-backup.service/.timer` and `nofx-clock-guard.service/.timer`. The templates use placeholder `ExecStart=__NOFX_DIR__/nofx-bin` (`deploy/nofx.service:41`).
6. **The DB is 100 % nofx-free.** No schema object contains nofx; `traders` id/name = 0 rows, `strategies` name = 0, `plans` plan_id = 0 (commands in §G3). Group 3's "DB migration" does not exist.
7. **Only 5 raw.githubusercontent URLs repo-wide**, all inside `docs/superpowers/reports/` (prior reports linking their own commits).
8. **A live dispatch holds the main-tree lock.** `~/nofx-main.lock`: `owner=hoang`, session "nofx-b3 wake-predicate cutover", `pid=67825` **ALIVE** (`kill -0` verified), expiry 2026-09-03T22:46:16-0500. Read-only respected; not cleared (A2).
9. **`provider/nofxos/` still ships the deprecated nofxos.ai client** (`package nofxos`; `claw402.go:17-27` routes through `claw402.ai/api/v1/nofx/…`). CLAUDE.md marks nofxos.ai dead (HTTP 402). It is a Go package name → module-relative rename, or delete.
10. **Partner repo confirmed VL.** `~/vlautoagenttraderv1` exists; its remote `nofxlocal` points at `/home/hoang/nofx` (local path remote). VL-named already; receives changes via `format-patch → am` per CLAUDE.md.

---

## 2 · SECTION C — the table

Legend: **visible?** = what the owner SEES today (screen/log/alert/URL/none). **rename-safe?** = yes / alias-first / migration. **command** = the grep that produced the count (all run from `~/nofx-rebrand`).

### Group 1 — VISIBLE BRAND (zero-risk renames)

| # | location | what | count | visible? | rename-safe? | keys on | migration | command |
|---|---|---|---|---|---|---|---|---|
| 1.1 | `main.go:44` | Boot banner `🚀 NOFX - AI-Powered Trading System` | 1 | log (journald at every boot) | yes | nothing | string swap; next restart | `grep -n -i nofx main.go \| grep -v '"nofx/'` |
| 1.2 | `web/src/components/agent/ChatMessages.tsx:133`, `ChatInput.tsx:123-124,187`, `WelcomeScreen.tsx:118` | AgentBeta chat brand "NOFXi ·", "Ask NOFXi anything…", "NOFXi may make mistakes…" | 4 files | screen (AgentBeta tab) | yes | nothing | string swap | `grep -rIin NOFXi web/src/components/agent/` |
| 1.3 | `agent/prompt_persona.go:5` | Agent persona "You are NOFXi, the core intelligence hub of the NOFX platform" | 1 | chat output | yes | nothing | string swap | `grep -n NOFXi agent/prompt_persona.go` |
| 1.4 | `telegram/bot.go:376,387,419,440` | Telegram /start + /help texts "NOFX 就绪…", "NOFX is ready!", "NOFX 使用指南", "NOFX Help" | 4 | Telegram chat | yes | nothing | string swap | `grep -rIin nofx telegram/bot.go \| grep -v '"nofx/'` |
| 1.5 | `telegram/agent/prompt.go:10,19,24`, `telegram/agent/agent.go:22` | Telegram agent prompt "NOFX quantitative trading system…", tool description | 4 | Telegram chat | yes | nothing | string swap | `grep -rIin nofx telegram/agent/` |
| 1.6 | `README.md:1` | `<h1 align="center">NOFX</h1>` | 1 | URL (GitHub repo page) | yes | nothing | heading swap | `head -3 README.md` |
| 1.7 | `docs/i18n/*/README.md` (zh-CN, vi, uk, ru, ko, ja, …) | Translated README headings | 379 hits in `docs/i18n/` | URL (GitHub) | yes | nothing | sed sweep | `grep -rIio nofx docs/i18n \| wc -l` |
| 1.8 | `.github/SECURITY.md:3,5,27,60,156` | "Security at NOFX", policy prose, Twitter **@nofx_official** | 5+ | URL (GitHub) | yes (account handle itself: external rename) | Twitter handle | prose swap; handle rename is a separate Twitter action | `grep -rIin nofx .github/SECURITY.md` |
| 1.9 | `web/src/index.css:19,108,192` + class usages | `--nofx-gold` CSS variables and `text-nofx-*`/`bg-nofx-*` Tailwind-class namespace | 25 in index.css (of 842 web total) | screen (styling only; invisible as text) | alias-first | ~40 `.tsx` class strings | rename CSS vars + class names mechanically; colors unchanged | `grep -c nofx web/src/index.css` |
| 1.10 | `web/src/guide/content/welcome.ts:37` | Guide architecture diagram "Go bot (nofx-bin) ───…" | 1 | screen (Guide tab) | yes | nothing | string swap (pair with binary rename phase) | `grep -rn nofx web/src/guide/content/` |
| 1.11 | `web/public/favicon.svg`, `web/public/icons/*` | favicon + logo assets | 0 nofx text [A] | screen | — | — | none; `vl.svg` already exists | `grep -o 'nofx\|VL' web/public/favicon.svg` |

**Totals G1: 17 locations · ~35 direct string hits + CSS namespace (25 css + class uses) + docs/i18n 379 · ALL rename-safe (except the Twitter handle, which is an account action).**

### Group 2 — RENAME-SAFE IDENTIFIERS (mechanical, but paths are hardcoded — change together)

| # | location | what | count | visible? | rename-safe? | keys on | migration | command |
|---|---|---|---|---|---|---|---|---|
| 2.1 | `go.mod:1` | `module nofx` | 1 | none | yes | 545 importing files | one commit: module path + all imports | `head -3 go.mod` |
| 2.2 | repo `*.go` | imports `"nofx/…"` | **1,058 lines / 545 files** | none | yes | module path | same commit as 2.1; `go build ./...` green | `grep -rIin '"nofx/' --include='*.go' . \| wc -l` |
| 2.3 | main tree root | binaries `nofx`, `nofx-bin`, 10× `nofx-bin.old.<rev>`, `nofx-bin.prev.boot` | 13 files | ps/log (process name) | alias-first | systemd ExecStart, cutover scripts | rename to `vl-bin`; keep `nofx-bin` symlink one boot | `ls -1 ~/nofx \| grep -i nofx` |
| 2.4 | `deploy/leveltruth-cutover.sh:8,10,34-35` | hardcoded `/home/hoang/nofx`, `./nofx-bin.next`, `pgrep -f "nofx-bin$"`, `mv nofx-bin nofx-bin.old.leveltruth` | 5+ | none | alias-first | cutover discipline | update in the SAME change that renames the binary | `grep -rIin 'hoang/nofx\|nofx-bin' deploy/leveltruth-cutover.sh` |
| 2.5 | `Makefile:1,7,59,60,100` | make banner + `go build -o nofx`, `rm -f nofx` | 5 | none | yes | build habit | swap target name | `grep -n -i nofx Makefile` |
| 2.6 | `~/.config/systemd/user/` | units `nofx-backup.service/.timer`, `nofx-clock-guard.service/.timer` | 4 files | log (`systemctl --user` output) | alias-first | timers (`timers.target.wants`) | new `vl-*` units; old ones stopped/disabled same cutover | `ls ~/.config/systemd/user/` |
| 2.7 | `deploy/systemd-user/nofx-backup.service:3,7`, `nofx-clock-guard.service:9` | ExecStart hardcodes `/home/hoang/nofx/deploy/…` | 3 lines | none | alias-first | unit files | path swap together with repo path | `grep -n ExecStart deploy/systemd-user/*.service` |
| 2.8 | `deploy/nofx.service:33,41`, `deploy/nofx-web.service` | repo templates `Description=NOFX trading backend…`, `ExecStart=__NOFX_DIR__/nofx-bin` | 2 files | none (not installed) | yes | nothing | rename files + strings | `grep -n 'Description\|ExecStart' deploy/nofx*.service` |
| 2.9 | `deploy/journald-nofx.conf:1-2,22` | journald dropin, installed as `/etc/systemd/journald.conf.d/nofx.conf` (root-owned) | 1 file | log (journald) | alias-first | sudo install | rename file; **needs sudo** to replace installed dropin | `grep -n -i nofx deploy/journald-nofx.conf` |
| 2.10 | `deploy/nofx-clock-guard.sh:25` | `NOFX_CLOCK_STATE` default `/home/hoang/nofx/data/clock-guard-state.json` | 1 | none | alias-first | env + path | accept `VL_*` with `NOFX_*` fallback | `grep -n NOFX deploy/nofx-clock-guard.sh` |
| 2.11 | `deploy/nofx-db-backup.sh:9-79` | backup names `nofx-YYYY-MM-DD_HHMMSS.db.gz`, `NOFX_DB`, `NOFX_BACKUP_DIR` default `~/nofx-backups/auto` | 10+ lines | log ("nofx-backup: done…") | alias-first | timer + backup layout | new prefix `vl-`; keep reading old `nofx-*` for retention sweeps | `grep -n -i nofx deploy/nofx-db-backup.sh` |
| 2.12 | `~/.config/systemd/user` + `~/nofx-backups` | unit names + backup dir (auto/{daily,weekly}) | paths | log | alias-first | scripts above | symlink `~/vl-backups` → old for one retention cycle | `ls ~/nofx-backups` |
| 2.13 | `~/nofx-main.lock` | lock marker (376 B, held live by another session) | 1 path | none | alias-first | lock discipline (CLAUDE.md) | path constant swap; keep old-name read for in-flight sessions | `ls -la ~/nofx-main.lock` |
| 2.14 | `docker-compose.yml:3,7,11,23,32`, `docker-compose.{stable,prod}.yml`, `Dockerfile.railway` | service `nofx`, container `nofx-trading`, network `nofx-network`, `${NOFX_BACKEND_PORT:-8080}`, `${NOFX_FRONTEND_PORT:-3000}` | 10+ | none (docker not in use — Docker NOT required per CLAUDE.md) | yes | docker-compose users | name swap | `grep -rIin 'nofx' docker-compose*.yml docker/` |
| 2.15 | `docker/Dockerfile.backend:52,63,70` | `go build … -o nofx`, `COPY … /app/nofx`, `CMD ["./nofx"]` | 3 | none | yes | image internals | swap with binary | `grep -n nofx docker/Dockerfile.backend` |
| 2.16 | `nginx/nginx.conf:1,38` | `# NOFX Frontend` + `proxy_pass http://nofx:8080/api/` | 2 | none (nginx not in local path) | yes | compose service name | swap with 2.14 | `grep -rn -i nofx nginx/` |
| 2.17 | `start.sh:178-185` | reads `NOFX_FRONTEND_PORT` / `NOFX_BACKEND_PORT` from `.env` (defaults 3000/8080) | 8 | none | alias-first | `.env` keys | accept both prefixes | `grep -n NOFX start.sh` |
| 2.18 | `~/.env` + 4 backups | keys `NOFX_BACKEND_PORT`, `NOFX_FRONTEND_PORT`, `NOFX_TIMEZONE`, `JWT_SECRET` (names only; values redacted) | 5 files | none | alias-first | start.sh/docker reads | dual-read until callers migrated | `grep -oE '^[A-Za-z_]+=' ~/nofx/.env \| grep -i nofx` |
| 2.19 | `logger/logger.go:90` | log file names `nofx_YYYY-MM-DD.log` | 1 | log (data/logs dir) | yes | nothing | format-string swap | `grep -rn -i nofx logger/ \| grep -v 'nofx/'` |
| 2.20 | `telemetry/metrics.go:13,22,31,43,54,67` | Prometheus metric prefix `nofx_` (`nofx_decisions_total` ×5 metrics) | 6 | none today (no Prometheus scrape configured — [B]) | migration | external scrapers/alert rules if ever added | rename to `vl_`; if a scraper exists, emit both one release | `grep -rIon 'nofx_[a-z_]*' telemetry/ --include='*.go' \| grep -v _test` |
| 2.21 | `auth/auth.go:93` | JWT `Issuer: "nofxAI"` | 1 | none | migration | **all issued tokens** | new issuer `vlAI`; dual-accept old issuer until token TTL elapses | `grep -rn -i issuer auth/` |
| 2.22 | `web/package.json:2` | npm name `"nofx-web"` | 1 | none | yes | nothing (no publish) | swap | `grep -n '"name"' web/package.json` |
| 2.23 | `/home/hoang/nofx` + worktrees `~/nofx-*` | repo path | 1 + 7 worktrees | none | alias-first | ~15 hardcoded script lines (2.4, 2.7, 2.10, 2.11) | rename dir → symlink old path one cycle | `grep -rIin '/home/hoang/nofx\|hoang/nofx' deploy/ scripts/` |

**Totals G2: 23 rows · the recurring risk is PATH/UNIT/ENV hardcoding in `deploy/*.sh` + systemd units — every one named above; all handle via alias/symlink + dual-read, one boot of overlap.**

### Group 3 — MIGRATION-REQUIRED (a stored row, NT8, systemd, or GitHub keys on it)

| # | location | what | count | visible? | rename-safe? | keys on | migration | command |
|---|---|---|---|---|---|---|---|---|
| 3.1 | `provider/ninjatrader/tcp_server.go:1717` | TCP Hello `Source: "nofx-go"` (peer is `"vltrader-addon"`, comment at `tcp_framing.go:113`) | 1 | none | migration | NT8 C# AddOn validates/reports source | **lockstep**: Go emits `vl-go` AND accepts old, C# updated, NT8 restart + bot boot together | `grep -rIin '"nofx-go"\|vltrader-addon' provider/ninjatrader/*.go` |
| 3.2 | `ninjascript/VLTraderTCPClient.cs:391,402` | NT8 reads `%USERPROFILE%\NofxTrader\account.txt` (dir exists on Windows) | 2 | none | migration | NT8 file read | copy dir to `VlTrader\`, dual-read, NT8 restart | `grep -n NofxTrader ninjascript/VLTraderTCPClient.cs` |
| 3.3 | `web/src/lib/agentChatStorage.ts:1,15,19` | localStorage keys `nofxi-agent-chat`, `nofxi-agent-chat-draft` | 3 | none | migration | stored browser chat state | dual-read old keys on load, write new | `grep -n nofxi web/src/lib/agentChatStorage.ts` |
| 3.4 | `.github/workflows/pr-docker-check.yml:55,72,91,129,145`, `docker-build.yml:95-96` | CI image tags `nofx-${{matrix.name}}:pr-test`, `nofx-backend:pr-test-arm64`, GHCR path `…/nofx-<suffix>`, DockerHub `…/nofx-<suffix>` | 7 | none | migration | GHCR/DockerHub registries | retag after repo rename; old tags can persist | `grep -rIin nofx .github/workflows/ \| grep -v johnwick` |
| 3.5 | DB `data/data.db` | schema objects / tables / columns / trader ids+names / strategy names / plan ids containing nofx | **0** [A] | none | — | — | **no DB migration exists** | `sqlite3 'file:…?mode=ro' ".schema" \| grep -in nofx` + the three COUNT(*) sweeps in §D5 |
| 3.6 | API route prefixes | any `/nofx/…` route | **0** [A] (only a comment: `api/agent_routes.go:9`) | none | — | — | none | `grep -rIin 'nofx' api/ --include='*.go' \| grep -v '"nofx/' \| grep -i 'route\|path\|/api\|prefix'` |
| 3.7 | `web/src/guide/types.ts:6` | `GUIDE_BUILT_REV` value — a git sha, not a brand name | 0 brand hits | screen (Guide drift badge) | yes | drift-check compares against bot revision | none (values are hashes) | `grep -n GUIDE_BUILT_REV web/src/guide/types.ts` |

**Totals G3: 4 real migration rows (TCP source string, NT8 profile path, localStorage keys, CI image tags) + 3 zero-rows proven.** DB, API routes, and stored trader/strategy/plan values need nothing.

### Group 4 — EXTERNAL

| # | location | what | count | visible? | rename-safe? | keys on | migration | command |
|---|---|---|---|---|---|---|---|---|
| 4.1 | GitHub | repo `johnwick2921-cyber/nofx` | 1 | URL | yes (GitHub redirects after rename) | clones, raw URLs, CI | rename repo last; update raw URLs after redirect proven | `git remote -v` |
| 4.2 | GitHub | upstream `NoFxAiOS/nofx` (historical) | 1 | URL | n/a | fork lineage | leave; do not touch upstream | `git remote -v` |
| 4.3 | `docs/superpowers/reports/` | raw.githubusercontent URLs embedding `johnwick2921-cyber/nofx` | **5** | URL (clickable in prior reports) | yes | GitHub redirect | works via redirect; update for cleanliness | `grep -rIin 'raw.githubusercontent.com/johnwick2921-cyber/nofx' . --exclude-dir=.git \| wc -l` |
| 4.4 | branches | 25+ remote branches, incl. `origin/docs/brand-census` (prior census) | 25+ | URL | yes | nothing | branch names are cosmetic | `git branch -r` |
| 4.5 | `.github/CODEOWNERS:24` | `@NoFxAiOS` + upstream contributor handles | 1 | URL | migration | GitHub org/account names (external) | edit only after upstream accounts confirmed; never rename external handles | `grep -n NoFxAiOS .github/CODEOWNERS` |
| 4.6 | `.github/workflows/` | 11 workflow files, docker image tags (see 3.4), workflow README "for the NOFX project" | 11 files | URL (Actions UI) | yes | Actions | string swaps | `ls .github/workflows/` |
| 4.7 | `~/vlautoagenttraderv1` | partner repo, VL-named, remote `nofxlocal` → `/home/hoang/nofx` | 1 path | none | yes | local remote | propagate via format-patch after each phase (CLAUDE.md) | `git -C ~/vlautoagenttraderv1 remote -v` |
| 4.8 | GHCR / DockerHub | image names `nofx-*` (from 3.4) | 2 registries | none | migration | registries | retag | (see 3.4 command) |

### Group 5 — DOCS (sized; renamed by sed at the end)

Count per `docs/` directory (command: `for d in docs/*/; do printf "%s " "$d"; grep -rIio nofx "$d" | wc -l; done`):

| dir | count | dir | count |
|---|---|---|---|
| docs/superpowers | 10,297 | docs/legal | 58 |
| docs/i18n | 379 | docs/internal | 52 |
| docs/guides | 120 | docs/operations | 47 |
| docs/plans | 107 | docs/architecture | 40 |
| docs/community | 104 | docs/getting-started | 39 |
| docs/partner-sync | 13 | docs/maintainers | 24 |
| docs/api | 13 | docs/roadmap | 8 |
| docs/research | 2 | docs/agent-skills | 1 |
| docs/adr | 0 | docs/regime-wave | 0 |

**docs total 11,405** = 81 % of all occurrences. Of that, 6,464 are the two prior-census files themselves (delete the duplicate; regenerate or retain history as-is). The rest is prose → mechanical sed, zero risk.

---

## 3 · D2 — LIVE-SURFACE TRUTH (A15): what the owner sees TODAY that says "nofx"

- **Screen (Studio):** NOFXi labels in the AgentBeta chat — message attribution "NOFXi ·" (`ChatMessages.tsx:133`), input placeholder "Ask NOFXi anything… ⌘K" (`ChatInput.tsx:123-124`), disclaimer "NOFXi may make mistakes…" (`ChatInput.tsx:187`), welcome "What can I help with?"/zh variant (`WelcomeScreen.tsx:118`). Guide architecture diagram says "Go bot (nofx-bin)" (`guide/content/welcome.ts:37`). **The tab title, header brand, and icon are ALREADY VL** (`index.html:8`, `HeaderBar.tsx:95`, `icons/vl.svg`) — the gold theme remains CSS classes `--nofx-*` (invisible as text).
- **Logs:** boot banner `🚀 NOFX - AI-Powered Trading System` every restart (`main.go:44`); log files named `nofx_YYYY-MM-DD.log` (`logger/logger.go:90`); backup timer log line "nofx-backup: done (…)" (`nofx-db-backup.sh:79`); process name `./nofx-bin` in ps/journald; journald dropin installed as `nofx.conf`.
- **Alerts/Telegram:** `/start` → "✅ NOFX 就绪…/NOFX is ready!", `/help` → "NOFX 使用指南/NOFX Help"; agent replies speak as "NOFX quantitative trading system AI assistant".
- **URLs:** `github.com/johnwick2921-cyber/nofx`; GitHub README `<h1>NOFX</h1>`; 5 raw URLs inside prior reports; Actions UI shows workflow image tags `nofx-*`.
- **systemd --user output:** unit names `nofx-backup.service/.timer`, `nofx-clock-guard.service/.timer`.
- **What does NOT surface:** Prometheus metrics (no scraper configured [B]), docker/nginx (not used locally), DB rows (zero nofx), NT8 AddOns (already VL).

## 4 · D3 — proposed RENAME ORDER (nothing breaks while running)

| Phase | Scope | Boot needed? | NT8 restart? |
|---|---|---|---|
| **1 — visible brand** | G1 rows 1.1-1.11: banner, NOFXi chat labels, Telegram texts, README/i18n headings, SECURITY prose, guide diagram. Pure string swaps. | Yes (banner + Telegram strings read at start) | No |
| **2 — identifiers with compat alias** | binary → `vl-bin` (+ symlink `nofx-bin` one boot), systemd units → `vl-backup`/`vl-clock-guard` (+ stop old, `daemon-reload`), env `NOFX_*`→`VL_*` dual-read, log file name, lock/backup paths (symlinks), deploy scripts updated in the SAME commit (2.4, 2.7, 2.10, 2.11, 2.23) | Yes (units + binary swap; old aliases kept one boot) | No |
| **3 — module path + imports** | `go.mod` `module nofx`→`vl` + all 1,058 import lines in 545 files, npm name, Makefile. One mechanical commit, full suite green. | Yes (new binary from renamed tree) | No |
| **4 — migration-required** | 3.1 TCP `source:"nofx-go"`→`vl-go` (Go+C# lockstep, both accept old); 3.2 `NofxTrader\account.txt` (copy dir, dual-read); 3.3 localStorage dual-read; 3.4 CI image retag; 2.21 JWT issuer `nofxAI`→`vlAI` with dual-accept until token TTL. DB: none needed. | Yes (Go side of TCP) | **Yes** (C# AddOn + account dir) |
| **5 — external** | rename GitHub repo (redirects), update the 5 raw URLs, retag registries, CODEOWNERS after upstream accounts. Partner repo via format-patch after phases 2-4. | No | No |

Phases needing a boot: 1, 2, 3, 4 (Go). NT8 restart: phase 4 only. A phase-1+2 combined boot is possible; phase 4 must be its own cutover (wire protocol).

## 5 · D4 — what "VL" already covers, and the canonical forms

Existing VL forms (counts, command `grep -rIio 'vltrader\|vlai\|VL Intelligent\|"vl"' …` across `*.go|*.cs|*.tsx|*.ts|*.md`):

- `VLTrader` ×234, `VL ` ×111, `vltrader` ×80, `"VL"` ×18, `VLtrader` ×1 (**inconsistent casing — the one to fix**)
- `web/index.html:8` page title "VL Trader - AI Trading System" · `HeaderBar.tsx:95` header "VL" · `web/public/icons/vl.svg` (aria-label "VL")
- NT8 AddOns: `VLBarsSubscriptionManager.cs`, `VLContractResolver.cs`, `VLTraderTCPClient.cs` (+ `vltrader_tcp_PROTOCOL.md`, `_README.md`, `_VERIFY.md`)
- Wire: `source: "vltrader-addon"` (peer side, already VL)
- Partner repo `vlautoagenttraderv1`

**Canonical proposal (one form each):** product **VL Intelligent** (docs/UI prose) · binary/unit prefix **vl** (`vl-bin`, `vl-backup`, `vl-clock-guard`) · Go module **vl** · JWT issuer **vlAI** (mirrors the existing `nofxAI` pattern) · agent name **VLi** (mirrors "NOFXi") · Prometheus prefix **vl_** · env prefix **VL_** · NT8 names stay **VLTrader/VLBars…/VLContract…** (already canonical) · wire source **vl-go** ↔ **vltrader-addon**. This collapses every surface onto the existing VL convention instead of inventing a second one.

## 6 · D5 — re-runnable grep commands (census shrinks to zero per phase)

```bash
cd <repo>                                              # wherever the tree lives that phase
grep -rIio nofx    . --exclude-dir=.git | wc -l        # headline count (→ 0 at the end)
grep -rIio nofxai  . --exclude-dir=.git | wc -l        # issuer/API name form
grep -rIio nofxi   . --exclude-dir=.git | wc -l        # assistant brand form
grep -rIin '"nofx/' --include='*.go' . | wc -l         # module imports (phase 3)
grep -rIin 'NOFX_' --include='*.sh' --include='*.yml' --include='*.yaml' . | wc -l   # env prefixes
grep -rIil nofx . --exclude-dir=.git | wc -l           # files remaining
grep -rIin 'nofx' deploy/ scripts/ | grep -v '"nofx/' | wc -l   # hardcoded paths/units
grep -rIin 'raw.githubusercontent.com/johnwick2921-cyber/nofx' . --exclude-dir=.git | wc -l
sqlite3 'file:/home/hoang/nofx/data/data.db?mode=ro' ".schema" | grep -in nofx | wc -l
sqlite3 'file:/home/hoang/nofx/data/data.db?mode=ro' "SELECT COUNT(*) FROM traders WHERE lower(CAST(id AS TEXT)) LIKE '%nofx%' OR lower(CAST(name AS TEXT)) LIKE '%nofx%';"
sqlite3 'file:/home/hoang/nofx/data/data.db?mode=ro' "SELECT COUNT(*) FROM strategies WHERE lower(CAST(name AS TEXT)) LIKE '%nofx%';"
sqlite3 'file:/home/hoang/nofx/data/data.db?mode=ro' "SELECT COUNT(*) FROM plans WHERE CAST(plan_id AS TEXT) LIKE '%nofx%';"
ls ~/.config/systemd/user/ | grep -i nofx              # units (phase 2)
ls '/mnt/c/Users/hoang/Documents/NinjaTrader 8/bin/Custom/AddOns/'   # NT8 side (already VL)
```

---

*Census closed read-only. No files, config, DB rows, units, or assets were modified. Worktree removed after push; `git ls-remote origin docs/rebrand-census-0903` equals this report's commit sha (A18).*
