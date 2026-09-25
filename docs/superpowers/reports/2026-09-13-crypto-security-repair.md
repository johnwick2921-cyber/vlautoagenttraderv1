# Bounded x/crypto advisory repair — 2026-09-13

Commit **40ed5d95** in /tmp/nofx-audit-repairs-20260913 changes **go.mod and go.sum only**. Source frozen. No runtime, live database/settings/account, deployment, C# or wire changes. Root separately owns build/CI pins and final merged validation.

## Confirmed findings and primary sources

[A] Read PR117 check103740249864 annotations, retained /tmp/nofx-crypto-pr117-annotations.json: installed x/crypto0.53.0, CVE-2026-56854 fixed0.55.0, CVE-2026-56855 and CVE-2026-78662 fixed0.56.0, GO-2026-5932 no fixed version.

- **CVE-2026-56854 / GO-2026-6303:** SSH server source-address permissions were not enforced for several non-public-key authentication callbacks; fixed0.55.0. [Go CNA record](https://github.com/CVEProject/cvelistV5/blob/main/cves/2026/56xxx/CVE-2026-56854.json), [Go advisory](https://pkg.go.dev/vuln/GO-2026-6303).
- **CVE-2026-56855 / GO-2026-6355:** malicious SSH messages on established channels could deadlock connections; fixed0.56.0. [Go CNA record](https://github.com/CVEProject/cvelistV5/blob/main/cves/2026/56xxx/CVE-2026-56855.json), [Go advisory](https://pkg.go.dev/vuln/GO-2026-6355).
- **CVE-2026-78662 / GO-2026-6354:** malicious messages before SSH channel establishment could deadlock connections; fixed0.56.0. [Go CNA record](https://github.com/CVEProject/cvelistV5/blob/main/cves/2026/78xxx/CVE-2026-78662.json), [Go advisory](https://pkg.go.dev/vuln/GO-2026-6354).
- **GO-2026-5932:** unmaintained OpenPGP packages are unsafe by design; all versions affected, **no fixed version**. A module bump cannot remove this advisory from module-level scanners. [Primary Go vulnerability JSON](https://vuln.go.dev/ID/GO-2026-5932.json), [maintainer issue](https://go.dev/issue/44226).

## Exact dependency/toolchain change

[A] `go get golang.org/x/crypto@v0.56.0` selected the minimum fixed release covering all three SSH advisories. Downloaded source/module metadata identifies upstream commit86efde54dc7069251a8b007026c500d28e4239ce. Its [go.mod](https://github.com/golang/crypto/blob/v0.56.0/go.mod) requires Go>=1.26.0 and newer x/net/sys/term/text.

Root explicitly approved pinning the current1.26 patch rather than merely minimum1.26.0. `go` directive is now **1.26.8**, and verification actually ran **go1.26.8 linux/amd64** (/tmp/nofx-crypto-toolchain.log). [Official Go release history](https://go.dev/doc/devel/release#go1.26.8) lists1.26.8 released2026-09-01. This is a required toolchain upgrade from1.25.13, not a source-only patch with the old compiler.

Only module-graph-required selected updates occurred: x/crypto0.53→0.56; x/net0.56→0.57; x/sys0.46→0.47; x/text0.39→0.41; x/sync0.21→0.22. The selected transitive graph also raises x/term0.45, x/tools0.48 and x/mod0.38 through these modules; tools/mod checksums were added but no new direct dependencies were introduced. x/crypto's module requires net0.57/sys0.47/term0.45/text0.41; text0.41 requires tools0.48/mod0.38/sync0.22. Exact graph retained /tmp/nofx-crypto-module-graph.log. Historical checksums remain in go.sum; selected versions are shown by go list -m, not inferred from every historical checksum entry. No broad tidy or unrelated upgrade performed.

Root identified/owns the corresponding CI/Docker pin work: .github/workflows/pr-checks.yml and pr-checks-run.yml hardcoded Go1.21, docker/Dockerfile.backend Go1.25-alpine, security.yml govuln installation Go1.26.7. This commit does not claim those separate files were included.

## Reachability and verification

[A] Direct imports are bcrypt (auth/auth.go), sha3 (mcp/payment/x402.go and trader/lighter/types.go). Final `go list -deps -buildvcs=false ./...` and `go list -deps -test -buildvcs=false ./...` completed under1.26.8; **no ssh or openpgp packages appear** in either dependency closure. Logs /tmp/nofx-crypto-deps-final-04.log and /tmp/nofx-crypto-test-deps-final-04.log, with adjacent JSON recording command, rc0 and hashes. This is default linux/amd64 build/test selection, not every possible platform or build tag.

[A] `go build -buildvcs=false ./...` PASS rc0, /tmp/nofx-crypto-build-02.log and .json. Build diagnostic log is empty; success is established by captured process returncode, not empty output. buildvcs disabled only because this temporary worktree's VCS discovery can choose the unrelated /tmp/.git ancestor; no artifact was deployed.

[A] `go test -race ./auth ./mcp/... ./trader/lighter -count=1` PASS: mcp46.107s, mcp/provider1.034s, lighter1.057s. auth and mcp/payment compile but have no test files. /tmp/nofx-crypto-focused-02.log and .json. Narrow direct-consumer rerun also passed /tmp/nofx-crypto-direct-consumers-03.log; it was launched while the broader focused run was finishing, not presented as additional independent coverage.

[A] Pinned govulncheck1.8.0 installed only into /tmp/nofx-crypto-tools with Go1.26.8. `GOFLAGS=-buildvcs=false govulncheck -test ./...` and verbose equivalent completed rc0. The verbose result is precise: **0 reachable symbol vulnerabilities; 0 vulnerabilities in imported packages; 1 module-only finding, GO-2026-5932, x/crypto0.56.0, fixed N/A**. /tmp/nofx-crypto-govulncheck-02.log and /tmp/nofx-crypto-govulncheck-verbose-03.log, adjacent JSON metadata.

This resolves the three version-fixable SSH findings in the selected dependency version. It does **not** assert no vulnerabilities, remove the unmaintained OpenPGP advisory, validate every alternate build, certify a running binary, or substitute for root's final merged full test/CI/security rerun. A module-level Trivy/OpenPGP notice may appropriately remain despite no imported vulnerable package/call path in the inspected build.

[A] `go mod verify` also completed rc0, “all modules verified”, /tmp/nofx-crypto-module-integrity-04.log and .json. Verification metadata records committed go.mod SHA2560a93dc2922a5c201079de8339e9056439b1ba46d983b231e75cffac848cb3ace and go.sum SHA25637c6676ea1d65f0f0e75634a72983ffb32d35816527f140a61a0e8067afdbeed. Root's protected-file baseline must be updated to this reviewed go.mod digest; this is metadata maintenance, not weakening the guard.
