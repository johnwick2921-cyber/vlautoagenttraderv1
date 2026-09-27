package updaterworker

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ── the boot proof Watch does not give (C13 as ruled) ──────────────────────
//
// activation.Watch is GREEN on any log token that prefixes the sha — and the
// REFUSED boot line carries the same "rev <sha12>" token as the OK one
// (kernel/boot_integrity.go Line(): "🔐 BOOT INTEGRITY %s — rev %s…"; main.go
// prints the REFUSED form at ERROR and the process keeps serving /api/health
// "ok"). So a boot that REFUSED trading passes Watch. boot_verified therefore
// reads the log itself, from the offset recorded BEFORE the kill:
//
//   - the literal "BOOT INTEGRITY OK — rev <sha12> ·" must be there (the " ·"
//     after the rev excludes the " +dirty" form, which is never a release);
//   - NO "BOOT INTEGRITY REFUSED" line may follow the offset, for any rev
//     (stricter than "for it": a refused boot after our kill is ours to own).
//
// Both matches are ANCHORED to the boot line's own logger shape (#206 review
// fold, CTO verdict on 88944226): the compactFormatter renders
// "MM-DD HH:MM:SS [LEVEL] caller msg" (logger/logger.go), main.go prints the
// OK line at INFO and the REFUSED line at ERROR, the caller is
// "<build-dir>/main.go:<line>" — the directory the binary was BUILT in, which
// varies (live logs show clone-build/main.go, nofx-clean/main.go, nofx/main.go
// [A: read from data/nofx_2026-09-23.log]). The match starts at LINE START
// with the formatter's exact prefix, the caller must be the FIRST token after
// the level (attacker text can only appear after it), and the line must carry
// the NEW MainPID the worker read after the restart (kernel/boot_integrity.go
// prints "· pid <n>") — a value no request can know in advance, so client text
// logged with %s (even one carrying a raw newline) cannot forge the boot line.
// Four strings.Contains anywhere in the line was still a substring matcher:
// any other log line echoing client text that carries those pieces would force
// a good install into rollback.

const (
	bootOKPrefix  = "BOOT INTEGRITY OK — rev "
	bootRefused   = "BOOT INTEGRITY REFUSED"
	maxBootScan   = 64 << 20
	bootLineShort = 12 // kernel/boot_integrity.go prints rev[:12]

	// bootShape: ^MM-DD HH:MM:SS [LEVEL] <builddir>/main.go:<line> 🔐 BOOT INTEGRITY
	bootShape = `^\d{2}-\d{2} \d{2}:\d{2}:\d{2} \[%s\] \S*/main\.go:\d+ 🔐 `
)

// bootOKRe is the app's OWN OK boot line: the exact formatter prefix, the
// caller first after the level, the rev, the pid of the NEW MainPID and the
// "expected <sha12>" field (the RELEASE half — an OK line that does not
// expect this sha is not this install's line).
func bootOKRe(rev12 string, pid int) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(bootShape, "INFO") +
		regexp.QuoteMeta(bootOKPrefix+rev12+" · pid "+strconv.Itoa(pid)+" · built ")+`\S+`+
		regexp.QuoteMeta(" · expected "+rev12+" ·"))
}

// bootRefusedRe is the app's OWN REFUSED boot line by shape and pid.
func bootRefusedRe(pid int) *regexp.Regexp {
	return regexp.MustCompile(fmt.Sprintf(bootShape, "ERRO") +
		regexp.QuoteMeta(bootRefused+" — rev ")+`\S+ · pid `+strconv.Itoa(pid)+` `)
}

// isBootOKLine: the app's OWN OK boot line, by shape and pid.
func isBootOKLine(ln, sha12 string, pid int) bool {
	return bootOKRe(sha12, pid).MatchString(ln)
}

// isBootRefusedLine: the app's OWN REFUSED boot line, by shape and pid.
func isBootRefusedLine(ln string, pid int) bool {
	return bootRefusedRe(pid).MatchString(ln)
}

// verifyBootLine scans path from off. pid is the NEW MainPID the worker read
// after the restart — only its boot line counts. ev gets the READ facts.
func verifyBootLine(path string, off int64, sha string, pid int, ev map[string]string) error {
	if len(sha) < bootLineShort || !isSHA40(sha) {
		return fmt.Errorf("boot check: %q is not a release sha", sha)
	}
	if off < 0 {
		return errors.New("boot check: negative log offset")
	}
	if pid < 2 {
		return fmt.Errorf("boot check: the new MainPID %d cannot have written a boot line", pid)
	}
	want := bootOKPrefix + sha[:bootLineShort] + " ·"
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("boot check: the boot log: %w", err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("boot check: %w", err)
	}
	if fi.Size() < off {
		return fmt.Errorf("boot check: %s is %d bytes, shorter than the offset %d recorded before the kill (truncated or replaced)", path, fi.Size(), off)
	}
	if _, err := f.Seek(off, io.SeekStart); err != nil {
		return fmt.Errorf("boot check: %w", err)
	}
	ev["log_offset"] = strconv.FormatInt(off, 10)
	ev["boot_pid"] = strconv.Itoa(pid)
	sc := bufio.NewScanner(io.LimitReader(f, maxBootScan))
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	ok := 0
	for sc.Scan() {
		ln := sc.Text()
		if isBootRefusedLine(ln, pid) {
			return fmt.Errorf("boot check: a REFUSED boot line follows the kill: %q", clipText(strings.TrimSpace(ln)))
		}
		if isBootOKLine(ln, sha[:bootLineShort], pid) {
			ok++
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("boot check: reading %s: %w", path, err)
	}
	if ok == 0 {
		return fmt.Errorf("boot check: no %q line in %s after offset %d", strings.TrimSuffix(want, " ·"), path, off)
	}
	ev["boot_line"] = "OK rev " + sha[:bootLineShort]
	return nil
}
