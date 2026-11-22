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

func (c docCLI) patch(actor, id, revision, patch string, flags ...string) board.UpdateResult {
	c.t.Helper()
	args := append([]string{"doc", "patch", id, "--expect", revision, "--json"}, flags...)
	out, _ := c.run(0, actor, patch, args...)
	var result board.UpdateResult
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.ID != id {
		c.t.Fatalf("patch result: %q, %v", out, err)
	}
	return result
}

func TestPatchDocumentFormats(t *testing.T) {
	cli, _ := updateFixture(t)
	const unified = "--- document\n+++ document\n"

	md := cli.create(updateAdmin, "# Handoff", updateScope, "handoff", "md", "--status", "archived", "--milestone", "v1")
	base := cli.readRevision(updateWriter, md.ID)
	modified := cli.patch(updateWriter, md.ID, base.Revision, unified+"@@ -1 +1 @@\n-# Handoff\n+# Progress\n")
	if modified.Status != "draft" || modified.Milestone != "v1" || modified.FormattingChanged == nil || *modified.FormattingChanged {
		t.Fatalf("Markdown patch: %+v", modified)
	}
	if got := cli.readRevision(updateWriter, md.ID); got.Content != "# Progress\n" || got.Revision == base.Revision {
		t.Fatalf("patched Markdown: %+v", got)
	}

	csv := cli.create(updateWriter, "id,note\nb,\"first\nsecond\"\n", updateScope, "data", "csv")
	base = cli.readRevision(updateWriter, csv.ID)
	changed := cli.patch(updateWriter, csv.ID, base.Revision, unified+"@@ -2,2 +2,3 @@\n b,\"first\n-second\"\n+third\"\n+a,start\n")
	if changed.FormattingChanged == nil || !*changed.FormattingChanged {
		t.Fatalf("CSV formatting: %+v", changed)
	}
	if got := cli.readRevision(updateWriter, csv.ID); got.Content != "id,note\na,start\nb,\"first\nthird\"\n" {
		t.Fatalf("patched CSV: %+v", got)
	}

	doc := cli.create(updateWriter, `{"a":{"old":1,"keep":2},"n":1234567890123456789012345}`, updateScope, "settings", "json", "--schema", "spec")
	base = cli.readRevision(updateWriter, doc.ID)
	merged := cli.patch(updateWriter, doc.ID, base.Revision, `{"a":{"old":null,"new":[null,1.00]}}`)
	if merged.FormattingChanged != nil || merged.SchemaID != "spec" {
		t.Fatalf("JSON patch metadata: %+v", merged)
	}
	stored := cli.readRevision(updateWriter, doc.ID)
	if strings.Contains(stored.Content, `"old"`) || !strings.Contains(stored.Content, "1.00") || !strings.Contains(stored.Content, "1234567890123456789012345") {
		t.Fatalf("patched JSON: %+v", stored)
	}
	cli.patch(updateWriter, doc.ID, stored.Revision, "null")
	if got := cli.readRevision(updateWriter, doc.ID); got.Content != "null\n" {
		t.Fatalf("null root: %+v", got)
	}
}

func TestRejectedPatchesPreserveBoard(t *testing.T) {
	cli, root := updateFixture(t)
	doc := cli.create(updateWriter, "id,value\na,1", updateScope, "data", "csv")
	base := cli.readRevision(updateWriter, doc.ID)
	index := filepath.Join(root, ".gh-corkboard", "index", "documents.json")
	before, err := os.ReadFile(index)
	if err != nil {
		t.Fatal(err)
	}

	const head = "--- document\n+++ document\n"
	const change = head + "@@ -2 +2 @@\n-a,1\n+a,2\n"
	_, missing := cli.run(2, updateWriter, change, "doc", "patch", doc.ID)
	if !strings.Contains(missing, "doc get --with-revision") {
		t.Fatalf("missing revision diagnostic: %q", missing)
	}
	_, malformed := cli.run(2, updateWriter, change, "doc", "patch", doc.ID, "--expect", "ABC")
	if !strings.Contains(malformed, "lowercase hexadecimal") {
		t.Fatalf("malformed revision diagnostic: %q", malformed)
	}
	cli.run(2, updateReviewer, change, "doc", "patch", doc.ID, "--expect", base.Revision)

	for _, tc := range []struct{ expected, patch, diagnostic string }{
		{strings.Repeat("0", 64), change, "revision_mismatch"},
		{base.Revision, head + "@@ -1 +1 @@\n-a,1\n+a,2\n", "patch_context_mismatch"},
	} {
		_, got := cli.run(2, updateWriter, tc.patch, "doc", "patch", doc.ID, "--expect", tc.expected, "--json")
		if !strings.Contains(got, `"code":"`+tc.diagnostic+`"`) {
			t.Fatalf("diagnostic: %q", got)
		}
	}

	for _, patch := range []string{change + "trailing data\n", head + "@@ -2 +2 @@\n-a,1\n+too,many,fields\n"} {
		found, _ := cli.run(1, updateWriter, patch, "doc", "patch", doc.ID, "--expect", base.Revision, "--json")
		if !strings.Contains(found, `"code":"invalid_csv"`) {
			t.Fatalf("malformed patch: %q", found)
		}
	}
	after, err := os.ReadFile(index)
	if err != nil || !bytes.Equal(before, after) || cli.readRevision(updateWriter, doc.ID) != base {
		t.Fatalf("rejected patch changed stored state: %v", err)
	}
}
