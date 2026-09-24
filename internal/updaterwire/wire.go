// Package updaterwire is the W-ONE-BUTTON M3 channel between the Go app and
// the updater worker: a unix socket at <data>/updater/worker.sock carrying
// newline-delimited, typed JSON frames with a FIXED verb set
// {status, install, cancel-before-boundary}. Anything else is rejected.
//
// This package is the codec, the id allow-lists, the socket path, and the
// app-side Dial. The worker side (Listen/Serve) lives in
// nofx/internal/updaterwire/wireserver so that an import-direction census can
// prove the trading app never links it.
//
// Wire format, one frame per line, at most MaxFrameBytes before the newline:
//
//	{"v":1,"verb":"status","payload":{}}
//	{"v":1,"verb":"status","payload":{"job_id":"<job>"}}
//	{"v":1,"verb":"install","payload":{"release_id":"<rel>","job_id":"<job>"}}
//	{"v":1,"verb":"cancel-before-boundary","payload":{"job_id":"<job>"}}
//
//	{"v":1,"ok":true,"state":"<state>"}
//	{"v":1,"ok":false,"error":"<text>"}
//
// Decoding is strict: exact-case keys only (encoding/json alone would accept
// "Verb"), no unknown or duplicate keys, no trailing data, typed payloads
// validated per verb. install carries release_id + job_id ONLY — never a URL,
// path, command or MAC (authorization happens in the app before a frame is
// ever built; the worker re-reads the verified manifest itself in M4).
package updaterwire

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode/utf8"
)

// ProtocolVersion is the only "v" a frame may carry.
const ProtocolVersion = 1

// MaxFrameBytes caps one frame (excluding its newline): 16 KiB.
const MaxFrameBytes = 16 << 10

// Verb is a request verb. The set is fixed; see Verbs.
type Verb string

const (
	VerbStatus               Verb = "status"
	VerbInstall              Verb = "install"
	VerbCancelBeforeBoundary Verb = "cancel-before-boundary"
)

// Verbs is the complete verb set, in a fixed order.
func Verbs() []Verb {
	return []Verb{VerbStatus, VerbInstall, VerbCancelBeforeBoundary}
}

// Known reports whether v is one of the three verbs (exact bytes).
func (v Verb) Known() bool {
	switch v {
	case VerbStatus, VerbInstall, VerbCancelBeforeBoundary:
		return true
	}
	return false
}

// StatusPayload asks for the worker's state, or one job's when JobID is set.
type StatusPayload struct {
	JobID string `json:"job_id,omitempty"`
}

// InstallPayload starts an ALREADY-AUTHORIZED job for a release whose
// manifest the app verified. Exactly these two fields.
type InstallPayload struct {
	ReleaseID string `json:"release_id"`
	JobID     string `json:"job_id"`
}

// CancelPayload cancels a job that has not crossed its point of no return.
type CancelPayload struct {
	JobID string `json:"job_id"`
}

// Request is one decoded frame: Verb plus exactly the one payload that verb
// takes (the others nil).
type Request struct {
	Verb    Verb
	Status  *StatusPayload
	Install *InstallPayload
	Cancel  *CancelPayload
}

// NewStatus builds a status request (jobID "" = the worker as a whole).
func NewStatus(jobID string) Request {
	return Request{Verb: VerbStatus, Status: &StatusPayload{JobID: jobID}}
}

// NewInstall builds an install request.
func NewInstall(releaseID, jobID string) Request {
	return Request{Verb: VerbInstall, Install: &InstallPayload{ReleaseID: releaseID, JobID: jobID}}
}

// NewCancelBeforeBoundary builds a cancel-before-boundary request.
func NewCancelBeforeBoundary(jobID string) Request {
	return Request{Verb: VerbCancelBeforeBoundary, Cancel: &CancelPayload{JobID: jobID}}
}

// Response is the worker's answer. OK ⇒ State set, Error empty; !OK ⇒ Error
// set, State empty.
type Response struct {
	OK    bool
	State string
	Error string
}

// RejectedResponse is the one answer every refused frame gets — no detail
// about which check refused it goes back over the wire.
var RejectedResponse = Response{OK: false, Error: "rejected"}

// Frame refusals. RejectReason maps each to a fixed log token.
var (
	ErrOversize       = errors.New("updaterwire: frame exceeds 16 KiB")
	ErrTruncated      = errors.New("updaterwire: stream ended mid-frame")
	ErrMalformed      = errors.New("updaterwire: malformed frame")
	ErrTrailingData   = errors.New("updaterwire: data after the frame object")
	ErrUnknownField   = errors.New("updaterwire: unknown field")
	ErrDuplicateField = errors.New("updaterwire: duplicate field")
	ErrBadVersion     = errors.New("updaterwire: unsupported protocol version")
	ErrUnknownVerb    = errors.New("updaterwire: unknown verb")
	ErrBadPayload     = errors.New("updaterwire: invalid payload")
)

