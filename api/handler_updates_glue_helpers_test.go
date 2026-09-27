package api

// Shared fixtures for the U5b glue tests: a job file written by updaterjob's
// PRODUCTION writer (updaterjob.Write), and a REAL worker socket
// (wireserver.Listen/Serve) in the test's temp data dir. A _test.go import of
// the worker side is not production linkage (the trading-app import guard
// counts non-test files only; TestWorkerImportGuardCatchesDirectAndTransitiveImports
// pins that exemption).

import (
	"sync"
	"testing"
	"time"

	"nofx/internal/updaterjob"
	"nofx/internal/updaterwire"
	"nofx/internal/updaterwire/wireserver"
)

// writeTestJob writes a job the way the worker's install verb does
// (updaterjob.New → Write), then walks it one step (downloaded: started →
// receipt → done) so the file carries a receipt and a done timestamp.
func writeTestJob(t *testing.T, dataDir, jobID, releaseID string) updaterjob.Job {
	t.Helper()
	now := time.Date(2026, 9, 24, 18, 2, 11, 0, time.UTC)
	j, err := updaterjob.New(jobID, releaseID, now)
	if err != nil {
		t.Fatalf("updaterjob.New: %v", err)
	}
	if err := updaterjob.Write(dataDir, j); err != nil {
		t.Fatalf("updaterjob.Write (requested): %v", err)
	}
	if err := j.Enter(updaterjob.StateDownloaded, now.Add(time.Second)); err != nil {
		t.Fatalf("Enter downloaded: %v", err)
	}
	if err := updaterjob.Write(dataDir, j); err != nil {
		t.Fatalf("updaterjob.Write (downloaded/started): %v", err)
	}
	r := updaterjob.Receipt{Step: "download", StartedAt: now.Add(time.Second), EndedAt: now.Add(2 * time.Second), OK: true, Evidence: map[string]string{"artifacts": "7"}}
	if err := j.AddReceipt(r, now.Add(2*time.Second)); err != nil {
		t.Fatalf("AddReceipt: %v", err)
	}
	if err := j.Finish(now.Add(2 * time.Second)); err != nil {
		t.Fatalf("Finish: %v", err)
	}
	if err := updaterjob.Write(dataDir, j); err != nil {
		t.Fatalf("updaterjob.Write (downloaded/done): %v", err)
	}
	got, err := updaterjob.Read(dataDir, jobID)
	if err != nil {
		t.Fatalf("updaterjob.Read after the production writer: %v", err)
	}
	return got
}

// testWorker is a real wireserver on <dataDir>/updater/worker.sock whose
// install verb answers what onInstall says; it records every install frame.
type testWorker struct {
	mu       sync.Mutex
	installs []updaterwire.InstallPayload
	other    []updaterwire.Verb
}

func (w *testWorker) calls() ([]updaterwire.InstallPayload, []updaterwire.Verb) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return append([]updaterwire.InstallPayload(nil), w.installs...), append([]updaterwire.Verb(nil), w.other...)
}

// startTestWorker listens with the production listener. onInstall returns
// (ok, state-or-error).
func startTestWorker(t *testing.T, dataDir string, onInstall func(releaseID, jobID string) (bool, string)) *testWorker {
	t.Helper()
	path, err := updaterwire.SocketPath(dataDir)
	if err != nil {
		t.Fatalf("SocketPath: %v", err)
	}
	ln, err := wireserver.Listen(path, t.Logf)
	if err != nil {
		t.Fatalf("wireserver.Listen(%s): %v", path, err)
	}
	w := &testWorker{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = ln.Serve(func(req updaterwire.Request) updaterwire.Response {
			w.mu.Lock()
			defer w.mu.Unlock()
			if req.Verb != updaterwire.VerbInstall || req.Install == nil {
				w.other = append(w.other, req.Verb)
				return updaterwire.Response{OK: false, Error: "unexpected verb"}
			}
			w.installs = append(w.installs, *req.Install)
			ok, s := onInstall(req.Install.ReleaseID, req.Install.JobID)
			if ok {
				return updaterwire.Response{OK: true, State: s}
			}
			return updaterwire.Response{OK: false, Error: s}
		})
	}()
	t.Cleanup(func() { _ = ln.Close(); <-done })
	return w
}
