package api

// W-ONE-BUTTON M4 3b-B U5b (d) — the job/receipt routes read the WORKER's
// job file, at the PRODUCTION router with the knob ON (canon 53). The file is
// written by updaterjob's production writer (updaterjob.Write, via
// writeTestJob). Knob OFF: M3's literal 404 before any filesystem access.

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"nofx/internal/updaterjob"
)

const testJobID = "0123456789abcdef"

func jsonMap(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("not a JSON object: %s (%v)", b, err)
	}
	return m
}

func TestJobRoutesReadTheWorkersFile(t *testing.T) {
	t.Setenv(updaterKnobEnv, "1")
	logs := captureLogs(t)
	e := newUpdEnv(t)
	j := writeTestJob(t, e.dataDir, testJobID, updRelease)

	// the job route serves updaterjob.View of the file, key for key
	w := e.do("GET", "/api/updates/jobs/"+testJobID, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET job = %d %s, want 200", w.Code, w.Body.String())
	}
	want, _ := json.Marshal(updaterjob.View(j))
	if got := jsonMap(t, w.Body.Bytes()); !reflect.DeepEqual(got, jsonMap(t, want)) {
		t.Fatalf("GET job = %s\nwant updaterjob.View = %s", w.Body.String(), want)
	}
	v := jsonMap(t, w.Body.Bytes())
	if v["job_id"] != testJobID || v["state"] != "downloaded" || v["receipt_url"] != "/api/updates/jobs/"+testJobID+"/receipt" {
		t.Fatalf("GET job = %s: want job_id, state downloaded and the receipt url", w.Body.String())
	}
	if ts, ok := v["timestamps"].(map[string]any); !ok || ts["requested"] == nil || ts["downloaded"] == nil {
		t.Fatalf("GET job timestamps = %v, want requested + downloaded", v["timestamps"])
	}

	// the receipt route serves updaterjob.Receipts
	w = e.do("GET", "/api/updates/jobs/"+testJobID+"/receipt", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET receipt = %d %s, want 200", w.Code, w.Body.String())
	}
	want, _ = json.Marshal(updaterjob.Receipts(j))
	if got := jsonMap(t, w.Body.Bytes()); !reflect.DeepEqual(got, jsonMap(t, want)) {
		t.Fatalf("GET receipt = %s\nwant updaterjob.Receipts = %s", w.Body.String(), want)
	}
	if rs, _ := jsonMap(t, w.Body.Bytes())["receipts"].([]any); len(rs) != 1 {
		t.Fatalf("GET receipt: %d receipts, want the one the worker wrote", len(rs))
	}

	// a valid id with no file ⇒ M3's 404 body, no ERROR (absence is not a fault)
	mark := len(logs())
	for _, p := range []string{"/api/updates/jobs/fedcba9876543210", "/api/updates/jobs/fedcba9876543210/receipt"} {
		if w := e.do("GET", p, ""); w.Code != http.StatusNotFound || w.Body.String() != m3NotFoundBody {
			t.Fatalf("%s (no such job) = %d %s, want 404 %s", p, w.Code, w.Body.String(), m3NotFoundBody)
		}
	}
	// an id the wire refuses ⇒ 404 before any read, no ERROR
	for _, id := range []string{"secret", "ABCDEF0123456789", "short", "-0123456789abcdef", strings.Repeat("a", 65)} {
		if w := e.do("GET", "/api/updates/jobs/"+id, ""); w.Code != http.StatusNotFound || w.Body.String() != m3NotFoundBody {
			t.Fatalf("GET job %q = %d %s, want 404", id, w.Code, w.Body.String())
		}
	}
	if tail := logs()[mark:]; strings.Contains(tail, "ERRO") {
		t.Fatalf("a missing or refused id logged an ERROR:\n%s", tail)
	}

	// a corrupt job file ⇒ the same 404 (no filesystem oracle) + an ERROR
	const corruptID = "c0ffee0000000001"
	const sentinel = "CORRUPT-JOB-SENTINEL-77"
	jobs, _ := updaterjob.JobsDir(e.dataDir)
	if err := os.WriteFile(filepath.Join(jobs, corruptID+".json"), []byte(`{"job_id":"`+sentinel+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	mark = len(logs())
	for _, p := range []string{"/api/updates/jobs/" + corruptID, "/api/updates/jobs/" + corruptID + "/receipt"} {
		w := e.do("GET", p, "")
		if w.Code != http.StatusNotFound || w.Body.String() != m3NotFoundBody || strings.Contains(w.Body.String(), sentinel) {
			t.Fatalf("%s (corrupt file) = %d %s, want exactly 404 %s", p, w.Code, w.Body.String(), m3NotFoundBody)
		}
	}
	if tail := logs()[mark:]; strings.Count(tail, "ERRO") < 2 || !strings.Contains(tail, corruptID) {
		t.Fatalf("a corrupt job file must log an ERROR per read naming the job:\n%s", tail)
	}

	// a loose jobs dir ⇒ 404 + ERROR, even for the good job
	if err := os.Chmod(jobs, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(jobs, 0o700) })
	mark = len(logs())
	if w := e.do("GET", "/api/updates/jobs/"+testJobID, ""); w.Code != http.StatusNotFound || w.Body.String() != m3NotFoundBody {
		t.Fatalf("loose jobs dir: GET job = %d %s, want 404", w.Code, w.Body.String())
	}
	if tail := logs()[mark:]; !strings.Contains(tail, "ERRO") {
		t.Fatalf("a loose jobs dir must log an ERROR:\n%s", tail)
	}
}

// OFF: the literal 404 comes before any filesystem access — a loose jobs dir
// (which ON logs as an ERROR) and a good job file are both invisible.
func TestJobRoutesKnobOffNeverTouchTheFilesystem(t *testing.T) {
	t.Setenv(updaterKnobEnv, "")
	logs := captureLogs(t)
	e := newUpdEnv(t)
	writeTestJob(t, e.dataDir, testJobID, updRelease)
	jobs, _ := updaterjob.JobsDir(e.dataDir)
	if err := os.Chmod(jobs, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(jobs, 0o700) })
	mark := len(logs())
	for _, p := range []string{"/api/updates/jobs/" + testJobID, "/api/updates/jobs/" + testJobID + "/receipt"} {
		if w := e.do("GET", p, ""); w.Code != http.StatusNotFound || w.Body.String() != m3NotFoundBody {
			t.Fatalf("knob OFF %s = %d %s, want M3's 404", p, w.Code, w.Body.String())
		}
	}
	if tail := logs()[mark:]; strings.Contains(tail, "ERRO") {
		t.Fatalf("knob OFF read the jobs dir (an ERROR was logged):\n%s", tail)
	}
}

