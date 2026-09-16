package researchsnapshot

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

type Archive struct {
	db       *sql.DB
	revision *string
}

func Open(path, revision string) (*Archive, error) {
	// A relative URL path serializes as file://data/... (an authority), not
	// a local file. Resolve filesystem paths before constructing the URI.
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: path}
	db, err := sql.Open("sqlite", u.String()+"?_pragma=busy_timeout(250)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	a := &Archive{db: db}
	if revision != "" {
		a.revision = Value(revision)
	}
	var version int
	if err = db.QueryRow("PRAGMA user_version").Scan(&version); err != nil || (version != 0 && version != SchemaVersion) {
		db.Close()
		return nil, fmt.Errorf("research schema unsupported: version=%d error=%v", version, err)
	}
	if _, err = db.Exec(`PRAGMA journal_mode=WAL;
CREATE TABLE IF NOT EXISTS research_facts (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 writer_revision TEXT, schema_version INTEGER NOT NULL,
 object TEXT NOT NULL, snapshot_id TEXT, event TEXT,
 observation_ms INTEGER, receipt_ms INTEGER, publication_ms INTEGER, permission_ms INTEGER,
 captured_ms INTEGER NOT NULL, fields_json TEXT, missing_json TEXT, null_fields INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS research_receipt ON research_facts(receipt_ms,id);
CREATE INDEX IF NOT EXISTS research_snapshot ON research_facts(snapshot_id,id);
CREATE INDEX IF NOT EXISTS research_captured ON research_facts(captured_ms,object);
` + fmt.Sprintf("PRAGMA user_version=%d;", SchemaVersion)); err != nil {
		db.Close()
		return nil, err
	}
	return a, nil
}

// OpenReadOnly does not run schema creation, migrations, or writer PRAGMAs.
func OpenReadOnly(path string) (*Archive, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: path}
	db, err := sql.Open("sqlite", u.String()+"?mode=ro")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return &Archive{db: db}, nil
}

func (a *Archive) Close() error { return a.db.Close() }

func (a *Archive) Save(ctx context.Context, facts []Fact) error {
	return a.saveAt(ctx, facts, time.Now())
}