var rejectReasons = []struct {
	err    error
	reason string
}{
	{ErrOversize, "oversize"},
	{ErrTruncated, "truncated"},
	{ErrTrailingData, "trailing_data"},
	{ErrUnknownField, "unknown_field"},
	{ErrDuplicateField, "duplicate_field"},
	{ErrBadVersion, "bad_version"},
	{ErrUnknownVerb, "unknown_verb"},
	{ErrBadPayload, "bad_payload"},
	{ErrMalformed, "malformed"},
}

// RejectReason is the fixed, attacker-independent token logged for err.
func RejectReason(err error) string {
	for _, r := range rejectReasons {
		if errors.Is(err, r.err) {
			return r.reason
		}
	}
	return "error"
}

// FrameDigest is the ONLY form in which a refused frame may reach a log:
// its length and the first 16 hex of its SHA-256 — never its bytes.
func FrameDigest(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("len=%d sha256=%s", len(b), hex.EncodeToString(sum[:8]))
}

// ReadFrame reads one newline-terminated frame (newline stripped). A frame
// longer than MaxFrameBytes returns ErrOversize after reading at most
// MaxFrameBytes plus one buffer — the stream cannot resynchronise after that,
// so callers close it. A clean EOF before any byte returns io.EOF; EOF
// mid-frame returns ErrTruncated. On ErrOversize/ErrTruncated the bytes read
// so far are returned for FrameDigest.
func ReadFrame(br *bufio.Reader) ([]byte, error) {
	var buf []byte
	for {
		chunk, err := br.ReadSlice('\n')
		buf = append(buf, chunk...)
		if err == nil {
			frame := buf[:len(buf)-1]
			if len(frame) > MaxFrameBytes {
				return frame, ErrOversize
			}
			return frame, nil
		}
		if len(buf) > MaxFrameBytes {
			return buf, ErrOversize
		}
		switch {
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF):
			if len(buf) == 0 {
				return nil, io.EOF
			}
			return buf, ErrTruncated
		default:
			return buf, err
		}
	}
}

// strictObject reads raw as exactly one JSON object with exact-case keys from
// allowed, no duplicates, nothing after it. It is the guard encoding/json
// lacks: json.Unmarshal matches keys case-insensitively and keeps the LAST of
// duplicate keys, either of which lets two parsers disagree about a frame.
func strictObject(raw []byte, allowed map[string]bool) (map[string]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		return nil, ErrMalformed
	}
	out := map[string]json.RawMessage{}
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, ErrMalformed
		}
		key, ok := kt.(string)
		if !ok {
			return nil, ErrMalformed
		}
		if !allowed[key] {
			return nil, ErrUnknownField
		}
		if _, dup := out[key]; dup {
			return nil, ErrDuplicateField
		}
		var v json.RawMessage
		if err := dec.Decode(&v); err != nil {
			return nil, ErrMalformed
		}
		out[key] = v
	}
	if tok, err := dec.Token(); err != nil || tok != json.Delim('}') {
		return nil, ErrMalformed
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, ErrTrailingData
	}
	return out, nil
}

