# Partner Patch Bundle — MANIFEST 2026-08-20

- **Target fork:** `johnwick2921-cyber/vlautoagenttraderv1` (branch `main`)
- **Source repo:** nofx — sync point `3624a2a4` (Part A complete, 2026-08-13) == partner HEAD `2bf3342d`
- **Bundle endpoint:** nofx `17cd52e2` (the deployed revision; HEAD adds only two docs commits)
- **Series:** `patches/partner-2026-08-20/0001..0264` (264 patches, 263 commits + 1 supplement)
- **Apply method (standing rule):** owner delivers the folder; partner runs
  `git am --3way patches/partner-2026-08-20/*.patch` from his `main` at `2bf3342d`.
- **No push to any remote.** Patches travel as files only.

## Apply-test proof (scratch clone of the partner fork, 2026-08-20)

| Check | Result |
|---|---|
| `git am --3way` on all 265 patch files | **265 applied / 0 three-way fallbacks / 0 conflicts** |
| Tree vs nofx HEAD (all code paths) | identical; 0 unexpected file differences |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| `go test ./... -count=1` | PASS (all packages ok, 0 FAIL) |
| `npm ci` + `npm run build` (web) | PASS (chunk-size warning only) |
| `npx vitest run` (web) | PASS — 32 files / 263 tests, all green |

## Special checks (partner known state)

1. **SendAccountsList / LFE-filter fix — ALREADY IN THEIR FORK.** The whole
   LFE guard chain predates the sync point and is byte-identical in both repos:
   C# `IsSimAccount` submit-site checks (10 sites, files identical),
   Go `isAccountTradeable` (`dedc67f0`, 2026-06-05), API `Non-SIM accounts not
   selectable` guard (`api/handler_account.go`). LFE can never cross as a
   submit target. **Nothing to add.**
2. **65-vs-60 min-confidence default fix — IN THE BUNDLE.** `6e7863f1` (6.1) +
   `7bdda06c` (E5 v1 strays). Unset strategies are now judged at the shared
   default **60** everywhere (gate clamp == futures prompt).
3. **June rollover / min-bar-depth fixes — ALREADY IN THEIR FORK.** Partner
   history carries the June campaign (`f05baa96` contract-roll postmortem,
   `597d12fd` bars_back 500→2000, `b3dc0a85` BarsRequest rebuild, `1584cc6a`
   debounce). No duplicates shipped.

## Security gate (Rule H) — final patch-file scan output

Grepped every final `.patch` for: `sk-…` API keys, `AKIA…`, `LFE`, owner
username `hoang`, `/home/hoang` paths, `DESKTOP-S4IA601`, owner prompt-box text
(`Focus On Quality…`), legacy `cm_…` nofxos key, owner UUIDs, SQL `INSERT INTO`,
DB dumps, and anything under `docs/superpowers/`.

- All secret/credential patterns: **0 hits**.
- `LFE` in patch text: **0 hits** (guard chain already in fork; nothing new crosses).
- Owner paths/names: **0 hits** — five patches were rewritten pre-scan
  (test fixtures neutralized `0001`, `0066`, `0195`; clock-guard deploy files
  genericized `$HOME`/`%h` in the clock-guard patch), one commit was excluded
  outright (`7e152559` forensic tool carrying an owner live-DB model UUID),
  and 10 mixed commits were path-limited to code-only (docs/`superpowers`
  artifacts, `deploy/RELEASE`, screenshots dropped).
- `deploy/RELEASE` diff hunks: stripped from 2 patches (boot-integrity +
  session-toggle) — the partner sets his own revision.
- Commit-message/comment *references* to doc paths remain (metadata only, no content).
- Author identity line (`johnwick2921-cyber <…@gmail.com>`) is inherent to
  `git format-patch` and is the same public identity on both forks.

## Patch series (seq | patch file | commit | one-liner | conflicts expected)

Conflicts column is proven **N** for the full series by the apply-test above.

