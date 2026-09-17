package main

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"nofx/logger"
)

// loadDotEnv loads path into the process environment and reports the
// outcome ONCE. It fails open exactly as the bare godotenv.Load() did: on
// any error nothing is set and every variable falls back to the process
// environment — the difference is that the operator is now told.
//
// Born on the 2026-09-16 partner install: a malformed RSA_PRIVATE_KEY line
// made Load fail, the error was discarded, and the boot died as "secrets
// missing" in a crash loop with no hint which line was bad.
//
// The godotenv error is never logged verbatim: for a parse error it quotes
// the remainder of the FILE from the bad statement onward (see
// redactDotEnvErr), which on the partner-install shape is the private key
// body and every secret after it — and WARN lines reach journald, the
// daily log file and the DB sink.
func loadDotEnv(path string) {
	err := godotenv.Load(path)
	switch {
	case err == nil:
	case errors.Is(err, os.ErrNotExist):
		logger.Infof(".env absent — process environment only")
	default:
		detail := redactDotEnvErr(err)
		if line := dotEnvErrorLine(path); line > 0 {
			detail = "line " + strconv.Itoa(line) + ": " + detail
		}
		logger.Warnf("⚠️ .env NOT loaded: %s — every variable falls back to the process environment", detail)
	}
}

// redactDotEnvErr keeps the diagnosis and drops the file contents from a
// godotenv v1.5.1 error. Its two parse errors are
//
//	unexpected character %q in variable name near %q   (%q = rest of file)
//	unterminated quoted value %s                       (%s = the value)
//
// Anything else (os.Open errors carry only the path) passes through.
func redactDotEnvErr(err error) string {
	msg := err.Error()
	if i := strings.Index(msg, " near "); i >= 0 {
		msg = msg[:i]
	}
	if strings.HasPrefix(msg, "unterminated quoted value") {
		msg = "unterminated quoted value"
	}
	return msg
}

// dotEnvErrorLine returns the 1-based line where path stops parsing, or 0
// when it cannot be read or parses whole. godotenv does not report line
// numbers, so it is recovered by parsing growing prefixes of the file. A
// quoted value may span lines, so prefixes that end inside the quotes fail
// and then recover; a genuine error never recovers, hence the answer is the
// line after the LAST prefix that parses — not the first prefix that fails.
func dotEnvErrorLine(path string) int {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	src := strings.ReplaceAll(string(raw), "\r\n", "\n")
	lines := strings.Split(src, "\n")
	lastOK := 0
	for n := range lines {
		prefix := strings.Join(lines[:n+1], "\n")
		if _, err := godotenv.Unmarshal(prefix); err == nil {
			lastOK = n + 1
		}
	}
	if lastOK >= len(lines) {
		return 0
	}
	return lastOK + 1
}
