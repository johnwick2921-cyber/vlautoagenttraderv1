package updaterworker

// sshsig.go — W-ONE-BUTTON M4 (3b-B, unit U3): verify the OpenSSH SSHSIG
// signature that 3a's release workflow puts beside every manifest
// (`ssh-keygen -Y sign -f <key> -n release manifest.json`, release.yml), with
// the standard library only.
//
// POLICY (brief C6, CTO ruling 1790259689740 — stricter than ssh-keygen, never
// looser). A signature is accepted only when ALL of these hold:
//
//   - the armor is "-----BEGIN SSH SIGNATURE-----" … "-----END SSH SIGNATURE-----";
//   - the blob is magic "SSHSIG", version 1, with nothing after the signature;
//   - the signing key is a plain ssh-ed25519 key (no certificate, no sk-, no
//     RSA/ECDSA);
//   - the namespace is "release" (checked, AND bound into the signed data);
//   - the reserved field is empty (what ssh-keygen writes);
//   - the hash algorithm is sha512 (ssh-keygen also accepts sha256; we do not);
//   - the signing key is BYTE-EQUAL to an ssh-ed25519 key the allowed-signers
//     file lists for the principal "release" — by exact principal, never by
//     pattern: a wildcard line ssh-keygen would honour admits nothing here,
//     and a line whose principal list carries ANY negation ("!…") admits
//     nothing either (ssh-keygen vetoes the line only when the negated
//     pattern matches; not implementing patterns, we veto it always);
//   - ed25519 verifies over "SSHSIG" ‖ string(namespace) ‖ string("") ‖
//     string("sha512") ‖ string(SHA-512(message)) (PROTOCOL.sshsig).
//
// The allowed-signers file is READ AT RUN TIME from the path the caller passes
// (the worker passes <installDir>/deploy/release_allowed_signers). Absent ⇒
// refused: no release can be installed until the owner commits the file
// (deploy/release/README.md, "Owner action, once").
//
// Cross-checked against the real tool: TestSSHSIGVerifiesAnSshKeygenSignature
// and every TestSSHSIGRefuses* case also run `ssh-keygen -Y verify`, so each
// refusal is shown to be the one its name claims.

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const (
	// ReleaseSignaturePrincipal is the ONLY principal whose keys may sign a
	// release (release.yml verifies with -I release).
	ReleaseSignaturePrincipal = "release"
	// ReleaseSignatureNamespace is the ONLY namespace a release signature may
	// carry (release.yml signs with -n release).
	ReleaseSignatureNamespace = "release"
	// ReleaseSignatureHashAlg is the ONLY hash algorithm accepted
	// (ssh-keygen's default; the sha256 it would also accept is refused).
	ReleaseSignatureHashAlg = "sha512"

	sshsigMagic   = "SSHSIG"
	sshsigVersion = 1
	sshEd25519    = "ssh-ed25519"
	armorBegin    = "-----BEGIN SSH SIGNATURE-----"
	armorEnd      = "-----END SSH SIGNATURE-----"

	// MaxAllowedSignersBytes / MaxSignatureBytes cap what is read: a trust
	// anchor or a signature larger than this is refused, never truncated.
	MaxAllowedSignersBytes = 64 << 10
	MaxSignatureBytes      = 64 << 10
)

// Each refusal is a sentinel so a test can prove WHICH check refused.
var (
	ErrNoAllowedSigners     = errors.New("sshsig: no allowed-signers file")
	ErrAllowedSignersUnsafe = errors.New("sshsig: allowed-signers file is unsafe")
	ErrAllowedSignersBad    = errors.New("sshsig: allowed-signers file is malformed")
	ErrSigFormat            = errors.New("sshsig: signature is malformed")
	ErrSigKeyType           = errors.New("sshsig: signing key is not a plain ssh-ed25519 key")
	ErrSigNamespace         = errors.New("sshsig: wrong namespace")
	ErrSigHashAlg           = errors.New("sshsig: hash algorithm is not sha512")
	ErrSigPrincipal         = errors.New("sshsig: allowed-signers lists no ssh-ed25519 key for principal release")
	ErrSigForeignKey        = errors.New("sshsig: signing key is not an allowed release key")
	ErrSigInvalid           = errors.New("sshsig: signature does not verify")
)

