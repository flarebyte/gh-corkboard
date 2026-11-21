package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/board"
)

func TestIndexAuditAndResyncDrift(t *testing.T) {
	cli, root := updateFixture(t)
	scope := filepath.Join(root, ".gh-corkboard", "topics", "specs")
	index := filepath.Join(root, ".gh-corkboard", "index", "documents.json")
	changed := cli.create(updateAdmin, "old", updateScope, "changed", "md", "--status", "archived", "--concept", "Cursor/Change", "--milestone", "v1", "--assignee", updateWriter)
	gone := cli.create(updateAdmin, "gone", updateScope, "missing", "md")
	cli.update(updateReviewer, changed.ID, "", "--review-comment", "Check this", "--review-status", "warning")
	cli.run(2, updateWriter, "", "index", "audit")
	cli.run(2, updateWriter, "", "index", "resync")
	raw, _ := cli.run(0, updateAdmin, "", "index", "audit", "--json")
	if raw != "{\"findings\":[]}\n" {
		t.Fatalf("clean audit: %q", raw)
	}

	local := "local-results.spec.csv"
	for name, content := range map[string]string{changed.ID + "--changed.md": "changed\n", local: "id,value\nb,01", "readme.txt": "leave in place"} {
		if err := os.WriteFile(filepath.Join(scope, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Remove(filepath.Join(scope, gone.ID+"--missing.md")); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}

	raw, _ = cli.run(1, updateAdmin, "", "index", "audit", "--json")
	for _, code := range []string{"missing_document", "unindexed_document", "digest_mismatch", "unsupported_entry"} {
		if !strings.Contains(raw, `"code":"`+code+`"`) {
			t.Fatalf("missing %s in %q", code, raw)
		}
	}
	after, err := os.ReadFile(index)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("audit changed catalog: %v", err)
	}
	if _, err := os.Stat(filepath.Join(scope, local)); err != nil {
		t.Fatalf("audit renamed local file: %v", err)
	}

	raw, _ = cli.run(0, updateAdmin, "", "index", "resync", "--json")
	var result board.ResyncResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Added) != 1 || len(result.Removed) != 1 || len(result.Updated) != 1 || len(result.ResetToDraft) != 1 ||
		result.Removed[0].ID != gone.ID || result.Updated[0].ID != changed.ID || result.ResetToDraft[0].Status != "draft" ||
		len(result.Findings) != 1 || result.Findings[0].Code != "unsupported_entry" {
		t.Fatalf("resync report: %q", raw)
	}
	storage := cli.readRevision(updateAdmin, result.Added[0].ID)
	if storage.Content != "id,value\nb,01" {
		t.Fatalf("adopted content: %+v", storage)
	}
	raw, _ = cli.run(0, updateAdmin, "", "doc", "search", "--json")
	var matches struct{ Documents []board.DocumentMetadata }
	if err := json.Unmarshal([]byte(raw), &matches); err != nil || len(matches.Documents) != 2 {
		t.Fatalf("search after resync: %q, %v", raw, err)
	}
	got := matches.Documents[0]
	if got.ID != changed.ID || got.Status != "draft" || got.Milestone != "v1" || got.Concept != "Cursor/Change" ||
		got.Assignee != updateWriter || got.Reviews[updateReviewer].Comment != "Check this" {
		t.Fatalf("surviving metadata: %+v", got)
	}
	if err := os.Remove(filepath.Join(scope, "readme.txt")); err != nil {
		t.Fatal(err)
	}
	text, _ := cli.run(0, updateAdmin, "", "index", "audit")
	if text != "Index is in sync.\n" {
		t.Fatalf("post-resync audit: %q", text)
	}
}

func TestIndexRebuildMissingAndKeepMalformed(t *testing.T) {
	cli, root := updateFixture(t)
	doc := cli.create(updateWriter, "original", updateScope, "rebuild", "md", "--milestone", "v1")
	index := filepath.Join(root, ".gh-corkboard", "index", "documents.json")
	if err := os.Remove(index); err != nil {
		t.Fatal(err)
	}
	text, _ := cli.run(1, updateAdmin, "", "index", "audit")
	if !strings.Contains(text, "Missing index: index/documents.json") {
		t.Fatalf("missing-index audit: %q", text)
	}
	out, _ := cli.run(0, updateAdmin, "", "index", "resync")
	if !strings.Contains(out, "Indexed 1 new document as draft.\n"+doc.ID) {
		t.Fatalf("rebuild result: %q", out)
	}
	raw, _ := cli.run(0, updateAdmin, "", "doc", "search", "--json")
	if !strings.Contains(raw, doc.ID) || strings.Contains(raw, `"milestone"`) {
		t.Fatalf("rebuilt metadata: %q", raw)
	}

	for _, broken := range []string{`{"version":1,"version":1}`, `{"version":2,"documents":[]}`} {
		if err := os.WriteFile(index, []byte(broken), 0600); err != nil {
			t.Fatal(err)
		}
		out, _ := cli.run(1, updateAdmin, "", "index", "audit", "--json")
		if !strings.Contains(out, `"code":"invalid_index"`) {
			t.Fatalf("malformed audit: %q", out)
		}
		cli.run(2, updateAdmin, "", "index", "resync")
		saved, err := os.ReadFile(index)
		if err != nil || string(saved) != broken {
			t.Fatalf("resync changed invalid catalog: %q, %v", saved, err)
		}
	}
}