// decodeString decodes raw as a JSON string (a number/null/object is refused).
func decodeString(raw json.RawMessage) (string, bool) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] != '"' {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

var (
	envelopeKeys = map[string]bool{"v": true, "verb": true, "payload": true}
	payloadKeys  = map[Verb]map[string]bool{
		VerbStatus:               {"job_id": true},
		VerbInstall:              {"release_id": true, "job_id": true},
		VerbCancelBeforeBoundary: {"job_id": true},
	}
)

// DecodeRequest parses one frame (without its newline). Every refusal wraps
// one of the Err* sentinels and never quotes the frame.
func DecodeRequest(frame []byte) (Request, error) {
	if len(frame) > MaxFrameBytes {
		return Request{}, ErrOversize
	}
	if !utf8.Valid(frame) {
		return Request{}, ErrMalformed
	}
	env, err := strictObject(frame, envelopeKeys)
	if err != nil {
		return Request{}, err
	}
	for k := range envelopeKeys {
		if _, ok := env[k]; !ok {
			return Request{}, fmt.Errorf("%w: missing %s", ErrMalformed, k)
		}
	}
	if string(bytes.TrimSpace(env["v"])) != "1" {
		return Request{}, ErrBadVersion
	}
	vs, ok := decodeString(env["verb"])
	verb := Verb(vs)
	if !ok || !verb.Known() {
		return Request{}, ErrUnknownVerb
	}
	praw := bytes.TrimSpace(env["payload"])
	if len(praw) == 0 || praw[0] != '{' {
		return Request{}, fmt.Errorf("%w: payload must be an object", ErrBadPayload)
	}
	fields, err := strictObject(praw, payloadKeys[verb])
	if err != nil {
		return Request{}, err
	}
	str := func(k string) (string, bool, error) {
		raw, present := fields[k]
		if !present {
			return "", false, nil
		}
		s, ok := decodeString(raw)
		if !ok {
			return "", true, fmt.Errorf("%w: %s must be a string", ErrBadPayload, k)
		}
		return s, true, nil
	}
	var req Request
	switch verb {
	case VerbStatus:
		job, present, err := str("job_id")
		if err != nil {
			return Request{}, err
		}
		if present && !ValidJobID(job) {
			return Request{}, fmt.Errorf("%w: job_id", ErrBadPayload)
		}
		req = NewStatus(job)
	case VerbInstall:
		rel, _, err := str("release_id")
		if err != nil {
			return Request{}, err
		}
		job, _, err := str("job_id")
		if err != nil {
			return Request{}, err
		}
		req = NewInstall(rel, job)
	case VerbCancelBeforeBoundary:
		job, _, err := str("job_id")
		if err != nil {
			return Request{}, err
		}
		req = NewCancelBeforeBoundary(job)
	}
	if err := req.Validate(); err != nil {
		return Request{}, err
	}
	return req, nil
}

// Validate checks the verb and that exactly its payload is present and valid.
func (r Request) Validate() error {
	if !r.Verb.Known() {
		return ErrUnknownVerb
	}
	n := 0
	for _, set := range []bool{r.Status != nil, r.Install != nil, r.Cancel != nil} {
		if set {
			n++
		}
	}
	if n != 1 {
		return fmt.Errorf("%w: exactly one payload", ErrBadPayload)
	}
	switch r.Verb {
	case VerbStatus:
		if r.Status == nil || (r.Status.JobID != "" && !ValidJobID(r.Status.JobID)) {
			return fmt.Errorf("%w: status", ErrBadPayload)
		}
	case VerbInstall:
		if r.Install == nil || !ValidReleaseID(r.Install.ReleaseID) || !ValidJobID(r.Install.JobID) {
			return fmt.Errorf("%w: install", ErrBadPayload)
		}
	case VerbCancelBeforeBoundary:
		if r.Cancel == nil || !ValidJobID(r.Cancel.JobID) {
			return fmt.Errorf("%w: cancel-before-boundary", ErrBadPayload)
		}
	}
	return nil
}

type wireRequest struct {
	V       int    `json:"v"`
	Verb    Verb   `json:"verb"`
	Payload any    `json:"payload"`
}

// EncodeRequest validates r and returns its frame including the newline. An
// invalid request never reaches the wire.
func EncodeRequest(r Request) ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	w := wireRequest{V: ProtocolVersion, Verb: r.Verb}
	switch r.Verb {
	case VerbStatus:
		w.Payload = r.Status
	case VerbInstall:
		w.Payload = r.Install
	case VerbCancelBeforeBoundary:
		w.Payload = r.Cancel
	}
	b, err := json.Marshal(w)
	if err != nil {
		return nil, err
	}
	if len(b) > MaxFrameBytes {
		return nil, ErrOversize
	}
	return append(b, '\n'), nil
}

var (
	stateRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)
	errorRe = regexp.MustCompile(`^[a-z][a-z0-9_ .-]{0,127}$`)
)

func (r Response) validate() error {
	if r.OK {
		if !stateRe.MatchString(r.State) || r.Error != "" {
			return fmt.Errorf("%w: ok response needs a state and no error", ErrBadPayload)
		}
		return nil
	}
	if !errorRe.MatchString(r.Error) || r.State != "" {
		return fmt.Errorf("%w: failed response needs an error and no state", ErrBadPayload)
	}
	return nil
}

type wireResponse struct {
	V     int    `json:"v"`
	OK    bool   `json:"ok"`
	State string `json:"state,omitempty"`
	Error string `json:"error,omitempty"`
}

// EncodeResponse validates r and returns its frame including the newline.
func EncodeResponse(r Response) ([]byte, error) {
	if err := r.validate(); err != nil {
		return nil, err
	}
	b, err := json.Marshal(wireResponse{V: ProtocolVersion, OK: r.OK, State: r.State, Error: r.Error})
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

var responseKeys = map[string]bool{"v": true, "ok": true, "state": true, "error": true}

// DecodeResponse parses one response frame with the same strictness as
// DecodeRequest.
func DecodeResponse(frame []byte) (Response, error) {
	if len(frame) > MaxFrameBytes {
		return Response{}, ErrOversize
	}
	if !utf8.Valid(frame) {
		return Response{}, ErrMalformed
	}
	f, err := strictObject(frame, responseKeys)
	if err != nil {
		return Response{}, err
	}
	if string(bytes.TrimSpace(f["v"])) != "1" {
		return Response{}, ErrBadVersion
	}
	var r Response
	switch string(bytes.TrimSpace(f["ok"])) {
	case "true":
		r.OK = true
	case "false":
	default:
		return Response{}, fmt.Errorf("%w: ok", ErrBadPayload)
	}
	if raw, ok := f["state"]; ok {
		s, ok := decodeString(raw)
		if !ok {
			return Response{}, fmt.Errorf("%w: state", ErrBadPayload)
		}
		r.State = s
	}
	if raw, ok := f["error"]; ok {
		s, ok := decodeString(raw)
		if !ok {
			return Response{}, fmt.Errorf("%w: error", ErrBadPayload)
		}
		r.Error = s
	}
	if err := r.validate(); err != nil {
		return Response{}, err
	}
	return r, nil
}
