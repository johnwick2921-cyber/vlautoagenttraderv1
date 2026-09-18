# Building nofx on a machine without a C compiler (partner mirror)

**Wave:** W-CGOFREE-SQLITE-UPSTREAM (2026-09-17). Not a knob — no guide entry.

The default build links `gorm.io/driver/sqlite`, which sits on `mattn/go-sqlite3`
and needs cgo (a working `gcc`). `modernc.org/sqlite`, used for the raw
`database/sql` path, is already pure Go. Machines without gcc (the partner
mirror "Binnie") build with the `cgofree` tag, which swaps BOTH sqlite backends
for pure-Go equivalents:

```
go build -tags cgofree -o nofx-bin .
go test  -tags cgofree ./store/... ./researchsnapshot/...
```

| tag | database/sql `"sqlite"` driver | GORM dialector |
|-----|-------------------------------|----------------|
| (default) | `modernc.org/sqlite` (pure Go) | `gorm.io/driver/sqlite` (mattn, cgo) |
| `cgofree` | `github.com/glebarez/go-sqlite` (pure Go) | `github.com/glebarez/sqlite` (pure Go) |

`glebarez/go-sqlite` is a fork of the `modernc.org/sqlite` driver layer over the
same `modernc.org/sqlite/lib` C translation, so both tag sets run the same
SQLite engine; the owner's machine stays on the default because it is the
binary that has been live since day one and nothing in this wave changes it.

**Rule:** the ONLY package that imports a sqlite driver is
`store/sqlitedriver`. Everything else calls `sqlitedriver.Open(dsn)` or
`sqlitedriver.GormDialector(dsn)`. A second blank import
(`_ "modernc.org/sqlite"`, `_ "github.com/glebarez/go-sqlite"`) anywhere else
panics at init with `sql: Register called twice for driver sqlite` — that was
Binnie's first-boot panic after `researchsnapshot/archive.go` grew its own
import. `store/sqlitedriver.TestSingleRegistration` pins the invariant.

Partner update on a machine without gcc (replaces Binnie's private rebase branch).
The mirror procedure is unchanged — build from a CLEAN head, THEN re-arm the two
markers on disk (uncommitted), THEN build the dist — only the build gets the tag:

```
git checkout HEAD -- deploy/RELEASE web/src/guide/types.ts     # drop the previous on-disk re-arm
git checkout main && git pull --ff-only origin main             # partner main (a mirror of nofx's running tree)
go build -tags cgofree -o nofx-bin.new . && go version -m nofx-bin.new | grep vcs.modified=false
HEAD_SHA=$(git rev-parse HEAD)
echo -n "$HEAD_SHA" > deploy/RELEASE                             # BOOT INTEGRITY compares the binary rev to this
sed -i "s/^export const GUIDE_BUILT_REV = '[0-9a-f]*'/export const GUIDE_BUILT_REV = '$HEAD_SHA'/" web/src/guide/types.ts
(cd web && npm run build)                                        # the 🖥 line must read bundle-rev == binary rev
mv -n nofx-bin nofx-bin.old.$(go version -m nofx-bin | awk -F= '/vcs.revision/{print substr($2,1,8)}') && mv nofx-bin.new nofx-bin
```

Then restart the bot the way that machine runs it (systemd where it exists;
Binnie has no sudo and runs `setsid nohup ./nofx-bin &`, so stop the old pid and
relaunch). Proofs to paste: `BOOT INTEGRITY OK — rev X · expected X · goldens
PASS`, `🖥 … bundle-rev=X matches the binary`, `hello handshake OK`, positions
`count=0`. The full step-by-step with rc checks is the CTO's
`update-partner-machine.sh` (2026-09-17); it needs only `-tags cgofree` added
to its `go build` line on this machine.

Web side, same wave: `EquityChart` no longer throws on a missing
`total_equity` (renders the empty state / `0.00`), and a root
`<ErrorBoundary>` under `App` shows a one-line error with a Reload button
instead of a white screen.