| seq | patch file | commits | what it does (partner-facing one-liner) | conflicts |
|---|---|---|---|---|
| 1 | `0001-8e8591a8.patch` | `8e8591a8` | =?UTF-8?q?fix(web):=20trader=20selection=20persists=20?= | N |
| 2 | `0002-e3276a7d.patch` | `e3276a7d` | feat(web): active-trader marker in the traders list | N |
| 3 | `0003-d1851dac.patch` | `d1851dac` | =?UTF-8?q?feat(dayplan):=20P0.1=20=E2=80=94=20day=5Fplan?= | N |
| 4 | `0004-0a974d31.patch` | `0a974d31` | =?UTF-8?q?feat(dayplan):=20P0.2=20=E2=80=94=20plans/plan?= | N |
| 5 | `0005-6a0d233b.patch` | `6a0d233b` | =?UTF-8?q?feat(dayplan):=20P0.3=20=E2=80=94=20CT-anchored?= | N |
| 6 | `0006-b51ab5c2.patch` | `b51ab5c2` | =?UTF-8?q?feat(dayplan):=20P0.4=20=E2=80=94=20scenario-fa?= | N |
| 7 | `0007-041e4450.patch` | `041e4450` | =?UTF-8?q?feat(dayplan):=20P0.5=20=E2=80=94=20level-state?= | N |
| 8 | `0008-89d7e1b3.patch` | `89d7e1b3` | =?UTF-8?q?test(dayplan):=20config-truth=20=E2=80=94=20day?= | N |
| 9 | `0009-14adc47e.patch` | `14adc47e` | =?UTF-8?q?feat(dayplan):=20P1.1=20=E2=80=94=20session-tag?= | N |
| 10 | `0010-e1cd5993.patch` | `e1cd5993` | =?UTF-8?q?feat(dayplan):=20P1.2=20=E2=80=94=20round=20num?= | N |
| 11 | `0011-3058e131.patch` | `3058e131` | =?UTF-8?q?feat(dayplan):=20P1.5=20=E2=80=94=20confluence?= | N |
| 12 | `0012-15e55520.patch` | `15e55520` | =?UTF-8?q?feat(dayplan):=20P1.6=20=E2=80=94=20regime=20bl?= | N |
| 13 | `0013-9436ea79.patch` | `9436ea79` | =?UTF-8?q?test(dayplan):=20P1=20sample=20=E2=80=94=20full?= | N |
| 14 | `0014-f138b7f3.patch` | `f138b7f3` | =?UTF-8?q?feat(dayplan):=20P1.4=20=E2=80=94=20EQH/EQL=20+?= | N |
| 15 | `0015-c42a629c.patch` | `c42a629c` | =?UTF-8?q?feat(dayplan):=20P1.7=20=E2=80=94=20wire=20KEY?= | N |
| 16 | `0016-a450adfc.patch` | `a450adfc` | =?UTF-8?q?feat(dayplan):=20P1.3=20=E2=80=94=20durable=20s?= | N |
| 17 | `0017-09579af3.patch` | `09579af3` | =?UTF-8?q?feat(dayplan):=20P1.8=20=E2=80=94=20calendar=20?= | N |
| 18 | `0018-069500c9.patch` | `069500c9` | =?UTF-8?q?feat(dayplan):=20P2.1=20=E2=80=94=20true=20bar-?= | N |
| 19 | `0019-3bb7a730.patch` | `3bb7a730` | =?UTF-8?q?feat(dayplan):=20P2.2=20=E2=80=94=20skip-while-?= | N |
| 20 | `0020-21c57118.patch` | `21c57118` | =?UTF-8?q?feat(dayplan):=20P2.3=20=E2=80=94=20day-trader?= | N |
| 21 | `0021-a43d006d.patch` | `a43d006d` | =?UTF-8?q?feat(dayplan):=20P2.4=20=E2=80=94=20MAE/MFE=20+?= | N |
| 22 | `0022-b0151d98.patch` | `b0151d98` | =?UTF-8?q?feat(dayplan):=20P2.5=20=E2=80=94=20day=5Fplan?= | N |
| 23 | `0023-389c0c9e.patch` | `389c0c9e` | =?UTF-8?q?feat(dayplan):=20P3.1=20=E2=80=94=20session=20g?= | N |
| 24 | `0024-05703a08.patch` | `05703a08` | =?UTF-8?q?feat(dayplan):=20P3.2=20=E2=80=94=20planner=20m?= | N |
| 25 | `0025-5d06b063.patch` | `5d06b063` | =?UTF-8?q?feat(dayplan):=20P3.3=20=E2=80=94=20read=20jobs?= | N |
| 26 | `0026-d0d96a10.patch` | `d0d96a10` | =?UTF-8?q?feat(dayplan):=20P3.4=20=E2=80=94=20executor=20?= | N |
| 27 | `0027-d2d4a975.patch` | `d2d4a975` | =?UTF-8?q?feat(dayplan):=20P3.5=20=E2=80=94=20advisory=20?= | N |
| 28 | `0028-0a292148.patch` | `0a292148` | =?UTF-8?q?feat(dayplan):=20P3.6=20core=20=E2=80=94=20acti?= | N |
| 29 | `0029-f9f5460f.patch` | `f9f5460f` | =?UTF-8?q?feat(dayplan):=20P3.6-A=20=E2=80=94=20digest=20?= | N |
| 30 | `0030-29727291.patch` | `29727291` | =?UTF-8?q?feat(dayplan):=20P3.6-B=20=E2=80=94=20scenario?= | N |
| 31 | `0031-9f000637.patch` | `9f000637` | =?UTF-8?q?feat(dayplan):=20P3.6-C=20=E2=80=94=20sticky=20?= | N |
| 32 | `0032-e19403fc.patch` | `e19403fc` | =?UTF-8?q?feat(dayplan):=20P3.6-D=20=E2=80=94=20night=20m?= | N |
| 33 | `0033-7674261b.patch` | `7674261b` | =?UTF-8?q?feat(dayplan):=20P4.1=20=E2=80=94=20/api/plan/*?= | N |
| 34 | `0034-40ec67a3.patch` | `40ec67a3` | =?UTF-8?q?fix(dayplan):=20P4.1=20=E2=80=94=20scope=20aler?= | N |
| 35 | `0035-f1ccb76e.patch` | `f1ccb76e` | =?UTF-8?q?feat(dayplan):=20P4=20FE=20foundation=20?= | N |
| 36 | `0036-842edb57.patch` | `842edb57` | =?UTF-8?q?feat(dayplan):=20P4.2=20=E2=80=94=20SessionTime?= | N |
| 37 | `0037-fd27f63a.patch` | `fd27f63a` | =?UTF-8?q?feat(dayplan):=20P4.3=20=E2=80=94=20SessionPlan?= | N |
| 38 | `0038-d4564cf5.patch` | `d4564cf5` | =?UTF-8?q?feat(dayplan):=20P4.4=20=E2=80=94=20in-app=20al?= | N |
| 39 | `0039-859353b3.patch` | `859353b3` | =?UTF-8?q?feat(dayplan):=20P4.5=20=E2=80=94=20Studio=20Da?= | N |
| 40 | `0040-899f14f5.patch` | `899f14f5` | =?UTF-8?q?feat(dayplan):=20P5.1=20=E2=80=94=20overlay=20A?= | N |
| 41 | `0041-94fa6daf.patch` | `94fa6daf` | =?UTF-8?q?feat(dayplan):=20P5.4=20backend=20=E2=80=94=20A?= | N |
| 42 | `0042-ad9dd424.patch` | `ad9dd424` | =?UTF-8?q?feat(dayplan):=20P5.5=20=E2=80=94=20adherence?= | N |
| 43 | `0043-fa9344b2.patch` | `fa9344b2` | =?UTF-8?q?feat(dayplan):=20P5.6=20=E2=80=94=20stats=20hon?= | N |
| 44 | `0044-3126b61e.patch` | `3126b61e` | =?UTF-8?q?feat(dayplan):=20P5.2+P5.3+P5.4=20FE=20?= | N |
| 45 | `0045-e99a145f.patch` | `e99a145f` | feat(dayplan): P5.2 bulk add + P5.7 blind-mark grammar parser | N |
| 46 | `0046-41afa1b6.patch` | `41afa1b6` | =?UTF-8?q?fix(dayplan):=20P5=20hardening=20=E2=80=94=20ad?= | N |
| 47 | `0047-79a58523.patch` | `79a58523` | =?UTF-8?q?fix(dayplan):=20Studio=20Day=20Plan=20settings?= | N |
| 48 | `0048-18e9e5ac.patch` | `18e9e5ac` | feat(dayplan): planner assembler honors day_plan config | N |
| 49 | `0049-57b518d8.patch` | `57b518d8` | =?UTF-8?q?fix(dayplan):=20W1=20=E2=80=94=20Sunday-read=20?= | N |
| 50 | `0050-0c8bae59.patch` | `0c8bae59` | =?UTF-8?q?fix(dayplan):=20W2=20=E2=80=94=20pin=20the=20EX?= | N |
| 51 | `0051-89e2809f.patch` | `89e2809f` | =?UTF-8?q?feat(dayplan):=20W3=20=E2=80=94=20calendar=20pr?= | N |
| 52 | `0052-5729e2b5.patch` | `5729e2b5` | =?UTF-8?q?feat(dayplan):=20W4=20=E2=80=94=20owner=20overl?= | N |
| 53 | `0053-e165f35e.patch` | `e165f35e` | =?UTF-8?q?feat(dayplan):=20W5=20=E2=80=94=20learning=20lo?= | N |
| 54 | `0054-da3fb458.patch` | `da3fb458` | =?UTF-8?q?feat(dayplan):=20W6=20=E2=80=94=20wire=20alert?= | N |
| 55 | `0055-77ffbe8f.patch` | `77ffbe8f` | =?UTF-8?q?feat(dayplan):=20W7=20=E2=80=94=20wire=20the=20?= | N |
| 56 | `0056-fb5bd624.patch` | `fb5bd624` | =?UTF-8?q?feat(dayplan):=20W8=20=E2=80=94=20gates=20read?= | N |
| 57 | `0057-1d476407.patch` | `1d476407` | =?UTF-8?q?feat(dayplan):=20W9=20=E2=80=94=20wire=20the=20?= | N |
| 58 | `0058-0f79fb4f.patch` | `0f79fb4f` | =?UTF-8?q?feat(dayplan):=20W10=20=E2=80=94=20supply=20the?= | N |
| 59 | `0059-cbf12870.patch` | `cbf12870` | =?UTF-8?q?feat(dayplan):=20W11=20=E2=80=94=20planner=20in?= | N |
| 60 | `0060-2985e50f.patch` | `2985e50f` | =?UTF-8?q?fix(dayplan):=20F0.1=20=E2=80=94=20calendar=20i?= | N |
| 61 | `0061-3d91e574.patch` | `3d91e574` | =?UTF-8?q?feat(dayplan):=20F0.2=20=E2=80=94=20static=20T1?= | N |
| 62 | `0062-5578966a.patch` | `5578966a` | =?UTF-8?q?feat(dayplan):=20F0.3=20=E2=80=94=20Sunday=20fe?= | N |
| 63 | `0063-c886b359.patch` | `c886b359` | =?UTF-8?q?test(dayplan):=20F0.4=20=E2=80=94=20calendar=20?= | N |
| 64 | `0064-a7bdae50.patch` | `a7bdae50` | =?UTF-8?q?feat(dayplan):=20W11b=20=E2=80=94=20surface=20p?= | N |
| 65 | `0065-298d75b0.patch` | `298d75b0` | =?UTF-8?q?test(dayplan):=20W12=20=E2=80=94=20math=20corre?= | N |
| 66 | `0066-205b1753.patch` | `205b1753` | =?UTF-8?q?docs:=20acceptance=20gate=20v2=20=E2=80=94=20dr?= | N |
| 67 | `0067-81e54fc9.patch` | `81e54fc9` | =?UTF-8?q?fix(security):=20S1=20=E2=80=94=20bind=20the=20?= | N |
| 68 | `0068-821c895a.patch` | `821c895a` | =?UTF-8?q?fix(security):=20S2=20=E2=80=94=20reset-passwor?= | N |
| 69 | `0069-0d97a672.patch` | `0d97a672` | docs(security): document API_SERVER_HOST and | N |
| 70 | `0070-92969990.patch` | `92969990` | =?UTF-8?q?fix(security):=20S5=20=E2=80=94=20stop=20writin?= | N |
| 71 | `0071-8dcc4521.patch` | `8dcc4521` | =?UTF-8?q?fix(security):=20S4=20=E2=80=94=20close=20the?= | N |
| 72 | `0072-05f8ce60.patch` | `05f8ce60` | =?UTF-8?q?fix(plan):=20blank=20Plan=20Card=20on=20a=20sec?= | N |
| 73 | `0073-db5949fa.patch` | `db5949fa` | =?UTF-8?q?feat(dayplan):=20W13.1=20=E2=80=94=20plan=20re-?= | N |
| 74 | `0074-14b26aae.patch` | `14b26aae` | =?UTF-8?q?feat(dayplan):=20W13.2=20=E2=80=94=20re-align?= | N |
| 75 | `0075-a6f39d7b.patch` | `a6f39d7b` | feat(sandbox): isolated hands-on sandbox on :3001 (own DB, no | N |
| 76 | `0076-517df64c.patch` | `517df64c` | =?UTF-8?q?fix(sandbox):=20empty=20plan=20card=20=E2=80=94?= | N |
| 77 | `0077-a8af893b.patch` | `a8af893b` | =?UTF-8?q?feat(safety):=20P1=20boot=20integrity=20asserti?= | N |
| 78 | `0078-66ccb500.patch` | `66ccb500` | =?UTF-8?q?fix(session):=20P3=20session-end=20drift=20?= | N |
| 79 | `0079-542dfdc8.patch` | `542dfdc8` | feat(regime): P2 dark-regime alert + DEGRADED plan flag | N |
| 80 | `0080-bc360a38.patch` | `bc360a38` | =?UTF-8?q?feat(dayplan):=20W15.A=20=E2=80=94=20a=20REAL?= | N |
| 81 | `0081-d396201e.patch` | `d396201e` | =?UTF-8?q?fix(dayplan):=20W15.B=20=E2=80=94=20every=20dea?= | N |
| 82 | `0082-e943f9c3.patch` | `e943f9c3` | =?UTF-8?q?fix(dayplan):=20W15.C=20=E2=80=94=20the=20plan?= | N |
| 83 | `0083-a84d6ae2.patch` | `a84d6ae2` | =?UTF-8?q?fix(security):=20bind=20the=20live=20dev=20UI?= | N |
| 84 | `0084-9dffec72.patch` | `9dffec72` | test(dayplan): pin three W15.A follow-on defects with | N |
| 85 | `0085-e33532e2.patch` | `e33532e2` | =?UTF-8?q?fix(dayplan):=20scope=20the=20owner=20door=20to?= | N |
| 86 | `0086-2427c850.patch` | `2427c850` | fix(dayplan): Ask-Planner must read plan_final, not the base | N |
| 87 | `0087-637b137a.patch` | `637b137a` | fix(dayplan): a NO-TRADE plan must not render as gold | N |
| 88 | `0088-f7fa2d3c.patch` | `f7fa2d3c` | fix(trader): a gate-REFUSED entry must not be recorded as a | N |
| 89 | `0089-0b8443f9.patch` | `0b8443f9` | =?UTF-8?q?fix(dayplan):=20R2=20=E2=80=94=20declining=20a?= | N |
| 90 | `0090-735ce88e.patch` | `735ce88e` | style: gofmt store/plan_qa.go (R2 follow-up) | N |
| 91 | `0091-5db9d3c9.patch` | `5db9d3c9` | =?UTF-8?q?fix(dayplan):=20R6=20=E2=80=94=20one=20planner?= | N |
| 92 | `0092-4381c801.patch` | `4381c801` | =?UTF-8?q?fix(trader):=20R5=20=E2=80=94=20a=20post-fill?= | N |
| 93 | `0093-9e7b3259.patch` | `9e7b3259` | =?UTF-8?q?feat(dayplan):=20R3=20=E2=80=94=20"why=20was=20?= | N |
| 94 | `0094-0504516f.patch` | `0504516f` | =?UTF-8?q?fix(dayplan):=20R4=20=E2=80=94=20scope=20the=20?= | N |
| 95 | `0095-97e4f541.patch` | `97e4f541` | =?UTF-8?q?fix(kernel):=20R7=20=E2=80=94=20remove=20the=20?= | N |
| 96 | `0096-c4996495.patch` | `c4996495` | =?UTF-8?q?feat(dayplan):=20R1=20=E2=80=94=20real=20per-sc?= | N |
| 97 | `0097-7e5932fc.patch` | `7e5932fc` | fix(risk): an UNSET risk value must never be the loosest | N |
| 98 | `0098-b4b28895.patch` | `b4b28895` | test(risk): the regression test that ends the | N |
| 99 | `0099-2c15f000.patch` | `2c15f000` | =?UTF-8?q?feat(cmd):=20dayplan-sessions=20=E2=80=94=20app?= | N |
| 100 | `0100-7aa521a1.patch` | `7aa521a1` | fix(nt8): end the bar-watchdog livelock that ran 100k+ | N |
| 101 | `0101-06f3343e.patch` | `06f3343e` | fix(dayplan): a plan may only die on evidence that POSTDATES | N |
| 102 | `0102-f6521df9.patch` | `f6521df9` | fix(plan-card): an empty plan must say WHY, and the chart | N |
| 103 | `0103-20a64c51.patch` | `20a64c51` | fix(dayplan): "2x5m" must mean two FIVE-minute closes, not | N |
| 104 | `0104-8b24c85e.patch` | `8b24c85e` | fix(dayplan): the executor prompt quotes the real re-plan | N |
| 105 | `0105-745f84f7.patch` | `745f84f7` | feat(plan): the read path for plan version history (ITEM 15, | N |
| 106 | `0106-47dbb269.patch` | `47dbb269` | =?UTF-8?q?fix(bars):=20refuse=20NT8's=20empty-minute=20pl?= | N |
| 107 | `0107-f630ceea.patch` | `f630ceea` | feat(plan): version chips open the version they name (ITEM | N |
| 108 | `0108-499dd59f.patch` | `499dd59f` | fix(dayplan): a re-plan that strands owner edits now says so | N |
| 109 | `0109-403a9205.patch` | `403a9205` | =?UTF-8?q?fix(plan):=20the=20FOOTER=20version=20chips=20w?= | N |
| 110 | `0110-678b880f.patch` | `678b880f` | =?UTF-8?q?fix(dayplan):=20post-mortem=20on=20the=206=20AS?= | N |
| 111 | `0111-301df4be.patch` | `301df4be` | =?UTF-8?q?fix(dayplan):=20the=20cap=20DID=20work=20?= | N |
| 112 | `0112-243787d8.patch` | `243787d8` | feat(plan): Ask-Planner opens with no active plan (labelled, | N |
| 113 | `0113-9364979a.patch` | `9364979a` | =?UTF-8?q?feat(plan):=20=E2=9F=B3=20Re-read=20=E2=80=94?= | N |
| 114 | `0114-ca819d91.patch` | `ca819d91` | =?UTF-8?q?fix(decisions):=20every=20decision=20carries=20?= | N |
| 115 | `0115-3594c2af.patch` | `3594c2af` | =?UTF-8?q?feat(alerts):=20the=20feed=20can=20be=20cleared?= | N |
| 116 | `0116-3ea0a442.patch` | `3ea0a442` | feat(plan): owner levels survive a re-plan, re-anchored by | N |
| 117 | `0117-2ddf3a58.patch` | `2ddf3a58` | fix(decision): recover prose-embedded JSON and retry/alert on | N |
| 118 | `0118-570c6c32.patch` | `570c6c32` | fix(h8): sessionRunnable is the single source of truth for | N |
| 119 | `0119-2b4162f6.patch` | `2b4162f6` | fix(latency): decision call capped inside the bar window + | N |
| 120 | `0120-6934a18e.patch` | `6934a18e` | fix(h9): the planner fetches the CONFIGURED timeframes and | N |
| 121 | `0121-f13ecb86.patch` | `f13ecb86` | fix(h4+h5): max_levels/scenario_cap reach validation AND the | N |
| 122 | `0122-8338aaea.patch` | `8338aaea` | fix(h1+h2): proximity_filter_atr governs level generation AND | N |
| 123 | `0123-a47a51a6.patch` | `a47a51a6` | fix(h7): executor prompt reads the PERSISTED admin registry, | N |
| 124 | `0124-9199e9f0.patch` | `9199e9f0` | =?UTF-8?q?fix(p7):=20NO-TRADE=20plans=20still=20carry=20t?= | N |
| 125 | `0125-020e407a.patch` | `020e407a` | =?UTF-8?q?feat(p6):=20the=20owner=20reset=20=E2=80=94=20a?= | N |
| 126 | `0126-480a4d51.patch` | `480a4d51` | fix(h10): acceptance counted on the rule's timeframe | N |
| 127 | `0127-77623924.patch` | `77623924` | test(h10): rule-timeframe fixtures + no-hardcoded-interval | N |
| 128 | `0128-99fd67e1.patch` | `99fd67e1` | =?UTF-8?q?fix(p0a):=20cross-trader=20plan=20governance=20?= | N |
| 129 | `0129-1e8cc591.patch` | `1e8cc591` | =?UTF-8?q?fix(p0b):=20the=20ASIA=20clock=20=E2=80=94=2016?= | N |
| 130 | `0130-91748082.patch` | `91748082` | =?UTF-8?q?fix(p1):=20H8=20residuals=20=E2=80=94=20the=20A?= | N |
| 131 | `0131-1a95a444.patch` | `1a95a444` | p1c: consumed levels role-flip and stay; consumption windowed | N |
| 132 | `0132-5fe152d7.patch` | `5fe152d7` | p1d: explicit prior-session aging for consumed levels | N |
| 133 | `0133-a85d7363.patch` | `a85d7363` | p1b: dayplan-level-repair purge tool | N |
| 134 | `0134-3b7a1d7c.patch` | `3b7a1d7c` | p1e: tests for wick/touch/window consumption, aging, | N |
| 135 | `0135-8559ea0c.patch` | `8559ea0c` | p0a2: trader-scoped plan identity (plans PK no longer | N |
| 136 | `0136-65834726.patch` | `65834726` | b-fix: wire alert-feed prune; delete dead HandoverBanner | N |
| 137 | `0137-9c10c149.patch` | `9c10c149` | c-fix: write ai_latency_ms + minimal discipline consumers | N |
| 138 | `0138-6a0cf41d.patch` | `6a0cf41d` | fix duplicate package clause in dayplan-level-repair | N |
| 139 | `0139-1c418a1f.patch` | `1c418a1f` | d-sweep-fix: trader-scoped read claim, config max_levels, | N |
| 140 | `0140-aeaf7076.patch` | `aeaf7076` | why-no-trades: plan advisory must read as advisory, not | N |
| 141 | `0141-28b79f09.patch` | `28b79f09` | =?UTF-8?q?ui-verify:=20reset=20was=20works-but-silent=20?= | N |
| 142 | `0142-179fb229.patch` | `179fb229` | ui-verify: explain the default-strategy lock; align | N |
| 143 | `0143-97ead6e4.patch` | `97ead6e4` | fix(plan): edit sheet prefills exact instruction + owner | N |
| 144 | `0144-8d5cfa1f.patch` | `8d5cfa1f` | fix(plan): write owner note/scenario-tag edits through to | N |
| 145 | `0145-8440dfb8.patch` | `8440dfb8` | =?UTF-8?q?fix(p5.4):=20ONE=20PROMPT,=20ONE=20SNAPSHOT=20?= | N |
| 146 | `0146-a3a2c929.patch` | `a3a2c929` | fix(p5.5): preserve the last non-empty model output across | N |
| 147 | `0147-91333cfb.patch` | `91333cfb` | =?UTF-8?q?fix(p1c):=20consumption=20always=20requires=20a?= | N |
| 148 | `0148-4efd1137.patch` | `4efd1137` | fix(chart): equity-history default is a 90-day window, not | N |
| 149 | `0149-66a9664f.patch` | `66a9664f` | =?UTF-8?q?fix(chart):=20raise=20the=20FE=20equity-curve?= | N |
| 150 | `0150-4314d4fd.patch` | `4314d4fd` | fix(web): plan mini chart klines limit 500->1500 (show full | N |
| 151 | `0151-d6905dd3.patch` | `d6905dd3` | =?UTF-8?q?fix(kernel):=20touch-gate=20StillValid=20?= | N |
| 152 | `0152-c8f91d41.patch` | `c8f91d41` | =?UTF-8?q?fix(tz):=20P0=20=E2=80=94=20CT=20canonical=20ev?= | N |
| 153 | `0153-f6478923.patch` | `f6478923` | =?UTF-8?q?fix(ai):=20wire-proven=20truncation=20was=20zer?= | N |
| 154 | `0154-1b5139c0.patch` | `1b5139c0` | P0: AI params fully config-driven + decision-first contract | N |
| 155 | `0155-26c5d35a.patch` | `26c5d35a` | mcp: log completion tokens + finish_reason on every AI call | N |
| 156 | `0156-5b1a9927.patch` | `5b1a9927` | agent sub-call token caps env-driven + startup audit | N |
| 157 | `0157-1a6dcf74.patch` | `1a6dcf74` | =?UTF-8?q?fix(P0):=20zero-trades=20root=20cause=20?= | N |
| 158 | `0158-54f0cb6c.patch` | `54f0cb6c` | P0 planner-quality: both-side seating + cluster collapse + | N |
| 159 | `0159-5408abd1.patch` | `5408abd1` | =?UTF-8?q?P0=20planner-quality:=20facts-validated=20plans?= | N |
| 160 | `0160-73976c96.patch` | `73976c96` | P0 planner-quality: death/flip text is now machine-evaluated | N |
| 161 | `0161-e8a4b710.patch` | `e8a4b710` | =?UTF-8?q?P0=20calendar=20fail-closed:=20missing=20slice?= | N |
| 162 | `0162-c42b7280.patch` | `c42b7280` | P0 planner-quality: continuation trigger must be reachable | N |
| 163 | `0163-49413c0d.patch` | `49413c0d` | P0-cleanup item 2c+2a-core: refusals never bare waits + | N |
| 164 | `0164-7bcb238d.patch` | `7bcb238d` | P0-cleanup item 2b/2d: one error surface + daily error digest | N |
| 165 | `0165-547a2e89.patch` | `547a2e89` | =?UTF-8?q?P0-cleanup=20item=201:=20learning=20loop=20clos?= | N |
| 166 | `0166-4893ce37.patch` | `4893ce37` | =?UTF-8?q?P0-cleanup=20item=203:=20soft-alert=20guardrail?= | N |
| 167 | `0167-ef334135.patch` | `ef334135` | P0-cleanup item 4: level_state trader-scoped (cross-trader | N |
| 168 | `0168-bb966a04.patch` | `bb966a04` | P0-cleanup item 5: cosmetics + prior audit reports landed | N |
| 169 | `0169-33d7eed2.patch` | `33d7eed2` | =?UTF-8?q?fix(timegate):=20ai-call-timeout=20=E2=80=94=20?= | N |
| 170 | `0170-862bcd41.patch` | `862bcd41` | =?UTF-8?q?fix(timegate):=20last-entry=20cutoff=20?= | N |
| 171 | `0171-889c2437.patch` | `889c2437` | =?UTF-8?q?fix(timegate):=20eod-flat=20=E2=80=94=20day-sco?= | N |
| 172 | `0172-12f92bf7.patch` | `12f92bf7` | =?UTF-8?q?fix(timegate):=20ui-timelines=20=E2=80=94=20bro?= | N |
| 173 | `0173-fb9baa81.patch` | `fb9baa81` | =?UTF-8?q?fix(timegate):=20ai-timeout=20tests=20=E2=80=94?= | N |
| 174 | `0174-8944e07a.patch` | `8944e07a` | =?UTF-8?q?fix(timegate):=20ai-params=20wiring=20=E2=80=94?= | N |
| 175 | `0175-039f4bb8.patch` | `039f4bb8` | =?UTF-8?q?fix(timegate):=20session-digest=20close=20test?= | N |
| 176 | `0176-844c9580.patch` | `844c9580` | =?UTF-8?q?fix(timegate):=20weekly=20matched-random=20free?= | N |
| 177 | `0177-8218b4ea.patch` | `8218b4ea` | =?UTF-8?q?fix(timegate):=20comment=20truth=20=E2=80=94=20?= | N |
| 178 | `0178-4ebd779a.patch` | `4ebd779a` | =?UTF-8?q?fix(timegate):=20clock-health=20line=20?= | N |
| 179 | `0179-22ea40ad.patch` | `22ea40ad` | fix(bars): NT8 close-stamps converted to OPEN stamps once, at | N |
| 180 | `0180-e9084e38.patch` | `e9084e38` | =?UTF-8?q?fix(charts):=20ONE=20CT=20label=20site=20?= | N |
| 181 | `0181-a6b0ae24.patch` | `a6b0ae24` | =?UTF-8?q?fix(stale):=20B4=20age=20contract=20=E2=80=94?= | N |
| 182 | `0182-d4396912.patch` | `d4396912` | =?UTF-8?q?feat(feed):=20U1=20=E2=80=94=20feed=20loss=20an?= | N |
| 183 | `0183-00a4bef6.patch` | `00a4bef6` | =?UTF-8?q?fix(feed):=20liveness=20bar=5Fage=20reads=20the?= | N |
| 184 | `0184-44808dc0.patch` | `44808dc0` | =?UTF-8?q?fix(intrade):=20feed=20watch=20moves=20to=20the?= | N |
| 185 | `0185-37b3b6bd.patch` | `37b3b6bd` | =?UTF-8?q?fix(intrade):=20feed=20alert=20tightens=20to=20?= | N |
| 186 | `0186-c335f897.patch` | `c335f897` | =?UTF-8?q?fix(intrade):=20skip-while-open=20relocates=20B?= | N |
| 187 | `0187-d154bb44.patch` | `d154bb44` | =?UTF-8?q?fix(skip-gate):=20position-state=20reconciliati?= | N |
| 188 | `0188-51c2a9bf.patch` | `51c2a9bf` | =?UTF-8?q?feat(clock):=20root-free=20clock-guard=20user?= | N |
| 189 | `0189-d9978599.patch` | `d9978599` | =?UTF-8?q?feat(clock):=20CLOCK=5FWARN=5FMS=20early=20warn?= | N |
| 190 | `0190-71ee0444.patch` | `71ee0444` | =?UTF-8?q?feat(pause):=20stop=5Funtil=20gets=20its=20prod?= | N |
| 191 | `0191-ef66d44e.patch` | `ef66d44e` | =?UTF-8?q?feat(roll):=20contract-roll=20entry=20block=20f?= | N |
| 192 | `0192-a37aff45.patch` | `a37aff45` | =?UTF-8?q?feat(halfdays):=20the=20HalfDays=20map=20gets?= | N |
| 193 | `0193-d4ad9875.patch` | `d4ad9875` | =?UTF-8?q?fix(pause):=20until=5Fct=20parses=20by=20hand?= | N |
| 194 | `0194-d9a23407.patch` | `d9a23407` | =?UTF-8?q?feat(ai402):=20402=20becomes=20an=20OUTAGE,=20n?= | N |
| 195 | `0195-85ec608f.patch` | `85ec608f` | =?UTF-8?q?feat(logdb):=20WARN+ERROR=20ship=20to=20the=20l?= | N |
| 196 | `0196-ce1a3988.patch` | `ce1a3988` | =?UTF-8?q?fix(snapshot):=20ONE=20instant=20per=20cycle=20?= | N |
| 197 | `0197-1021c65b.patch` | `1021c65b` | =?UTF-8?q?fix(stale):=20B4=20refuses=20the=20hidden=20clo?= | N |
| 198 | `0198-59e98d3c.patch` | `59e98d3c` | =?UTF-8?q?test(gates):=20E5=20interaction-edge=20guard=20?= | N |
| 199 | `0199-c0e5ce43.patch` | `c0e5ce43` | =?UTF-8?q?fix(test):=20E5=20gate-order=20anchors=20use=20?= | N |
| 200 | `0200-e065b01e.patch` | `e065b01e` | =?UTF-8?q?feat(cadence):=20P10=20owner=20ruling=20?= | N |
| 201 | `0201-a4aa9e7c.patch` | `a4aa9e7c` | =?UTF-8?q?feat(cadence):=20P10.3/10.5=20=E2=80=94=20caden?= | N |
| 202 | `0202-e3e22a15.patch` | `e3e22a15` | =?UTF-8?q?fix(halfdays):=20E7-v2=20HIGH=20=E2=80=94=20cal?= | N |
| 203 | `0203-4d17cbae.patch` | `4d17cbae` | =?UTF-8?q?fix(halfdays):=20E7-v2=20=E2=80=94=20pull-in=20?= | N |
| 204 | `0204-cc7e45db.patch` | `cc7e45db` | =?UTF-8?q?fix(roll):=20E7-v2=20=E2=80=94=20daysLeft=20rou?= | N |
| 205 | `0205-26a90967.patch` | `26a90967` | =?UTF-8?q?fix(pause):=20E7-v2=20trio=20=E2=80=94=20until?= | N |
| 206 | `0206-14f93d11.patch` | `14f93d11` | =?UTF-8?q?fix(halfdays):=20E7-v2=20medium=20=E2=80=94=20a?= | N |
| 207 | `0207-f6447076.patch` | `f6447076` | =?UTF-8?q?feat(boot):=20E1=20completion=20=E2=80=94=20per?= | N |
| 208 | `0208-05bbca7b.patch` | `05bbca7b` | =?UTF-8?q?feat(logs):=20honest=20logs=20=E2=80=94=20journ?= | N |
| 209 | `0209-6c5ef518.patch` | `6c5ef518` | =?UTF-8?q?feat(discard-burn):=20dodge=20cycles=20that=20w?= | N |
| 210 | `0210-2625a04d.patch` | `2625a04d` | =?UTF-8?q?feat(watcher):=20in-position=20AI=20goes=20watc?= | N |
| 211 | `0211-7bcefb5d.patch` | `7bcefb5d` | fix(fmt): gofmt realign of the AutoTrader field block after | N |
| 212 | `0212-8f7edd1b.patch` | `8f7edd1b` | =?UTF-8?q?feat(trailing):=20trailing=20profit=20in=20Risk?= | N |
| 213 | `0213-6b4ab842.patch` | `6b4ab842` | =?UTF-8?q?feat(post-exit):=20one=20immediate=20full=20dec?= | N |
| 214 | `0214-8dd0dabb.patch` | `8dd0dabb` | =?UTF-8?q?fix(ask-planner):=20stuck=20send=20=E2=80=94=20?= | N |
| 215 | `0215-6e7863f1.patch` | `6e7863f1` | =?UTF-8?q?fix(minconf):=206.1=20=E2=80=94=20one=20shared?= | N |
| 216 | `0216-e2367618.patch` | `e2367618` | =?UTF-8?q?fix(cadence):=206.2=20=E2=80=94=20trader=20edit?= | N |
| 217 | `0217-d92bac9d.patch` | `d92bac9d` | =?UTF-8?q?fix(guardrails):=206.3=20=E2=80=94=20CheckSoft?= | N |
| 218 | `0218-0d6b58d3.patch` | `0d6b58d3` | =?UTF-8?q?fix(caps):=206.4=20ruling=20B=20=E2=80=94=20rem?= | N |
| 219 | `0219-d005fd40.patch` | `d005fd40` | =?UTF-8?q?fix(guardrails):=206.5=20=E2=80=94=20the=20$500?= | N |
| 220 | `0220-12ee138d.patch` | `12ee138d` | =?UTF-8?q?fix(docs):=206.6=20=E2=80=94=20three=20comments?= | N |
| 221 | `0221-39e99c39.patch` | `39e99c39` | =?UTF-8?q?feat(backfill):=206.7=20=E2=80=94=20one-time=20?= | N |
| 222 | `0222-1312fb79.patch` | `1312fb79` | =?UTF-8?q?chore(deprecate):=206.8=20sweep=20=E2=80=94=20r?= | N |
| 223 | `0223-6c734afb.patch` | `6c734afb` | =?UTF-8?q?fix(trailing):=20stage=20the=20Phase-3=20stub?= | N |
| 224 | `0224-1d67a675.patch` | `1d67a675` | =?UTF-8?q?fix(post-exit):=20E7=20self-grep=20=E2=80=94=20?= | N |
| 225 | `0225-7cefc00f.patch` | `7cefc00f` | =?UTF-8?q?fix(watcher-eyes):=20U1=20=E2=80=94=20watch=20c?= | N |
| 226 | `0226-39554ea8.patch` | `39554ea8` | fix(fmt): gofmt realign after the watcher-eyes edits (comment | N |
| 227 | `0227-4d4bca00.patch` | `4d4bca00` | fix(fe): shared guardedCall never-latch helper (extracted per | N |
| 228 | `0228-23a7be65.patch` | `23a7be65` | =?UTF-8?q?fix(fe):=20complete=20the=20Ask-Planner=20refit?= | N |
| 229 | `0229-d4c8c44f.patch` | `d4c8c44f` | =?UTF-8?q?fix(reset-dialog):=20the=20stuck=20confirm=20?= | N |
| 230 | `0230-610ed72a.patch` | `610ed72a` | =?UTF-8?q?fix(reread-dialog):=20same=20stuck-confirm=20cl?= | N |
| 231 | `0231-2bda99a5.patch` | `2bda99a5` | =?UTF-8?q?fix(edit-sheet):=20save/delete=20never-latch=20?= | N |
| 232 | `0232-4deebe3e.patch` | `4deebe3e` | =?UTF-8?q?fix(realign):=20never-latch=20=E2=80=94=20runRe?= | N |
| 233 | `0233-029ccb85.patch` | `029ccb85` | =?UTF-8?q?fix(plan-door):=20bulk-add=20rows=20+=20Planner?= | N |
| 234 | `0234-239cd7ab.patch` | `239cd7ab` | =?UTF-8?q?fix(F4):=205m=5Fclose=20evaluates=20as=20author?= | N |
| 235 | `0235-6119b7d5.patch` | `6119b7d5` | =?UTF-8?q?fix(T5):=20C2=20clock-drift=20evaluates=20the?= | N |
| 236 | `0236-f6a59c8d.patch` | `f6a59c8d` | =?UTF-8?q?fix(F13+A11):=20delete=20the=20dead=20activePla?= | N |
| 237 | `0237-dc68f4ee.patch` | `dc68f4ee` | =?UTF-8?q?fix(F12):=20cited=5Fscenario=20joins=20the=20fu?= | N |
| 238 | `0238-e5b68e25.patch` | `e5b68e25` | =?UTF-8?q?fix(F2+F8):=20scenario-dot=20honesty=20+=20anch?= | N |
| 239 | `0239-b2a42138.patch` | `b2a42138` | =?UTF-8?q?fix(F5+F11):=20write-time=20truth=20=E2=80=94?= | N |
| 240 | `0240-27b357ff.patch` | `27b357ff` | =?UTF-8?q?fix(F14):=20overlay=20failures=20are=20never=20?= | N |
| 241 | `0241-6cffb5e8.patch` | `6cffb5e8` | fix(F14): missing logger import for the fallback WARN | N |
| 242 | `0242-5d5cdd00.patch` | `5d5cdd00` | =?UTF-8?q?fix(F14):=20hoist=20the=20overlay-error=20colle?= | N |
| 243 | `0243-4c9bd78c.patch` | `4c9bd78c` | =?UTF-8?q?fix(A10):=20comment=20truth=20+=20gate=20dedup?= | N |
| 244 | `0244-53da0e86.patch` | `53da0e86` | =?UTF-8?q?fix(T3):=20ONE=20timeframe=20table=20(kernel/ti?= | N |
| 245 | `0245-cf51baf7.patch` | `cf51baf7` | =?UTF-8?q?fix(T6):=20one=20feed-aliveness=20policy=20?= | N |
| 246 | `0246-001762ca.patch` | `001762ca` | =?UTF-8?q?fix(F6):=20strict=20adherence=20upgraded=20to?= | N |
| 247 | `0247-ad5a5eb8.patch` | `ad5a5eb8` | =?UTF-8?q?feat(C1/F3):=20structured=20scenario=20confirma?= | N |
| 248 | `0248-a8096585.patch` | `a8096585` | =?UTF-8?q?docs(D1/F7):=20scenario=20state=20stays=20ADVIS?= | N |
| 249 | `0249-130870bd.patch` | `130870bd` | =?UTF-8?q?docs(D2/F9):=20target=5Fchain=20stated=20as=20G?= | N |
| 250 | `0250-b335d844.patch` | `b335d844` | docs(D2): regenerate the remaining futures golden (the D2 | N |
| 251 | `0251-2d8dae1f.patch` | `2d8dae1f` | =?UTF-8?q?docs(D3/F10):=20scenario=20quality=20labeled=20?= | N |
| 252 | `0252-7bdda06c.patch` | `7bdda06c` | =?UTF-8?q?fix(E5):=20v1=20strays=20=E2=80=94=20the=20chat?= | N |
| 253 | `0253-1a9f81c4.patch` | `1a9f81c4` | =?UTF-8?q?fix(E5/v1#8):=20the=20running=20revision=20is?= | N |
| 254 | `0254-227ffa6e.patch` | `227ffa6e` | =?UTF-8?q?chore(E2):=20RISK=5FMAX=5FCONTRACTS=5FPER=5FORD?= | N |
| 255 | `0255-ad2d0115.patch` | `ad2d0115` | chore(E2): drop the removed field from the RiskLimits test | N |
| 256 | `0256-b48cd2a6.patch` | `b48cd2a6` | =?UTF-8?q?chore(E3):=20the=20write-only=20trader-row=20pr?= | N |
| 257 | `0257-3e01b503.patch` | `3e01b503` | =?UTF-8?q?chore(E3):=20the=20API=20handler=20stops=20feed?= | N |
| 258 | `0258-f82a2418.patch` | `f82a2418` | =?UTF-8?q?fix(E4):=20the=20pre-existing=20FE=20test=20pai?= | N |
| 259 | `0259-4e32cd5a.patch` | `4e32cd5a` | =?UTF-8?q?fix(E1):=20Studio=20save=20honesty=20=E2=80=94?= | N |
| 260 | `0260-b9196b93.patch` | `b9196b93` | =?UTF-8?q?fix(pnl):=20close-sync=20attributes=20ONLY=20th?= | N |
| 261 | `0261-ca6e990f.patch` | `ca6e990f` | =?UTF-8?q?fix(pnl):=20additive=20correction=20layer=20?= | N |
| 262 | `0262-f024955a.patch` | `f024955a` | =?UTF-8?q?feat(pnl):=20per-close=20integrity=20guard=20?= | N |
| 263 | `0263-5bc5c945.patch` | `5bc5c945` | =?UTF-8?q?fix(pnl):=20missing=20telemetry=20import=20?= | N |
| 264 | `0264-17cd52e2.patch` | `17cd52e2` | =?UTF-8?q?fix(pnl):=20every=20remaining=20aggregate=20rea?= | N |

| 264 | `0264-supplement-faqdata.patch` | — | supplement: add `web/src/data/faqData.ts` — your fork's FAQ components already import it but the data file was never committed to your history (nofx has had it since 2026-03-15); without it your FE build fails TS2307 | N |

## Rewritten / path-limited patches (10 + 1 supplement)

The following patches had non-code content stripped (Rule H) or were edited;
code content is identical to the original commit:

- `0062-5578966a`, `0065-298d75b0`, `0066-205b1753`, `0168-bb966a04` — `docs/superpowers` reports/artifacts dropped.
- `0109-403a9205`, `0113-9364979a` — screenshot assets dropped.
- `0077-a8af893b`, `0080-bc360a38` — `deploy/RELEASE` hunks dropped (boot integrity + session toggle code retained).
- `0078-66ccb500` — `docs/VL-DAYPLAN-FULL-SPEC.md` hunk dropped (your fork never received that file).
- `0075-a6f39d7b` — `.gitignore` hunk dropped (your fork's ignore file is yours).
- clock-guard patch — deploy script paths genericized (`$HOME/nofx`, `%h/nofx`) from the source machine's absolute paths.
- `0001-8e8591a8`, `0066-205b1753`, `0195-85ec608f` — test fixtures neutralized (trader names replaced with `zeta`).
- **`0264-supplement-faqdata.patch`** — synthetic patch adding the missing FAQ data file (content identical to nofx HEAD, pre-scanned clean).

## Classification of every commit since the sync point (354 total)

| Bucket | Count | Justification |
|---|---|---|
| PROPAGATE (in the series) | 264* | all non-doc code: day-plan campaign, final-bundle phases, wire fixes, PnL integrity, UI repairs — everything in the upgrade guide |
| OWNER-ONLY (not sent) | 89 | `docs/superpowers/` worklogs/reports/plans, root `.md` worklogs, `docs/` specs, screenshots, `deploy/RELEASE` markers — zero code |
| PARTNER-DIVERGENT | 0 | no nofx commit touches `web/src/data/faqData.ts` or any other path your fork modified independently (verified against `2bf3342d`) |
| EXCLUDED (Rule H) | 1 | `7e152559` `cmd/decisive-test` forensic tool — hardcodes a model UUID read from the owner's live DB |

\*264 = 263 source commits + 1 supplement patch.

## Appendix — full commit table since the sync point (`3624a2a4..17cd52e2` + 2 docs)

hash | date | area | one-line | bucket

| `ca1f38c6` | 2026-08-14 | docs | docs: day-plan pre-build recon — 12 integration anchors verified at HEAD | OWNER-ONLY |
| `dde07f4e` | 2026-08-14 | docs | docs: no-entry investigation — C2 clock-drift is the sole blocker | OWNER-ONLY |
| `9fd32410` | 2026-08-14 | docs | docs: clock-cure recheck — CURED, awaiting first setup | OWNER-ONLY |
| `8e8591a8` | 2026-08-14 | web | fix(web): trader selection persists — no more snap-back to last-created | PROPAGATE |
| `e3276a7d` | 2026-08-14 | web | feat(web): active-trader marker in the traders list | PROPAGATE |
| `c051b975` | 2026-08-14 | docs | docs: trader-select-fix report — persistence + stable default + marker | OWNER-ONLY |
| `97512286` | 2026-08-14 | dayplan | docs(dayplan): commit build contract spec + campaign heartbeat | OWNER-ONLY |
| `d1851dac` | 2026-08-14 | dayplan | feat(dayplan): P0.1 — day_plan config on strategy rows (ROOT, both codecs) | PROPAGATE |
| `0a974d31` | 2026-08-14 | dayplan | feat(dayplan): P0.2 — plans/plan_overlays append-only store + decision FK + WAL | PROPAGATE |
| `6a0d233b` | 2026-08-14 | dayplan | feat(dayplan): P0.3 — CT-anchored session registry (NY-only default) | PROPAGATE |
| `b51ab5c2` | 2026-08-14 | dayplan | feat(dayplan): P0.4 — scenario-fact evaluator (the keystone) | PROPAGATE |
| `041e4450` | 2026-08-14 | dayplan | feat(dayplan): P0.5 — level-state table (identity-keyed, cross-session) | PROPAGATE |
| `89d7e1b3` | 2026-08-14 | dayplan | test(dayplan): config-truth — day_plan survives MergeStrategyConfig | PROPAGATE |
| `d8e2f88c` | 2026-08-14 | dayplan | docs(dayplan): P0 FOUNDATIONS complete — checkpoint report + heartbeat | OWNER-ONLY |
| `14adc47e` | 2026-08-14 | dayplan | feat(dayplan): P1.1 — session-tagged bars + multi-day level extractor | PROPAGATE |
| `e1cd5993` | 2026-08-14 | dayplan | feat(dayplan): P1.2 — round numbers + gap tracker + OR/IB detectors | PROPAGATE |
| `3058e131` | 2026-08-14 | dayplan | feat(dayplan): P1.5 — confluence scorer -> graded TOP-8 + KEY LEVELS renderer | PROPAGATE |
| `15e55520` | 2026-08-14 | dayplan | feat(dayplan): P1.6 — regime block (7 auto fields, graceful degradation) | PROPAGATE |
| `9436ea79` | 2026-08-14 | dayplan | test(dayplan): P1 sample — full detector->scorer->render pipeline block | PROPAGATE |
| `2edee42a` | 2026-08-14 | dayplan | docs(dayplan): P1 checkpoint — MAP backbone (4/8), report + heartbeat | OWNER-ONLY |
| `f138b7f3` | 2026-08-14 | dayplan | feat(dayplan): P1.4 — EQH/EQL + S/D zones + FVG + OB detectors | PROPAGATE |
| `c42a629c` | 2026-08-14 | dayplan | feat(dayplan): P1.7 — wire KEY LEVELS into the live futures prompt | PROPAGATE |
| `a450adfc` | 2026-08-14 | dayplan | feat(dayplan): P1.3 — durable session-profile store + snapshot writer + nPOC | PROPAGATE |
| `09579af3` | 2026-08-14 | dayplan | feat(dayplan): P1.8 — calendar fetcher (ForexFactory + static fallback) | PROPAGATE |
| `3bcd1132` | 2026-08-14 | dayplan | docs(dayplan): P1 COMPLETE (8/8) — final report + heartbeat | OWNER-ONLY |
| `069500c9` | 2026-08-14 | dayplan | feat(dayplan): P2.1 — true bar-close cadence (gated, dormant) | PROPAGATE |
| `3bb7a730` | 2026-08-14 | dayplan | feat(dayplan): P2.2 — skip-while-open gate (build; RECON #5 did not exist) | PROPAGATE |
| `21c57118` | 2026-08-14 | dayplan | feat(dayplan): P2.3 — day-trader clock fields (last_entry + eod_flat) | PROPAGATE |
| `a43d006d` | 2026-08-14 | dayplan | feat(dayplan): P2.4 — MAE/MFE + entry-confidence logging (additive) | PROPAGATE |
| `b0151d98` | 2026-08-14 | dayplan | feat(dayplan): P2.5 — day_plan ARM tool (guarded, idempotent, dry-run) | PROPAGATE |
| `a62ef130` | 2026-08-14 | dayplan | docs(dayplan): P2 COMPLETE (5/5) — report + ★ RESTART 1 handoff + heartbeat | OWNER-ONLY |
| `389c0c9e` | 2026-08-14 | dayplan | feat(dayplan): P3.1 — session gates (enabled-window-only entries + no-trade) | PROPAGATE |
| `05703a08` | 2026-08-14 | dayplan | feat(dayplan): P3.2 — planner model binding (RECON #12) | PROPAGATE |
| `cad1dc2f` | 2026-08-14 | dayplan | docs(dayplan): P3 checkpoint — session gates + model binding (2/6) | OWNER-ONLY |
| `5d06b063` | 2026-08-14 | dayplan | feat(dayplan): P3.3 — read jobs + the planner call (schema/assembler/call) | PROPAGATE |
| `c65d9b2b` | 2026-08-14 | dayplan | docs(dayplan): P3 planner-core checkpoint (P3.3 done, 3/6) | OWNER-ONLY |
| `d0d96a10` | 2026-08-15 | dayplan | feat(dayplan): P3.4 — executor plan injection + RECON #4 reorder | PROPAGATE |
| `d2d4a975` | 2026-08-15 | dayplan | feat(dayplan): P3.5 — advisory mode (cite scenario_id + match-rate) | PROPAGATE |
| `0a292148` | 2026-08-15 | dayplan | feat(dayplan): P3.6 core — activation window + death re-plan + restart recovery | PROPAGATE |
| `0dfcd532` | 2026-08-15 | dayplan | docs(dayplan): P3 executor injection + advisory + lifecycle-core | OWNER-ONLY |
| `f9f5460f` | 2026-08-15 | dayplan | feat(dayplan): P3.6-A — digest writers + tapered week chain | PROPAGATE |
| `29727291` | 2026-08-15 | dayplan | feat(dayplan): P3.6-B — scenario re-arm via level-state | PROPAGATE |
| `9f000637` | 2026-08-15 | dayplan | feat(dayplan): P3.6-C — sticky owner levels | PROPAGATE |
| `e19403fc` | 2026-08-15 | dayplan | feat(dayplan): P3.6-D — night mode (explicit state + clean restart) | PROPAGATE |
| `3f37ab54` | 2026-08-15 | dayplan | docs(dayplan): P3 COMPLETE — P3.6 finish (digests/re-arm/sticky/night) | OWNER-ONLY |
| `c112df75` | 2026-08-15 | dayplan | docs(dayplan): commit P4 Plan Card design system (tokens + components are LAW) | OWNER-ONLY |
| `7674261b` | 2026-08-15 | dayplan | feat(dayplan): P4.1 — /api/plan/* API (today/history/alerts + ack) | PROPAGATE |
| `40ec67a3` | 2026-08-15 | dayplan | fix(dayplan): P4.1 — scope alert-ack by trader_id (IDOR guard) | PROPAGATE |
| `f1ccb76e` | 2026-08-15 | dayplan | feat(dayplan): P4 FE foundation — tokens, plan API client, i18n, hooks | PROPAGATE |
| `842edb57` | 2026-08-15 | dayplan | feat(dayplan): P4.2 — SessionTimelineStrip + SessionTabs + HandoverBanner | PROPAGATE |
| `fd27f63a` | 2026-08-15 | dayplan | feat(dayplan): P4.3 — SessionPlanCard (bias/chart/levels/scenarios/rules) | PROPAGATE |
| `d4564cf5` | 2026-08-15 | dayplan | feat(dayplan): P4.4 — in-app alert center (bell + feed + P0 banner) | PROPAGATE |
| `859353b3` | 2026-08-15 | dayplan | feat(dayplan): P4.5 — Studio Day Plan block (config + sessions accordion) | PROPAGATE |
| `72e11543` | 2026-08-15 | dayplan | docs(dayplan): P4 · THE CARD completion report | OWNER-ONLY |
| `899f14f5` | 2026-08-15 | dayplan | feat(dayplan): P5.1 — overlay API (RFC-6902 apply + test-op + B2 armor) | PROPAGATE |
| `94fa6daf` | 2026-08-15 | dayplan | feat(dayplan): P5.4 backend — Ask-Planner (anti-sycophancy, verdict log) | PROPAGATE |
| `ad9dd424` | 2026-08-15 | dayplan | feat(dayplan): P5.5 — adherence grade A–F per closed trade | PROPAGATE |
| `fa9344b2` | 2026-08-15 | dayplan | feat(dayplan): P5.6 — stats honesty gate (matched-random, WARMING) | PROPAGATE |
| `3126b61e` | 2026-08-15 | dayplan | feat(dayplan): P5.2+P5.3+P5.4 FE — the owner door (edit / conflict / ask-planner) | PROPAGATE |
| `e99a145f` | 2026-08-15 | dayplan | feat(dayplan): P5.2 bulk add + P5.7 blind-mark grammar parser | PROPAGATE |
| `41afa1b6` | 2026-08-15 | dayplan | fix(dayplan): P5 hardening — adversarial-review findings (12 confirmed) | PROPAGATE |
| `aef31090` | 2026-08-15 | dayplan | docs(dayplan): P5 · THE DOOR completion report + ★ RESTART 2 handoff | OWNER-ONLY |
| `79a58523` | 2026-08-15 | dayplan | fix(dayplan): Studio Day Plan settings not editable — normalize dropped day_plan | PROPAGATE |
| `18e9e5ac` | 2026-08-15 | dayplan | feat(dayplan): planner assembler honors day_plan config (config-truth) | PROPAGATE |
| `10ce3886` | 2026-08-15 | dayplan | docs(dayplan): Day Plan settings-not-editable fix report | OWNER-ONLY |
| `7344820d` | 2026-08-15 | dayplan | docs(dayplan): full function audit vs contract (read-only, pre-Monday) | OWNER-ONLY |
| `57b518d8` | 2026-08-15 | dayplan | fix(dayplan): W1 — Sunday-read guard + daily-digest window (URGENT pre-Sun-17:00) | PROPAGATE |
| `0c8bae59` | 2026-08-15 | dayplan | fix(dayplan): W2 — pin the EXACT planner model id, reset stats on change (§125/§128) | PROPAGATE |
| `50d63061` | 2026-08-15 | dayplan | docs(dayplan): wire-up train report — W1+W2 done, deploy checkpoint | OWNER-ONLY |
| `89e2809f` | 2026-08-15 | dayplan | feat(dayplan): W3 — calendar producer + T1 red-news HARD blackout end-to-end | PROPAGATE |
| `5729e2b5` | 2026-08-15 | dayplan | feat(dayplan): W4 — owner overlays reach the executor (plan_final, not base) | PROPAGATE |
| `e165f35e` | 2026-08-15 | dayplan | feat(dayplan): W5 — learning loop fires on REAL exits (OCO / EOD / manual) | PROPAGATE |
| `da3fb458` | 2026-08-15 | dayplan | feat(dayplan): W6 — wire alert Emit at 6 production call-sites | PROPAGATE |
| `77ffbe8f` | 2026-08-15 | dayplan | feat(dayplan): W7 — wire the level-state writer (cross-session burn memory) | PROPAGATE |
| `fb5bd624` | 2026-08-15 | dayplan | feat(dayplan): W8 — gates read the admin session registry (was hardcoded default) | PROPAGATE |
| `1d476407` | 2026-08-15 | dayplan | feat(dayplan): W9 — wire the 6 display-only DayPlanConfig fields to real behavior | PROPAGATE |
| `0f79fb4f` | 2026-08-15 | dayplan | feat(dayplan): W10 — supply the realized-vol baseline (RV stops "warming") | PROPAGATE |
| `8c335333` | 2026-08-15 | docs | docs: wire-up train report — W3–W10 sections + FINAL deploy line | OWNER-ONLY |
| `cf66b016` | 2026-08-15 | docs | docs: red-news pipeline verify ×2 — chain broken at link 1 (restart predates W3); week matches owner | OWNER-ONLY |
| `cbf12870` | 2026-08-15 | dayplan | feat(dayplan): W11 — planner indicator mirror (owner override, spec §regime) | PROPAGATE |
| `2985e50f` | 2026-08-15 | dayplan | fix(dayplan): F0.1 — calendar ignition: producer hoisted above the CME session gate + plain 📅 logs | PROPAGATE |
| `3d91e574` | 2026-08-15 | dayplan | feat(dayplan): F0.2 — static T1 fallback STORES rows (source=static); repo-shipped template | PROPAGATE |
| `b30fb8cc` | 2026-08-15 | docs | docs: acceptance gate — STEP 0 FAIL (W12/F0 nonexistent, dirty tree w/ uncommitted W11b, deployed bi | OWNER-ONLY |
| `5578966a` | 2026-08-15 | dayplan | feat(dayplan): F0.3 — Sunday feed-roll freshness: static/none slices are stale, live fetch upgrades  | PROPAGATE |
| `c886b359` | 2026-08-15 | dayplan | test(dayplan): F0.4 — calendar ignition matrix: fetch-ok / 404-fallback-static / skip-fresh / static | PROPAGATE |
| `e2bf117c` | 2026-08-15 | docs | docs: F0 calendar ignition report — gate-order root cause, 4-commit fix train, deploy handoff | OWNER-ONLY |
| `1cf39bff` | 2026-08-15 | docs | docs: F0 report — correct F0.3 hash after concurrent-session rebase (174a8884 → 5578966a) | OWNER-ONLY |
| `dd095044` | 2026-08-15 | docs | docs: acceptance gate addendum — concurrent W-train session landing F0.x live (gate fired early); am | OWNER-ONLY |
| `a7bdae50` | 2026-08-15 | dayplan | feat(dayplan): W11b — surface persisted level freshness + overnight-gap regime input | PROPAGATE |
| `877998a3` | 2026-08-15 | docs | docs: W11 indicator-mirror report — golden regen finding (byte-identical disabled) + real sample blo | OWNER-ONLY |
| `298d75b0` | 2026-08-15 | dayplan | test(dayplan): W12 — math correctness audit (14 formulas, 3 oracles each) | PROPAGATE |
| `b9c05e36` | 2026-08-15 | docs | docs: design conformance audit — built system vs blueprints (READ-ONLY) | OWNER-ONLY |
| `205b1753` | 2026-08-15 | docs | docs: acceptance gate v2 — dress rehearsal PASS on deployed binary; 8 findings (1 CRITICAL auth, 2 H | PROPAGATE |
| `81e54fc9` | 2026-08-15 | security | fix(security): S1 — bind the API to 127.0.0.1 by default (was 0.0.0.0) | PROPAGATE |
| `821c895a` | 2026-08-15 | security | fix(security): S2 — reset-password disabled; reset-account behind JWT + env flag + confirm token | PROPAGATE |
| `0d97a672` | 2026-08-15 | security | docs(security): document API_SERVER_HOST and ALLOW_ACCOUNT_RESET in .env.example | PROPAGATE |
| `92969990` | 2026-08-15 | security | fix(security): S5 — stop writing plaintext API keys and wallet private keys to the logs | PROPAGATE |
| `8dcc4521` | 2026-08-15 | security | fix(security): S4 — close the unauthenticated RSA decryption oracle (/api/crypto/decrypt) | PROPAGATE |
| `fb3d5f65` | 2026-08-15 | security | docs(security): P0 fix report — takeover chain closed, 2 extra holes found, 7 open items | OWNER-ONLY |
| `b75b0cb9` | 2026-08-16 | docs | docs: demo plan seed — clickable day plan on the live dashboard, isolated from Monday | OWNER-ONLY |
| `38ca583c` | 2026-08-16 | docs | docs: demo seed addendum — isolated preview stack so the card is viewable now | OWNER-ONLY |
| `05f8ce60` | 2026-08-16 | plan | fix(plan): blank Plan Card on a second backend — NT port singleton + registry | PROPAGATE |
| `db5949fa` | 2026-08-16 | dayplan | feat(dayplan): W13.1 — plan re-alignment on owner edit (backend, proposal-only) | PROPAGATE |
| `14b26aae` | 2026-08-16 | dayplan | feat(dayplan): W13.2 — re-align UI (inline status · proposal card · Apply-gated) | PROPAGATE |
| `c3f4092a` | 2026-08-16 | dayplan | docs(dayplan): W13 — contract + report for re-alignment on owner edit | OWNER-ONLY |
| `a6f39d7b` | 2026-08-16 | sandbox | feat(sandbox): isolated hands-on sandbox on :3001 (own DB, no orders, canned AI) | PROPAGATE |
| `02230136` | 2026-08-16 | docs | docs: sandbox preview report — isolation receipts + owner step list | OWNER-ONLY |
| `517df64c` | 2026-08-16 | sandbox | fix(sandbox): empty plan card — synthetic bars + owner provenance + scenario status | PROPAGATE |
| `3155b766` | 2026-08-16 | docs | docs: sandbox add-on — diagnosis (empty bars, not W13) + full seed + owner steps | OWNER-ONLY |
| `a8af893b` | 2026-08-16 | safety | feat(safety): P1 boot integrity assertion — refuse to trade on a stale binary | PROPAGATE |
| `66ccb500` | 2026-08-16 | session | fix(session): P3 session-end drift — NY ends where it flats (14:45 CT = 15:45 ET) | PROPAGATE |
| `542dfdc8` | 2026-08-16 | regime | feat(regime): P2 dark-regime alert + DEGRADED plan flag | PROPAGATE |
| `1a87a9e4` | 2026-08-16 | docs | docs: pre-open fixes report — P1-P4, adversarial findings, screenshots, deploy handoff | OWNER-ONLY |
| `bc360a38` | 2026-08-16 | dayplan | feat(dayplan): W15.A — a REAL session enable toggle on every session row | PROPAGATE |
| `d396201e` | 2026-08-16 | dayplan | fix(dayplan): W15.B — every dead day-plan control made live | PROPAGATE |
| `e943f9c3` | 2026-08-16 | dayplan | fix(dayplan): W15.C — the plan card's session tabs were a lie | PROPAGATE |
| `5472f316` | 2026-08-16 | docs | docs: W15 control inspection — 11 dead controls found, 11 fixed | OWNER-ONLY |
| `6fe92f16` | 2026-08-16 | deploy | docs(deploy): RESTORE.md — binary rollback + the MANDATORY RELEASE re-arm | OWNER-ONLY |
| `a84d6ae2` | 2026-08-16 | security | fix(security): bind the live dev UI to loopback — it was proxying the LAN into :8080 | PROPAGATE |
| `9dffec72` | 2026-08-16 | dayplan | test(dayplan): pin three W15.A follow-on defects with executable receipts | PROPAGATE |
| `e6afcf77` | 2026-08-16 | docs | docs: CTO final verification — hardcode hunt, 8 pipeline traces, defaults audit | OWNER-ONLY |
| `e33532e2` | 2026-08-16 | dayplan | fix(dayplan): scope the owner door to the LIVE session — regression from W15.C | PROPAGATE |
| `2427c850` | 2026-08-16 | dayplan | fix(dayplan): Ask-Planner must read plan_final, not the base plan | PROPAGATE |
| `637b137a` | 2026-08-16 | dayplan | fix(dayplan): a NO-TRADE plan must not render as gold "ACTIVE" | PROPAGATE |
| `c045b97b` | 2026-08-16 | docs | docs: CTO verification — fold in the control matrix + the 3 fixes it produced | OWNER-ONLY |
| `9acba3d9` | 2026-08-16 | docs | docs: CTO verification — fold in 5.3 exit integrity + 5.10 observability; flag 5.1 as not completed | OWNER-ONLY |
| `f7fa2d3c` | 2026-08-16 | trader | fix(trader): a gate-REFUSED entry must not be recorded as a successful execution | PROPAGATE |
| `04475d66` | 2026-08-16 | docs | docs: CTO verification — complete. All 12 checklist items verdicted, 41 remaining | OWNER-ONLY |
| `0b8443f9` | 2026-08-16 | dayplan | fix(dayplan): R2 — declining a planner proposal is now recorded, and says so | PROPAGATE |
| `735ce88e` | 2026-08-16 | style | style: gofmt store/plan_qa.go (R2 follow-up) | PROPAGATE |
| `5db9d3c9` | 2026-08-16 | dayplan | fix(dayplan): R6 — one planner call per (trade_date, session) at a time | PROPAGATE |
| `4381c801` | 2026-08-16 | trader | fix(trader): R5 — a post-fill DB write failure now freezes and alerts, not INFO-logs | PROPAGATE |
| `9e7b3259` | 2026-08-16 | dayplan | feat(dayplan): R3 — "why was my entry refused?" is now answerable from the UI | PROPAGATE |
| `0504516f` | 2026-08-16 | dayplan | fix(dayplan): R4 — scope the session digest to the trader whose P&L it reports | PROPAGATE |
| `97e4f541` | 2026-08-16 | kernel | fix(kernel): R7 — remove the unreachable T22 stale/drift block, pin what does run | PROPAGATE |
| `c4996495` | 2026-08-16 | dayplan | feat(dayplan): R1 — real per-scenario status, so plays stop all reading "armed" | PROPAGATE |
| `909d3a48` | 2026-08-16 | docs | docs: W16 truthful-reporting report — all 7 shipped, per-item commits | OWNER-ONLY |
| `02ebb9e3` | 2026-08-16 | docs | docs: VL-VERIFICATION-CHECKLIST v1 — the standing verification spec | OWNER-ONLY |
| `e2dd882a` | 2026-08-16 | docs | docs: checklist run 1 — 39 verified, 13 not done, 0 blocking | OWNER-ONLY |
| `7e5932fc` | 2026-08-16 | risk | fix(risk): an UNSET risk value must never be the loosest setting | PROPAGATE |
| `b4b28895` | 2026-08-16 | risk | test(risk): the regression test that ends the screen-vs-engine class | PROPAGATE |
| `a8c308ee` | 2026-08-16 | docs | docs: correct the false 'empty risk_control' headline in both prior reports | OWNER-ONLY |
| `2c15f000` | 2026-08-16 | cmd | feat(cmd): dayplan-sessions — apply per-session overrides through the store layer | PROPAGATE |
| `068bae2e` | 2026-08-16 | docs | docs: P0 investigation — no P0; the audit query was wrong, not the config | OWNER-ONLY |
| `7aa521a1` | 2026-08-16 | nt8 | fix(nt8): end the bar-watchdog livelock that ran 100k+ rebuilds over the weekend | PROPAGATE |
| `06f3343e` | 2026-08-16 | dayplan | fix(dayplan): a plan may only die on evidence that POSTDATES it | PROPAGATE |
| `f6521df9` | 2026-08-16 | plan-card | fix(plan-card): an empty plan must say WHY, and the chart must still render | PROPAGATE |
| `20a64c51` | 2026-08-16 | dayplan | fix(dayplan): "2x5m" must mean two FIVE-minute closes, not two bars | PROPAGATE |
| `8b24c85e` | 2026-08-16 | dayplan | fix(dayplan): the executor prompt quotes the real re-plan cap, not a literal 2 | PROPAGATE |
| `745f84f7` | 2026-08-16 | plan | feat(plan): the read path for plan version history (ITEM 15, backend) | PROPAGATE |
| `47dbb269` | 2026-08-16 | bars | fix(bars): refuse NT8's empty-minute placeholders — no synthetic bars, ever | PROPAGATE |
| `f630ceea` | 2026-08-16 | plan | feat(plan): version chips open the version they name (ITEM 15, frontend) | PROPAGATE |
| `499dd59f` | 2026-08-16 | dayplan | fix(dayplan): a re-plan that strands owner edits now says so (P1 alert) | PROPAGATE |
| `5fc5ae9a` | 2026-08-16 | checklist | docs(checklist): plan history browser is built — D16 has an implementation | OWNER-ONLY |
| `359ace1c` | 2026-08-16 | docs | docs: P0 report — levels died in the death check; the chart gaps are a separate fault | OWNER-ONLY |
| `403a9205` | 2026-08-16 | plan | fix(plan): the FOOTER version chips were still dead — both rows now open a version | PROPAGATE |
| `678b880f` | 2026-08-16 | dayplan | fix(dayplan): post-mortem on the 6 ASIA deaths — replay, regression test, guard | PROPAGATE |
| `8e7b816a` | 2026-08-16 | docs | docs: chips + 6-deaths post-mortem | OWNER-ONLY |
| `301df4be` | 2026-08-16 | dayplan | fix(dayplan): the cap DID work — v6 is a NO-TRADE marker, now shown as one | PROPAGATE |
| `243787d8` | 2026-08-16 | plan | feat(plan): Ask-Planner opens with no active plan (labelled, read-only) | PROPAGATE |
| `9364979a` | 2026-08-16 | plan | feat(plan): ⟳ Re-read — the owner's manual escape hatch (ITEM 3) | PROPAGATE |
| `ca819d91` | 2026-08-16 | decisions | fix(decisions): every decision carries its "why" — wait included (ITEM 6) | PROPAGATE |
| `3594c2af` | 2026-08-16 | alerts | feat(alerts): the feed can be cleared — the record cannot (ITEM 5) | PROPAGATE |
| `3ea0a442` | 2026-08-16 | plan | feat(plan): owner levels survive a re-plan, re-anchored by price (ITEM 4) | PROPAGATE |
| `184fe200` | 2026-08-16 | docs | docs: cap semantics · ask-with-no-plan · re-read · overlays · alerts · reasoning | OWNER-ONLY |
| `7ad50607` | 2026-08-17 | docs | docs: 9-shadow-config investigation (read-only) — 7 SHADOWS-CONFIG, H8 live-harmful, replan_cap RESO | OWNER-ONLY |
| `2ddf3a58` | 2026-08-17 | decision | fix(decision): recover prose-embedded JSON and retry/alert on prose-only misses (P0) | PROPAGATE |
| `570c6c32` | 2026-08-17 | h8 | fix(h8): sessionRunnable is the single source of truth for session enablement | PROPAGATE |
| `2b4162f6` | 2026-08-17 | latency | fix(latency): decision call capped inside the bar window + stale-bar discard (P0) | PROPAGATE |
| `6934a18e` | 2026-08-17 | h9 | fix(h9): the planner fetches the CONFIGURED timeframes and claims only what it read | PROPAGATE |
| `f13ecb86` | 2026-08-17 | h4+h5 | fix(h4+h5): max_levels/scenario_cap reach validation AND the prompt (12/5 hard ceilings) | PROPAGATE |
| `8338aaea` | 2026-08-17 | h1+h2 | fix(h1+h2): proximity_filter_atr governs level generation AND seating, not just activation | PROPAGATE |
| `a47a51a6` | 2026-08-17 | h7 | fix(h7): executor prompt reads the PERSISTED admin registry, not the hardcoded default | PROPAGATE |
| `9199e9f0` | 2026-08-17 | p7 | fix(p7): NO-TRADE plans still carry the map — levels are facts, the plan is opinion | PROPAGATE |
| `020e407a` | 2026-08-17 | p6 | feat(p6): the owner reset — abandon the chain, re-arm the budget, fresh plan | PROPAGATE |
| `6cc9ce11` | 2026-08-17 | docs | docs: combined fix train report (2026-08-17) | OWNER-ONLY |
| `480a4d51` | 2026-08-17 | h10 | fix(h10): acceptance counted on the rule's timeframe everywhere | PROPAGATE |
| `77623924` | 2026-08-17 | h10 | test(h10): rule-timeframe fixtures + no-hardcoded-interval guard | PROPAGATE |
| `25e07822` | 2026-08-17 | docs | docs: 2026-08-18 full re-audit (read-only) — 24 findings, 3 blocking | OWNER-ONLY |
| `6dcee05a` | 2026-08-17 | docs | docs: live-watch report 2026-08-17 — plan reached the executor (NY) but via a first-trader-captured  | OWNER-ONLY |
| `99fd67e1` | 2026-08-17 | p0a | fix(p0a): cross-trader plan governance — trader-scoped lookups + per-trader providers | PROPAGATE |
| `1e8cc591` | 2026-08-17 | p0b | fix(p0b): the ASIA clock — 16:55 closed-market read fires; one instance, one chain | PROPAGATE |
| `91748082` | 2026-08-17 | p1 | fix(p1): H8 residuals — the API layer resolves sessionRunnable, never the raw flag | PROPAGATE |
| `32380158` | 2026-08-17 | docs | docs: cross-trader / asia-clock / h8-residuals report (2026-08-18) | OWNER-ONLY |
| `1a95a444` | 2026-08-17 | p | p1c: consumed levels role-flip and stay; consumption windowed on row birth | PROPAGATE |
| `5fe152d7` | 2026-08-17 | p | p1d: explicit prior-session aging for consumed levels | PROPAGATE |
| `a85d7363` | 2026-08-17 | p | p1b: dayplan-level-repair purge tool | PROPAGATE |
| `3b7a1d7c` | 2026-08-17 | p | p1e: tests for wick/touch/window consumption, aging, role-flip | PROPAGATE |
| `8559ea0c` | 2026-08-17 | p | p0a2: trader-scoped plan identity (plans PK no longer cross-trader) | PROPAGATE |
| `65834726` | 2026-08-17 | b | b-fix: wire alert-feed prune; delete dead HandoverBanner | PROPAGATE |
| `9c10c149` | 2026-08-17 | c | c-fix: write ai_latency_ms + minimal discipline consumers | PROPAGATE |
| `6a0cf41d` | 2026-08-17 | fix | fix duplicate package clause in dayplan-level-repair | PROPAGATE |
| `1c418a1f` | 2026-08-17 | d | d-sweep-fix: trader-scoped read claim, config max_levels, honest audit columns | PROPAGATE |
| `c4abff35` | 2026-08-17 | report | report: false burned levels + remaining audit findings (2026-08-18); arm RELEASE 1c418a1f | OWNER-ONLY |
| `aeaf7076` | 2026-08-17 | why | why-no-trades: plan advisory must read as advisory, not direction | PROPAGATE |
| `7e152559` | 2026-08-17 | tool | tool: decisive-test — replay stored cycles through the model with the plan block stripped | EXCLUDED (forensic tool w/ owner DB values) |
| `8e930a93` | 2026-08-17 | report | report: why no trades — decisive test + full-session autopsy (2026-08-18) | OWNER-ONLY |
| `28b79f09` | 2026-08-17 | ui | ui-verify: reset was works-but-silent — claim-retry, honest note, reading banner | PROPAGATE |
| `179fb229` | 2026-08-17 | ui | ui-verify: explain the default-strategy lock; align level-state alert wording | PROPAGATE |
| `0c9cd04c` | 2026-08-17 | report | report: full UI verification — reset works-but-silent fixed, 21 LIVE / 2 fixed / 0 broken | OWNER-ONLY |
| `10da1b8c` | 2026-08-17 | report | report: full verification run (read-only) — 34 pass, 13 findings, 2 high, binary 2 behind HEAD | OWNER-ONLY |
| `97ead6e4` | 2026-08-17 | plan | fix(plan): edit sheet prefills exact instruction + owner note/tag | PROPAGATE |
| `8d5cfa1f` | 2026-08-17 | plan | fix(plan): write owner note/scenario-tag edits through to owner_levels | PROPAGATE |
| `67469751` | 2026-08-17 | docs | docs: reverse-trace audit 2026-08-17 — 22 values traced, 0 invented, 0 causeless actions | OWNER-ONLY |
| `cd5fd153` | 2026-08-17 | report | report: end-to-end audit 2026-08-18 — one real decision traced bar-to-order, 17 hops verified, 1 mis | OWNER-ONLY |
| `59a6b8eb` | 2026-08-17 | docs | docs: total sweep — 8 error types censused, both guardrail_skip causes autopsied, ASIA enabled verdi | OWNER-ONLY |
| `8440dfb8` | 2026-08-17 | p5.4 | fix(p5.4): ONE PROMPT, ONE SNAPSHOT — SVP, KEY LEVELS and PLAN STATUS share one bars fetch | PROPAGATE |
| `a3a2c929` | 2026-08-17 | p5.5 | fix(p5.5): preserve the last non-empty model output across schema retries | PROPAGATE |
| `b37697c8` | 2026-08-17 | spec | docs(spec): min_confidence 60 owner-dated 2026-08-18 (was 65) — contract matches live config | OWNER-ONLY |
| `32892232` | 2026-08-17 | docs | docs: final build report — 4 of 8 landed, deployed b37697c8, boot integrity OK | OWNER-ONLY |
| `99dc04b8` | 2026-08-17 | docs | docs: consumed-without-touch audit — 0 of 7 burns justified, root cause in level facts + state write | OWNER-ONLY |
| `91333cfb` | 2026-08-17 | p1c | fix(p1c): consumption always requires an in-window touch after birth — one meaning everywhere | PROPAGATE |
| `256e8a90` | 2026-08-17 | docs | docs: consumption touch-gate fix landed — 12 false burns purged, display birth-scoped, deployed 9133 | OWNER-ONLY |
| `4efd1137` | 2026-08-17 | chart | fix(chart): equity-history default is a 90-day window, not the latest 500 snapshots | PROPAGATE |
| `66a9664f` | 2026-08-17 | chart | fix(chart): raise the FE equity-curve display cap 2000→10000 | PROPAGATE |
| `4314d4fd` | 2026-08-18 | web | fix(web): plan mini chart klines limit 500->1500 (show full NT8 cache) | PROPAGATE |
| `d6905dd3` | 2026-08-18 | kernel | fix(kernel): touch-gate StillValid — untouched levels no longer show consumed after 2 bars | PROPAGATE |
| `c8f91d41` | 2026-08-18 | tz | fix(tz): P0 — CT canonical everywhere; labelled clock in every prompt, single timezone source | PROPAGATE |
| `e7c54776` | 2026-08-18 | docs | docs: P0 timezone CT-canonical report — 9 suppressed decisions, sweep table, exit bar | OWNER-ONLY |
| `b4f8e05d` | 2026-08-18 | docs | docs: VL master audit checklist v1 (08-19) — spec for the full audit | OWNER-ONLY |
| `7377742a` | 2026-08-18 | docs | docs: VL master audit — 76 rows verified, 22 findings, 3 money/trade-risk, 6 unproven (read-only run | OWNER-ONLY |
| `e129f141` | 2026-08-18 | docs | docs: partner-vs-us A/B — pipeline exonerated, model proposes 0 in all 4 prompt shapes; 107 rule-com | OWNER-ONLY |
| `b60789df` | 2026-08-18 | docs | docs: aug-14 bisect — 79 calls across 7 prompt variants, 0 proposals; no suppressing text found in t | OWNER-ONLY |
| `f6478923` | 2026-08-18 | ai | fix(ai): wire-proven truncation was zeroing every decision — raise AI_MAX_TOKENS 2000→8000 + cap rea | PROPAGATE |
| `d8964193` | 2026-08-18 | docs | docs: total root-cause — wire-proven truncation zeroed decisions; fix deployed f6478923 | OWNER-ONLY |
| `1b5139c0` | 2026-08-18 | ? | P0: AI params fully config-driven + decision-first contract | PROPAGATE |
| `26c5d35a` | 2026-08-18 | mcp | mcp: log completion tokens + finish_reason on every AI call | PROPAGATE |
| `5b1a9927` | 2026-08-18 | agent | agent sub-call token caps env-driven + startup audit | PROPAGATE |
| `1a6dcf74` | 2026-08-18 | P0 | fix(P0): zero-trades root cause — WSL clock drift was blocking every entry | PROPAGATE |
| `7a85a745` | 2026-08-18 | docs | docs: total e2e investigation — clock-drift guard was converting every entry to wait; AI-params live | OWNER-ONLY |
| `9f9deebc` | 2026-08-18 | docs | docs: e2e report final — A/B 9/9 (b reproduces the 08:47 entry), replay complete | OWNER-ONLY |
| `d9c7c7f1` | 2026-08-18 | docs | docs: trim e2e report to spec (≤35 lines) with final A/B numbers | OWNER-ONLY |
| `c01802b5` | 2026-08-18 | docs | docs: e2e report at spec limit (35 lines) | OWNER-ONLY |
| `54f0cb6c` | 2026-08-18 | ? | P0 planner-quality: both-side seating + cluster collapse + HTF zones seat (items 1/4/5) | PROPAGATE |
| `5408abd1` | 2026-08-18 | ? | P0 planner-quality: facts-validated plans — both-side levels, gap continuation, reachable targets (i | PROPAGATE |
| `73976c96` | 2026-08-18 | ? | P0 planner-quality: death/flip text is now machine-evaluated (item 3) | PROPAGATE |
| `e8a4b710` | 2026-08-18 | ? | P0 calendar fail-closed: missing slice → static T1 blackout + P0 alert (item 6) | PROPAGATE |
| `c42b7280` | 2026-08-18 | ? | P0 planner-quality: continuation trigger must be reachable (item 2b) | PROPAGATE |
| `e02ed157` | 2026-08-18 | docs | docs: planner-quality fix report — receipts, fixes, before/after table | OWNER-ONLY |
| `49413c0d` | 2026-08-18 | ? | P0-cleanup item 2c+2a-core: refusals never bare waits + structured error events | PROPAGATE |
| `7bcb238d` | 2026-08-18 | ? | P0-cleanup item 2b/2d: one error surface + daily error digest line | PROPAGATE |
| `547a2e89` | 2026-08-18 | ? | P0-cleanup item 1: learning loop closed — plan attribution + MAE/MFE/adherence in digests | PROPAGATE |
| `4893ce37` | 2026-08-18 | ? | P0-cleanup item 3: soft-alert guardrails — owner sees what the cage would catch | PROPAGATE |
| `ef334135` | 2026-08-18 | ? | P0-cleanup item 4: level_state trader-scoped (cross-trader class closed) | PROPAGATE |
| `bb966a04` | 2026-08-18 | ? | P0-cleanup item 5: cosmetics + prior audit reports landed | PROPAGATE |
| `f730d434` | 2026-08-18 | docs | docs: cleanup-train report — 6 of 6 landed, deployed bb966a049d9e | OWNER-ONLY |
| `c2efefc9` | 2026-08-18 | docs | docs: cleanup-train report at spec limit | OWNER-ONLY |
| `49dd83c9` | 2026-08-18 | docs | docs: forensic multi-agent verification — 12 of 13 causes verified live, 13 findings, 5 unproven | OWNER-ONLY |
| `33d7eed2` | 2026-08-18 | timegate | fix(timegate): ai-call-timeout — hardcoded 180s cap shadowed config and killed 150s+ reasoning reads | PROPAGATE |
| `862bcd41` | 2026-08-18 | timegate | fix(timegate): last-entry cutoff — day-scoped 13:00 CT literal refused every Asia-evening entry | PROPAGATE |
| `889c2437` | 2026-08-18 | timegate | fix(timegate): eod-flat — day-scoped 14:45 CT twin would flatten Asia positions on sight | PROPAGATE |
| `12f92bf7` | 2026-08-18 | timegate | fix(timegate): ui-timelines — browser-local rendering broke the Houston anchor for remote viewers | PROPAGATE |
| `fb9baa81` | 2026-08-18 | timegate | fix(timegate): ai-timeout tests — T5/T6/T7 against a real HTTP server | PROPAGATE |
| `8944e07a` | 2026-08-18 | timegate | fix(timegate): ai-params wiring — TimeoutSet ignored the canonical env; Request path dropped configu | PROPAGATE |
| `039f4bb8` | 2026-08-18 | timegate | fix(timegate): session-digest close test — wrap-blind >= end marked running ASIA as closed | PROPAGATE |
| `844c9580` | 2026-08-18 | timegate | fix(timegate): weekly matched-random freeze — Weekday/ISOWeek ran in host tz, not CT | PROPAGATE |
| `8218b4ea` | 2026-08-18 | timegate | fix(timegate): comment truth — stale planner-client claim, dead FlatCT field, zero-only dailyPnL | PROPAGATE |
| `4ebd779a` | 2026-08-18 | timegate | fix(timegate): clock-health line — boot + session-roll observability for Go/NT8/timesync clocks | PROPAGATE |
| `b994e30b` | 2026-08-18 | docs | docs: timegate audit report — 57 unique gates, 8 BUG rows fixed, clock-health caught live WSL skew | OWNER-ONLY |
| `22ea40ad` | 2026-08-19 | bars | fix(bars): NT8 close-stamps converted to OPEN stamps once, at ingest (S2) | PROPAGATE |
| `e9084e38` | 2026-08-19 | charts | fix(charts): ONE CT label site — PlanMiniChart rendered UTC, +5h vs NT8 (S1) | PROPAGATE |
| `a6b0ae24` | 2026-08-19 | stale | fix(stale): B4 age contract — end-stamp-era math lost a period of slack; post-call clock conflated l | PROPAGATE |
| `d4396912` | 2026-08-19 | feed | feat(feed): U1 — feed loss announces itself (P0 alert · planner preflight · 60s liveness line) | PROPAGATE |
| `00a4bef6` | 2026-08-19 | feed | fix(feed): liveness bar_age reads the OPEN stamp — the close-stamp variant went negative on forming  | PROPAGATE |
| `44808dc0` | 2026-08-19 | intrade | fix(intrade): feed watch moves to the 60s wall-clock monitor — in runCycle it was doubly dead (skip- | PROPAGATE |
| `37b3b6bd` | 2026-08-19 | intrade | fix(intrade): feed alert tightens to INTRADE_FEED_ALERT_S (120s) while holding — in-position the ban | PROPAGATE |
| `c335f897` | 2026-08-19 | intrade | fix(intrade): skip-while-open relocates BELOW snapshot+equity — in-position cycles now build context | PROPAGATE |
| `b5aa0f48` | 2026-08-19 | docs | docs: in-position silence report — skip-while-open froze equity/decisions/guards for the life of eve | OWNER-ONLY |
| `d154bb44` | 2026-08-19 | skip-gate | fix(skip-gate): position-state reconciliation before skip-while-open — the gate never trusts a stale | PROPAGATE |
| `51c2a9bf` | 2026-08-19 | clock | feat(clock): root-free clock-guard user timer — 15-min drift detector (host RTC vs WSL + NTP offset) | PROPAGATE |
| `d9978599` | 2026-08-19 | clock | feat(clock): CLOCK_WARN_MS early warning (30s = half of C2 tolerance) + boot clock-guard block — dri | PROPAGATE |
| `71ee0444` | 2026-08-19 | pause | feat(pause): stop_until gets its producer — owner pause API + dashboard one-tap control; NEW entries | PROPAGATE |
| `ef66d44e` | 2026-08-19 | roll | feat(roll): contract-roll entry block for continuous MNQ — the dated-code T19 gate never fires on th | PROPAGATE |
| `a37aff45` | 2026-08-19 | halfdays | feat(halfdays): the HalfDays map gets its producer — official 2026 CME early-close table (Labor Day  | PROPAGATE |
| `d4ad9875` | 2026-08-19 | pause | fix(pause): until_ct parses by hand — the TZ-guard forbids bare time layouts outside kernel/tz.go (c | PROPAGATE |
| `d9a23407` | 2026-08-19 | ai402 | feat(ai402): 402 becomes an OUTAGE, not 139 log lines — typed decision_records error_class, one P0 ' | PROPAGATE |
| `85ec608f` | 2026-08-19 | logdb | feat(logdb): WARN+ERROR ship to the log_events table — logrus hook tee (zero call-site changes), sel | PROPAGATE |
| `ce1a3988` | 2026-08-19 | snapshot | fix(snapshot): ONE instant per cycle — snapshotNow hoists above the market fetch, the 1m window read | PROPAGATE |
| `1021c65b` | 2026-08-19 | stale | fix(stale): B4 refuses the hidden clock — SnapshotMs absent is now WARN + fail-open, never a silent  | PROPAGATE |
| `59e98d3c` | 2026-08-19 | gates | test(gates): E5 interaction-edge guard — stop_until outranks contract-roll in the entry stack, so a  | PROPAGATE |
| `c0e5ce43` | 2026-08-19 | test | fix(test): E5 gate-order anchors use the emitted refusal strings — the phrase-level anchors matched  | PROPAGATE |
| `34cf4d4f` | 2026-08-19 | docs | docs: ledger-close dispatch report — per-phase findings, canonical 42-gate order, official half-day  | OWNER-ONLY |
| `e065b01e` | 2026-08-19 | cadence | feat(cadence): P10 owner ruling — the Studio scan interval IS the decision cadence; bar-close demote | PROPAGATE |
| `a4aa9e7c` | 2026-08-19 | cadence | feat(cadence): P10.3/10.5 — cadence_mode through the trader API (create+update, remove+reload on sav | PROPAGATE |
| `f9e449dd` | 2026-08-19 | docs | docs: P10 section — owner ruling, mode semantics, prompt honesty, cost note, E13-E17 slots | OWNER-ONLY |
| `e3e22a15` | 2026-08-19 | halfdays | fix(halfdays): E7-v2 HIGH — calendar dates convert to session-day KEYS at seed (CMESessionDayKey = 1 | PROPAGATE |
| `4d17cbae` | 2026-08-19 | halfdays | fix(halfdays): E7-v2 — pull-in compares in session-relative minutes; an out-of-window early close no | PROPAGATE |
| `cc7e45db` | 2026-08-19 | roll | fix(roll): E7-v2 — daysLeft rounds to the nearest calendar day; the CT-midnight span loses an hour a | PROPAGATE |
| `26a90967` | 2026-08-19 | pause | fix(pause): E7-v2 trio — until_ct strict HH:MM parse (trailing garbage rejected), tomorrow-wrap via  | PROPAGATE |
| `14f93d11` | 2026-08-19 | halfdays | fix(halfdays): E7-v2 medium — a row deleted from half_days.json is pruned from the registry via a pr | PROPAGATE |
| `f6447076` | 2026-08-19 | boot | feat(boot): E1 completion — per-trader ledger boot block (sessions/cutoffs CT, stop_until, cadence,  | PROPAGATE |
| `05bbca7b` | 2026-08-19 | logs | feat(logs): honest logs — journald rate-limit raise (owner applies with sudo) + WARN promotion of ow | PROPAGATE |
| `6c5ef518` | 2026-08-19 | discard-burn | feat(discard-burn): dodge cycles that would span a bar close (avg×1.2, defer to close+1s), re-evalua | PROPAGATE |
| `2625a04d` | 2026-08-19 | watcher | feat(watcher): in-position AI goes watch-only — position_mode per trader (ai_watch default | bracket | PROPAGATE |
| `7bcefb5d` | 2026-08-19 | fmt | fix(fmt): gofmt realign of the AutoTrader field block after Phase 2/3 field additions (comment colum | PROPAGATE |
| `8f7edd1b` | 2026-08-19 | trailing | feat(trailing): trailing profit in Risk Control — best-price ∓ mult×ATR(period,5m) ratchet on the 60 | PROPAGATE |
| `6b4ab842` | 2026-08-19 | post-exit | feat(post-exit): one immediate full decision cycle after every confirmed position close (close_sync  | PROPAGATE |
| `8dd0dabb` | 2026-08-19 | ask-planner | fix(ask-planner): stuck send — 30s axios timeout vs 300s planner budget threw past an unguarded awai | PROPAGATE |
| `6e7863f1` | 2026-08-19 | minconf | fix(minconf): 6.1 — one shared min-confidence default (60) for gate clamp AND futures prompt; unset  | PROPAGATE |
| `e2367618` | 2026-08-19 | cadence | fix(cadence): 6.2 — trader edit PUT now persists cadence_mode (was create-only, the census register  | PROPAGATE |
| `d92bac9d` | 2026-08-19 | guardrails | fix(guardrails): 6.3 — CheckSoft now audits blackout + consistency too: every CONFIGURED limit rende | PROPAGATE |
| `0d6b58d3` | 2026-08-19 | caps | fix(caps): 6.4 ruling B — remove the two dead cap toggles (zero readers) from Studio; rows render 'a | PROPAGATE |
| `d005fd40` | 2026-08-19 | guardrails | fix(guardrails): 6.5 — the $500 RISK_MAX_DAILY_LOSS_USD fallback is now stated: boot ledger line nam | PROPAGATE |
| `12ee138d` | 2026-08-19 | docs | fix(docs): 6.6 — three comments claimed a 10-contract default; the researched code default is 2 (max | PROPAGATE |
| `39e99c39` | 2026-08-19 | backfill | feat(backfill): 6.7 — one-time flag-guarded entry_confidence recovery for historical closed position | PROPAGATE |
| `1312fb79` | 2026-08-19 | deprecate | chore(deprecate): 6.8 sweep — remove dead AutoStartRunningTraders (zero callers) + the DATABENTO_DAT | PROPAGATE |
| `6c734afb` | 2026-08-19 | trailing | fix(trailing): stage the Phase-3 stub deletion — auto_trader_trailing.go owns currentTrailLevel now  | PROPAGATE |
| `1d67a675` | 2026-08-19 | post-exit | fix(post-exit): E7 self-grep — the OnPositionClosed hook assignment goes through sync.Once (multiple | PROPAGATE |
| `a374d45c` | 2026-08-19 | docs | docs: final-bundle report — phase ledger, cutover record, E1-E7 evidence (E4 live dodge proof), 41-m | OWNER-ONLY |
| `599fd460` | 2026-08-19 | docs | docs: stamp PR #55 (parsed from the gh pr create output URL) | OWNER-ONLY |
| `7cefc00f` | 2026-08-19 | watcher-eyes | fix(watcher-eyes): U1 — watch cycles fetch real market data (kernel.EnsureMarketData: MarketDataMap  | PROPAGATE |
| `39554ea8` | 2026-08-19 | fmt | fix(fmt): gofmt realign after the watcher-eyes edits (comment columns only) | PROPAGATE |
| `69d35170` | 2026-08-19 | docs | docs: final-bundle soak addendum — full-hour numbers (3.4% discards vs 18.5%, 0 lost entries) + watc | OWNER-ONLY |
| `4d4bca00` | 2026-08-19 | fe | fix(fe): shared guardedCall never-latch helper (extracted per twin-path rule) + 320s budgets on the  | PROPAGATE |
| `23a7be65` | 2026-08-19 | fe | fix(fe): complete the Ask-Planner refit onto guardedCall (the prior anchor missed prettier's reforma | PROPAGATE |
| `d4c8c44f` | 2026-08-19 | reset-dialog | fix(reset-dialog): the stuck confirm — guardedCall + try/finally (busy can never latch), double-clic | PROPAGATE |
| `610ed72a` | 2026-08-19 | reread-dialog | fix(reread-dialog): same stuck-confirm class as reset — guardedCall + finally, dedupe, disabled Yes  | PROPAGATE |
| `2bda99a5` | 2026-08-19 | edit-sheet | fix(edit-sheet): save/delete never-latch — guardedCall on overlay + owner-level posts; a thrown time | PROPAGATE |
| `4deebe3e` | 2026-08-19 | realign | fix(realign): never-latch — runRealign's planner-backed call could strand the phase at 'reviewing' o | PROPAGATE |
| `029ccb85` | 2026-08-19 | plan-door | fix(plan-door): bulk-add rows + PlannerReply apply/decline onto guardedCall — a thrown request count | PROPAGATE |
| `239cd7ab` | 2026-08-20 | F4 | fix(F4): 5m_close evaluates as authored — one 5m close (was silently 2x5m; the death log printed the | PROPAGATE |
| `6119b7d5` | 2026-08-20 | T5 | fix(T5): C2 clock-drift evaluates the snapshot instant vs the snapshot bars — a legal 200-300s AI ca | PROPAGATE |
| `f6a59c8d` | 2026-08-20 | F13+A11 | fix(F13+A11): delete the dead activePlanIsDead path (weaker duplicate death definition, zero callers | PROPAGATE |
| `dc68f4ee` | 2026-08-20 | F12 | fix(F12): cited_scenario joins the futures OUTPUT CONTRACT (example JSON + field description) and th | PROPAGATE |
| `e5b68e25` | 2026-08-20 | F2+F8 | fix(F2+F8): scenario-dot honesty + anchor mining — verdicts carry their BASIS (machine when the anch | PROPAGATE |
| `b2a42138` | 2026-08-20 | F5+F11 | fix(F5+F11): write-time truth — structured death/flip prices MUST match their prose twins (validator | PROPAGATE |
| `27b357ff` | 2026-08-20 | F14 | fix(F14): overlay failures are never silent — read-merge SKIPPED patches WARN + surface as overlay_e | PROPAGATE |
| `6cffb5e8` | 2026-08-20 | F14 | fix(F14): missing logger import for the fallback WARN (fix-forward on the prior commit) | PROPAGATE |
| `5d5cdd00` | 2026-08-20 | F14 | fix(F14): hoist the overlay-error collector to function scope (fix-forward — the response map could  | PROPAGATE |
| `4c9bd78c` | 2026-08-20 | A10 | fix(A10): comment truth + gate dedup — the bridge no longer claims NT8 bars are closed (the newest i | PROPAGATE |
| `53da0e86` | 2026-08-20 | T3 | fix(T3): ONE timeframe table (kernel/timeframes.go) — the three drifted private copies delegated (st | PROPAGATE |
| `cf51baf7` | 2026-08-20 | T6 | fix(T6): one feed-aliveness policy — flat FEED_ALERT_S=600 / in-position INTRADE_FEED_ALERT_S=120 co | PROPAGATE |
| `001762ca` | 2026-08-20 | F6 | fix(F6): strict adherence upgraded to spec — the entry is band-judged against the cited scenario (an | PROPAGATE |
| `ad5a5eb8` | 2026-08-20 | C1/F3 | feat(C1/F3): structured scenario confirmation — confirm{rule: touch|1x5m_close|2x5m_close|15m_close, | PROPAGATE |
| `a8096585` | 2026-08-20 | D1/F7 | docs(D1/F7): scenario state stays ADVISORY by ruling — card header labeled '(advisory)' with the rat | PROPAGATE |
| `130870bd` | 2026-08-20 | D2/F9 | docs(D2/F9): target_chain stated as GUIDANCE in both contracts — the executor AI sets the actual TP, | PROPAGATE |
| `b335d844` | 2026-08-20 | D2 | docs(D2): regenerate the remaining futures golden (the D2 guidance line; the -run filter missed the  | PROPAGATE |
| `2d8dae1f` | 2026-08-20 | D3/F10 | docs(D3/F10): scenario quality labeled informational (tooltip) — enum-validated, rendered, consumed  | PROPAGATE |
| `7bdda06c` | 2026-08-20 | E5 | fix(E5): v1 strays — the chat-path OHLCV table renders CT like every trading prompt (the last Time(U | PROPAGATE |
| `1a9f81c4` | 2026-08-20 | E5/v1#8 | fix(E5/v1#8): the running revision is exposed on the trader status API (cached at the boot assertion | PROPAGATE |
| `227ffa6e` | 2026-08-20 | E2 | chore(E2): RISK_MAX_CONTRACTS_PER_ORDER removed end-to-end — grep-proven zero readers (decl+copy onl | PROPAGATE |
| `ad2d0115` | 2026-08-20 | E2 | chore(E2): drop the removed field from the RiskLimits test fixture (fix-forward) | PROPAGATE |
| `b48cd2a6` | 2026-08-20 | E3 | chore(E3): the write-only trader-row prompt pair removed from code — at.customPrompt/overrideBasePro | PROPAGATE |
| `3e01b503` | 2026-08-20 | E3 | chore(E3): the API handler stops feeding the removed in-memory prompt pair (column persist stays for | PROPAGATE |
| `f82a2418` | 2026-08-20 | E4 | fix(E4): the pre-existing FE test pair — the logo test asserted the pre-rebrand alt text (component  | PROPAGATE |
| `4e32cd5a` | 2026-08-20 | E1 | fix(E1): Studio save honesty — success toast ONLY on HTTP 200 via the shared guardedCall, ANY failur | PROPAGATE |
| `c28a1c9e` | 2026-08-20 | docs | docs: fail-register-close report — triage table, 26-commit ledger, V1 first-attempt grammar pass wit | OWNER-ONLY |
| `b9196b93` | 2026-08-20 | pnl | fix(pnl): close-sync attributes ONLY the owning row's quantity — a manual NT8 flatten's frame (qty=2 | PROPAGATE |
| `ca6e990f` | 2026-08-20 | pnl | fix(pnl): additive correction layer — pnl_corrected + note columns (originals never edited), one-tim | PROPAGATE |
| `f024955a` | 2026-08-20 | pnl | feat(pnl): per-close integrity guard — every close verifies recorded vs recomputed (stored prices x  | PROPAGATE |
| `5bc5c945` | 2026-08-20 | pnl | fix(pnl): missing telemetry import — guard commit went out unbuildable (fix-forward) | PROPAGATE |
| `17cd52e2` | 2026-08-20 | pnl | fix(pnl): every remaining aggregate reader honors corrections — loss-streak halt, full stats, recent | PROPAGATE |
| `28ff303b` | 2026-08-20 | docs | docs: P0 pnl-record-integrity report — #526 truth table (21x overstated), E6 quantity-attribution ro | OWNER-ONLY |
| `a491dc70` | 2026-08-20 | docs | docs: V4 soak verdict — 30-min clean, phantom daily-loss would-trip line GONE, corrected sum live in | OWNER-ONLY |

---
*Generated 2026-08-20 · read-only on nofx · owner delivers to the partner.*
