package board

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/testrepo"
)

func TestAccessValidation(t *testing.T) {
	index := testIndex(t)

	const valid = `{"version":1,"principals":[{"id":"human:owner","role":"admin"},{"id":"agent:writer"}],"topics":[],"grants":[]}`
	for _, input := range []string{
		`{"version":1,"principals":[{"id":"agent:owner","role":"admin"}],"topics":[],"grants":[]}`,
		`{"version":1,"principals":[{"id":"human:owner","role":"admin"},{"id":"human:owner"}],"topics":[],"grants":[]}`,
		`{"version":1,"principals":[{"id":"human:owner","role":"admin"},{"id":"agent:writer"}],"topics":[],"grants":[{"scope":"agent:writer","principal":"human:owner","permissions":["write"]}]}`,
		`{"version":1,"principals":[{"id":"human:owner","role":"admin"},{"id":"agent:writer","role":null}],"topics":[],"grants":[]}`,
	} {
		if err := index.WriteFile("access.json", []byte(input), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadAccess(index); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}

	oversized := append([]byte(valid), bytes.Repeat([]byte(" "), accessLimit)...)
	if err := index.WriteFile("access.json", oversized, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := loadAccess(index); err == nil {
		t.Fatal("accepted oversized policy")
	}
	if err := index.WriteFile("access.json", []byte(valid), 0600); err != nil {
		t.Fatal(err)
	}
	policy, err := loadAccess(index)
	if err != nil || policy.role("agent:writer") != "member" {
		t.Fatalf("default role: %+v, %v", policy, err)
	}
}

func TestCatalogValidatesDerivedPaths(t *testing.T) {
	index := testIndex(t)
	policy := accessPolicy{Topics: []string{"specs"}}
	doc := document{
		ID: "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb", Scope: "topic:specs", Name: "pagination", Format: "md",
		Status: "draft", Digest: "0a00000000000000000000000000000000000000000000000000000000000000",
		Reviews: map[string]review{},
	}
	for _, path := range []string{"../pagination.md", doc.canonicalPath()} {
		doc.Path = path
		if err := writeMetadata(index, "documents", catalog{Version: 1, Documents: []document{doc}}); err != nil {
			t.Fatal(err)
		}
		_, err := loadCatalog(index, policy)
		if (err == nil) != (path == doc.canonicalPath()) {
			t.Fatalf("path %q: %v", path, err)
		}
	}
}

func TestInitRejectsSymlinkedPolicy(t *testing.T) {
	dir := testrepo.New(t)
	if _, err := Init(dir, "human:owner"); err != nil {
		t.Fatal(err)
	}
	original := filepath.Join(dir, boardName, "index", "access.json")
	relocated := filepath.Join(dir, boardName, "index", "relocated.json")
	if err := os.Rename(original, relocated); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("relocated.json", original); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Init(dir, "human:owner"); err == nil {
		t.Fatal("accepted a symlinked policy")
	}
}

func testIndex(t *testing.T) *os.Root {
	t.Helper()
	dir := testrepo.New(t)
	if _, err := Init(dir, "human:owner"); err != nil {
		t.Fatal(err)
	}
	index, err := os.OpenRoot(filepath.Join(dir, boardName, "index"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { index.Close() })
	return index
}
