package updateauth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

// ErrMalformed wraps every strict-parse refusal.
var ErrMalformed = errors.New("updateauth: malformed")

func malformed(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrMalformed, fmt.Sprintf(format, a...))
}

// decodeStrictObject reads exactly ONE JSON object from r whose keys are
// exactly `keys` — each present once, compared byte-exact (encoding/json
// matches struct keys case-insensitively and lets a duplicate key silently
// win, so neither is trusted here) — followed by nothing but whitespace.
func decodeStrictObject(r io.Reader, keys ...string) (map[string]json.RawMessage, error) {
	want := make(map[string]bool, len(keys))
	for _, k := range keys {
		want[k] = true
	}
	dec := json.NewDecoder(r)
	tok, err := dec.Token()
	if err != nil {
		return nil, malformed("not JSON")
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, malformed("not a JSON object")
	}
	out := make(map[string]json.RawMessage, len(keys))
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, malformed("bad key")
		}
		k, ok := kt.(string)
		if !ok {
			return nil, malformed("bad key")
		}
		if !want[k] {
			return nil, malformed("unknown field %q", truncate(k, 64))
		}
		if _, dup := out[k]; dup {
			return nil, malformed("duplicate field %q", k)
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, malformed("bad value for %q", k)
		}
		out[k] = v
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, malformed("unterminated object")
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, malformed("trailing data after the object")
	}
	for _, k := range keys {
		if _, ok := out[k]; !ok {
			return nil, malformed("missing field %q", k)
		}
	}
	return out, nil
}

// rawString decodes a JSON string value (null, numbers, objects refused).
func rawString(v json.RawMessage) (string, error) {
	t := bytes.TrimSpace(v)
	if len(t) == 0 || t[0] != '"' {
		return "", malformed("not a string")
	}
	var s string
	if err := json.Unmarshal(t, &s); err != nil {
		return "", malformed("not a string")
	}
	return s, nil
}

// canonicalUint matches a canonical non-negative decimal: no sign, no
// leading zero, no fraction, no exponent, at most 18 digits (fits int64).
var canonicalUint = regexp.MustCompile(`^(0|[1-9][0-9]{0,17})$`)

// rawUnixSeconds decodes a JSON number that is a canonical positive decimal
// integer — the ONLY encoding expires_at has (the MAC is over its decimal
// text, so 1.7e9, 0017… or "1700000000" must never alias it).
func rawUnixSeconds(v json.RawMessage) (int64, error) {
	t := bytes.TrimSpace(v)
	if !canonicalUint.Match(t) {
		return 0, malformed("not a canonical unix-seconds integer")
	}
	n, err := strconv.ParseInt(string(t), 10, 64)
	if err != nil || n <= 0 {
		return 0, malformed("not a positive unix-seconds integer")
	}
	return n, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