// SignatureVerdict is what a verified signature proves.
type SignatureVerdict struct {
	Principal   string // always ReleaseSignaturePrincipal
	Namespace   string // always ReleaseSignatureNamespace
	HashAlg     string // always ReleaseSignatureHashAlg
	Fingerprint string // "SHA256:<unpadded base64>" of the signer's key blob — the `ssh-keygen -l` form
}

// String is the manifest's signature_verdict: "sshsig:release:SHA256:…".
func (v SignatureVerdict) String() string {
	return "sshsig:" + v.Principal + ":" + v.Fingerprint
}

// ReleaseAllowedSignersPath is where an installation keeps the release trust
// anchor: <installDir>/deploy/release_allowed_signers (release.yml verifies
// against the same committed file).
func ReleaseAllowedSignersPath(installDir string) string {
	return filepath.Join(installDir, "deploy", "release_allowed_signers")
}

// VerifySSHSIG verifies an armored SSHSIG signature over message against the
// allowed-signers file at allowedSignersPath, under the policy above. The
// allowed-signers file is read first: absent or unsafe refuses before the
// signature is even parsed.
func VerifySSHSIG(message, armored []byte, allowedSignersPath string) (SignatureVerdict, error) {
	keys, err := releaseSignerKeys(allowedSignersPath)
	if err != nil {
		return SignatureVerdict{}, err
	}
	blob, err := dearmorSSHSIG(armored)
	if err != nil {
		return SignatureVerdict{}, err
	}
	r := &sshReader{b: blob}
	magic, ok := r.raw(len(sshsigMagic))
	if !ok || string(magic) != sshsigMagic {
		return SignatureVerdict{}, fmt.Errorf("%w: magic is not %q", ErrSigFormat, sshsigMagic)
	}
	version, ok := r.u32()
	if !ok || version != sshsigVersion {
		return SignatureVerdict{}, fmt.Errorf("%w: version %d, want %d", ErrSigFormat, version, sshsigVersion)
	}
	pubBlob, ok1 := r.str()
	namespace, ok2 := r.str()
	reserved, ok3 := r.str()
	hashAlg, ok4 := r.str()
	sigBlob, ok5 := r.str()
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return SignatureVerdict{}, fmt.Errorf("%w: truncated signature object", ErrSigFormat)
	}
	if len(r.b) != 0 {
		return SignatureVerdict{}, fmt.Errorf("%w: %d byte(s) of trailing data", ErrSigFormat, len(r.b))
	}
	pub, err := parseEd25519Blob(pubBlob)
	if err != nil {
		return SignatureVerdict{}, err
	}
	if string(namespace) != ReleaseSignatureNamespace {
		return SignatureVerdict{}, fmt.Errorf("%w: %q, want %q", ErrSigNamespace, namespace, ReleaseSignatureNamespace)
	}
	if len(reserved) != 0 {
		return SignatureVerdict{}, fmt.Errorf("%w: reserved field is not empty", ErrSigFormat)
	}
	if string(hashAlg) != ReleaseSignatureHashAlg {
		return SignatureVerdict{}, fmt.Errorf("%w: %q", ErrSigHashAlg, hashAlg)
	}
	sr := &sshReader{b: sigBlob}
	sigType, okT := sr.str()
	rawSig, okS := sr.str()
	if !okT || !okS || len(sr.b) != 0 || string(sigType) != sshEd25519 || len(rawSig) != ed25519.SignatureSize {
		return SignatureVerdict{}, fmt.Errorf("%w: signature is not one ssh-ed25519 signature", ErrSigFormat)
	}
	if len(keys) == 0 {
		return SignatureVerdict{}, fmt.Errorf("%w (%s)", ErrSigPrincipal, allowedSignersPath)
	}
	listed := false
	for _, k := range keys {
		listed = listed || bytes.Equal(k, pubBlob)
	}
	if !listed {
		return SignatureVerdict{}, fmt.Errorf("%w: %s is not listed for principal %q in %s",
			ErrSigForeignKey, keyFingerprint(pubBlob), ReleaseSignaturePrincipal, allowedSignersPath)
	}
	// PROTOCOL.sshsig: the signed data binds OUR namespace and hash algorithm,
	// never the blob's — a check that is skipped still cannot be satisfied by
	// a signature made for another namespace.
	digest := sha512.Sum512(message)
	var signed []byte
	signed = append(signed, sshsigMagic...)
	signed = appendSSHString(signed, []byte(ReleaseSignatureNamespace))
	signed = appendSSHString(signed, nil)
	signed = appendSSHString(signed, []byte(ReleaseSignatureHashAlg))
	signed = appendSSHString(signed, digest[:])
	if !ed25519.Verify(pub, signed, rawSig) {
		return SignatureVerdict{}, ErrSigInvalid
	}
	return SignatureVerdict{
		Principal:   ReleaseSignaturePrincipal,
		Namespace:   ReleaseSignatureNamespace,
		HashAlg:     ReleaseSignatureHashAlg,
		Fingerprint: keyFingerprint(pubBlob),
	}, nil
}