// TestJobRoutesNeverServeAFile's probe, knob ON: no id — path-shaped,
// encoded, oversized, or naming an enrollment file — reaches a file, and a
// job-id-shaped file that is not a job is never echoed.
func TestJobRoutesNeverServeAFileWithTheKnobOn(t *testing.T) {
	t.Setenv(updaterKnobEnv, "1")
	e := newUpdEnv(t)
	writeTestJob(t, e.dataDir, testJobID, updRelease)
	const secret = "TOP-SECRET-SENTINEL-9f1c"
	jobs, _ := updaterjob.JobsDir(e.dataDir)
	for _, p := range []string{
		filepath.Join(e.dataDir, "updater", "secret.json"), filepath.Join(e.dataDir, "secret.json"),
		filepath.Join(jobs, "secret.json"), filepath.Join(jobs, "deadbeefdeadbeef.json"),
	} {
		if err := os.WriteFile(p, []byte(`{"s":"`+secret+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, p := range []string{
		"/api/updates/jobs/secret", "/api/updates/jobs/secret.json", "/api/updates/jobs/..%2fsecret.json",
		"/api/updates/jobs/..%2f..%2fsecret.json", "/api/updates/jobs/%2e%2e", "/api/updates/jobs/" + strings.Repeat("a", 300),
		"/api/updates/jobs/..%2fsecret.json/receipt", "/api/updates/jobs/device.key", "/api/updates/jobs/admin.json/receipt",
		"/api/updates/jobs/deadbeefdeadbeef", "/api/updates/jobs/deadbeefdeadbeef/receipt",
		"/api/updates/jobs/" + testJobID + "%2f..%2f..%2fsecret.json", "/api/updates/jobs/" + testJobID + ".json",
	} {
		w := e.do("GET", p, "")
		if w.Code == http.StatusOK || strings.Contains(w.Body.String(), secret) || strings.Contains(w.Body.String(), updAdminEmail) {
			t.Errorf("%s: %d %s", p, w.Code, w.Body.String())
		}
	}
	// positive control: the real job IS served
	if w := e.do("GET", "/api/updates/jobs/"+testJobID, ""); w.Code != http.StatusOK {
		t.Fatalf("positive control: GET the worker's job = %d %s", w.Code, w.Body.String())
	}
}
