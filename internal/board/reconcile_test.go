package board

import (
	"bytes"
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

const reconcileScope = "topic:specs"
const reconcileID = "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb"

func reconcileFixture(t *testing.T) (string, string) {
	t.Helper()
	dir := accessFixture(t)
	editAccess(t, dir, func(a *Access) error { return a.CreateTopic("specs") })
	return dir, filepath.Join(dir, boardName, "topics", "specs")
}

func putFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func runResync(t *testing.T, dir string) ResyncResult {
	t.Helper()
	var result ResyncResult
	editAccess(t, dir, func(a *Access) error {
		var err error
		result, err = a.ResyncIndex()
		return err
	})
	return result
}

func rejectResync(t *testing.T, dir string, contains ...string) {
	t.Helper()
	err := WithAccess(dir, owner, true, func(a *Access) error { _, err := a.ResyncIndex(); return err })
	if err == nil {
		t.Fatal("resync unexpectedly succeeded")
	}
	for _, text := range contains {
		if !strings.Contains(err.Error(), text) {
			t.Fatalf("resync error %q lacks %q", err, text)
		}
	}
}

func TestResyncReusesRenamedUUIDAfterPublicationFailure(t *testing.T) {
	dir, folder := reconcileFixture(t)
	putFile(t, folder, "local.spec.csv", "id,priority\nb,01")
	originalIndex, err := os.ReadFile(filepath.Join(dir, boardName, filepath.FromSlash(catalogPath)))
	if err != nil {
		t.Fatal(err)
	}
	failure := errors.New("simulated index write failure")
	err = WithAccess(dir, owner, true, func(a *Access) error {
		_, err := a.resyncIndex(func(*Documents, catalog, []byte) error { return failure })
		return err
	})
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "renamed 1") {
		t.Fatalf("partial result: %v", err)
	}

	entries, err := os.ReadDir(folder)
	if err != nil || len(entries) != 1 {
		t.Fatalf("partially reconciled files: %v, %v", entries, err)
	}
	name := entries[0].Name()
	adopted, ok := documentFromFilename(reconcileScope, name)
	if !ok || !uuidV4.MatchString(adopted.ID) || adopted.SchemaID != "spec" {
		t.Fatalf("rename did not persist identity: %q", name)
	}
	index, err := os.ReadFile(filepath.Join(dir, boardName, filepath.FromSlash(catalogPath)))
	if err != nil || !bytes.Equal(index, originalIndex) {
		t.Fatalf("failed publication changed index: %v", err)
	}

	result := runResync(t, dir)
	if len(result.Added) != 1 || result.Added[0].ID != adopted.ID {
		t.Fatalf("retry allocated another ID: %+v", result)
	}
	stored, err := os.ReadFile(filepath.Join(folder, name))
	if err != nil || string(stored) != "id,priority\nb,01" {
		t.Fatalf("resync formatted manual bytes: %q, %v", stored, err)
	}
}

func TestResyncConflictsAreDetectedBeforeWrites(t *testing.T) {
	for _, tc := range []struct {
		name, first, second, diagnostic string
	}{
		{"duplicate id", reconcileID + "--first.md", reconcileID + "--second.md", "Duplicate document UUID"},
		{"unassigned collision", "same.csv", reconcileID + "--same.md", "Conflicting document name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, scope := reconcileFixture(t)
			for _, name := range []string{"independent.md", tc.first, tc.second} {
				putFile(t, scope, name, "original")
			}
			rejectResync(t, dir, tc.diagnostic, tc.first, tc.second)
			entries, err := os.ReadDir(scope)
			if err != nil || len(entries) != 3 {
				t.Fatalf("unexpected files: %v, %v", entries, err)
			}
			if _, err := os.Stat(filepath.Join(scope, "independent.md")); err != nil {
				t.Fatalf("resync renamed before detecting conflict: %v", err)
			}
		})
	}
}

func TestResyncRejectsMovedIndexedUUID(t *testing.T) {
	dir, scope := reconcileFixture(t)
	original := reconcileID + "--original.md"
	putFile(t, scope, original, "notes")
	runResync(t, dir)
	changed := reconcileID + "--renamed.md"
	if err := os.Rename(filepath.Join(scope, original), filepath.Join(scope, changed)); err != nil {
		t.Fatal(err)
	}
	rejectResync(t, dir, "Indexed UUID at a different path", original, changed)
	if err := WithAccess(dir, owner, false, func(a *Access) error {
		findings, err := a.AuditIndex()
		if err != nil {
			return err
		}
		for _, finding := range findings {
			if finding.Code == "uuid_at_different_path" && path.Base(finding.Path) == changed && path.Base(finding.RelatedPath) == original {
				return nil
			}
		}
		t.Fatalf("missing moved UUID finding: %+v", findings)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRenameDoesNotReplaceExistingTarget(t *testing.T) {
	root := t.TempDir()
	putFile(t, root, "source.md", "notes")
	putFile(t, root, "target.md", "keep")
	folder, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer folder.Close()
	if err := renameEntryExclusive(folder, "source.md", "target.md"); err == nil {
		t.Fatal("rename replaced target")
	}
	content, err := folder.ReadFile("target.md")
	if err != nil || string(content) != "keep" {
		t.Fatalf("rename changed target: %q, %v", content, err)
	}
}

func TestMissingIndexCannotInferMilestoneNamespaces(t *testing.T) {
	dir, scope := reconcileFixture(t)
	if err := WithDocuments(dir, owner, true, func(d *Documents) error {
		for _, milestone := range []string{"v1", "v2"} {
			if _, err := d.Create(CreateOptions{Scope: reconcileScope, Name: "review", Format: "md", Milestone: milestone}, strings.NewReader(milestone)); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	index := filepath.Join(dir, boardName, filepath.FromSlash(catalogPath))
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	rejectResync(t, dir, "Conflicting document name")
	if _, err := os.Stat(index); !os.IsNotExist(err) {
		t.Fatalf("rebuilt ambiguous index: %v", err)
	}
	entries, err := os.ReadDir(scope)
	if err != nil || len(entries) != 2 {
		t.Fatalf("ambiguous files not preserved: %v, %v", entries, err)
	}
}

func TestResyncRejectsSymlinkedDocument(t *testing.T) {
	dir, scope := reconcileFixture(t)
	putFile(t, dir, "elsewhere.md", "keep")
	orphan := filepath.Join(scope, "orphan.md")
	if err := os.Symlink(filepath.Join(dir, "elsewhere.md"), orphan); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	rejectResync(t, dir, "Unsafe document entry", "orphan.md")
	if stat, err := os.Lstat(orphan); err != nil || stat.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink was modified: %v", err)
	}
}