// signersCheckedHook is a TEST SEAM, nil in production: it runs after the
// trust anchor has passed its checks and before its bytes are read — the
// window a swap of the file would use.
var signersCheckedHook func(path string)

// releaseSignerKeys reads the allowed-signers file and returns the key blob
// of every ssh-ed25519 key it lists for the principal "release". The file is
// a trust anchor: it must be a regular file (never a symlink) that neither
// group nor other can write, and it is judged and read as ONE file — opened
// once (O_NOFOLLOW, O_NONBLOCK: a symlink fails, a FIFO cannot block) through
// its directory, which must itself be a real directory (openTrustAnchor: a
// symlinked <install>/deploy is never followed), then every check is made on
// that descriptor, so a swap of the path after the checks changes nothing
// that is read.
//
// Only lines that name "release" in their principal list are parsed; a
// malformed one of those refuses the WHOLE file (a parser that skipped a
// release line it could not read would decide trust on a file nobody wrote).
// Lines for other principals are skipped UNPARSED: whatever they say, they
// admit nothing here. A release line with options (namespaces=,
// valid-before=, cert-authority, …) is refused: those are restrictions
// ssh-keygen enforces and this verifier does not implement, so ignoring them
// would widen trust.
// anchorOpenError maps a failed open of the trust anchor (isDir: of its
// directory) to the refusal: a symlink (ELOOP) or a non-directory (ENOTDIR)
// is UNSAFE; anything else means there is no file to trust (absent ⇒ refused).
func anchorOpenError(path string, isDir bool, err error) error {
	what := "the allowed-signers file"
	if isDir {
		what = "the allowed-signers directory"
	}
	switch {
	case errors.Is(err, syscall.ELOOP):
		return fmt.Errorf("%w: %s %s is a symlink: %w", ErrAllowedSignersUnsafe, what, path, err)
	case isDir && errors.Is(err, syscall.ENOTDIR):
		return fmt.Errorf("%w: %s %s is not a real directory: %w", ErrAllowedSignersUnsafe, what, path, err)
	}
	return fmt.Errorf("%w: %s: %w", ErrNoAllowedSigners, path, err)
}

