package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nofx/logger"
)

// captureLog redirects the process logger into a buffer for one test.
// loadDotEnv logs through the same package-level logger main uses, so the
// assertions below read the exact line an operator would see.
func captureLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := logger.Log.Out
	logger.Log.SetOutput(&buf)
	t.Cleanup(func() { logger.Log.SetOutput(prev) })
	return &buf
}

func writeDotEnv(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// The partner-install shape: an RSA private key pasted unquoted across
// several lines. Line 1 parses as KEY=value; line 2 is base64 whose '+' is
// not a legal variable-name character, so godotenv rejects the whole file
// and sets NOTHING (loadFile is all-or-nothing). The WARN must say so, name
// the line, and must not echo the file: godotenv's error quotes the entire
// remainder of the file from the bad statement onward.
func TestLoadDotEnv_MalformedFileWarnsWithLineAndWithoutContents(t *testing.T) {
	const good = "NOFX_TEST_DOTENV_GOOD"
	const secret = "sk-test-secret-must-not-appear-in-log"
	t.Cleanup(func() { os.Unsetenv(good) })
	os.Unsetenv(good)

	path := writeDotEnv(t, good+"=ok\n"+
		"RSA_PRIVATE_KEY=-----BEGIN RSA PRIVATE KEY-----\n"+
		"MIIEowIBAAKCAQEA+notreallyakey/AAAA\n"+
		"DEEPSEEK_API_KEY="+secret+"\n")

	buf := captureLog(t)
	loadDotEnv(path)
	out := buf.String()

	if !strings.Contains(out, "⚠️ .env NOT loaded:") {
		t.Fatalf("expected the WARN line, got:\n%s", out)
	}
	if !strings.Contains(out, "line 3") {
		t.Errorf("expected the bad line to be named (line 3), got:\n%s", out)
	}
	if !strings.Contains(out, "every variable falls back to the process environment") {
		t.Errorf("expected the fail-open explanation, got:\n%s", out)
	}
	for _, leak := range []string{secret, "MIIEow", "BEGIN RSA"} {
		if strings.Contains(out, leak) {
			t.Errorf("log line echoes file contents (%q):\n%s", leak, out)
		}
	}
	if _, set := os.LookupEnv(good); set {
		t.Errorf("a malformed .env must set nothing (fail open as before), but %s is set", good)
	}
}

func TestLoadDotEnv_ValidFileIsSilentAndLoads(t *testing.T) {
	const key = "NOFX_TEST_DOTENV_VALID"
	t.Cleanup(func() { os.Unsetenv(key) })
	os.Unsetenv(key)

	path := writeDotEnv(t, key+"=loaded\n")
	buf := captureLog(t)
	loadDotEnv(path)

	if out := buf.String(); strings.Contains(out, ".env") {
		t.Errorf("a valid .env must load silently, got:\n%s", out)
	}
	if got := os.Getenv(key); got != "loaded" {
		t.Errorf("%s = %q, want %q", key, got, "loaded")
	}
}

func TestLoadDotEnv_AbsentFileLogsInfo(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	buf := captureLog(t)
	loadDotEnv(path)
	out := buf.String()

	if !strings.Contains(out, "[INFO]") || !strings.Contains(out, ".env absent — process environment only") {
		t.Errorf("expected the INFO absent line, got:\n%s", out)
	}
	if strings.Contains(out, "⚠️") {
		t.Errorf("an absent .env must not WARN, got:\n%s", out)
	}
}

// A quoted value may legally span lines. Every prefix that ends inside the
// quotes fails to parse and then recovers, so the bad line is the one after
// the LAST prefix that parses — not the first prefix that fails.
func TestDotEnvErrorLine_MultiLineQuoteBeforeBadLine(t *testing.T) {
	path := writeDotEnv(t, "A=1\n"+
		"B=\"first\n"+
		"second\n"+
		"third\"\n"+
		"C=fine\n"+
		"+broken\n")
	if got := dotEnvErrorLine(path); got != 6 {
		t.Errorf("dotEnvErrorLine = %d, want 6", got)
	}
}

// The redactor must keep the diagnosis and drop the quoted file contents
// for both error shapes godotenv v1.5.1 produces.
func TestRedactDotEnvErr(t *testing.T) {
	cases := map[string]struct{ in, want string }{
		"unexpected character quotes the rest of the file": {
			in:   `unexpected character "+" in variable name near "MIIEow+secret\nAPI_KEY=sk-leak\n"`,
			want: `unexpected character "+" in variable name`,
		},
		"unterminated quote echoes the value": {
			in:   `unterminated quoted value "-----BEGIN RSA PRIVATE KEY-----`,
			want: `unterminated quoted value`,
		},
		"open error carries only the path": {
			in:   `open .env: permission denied`,
			want: `open .env: permission denied`,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := redactDotEnvErr(errors.New(tc.in)); got != tc.want {
				t.Errorf("redactDotEnvErr(%q)\n got %q\nwant %q", tc.in, got, tc.want)
			}
		})
	}
}