func (a *Archive) saveAt(ctx context.Context, facts []Fact, now time.Time) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("research archive panic (%T)", p)
		}
	}()
	if len(facts) == 0 {
		return nil
	}
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO research_facts
 (writer_revision,schema_version,object,snapshot_id,event,observation_ms,receipt_ms,publication_ms,permission_ms,captured_ms,fields_json,missing_json,null_fields)
 VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, f := range facts {
		if err = f.validate(); err != nil {
			return err
		}
		payload, e := json.Marshal(f.Fields)
		if e != nil {
			return e
		}
		missing, e := json.Marshal(f.Missing)
		if e != nil {
			return e
		}
		nullCount := 0
		for _, v := range f.Fields {
			if v == nil {
				nullCount++
			}
		}
		for _, v := range []*int64{f.Clocks.ObservationMS, f.Clocks.ReceiptMS, f.Clocks.PublicationMS, f.Clocks.PermissionMS} {
			if v == nil {
				nullCount++
			}
		}
		_, err = stmt.ExecContext(ctx, a.revision, SchemaVersion, f.Object, f.SnapshotID, f.Event,
			f.Clocks.ObservationMS, f.Clocks.ReceiptMS, f.Clocks.PublicationMS, f.Clocks.PermissionMS,
			now.UnixMilli(), string(payload), string(missing), nullCount)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

type StoredFact struct {
	ID             int64              `json:"id"`
	WriterRevision *string            `json:"writer_revision"`
	SchemaVersion  int                `json:"schema_version"`
	Object         string             `json:"object"`
	SnapshotID     *string            `json:"snapshot_id"`
	Event          *string            `json:"event"`
	Clocks         Clocks             `json:"clocks"`
	ClockCT        map[string]*string `json:"clock_ct"`
	CapturedMS     int64              `json:"captured_ms"`
	Fields         json.RawMessage    `json:"fields"`
	Missing        json.RawMessage    `json:"missing"`
	NullFields     int64              `json:"null_fields"`
}

type Manifest struct {
	SchemaVersion          int            `json:"schema_version"`
	FromMS                 int64          `json:"from_ms"`
	ToMS                   int64          `json:"to_ms"`
	RangeBasis             string         `json:"range_basis"`
	Counts                 map[string]int `json:"counts"`
	WriterRevisions        []*string      `json:"writer_revisions"`
	UnknownReceiptExcluded int64          `json:"unknown_receipt_excluded"`
	Checksum               string         `json:"objects_sha256"`
}

type Bundle struct {
	Manifest Manifest                `json:"manifest"`
	Objects  map[string][]StoredFact `json:"objects"`
}

// Export uses receipt time for range membership, so late backfill cannot enter
// an earlier decision's available evidence. Null receipt clocks are counted as
// excluded across the archive, never silently replaced with observation time.
func (a *Archive) Export(ctx context.Context, fromMS, toMS int64) ([]byte, error) {
	if toMS < fromMS {
		return nil, errors.New("end precedes start")
	}
	tx, err := a.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	b := Bundle{Manifest: Manifest{SchemaVersion: SchemaVersion, FromMS: fromMS, ToMS: toMS, RangeBasis: "receipt_ms [from,to)", Counts: map[string]int{}, WriterRevisions: []*string{}}, Objects: map[string][]StoredFact{}}
	for _, kind := range Objects {
		b.Objects[kind] = []StoredFact{}
		b.Manifest.Counts[kind] = 0
	}
	if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM research_facts WHERE receipt_ms IS NULL").Scan(&b.Manifest.UnknownReceiptExcluded); err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,writer_revision,schema_version,object,snapshot_id,event,
 observation_ms,receipt_ms,publication_ms,permission_ms,captured_ms,fields_json,missing_json,null_fields
 FROM research_facts WHERE receipt_ms>=? AND receipt_ms<? ORDER BY id`, fromMS, toMS)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		rows.Close()
		return nil, err
	}
	revisions := map[string]bool{}
	unknownRevision := false
	for rows.Next() {
		var r StoredFact
		var payload, missing string
		if err = rows.Scan(&r.ID, &r.WriterRevision, &r.SchemaVersion, &r.Object, &r.SnapshotID, &r.Event,
			&r.Clocks.ObservationMS, &r.Clocks.ReceiptMS, &r.Clocks.PublicationMS, &r.Clocks.PermissionMS,
			&r.CapturedMS, &payload, &missing, &r.NullFields); err != nil {
			rows.Close()
			return nil, err
		}
		r.Fields = json.RawMessage(payload)
		r.Missing = json.RawMessage(missing)
		r.ClockCT = map[string]*string{}
		for k, v := range map[string]*int64{"observation": r.Clocks.ObservationMS, "receipt": r.Clocks.ReceiptMS, "publication": r.Clocks.PublicationMS, "permission": r.Clocks.PermissionMS} {
			r.ClockCT[k] = nil
			if v != nil {
				r.ClockCT[k] = Value(time.UnixMilli(*v).In(loc).Format(time.RFC3339Nano))
			}
		}
		if _, ok := b.Objects[r.Object]; !ok {
			rows.Close()
			return nil, fmt.Errorf("unknown stored research object %q", r.Object)
		}
		b.Objects[r.Object] = append(b.Objects[r.Object], r)
		b.Manifest.Counts[r.Object]++
		if r.WriterRevision == nil {
			unknownRevision = true
		} else {
			revisions[*r.WriterRevision] = true
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	var keys []string
	for k := range revisions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if unknownRevision {
		b.Manifest.WriterRevisions = append(b.Manifest.WriterRevisions, nil)
	}
	for _, k := range keys {
		b.Manifest.WriterRevisions = append(b.Manifest.WriterRevisions, Value(k))
	}
	objects, err := json.Marshal(b.Objects)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256(objects)
	b.Manifest.Checksum = hex.EncodeToString(h[:])
	out, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func VerifyBundle(data []byte) error {
	var b Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	for _, kind := range Objects {
		rows, ok := b.Objects[kind]
		if !ok || b.Manifest.Counts[kind] != len(rows) {
			return fmt.Errorf("object count mismatch: %s", kind)
		}
	}
	objects, err := json.Marshal(b.Objects)
	if err != nil {
		return err
	}
	h := sha256.Sum256(objects)
	if b.Manifest.Checksum != hex.EncodeToString(h[:]) {
		return errors.New("research checksum mismatch")
	}
	return nil
}