func releaseSignerKeys(path string) ([][]byte, error) {
	f, err := openTrustAnchor(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrNoAllowedSigners, path, err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%w: %s is not a regular file (%s)", ErrAllowedSignersUnsafe, path, fi.Mode().Type())
	}
	if fi.Mode().Perm()&0o022 != 0 {
		return nil, fmt.Errorf("%w: %s is writable by group or other (%#o)", ErrAllowedSignersUnsafe, path, fi.Mode().Perm())
	}
	// #206 note sshsig.go:245: the mode check proves nothing if the file is
	// owned by ANOTHER uid — that uid can rewrite or chmod it at any time.
	// Accept only this uid or root (a root-owned 0644 anchor is legitimate
	// hardening; a root-owned anchor cannot be rewritten by an unprivileged
	// peer).
	if uid, ok := signersAnchorUID(fi); ok {
		if why := anchorOwnerBad(uid, os.Geteuid()); why != "" {
			return nil, fmt.Errorf("%w: %s: %s", ErrAllowedSignersUnsafe, path, why)
		}
	}
	if signersCheckedHook != nil {
		signersCheckedHook(path)
	}
	raw, err := io.ReadAll(io.LimitReader(f, MaxAllowedSignersBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrAllowedSignersBad, path, err)
	}
	if len(raw) > MaxAllowedSignersBytes {
		return nil, fmt.Errorf("%w: %s is larger than %d bytes", ErrAllowedSignersBad, path, MaxAllowedSignersBytes)
	}
	var keys [][]byte
	for i, line := range strings.Split(string(raw), "\n") {
		// Tokenized so that we never admit a line ssh-keygen refuses (CTO
		// ruling 1790279155144 (1)). Our fields are separated by SPACE and TAB
		// ONLY. ssh-keygen's field delimiter set is " \t\r\n", so ours is
		// deliberately STRICTER on CR (a CR inside a line stays in its field
		// and the line admits nothing — refusing what the tool accepts is the
		// permitted direction). strings.TrimSpace / strings.Fields would also
		// split on VT, FF and Unicode spaces (NBSP, NEL, …), admitting lines
		// the tool refuses; here those characters stay inside their field.
		// LEADING trim is space/tab exactly as the tool skips it: the tool
		// ends the first field at a leading CR (an empty principal) and
		// refuses, so a leading CR must never be trimmed away (verifier f13
		// defect 1). Trailing CR/LF is trimmed (CRLF files).
		line = strings.TrimLeft(strings.TrimRight(line, " \t\r\n"), " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// P0 (review fold): the principal field is extracted with
		// ssh-keygen's strdelimw quote semantics (OpenSSH 9.6p1, misc.c):
		// the token ends at the first whitespace or quote; if that first
		// delimiter is a quote, the OPENING quote is dropped and the token
		// ends at the NEXT quote (no closing quote ⇒ the line is invalid);
		// the quoted content is kept verbatim, commas included. A naive
		// comma split of the raw field trusted `release` segments inside a
		// quoted principal that ssh-keygen parses as one name — accepting
		// what the tool refuses. Deliberate deviation, per the standing
		// ruling: CR/LF are NOT token terminators here (stricter — the tool
		// accepts a CR-separated line, we refuse it).
		principals, rest, ok := sshKeygenPrincipalsToken(line)
		if !ok {
			return nil, fmt.Errorf("%w: %s:%d: unterminated quote in the principals field", ErrAllowedSignersBad, path, i+1)
		}
		fields := strings.FieldsFunc(rest, isAllowedSignersSep)
		forRelease, negated := false, false
		for _, p := range strings.Split(principals, ",") {
			forRelease = forRelease || p == ReleaseSignaturePrincipal
			negated = negated || strings.HasPrefix(p, "!")
		}
		// ssh-keygen lets a matching negation ("release,!rel*") veto the line;
		// patterns are not implemented here, so ANY negation vetoes it.
		if !forRelease || negated {
			continue
		}
		if len(fields) < 2 {
			return nil, fmt.Errorf("%w: %s:%d: a release line needs <principals> <keytype> <base64>", ErrAllowedSignersBad, path, i+1)
		}
		// The keytype is the FIRST token after the principals (the tool parses
		// the key field with sshkey_read; anything else is an option list it
		// skips before re-reading — options this verifier does not implement,
		// so the line refuses). This also closes the P0 second half:
		// `"release",evil ssh-ed25519 …` leaves `,evil` as that first token
		// and the tool refuses ("bad options: unknown key option") — trusting
		// the keytype two fields later would accept what the tool refuses.
		if !looksLikeKeyType(fields[0]) {
			return nil, fmt.Errorf("%w: %s:%d: options on a release line are not supported (%q)", ErrAllowedSignersBad, path, i+1, fields[0])
		}
		if fields[0] != sshEd25519 {
			continue // can never verify under this policy; admits nothing
		}
		blob, err := base64.StdEncoding.DecodeString(fields[1])
		if err != nil {
			return nil, fmt.Errorf("%w: %s:%d: key is not base64: %w", ErrAllowedSignersBad, path, i+1, err)
		}
		if _, err := parseEd25519Blob(blob); err != nil {
			return nil, fmt.Errorf("%w: %s:%d: %w", ErrAllowedSignersBad, path, i+1, err)
		}
		keys = append(keys, blob)
	}
	return keys, nil
}

// isAllowedSignersSep is the ONLY field separator of an allowed-signers line:
// space and tab — never unicode.IsSpace. ssh-keygen (sshsig.c, strdelimw)
// also splits on CR/LF; ours does not, deliberately stricter on CR.
func isAllowedSignersSep(r rune) bool { return r == ' ' || r == '\t' }

// anchorUID returns the file's owner uid when the platform reports one.
func anchorUID(fi os.FileInfo) (uint32, bool) {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return st.Uid, true
	}
	return 0, false
}

