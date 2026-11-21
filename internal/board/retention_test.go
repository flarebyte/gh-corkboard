package board

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func archivedFixture(t *testing.T, names ...string) (string, string, PrunePlan) {
	t.Helper()
	dir, scope := reconcileFixture(t)
	if err := WithDocuments(dir, owner, true, func(d *Documents) error {
		status := "archived"
		for _, name := range names {
			if _, err := d.Create(CreateOptions{Scope: reconcileScope, Name: name, Format: "md", Status: &status}, []byte(name)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var plan PrunePlan
	if err := WithAccess(dir, owner, false, func(a *Access) error {
		var err error
		plan, err = a.PreviewPrune()
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(plan.Candidates) != len(names) {
		t.Fatalf("preview: %+v", plan)
	}
	return dir, scope, plan
}

func currentAudit(t *testing.T, dir string) []IndexFinding {
	t.Helper()
	var findings []IndexFinding
	if err := WithAccess(dir, owner, false, func(a *Access) error {
		var err error
		findings, err = a.AuditIndex()
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return findings
}

func TestPruneRechecksBeforeUnlinkAndReconcilesPartialResult(t *testing.T) {
	dir, folder, plan := archivedFixture(t, "first", "second", "third")
	calls := 0
	var got PruneResult
	err := WithAccess(dir, owner, true, func(a *Access) error {
		var err error
		got, err = a.applyPrune(plan.ID, func(scope *os.Root, name string) error {
			calls++
			if calls == 1 {
				changed := filepath.Join(folder, filepath.Base(plan.Candidates[1].Path))
				if err := os.WriteFile(changed, []byte("changed\n"), 0600); err != nil {
					return err
				}
			}
			return removeDocumentFile(scope, name)
		}, (*Documents).publishCatalog)
		return err
	})
	var partial *PartialPruneError
	wantRemaining := []string{plan.Candidates[1].ID, plan.Candidates[2].ID}
	if !errors.As(err, &partial) || !strings.Contains(err.Error(), "document became changed") || calls != 1 ||
		!slices.Equal(got.Deleted, []string{plan.Candidates[0].ID}) || !slices.Equal(got.Remaining, wantRemaining) {
		t.Fatalf("partial prune: %+v, %v", got, err)
	}
	findings := currentAudit(t, dir)
	if len(findings) != 1 || findings[0].Code != "digest_mismatch" || findings[0].Path != plan.Candidates[1].Path {
		t.Fatalf("partial deletion did not reconcile: %+v", findings)
	}
}

func TestPruneReportsUnlinkAndIndexFailures(t *testing.T) {
	dir, _, plan := archivedFixture(t, "first", "second")
	unlinkFailure := errors.New("simulated unlink failure")
	publishFailure := errors.New("simulated index failure")
	calls := 0
	err := WithAccess(dir, owner, true, func(a *Access) error {
		_, err := a.applyPrune(plan.ID, func(root *os.Root, name string) error {
			calls++
			if calls == 2 {
				return unlinkFailure
			}
			return removeDocumentFile(root, name)
		}, func(*Documents, catalog, []byte) error { return publishFailure })
		return err
	})
	if !errors.Is(err, unlinkFailure) || !errors.Is(err, publishFailure) || !strings.Contains(err.Error(), "index audit") {
		t.Fatalf("lost partial failure: %v", err)
	}
	findings := currentAudit(t, dir)
	if len(findings) != 1 || findings[0].Code != "missing_document" || findings[0].Path != plan.Candidates[0].Path {
		t.Fatalf("missing audit evidence: %+v", findings)
	}
}

func TestSnapshotWithDriftAndInvalidIndex(t *testing.T) {
	dir, folder, plan := archivedFixture(t, "draft")
	index := filepath.Join(dir, boardName, "index")
	putFile(t, index, "documents.json", "malformed index")
	putFile(t, folder, "unindexed.json", "{broken")
	putFile(t, folder, temporaryPrefix+"unused", "temporary")
	var zipBytes bytes.Buffer
	if err := WithAccess(dir, owner, false, func(a *Access) error {
		if err := a.Snapshot(&zipBytes, &plan.ID); err == nil || zipBytes.Len() != 0 {
			t.Fatalf("invalid index passed guard: %v", err)
		}
		return a.Snapshot(&zipBytes, nil)
	}); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(zipBytes.Bytes()), int64(zipBytes.Len()))
	if err != nil {
		t.Fatal(err)
	}
	paths := map[string]string{}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		paths[file.Name] = string(data)
	}
	for _, prefix := range append([]string{""}, snapshotFolders...) {
		p := boardName + "/"
		if prefix != "" {
			p += prefix + "/"
		}
		if _, ok := paths[p]; !ok {
			t.Fatalf("missing directory %q", p)
		}
	}
	if paths[boardName+"/index/access.json"] == "" || paths[boardName+"/index/documents.json"] != "malformed index" ||
		paths[boardName+"/topics/specs/unindexed.json"] != "{broken" {
		t.Fatalf("snapshot contents: %+v", paths)
	}
	for path := range paths {
		if strings.Contains(path, "board.lock") || strings.Contains(path, temporaryPrefix) {
			t.Fatalf("bookkeeping in ZIP: %s", path)
		}
	}
}

type faultWriter struct{ err error }

func (w faultWriter) Write([]byte) (int, error) { return 0, w.err }

func TestSnapshotOutputFailure(t *testing.T) {
	dir, _, _ := archivedFixture(t, "draft")
	failure := errors.New("output unavailable")
	err := WithAccess(dir, owner, false, func(a *Access) error { return a.Snapshot(faultWriter{failure}, nil) })
	if !errors.Is(err, failure) {
		t.Fatalf("snapshot lost output error: %v", err)
	}
}

func TestSnapshotRejectsUnsafeEntry(t *testing.T) {
	dir, folder := reconcileFixture(t)
	putFile(t, dir, "outside.md", "private")
	if err := os.Symlink(filepath.Join(dir, "outside.md"), filepath.Join(folder, "link.md")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	err := WithAccess(dir, owner, false, func(a *Access) error { return a.Snapshot(io.Discard, nil) })
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("snapshot accepted symlink: %v", err)
	}
}
