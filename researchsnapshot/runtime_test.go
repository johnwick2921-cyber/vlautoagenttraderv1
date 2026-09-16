package researchsnapshot

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Exercise the same Start -> installed recorder -> CurrentBootLine path as main.
// Absolute TempDir paths alone missed the deployed default data/data.db path.
func TestResearchStartupRelativePathReportsWorkingSchema(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, path := range []string{"data/data.db.research.db", "data/space ?#%/archive.db"} {
		t.Run(path, func(t *testing.T) {
			Install(nil)
			closeRecorder := Start(path, nil)
			defer closeRecorder()
			now := time.Date(2026, 9, 8, 23, 30, 0, 0, time.UTC)
			line := CurrentBootLineAt(now)
			if Active() == nil || !strings.Contains(line, "schema="+strconv.Itoa(SchemaVersion)+" ·") {
				t.Fatalf("working recorder required at relative path %q; boot line: %s", path, line)
			}
			archive, ok := Active().sink.(*Archive)
			if !ok {
				t.Fatal("startup did not install the real SQLite archive")
			}
			fact := NewFact("candidate", "startup-pin", Value("startup-read"), Clocks{ObservationMS: Value(now.UnixMilli()), ReceiptMS: Value(now.UnixMilli())})
			fact.Set("final_score", 0.0)
			// Fixed captured clock makes the boot's today count deterministic.
			if err := archive.saveAt(context.Background(), []Fact{fact}, now); err != nil {
				t.Fatal(err)
			}
			line = CurrentBootLineAt(now)
			if strings.Contains(line, "schema=UNKNOWN") || !strings.Contains(line, "candidate=1") {
				t.Fatalf("boot must read schema and row counts from the working archive: %s", line)
			}
			t.Logf("working startup: %s", line)
			info, err := os.Stat(path)
			if err != nil || info.Size() == 0 {
				t.Fatalf("archive was not created at the requested filesystem path: %v", err)
			}
			reader, err := OpenReadOnly(path)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			raw, err := reader.Export(context.Background(), now.UnixMilli(), now.UnixMilli()+1)
			if err != nil {
				t.Fatal(err)
			}
			if err := VerifyBundle(raw); err != nil {
				t.Fatal(err)
			}
			var bundle Bundle
			if err := json.Unmarshal(raw, &bundle); err != nil {
				t.Fatal(err)
			}
			if bundle.Manifest.Counts["candidate"] != 1 {
				t.Fatal("writer and read-only opener resolved different archives")
			}
		})
	}
}

func TestResearchReadOnlyRelativePathReadsExistingArchive(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	path := filepath.Join("data", "read ?#%.db")
	archive, err := Open(filepath.Join(root, path), "path-pin")
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	f := NewFact("market", "path-pin", nil, Clocks{ReceiptMS: Value(int64(1))})
	if err = archive.Save(context.Background(), []Fact{f}); err != nil {
		t.Fatal(err)
	}
	reader, err := OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	raw, err := reader.Export(context.Background(), 0, 2)
	if err != nil {
		t.Fatalf("read-only relative path failed: %v", err)
	}
	var bundle Bundle
	if err = json.Unmarshal(raw, &bundle); err != nil {
		t.Fatal(err)
	}
	if bundle.Manifest.Counts["market"] != 1 {
		t.Fatal("existing archive was not read")
	}
}

func TestResearchFailedStartupStillWarnsAndReportsUnknown(t *testing.T) {
	t.Chdir(t.TempDir())
	Install(nil)
	if err := os.WriteFile("blocked", []byte("regular file"), 0600); err != nil {
		t.Fatal(err)
	}
	var warnings []string
	closeRecorder := Start(filepath.Join("blocked", "archive.db"), func(line string) { warnings = append(warnings, line) })
	defer closeRecorder()
	now := time.Date(2026, 9, 8, 23, 30, 0, 0, time.UTC)
	line := CurrentBootLineAt(now)
	if Active() != nil || !strings.Contains(line, "schema=UNKNOWN") {
		t.Fatalf("failed startup must not fabricate working schema: %s", line)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "WARN research snapshot archive unavailable") {
		t.Fatalf("failed startup must warn: %v", warnings)
	}
}