// signersAnchorUID is a TEST SEAM, anchorUID in production (chown to another
// uid needs root, so the ownership refusal is proven through this seam).
var signersAnchorUID = anchorUID

// anchorOwnerBad is "" when a root- or self-owned trust anchor is acceptable
// (#206 note sshsig.go:245): any other owner can rewrite the anchor at will,
// so the mode check would prove nothing about who controls the trusted keys.
func anchorOwnerBad(uid uint32, euid int) string {
	if uid == 0 || int(uid) == euid {
		return ""
	}
	return fmt.Sprintf("owned by uid %d, not this process (%d) or root", uid, euid)
}

// sshKeygenPrincipalsToken extracts the principal field of an allowed-signers
// line exactly as OpenSSH 9.6p1 strdelimw does (misc.c): the token ends at the
// first space/tab or quote; when that first delimiter is a quote, the opening
// quote is DROPPED and the token ends at the next quote — the quoted content
// is kept verbatim (commas and spaces included), and everything after the
// closing quote belongs to the following fields. No backslash escapes exist.
// ok=false means an unterminated quote, which the tool rejects as
// "invalid line". Deliberate deviation: CR/LF do NOT terminate the token
// (the tool treats them as whitespace; we stay stricter, per ruling
// 1790279155144 (1)).
func sshKeygenPrincipalsToken(line string) (principals, rest string, ok bool) {
	idx := strings.IndexAny(line, " \t\"")
	if idx < 0 {
		return line, "", true // token runs to end of line; no further fields
	}
	if line[idx] != '"' {
		return line[:idx], strings.TrimLeft(line[idx:], " \t"), true
	}
	// The tool memmoves the opening quote away and terminates the token at
	// the next quote; an unterminated quote makes strdelimw return NULL.
	// Everything before the opening quote is part of the token (OpenSSH's
	// strpbrk finds the quote but the token runs from the line start).
	body := line[idx+1:]
	closeIdx := strings.IndexByte(body, '"')
	if closeIdx < 0 {
		return "", "", false
	}
	principals = line[:idx] + body[:closeIdx]
	rest = strings.TrimLeft(body[closeIdx+1:], " \t")
	return principals, rest, true
}

// looksLikeKeyType reports whether an allowed-signers field is a key type
// rather than an option list (OpenSSH key type names).
func looksLikeKeyType(s string) bool {
	for _, p := range []string{"ssh-", "ecdsa-", "sk-", "rsa-", "x509v3-"} {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// parseEd25519Blob returns the 32-byte key of a plain ssh-ed25519 public key
// blob: string("ssh-ed25519") ‖ string(key), nothing after.
func parseEd25519Blob(blob []byte) (ed25519.PublicKey, error) {
	r := &sshReader{b: blob}
	kt, ok1 := r.str()
	k, ok2 := r.str()
	if !ok1 || !ok2 || len(r.b) != 0 {
		return nil, fmt.Errorf("%w: public key blob is malformed", ErrSigFormat)
	}
	if string(kt) != sshEd25519 {
		return nil, fmt.Errorf("%w: %q", ErrSigKeyType, kt)
	}
	if len(k) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("%w: ed25519 key is %d bytes", ErrSigFormat, len(k))
	}
	return ed25519.PublicKey(k), nil
}

// keyFingerprint is OpenSSH's SHA256 fingerprint of a key blob.
func keyFingerprint(blob []byte) string {
	sum := sha256.Sum256(blob)
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(sum[:])
}

// dearmorSSHSIG strips the armor ssh-keygen -Y sign writes: the BEGIN line
// first, base64 lines, the END line, then nothing but whitespace.
func dearmorSSHSIG(armored []byte) ([]byte, error) {
	if len(armored) > MaxSignatureBytes {
		return nil, fmt.Errorf("%w: larger than %d bytes", ErrSigFormat, MaxSignatureBytes)
	}
	s := string(armored)
	if !strings.HasPrefix(s, armorBegin+"\n") {
		return nil, fmt.Errorf("%w: does not begin with %q", ErrSigFormat, armorBegin)
	}
	s = s[len(armorBegin)+1:]
	// The END marker must begin a LINE (#206 review fold): ssh-keygen's
	// dearmor looks for "\n-----END SSH SIGNATURE-----" and refuses
	// 'missing footer' when the marker is glued to the last base64
	// characters. Searching for the bare marker accepted that shape here —
	// looser than the tool on byte-identical decoded data.
	needle := "\n" + armorEnd
	end := strings.Index(s, needle)
	if end < 0 {
		return nil, fmt.Errorf("%w: no %q line", ErrSigFormat, armorEnd)
	}
	marker := end + 1 // the "\n" belongs to the body, not the marker
	if strings.TrimSpace(s[marker+len(armorEnd):]) != "" {
		return nil, fmt.Errorf("%w: text after %q", ErrSigFormat, armorEnd)
	}
	body := strings.ReplaceAll(s[:end], "\n", "")
	blob, err := base64.StdEncoding.DecodeString(body)
	if err != nil {
		return nil, fmt.Errorf("%w: armor body: %w", ErrSigFormat, err)
	}
	return blob, nil
}

// sshReader reads the SSH wire encoding (RFC 4251 §5): uint32 big-endian,
// and string = uint32 length ‖ bytes.
type sshReader struct{ b []byte }

func (r *sshReader) raw(n int) ([]byte, bool) {
	if n < 0 || len(r.b) < n {
		return nil, false
	}
	v := r.b[:n]
	r.b = r.b[n:]
	return v, true
}

func (r *sshReader) u32() (uint32, bool) {
	v, ok := r.raw(4)
	if !ok {
		return 0, false
	}
	return binary.BigEndian.Uint32(v), true
}

func (r *sshReader) str() ([]byte, bool) {
	n, ok := r.u32()
	if !ok || uint64(n) > uint64(len(r.b)) {
		return nil, false
	}
	return r.raw(int(n))
}

func appendSSHString(dst, v []byte) []byte {
	dst = binary.BigEndian.AppendUint32(dst, uint32(len(v)))
	return append(dst, v...)
}
