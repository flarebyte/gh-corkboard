package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/flarebyte/gh-corkboard/internal/testrepo"
)

const updateAdmin = "human:owner"
const updateWriter = "agent:implementation"
const updateReviewer = "agent:test"
const updateScope = "topic:specs"

func updateFixture(t *testing.T) (docCLI, string) {
	t.Helper()
	root := testrepo.New(t)
	t.Chdir(root)
	cli := docCLI{t}
	cli.run(0, updateAdmin, "", "init")
	cli.run(0, updateAdmin, "", "access", "topic", "create", "specs")
	for _, subject := range []struct{ id, permission string }{
		{updateWriter, "write"}, {updateReviewer, "status"},
	} {
		cli.run(0, updateAdmin, "", "access", "principal", "add", subject.id)
		cli.run(0, updateAdmin, "", "access", "grant", updateScope, "--to", subject.id, "--permission", subject.permission)
	}
	return cli, root
}

func (c docCLI) readRevision(actor, id string) board.DocumentContent {
	c.t.Helper()
	out, _ := c.run(0, actor, "", "doc", "get", id, "--with-revision")
	var result board.DocumentContent
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.Revision == "" {
		c.t.Fatalf("revision: %q, %v", out, err)
	}
	return result
}

func (c docCLI) update(actor, id, input string, flags ...string) board.UpdateResult {
	c.t.Helper()
	out, _ := c.run(0, actor, input, append([]string{"doc", "update", id, "--json"}, flags...)...)
	var result board.UpdateResult
	if err := json.Unmarshal([]byte(out), &result); err != nil || result.ID != id {
		c.t.Fatalf("update: %q, %v", out, err)
	}
	return result
}

func TestUpdateGuardsAndPermissions(t *testing.T) {
	cli, root := updateFixture(t)
	doc := cli.create(updateWriter, "original", updateScope, "pagination", "md", "--milestone", "v1")
	another := cli.create(updateWriter, `{"other":true}`, updateScope, "pagination", "json", "--milestone", "v2")
	base := cli.readRevision(updateWriter, doc.ID)
	catPath := filepath.Join(root, ".gh-corkboard", "index", "documents.json")
	before, err := os.ReadFile(catPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		actor, id, message string
		flags              []string
	}{
		{updateWriter, doc.ID, "status permission", []string{"--stdin", "--status", "active"}},
		{updateReviewer, doc.ID, "write permission", []string{"--description", "no", "--status", "approved"}},
		{updateWriter, doc.ID, "lowercase hexadecimal", []string{"--expect", "BAD", "--stdin"}},
		{updateWriter, doc.ID, "at least one change", []string{"--expect", base.Revision}},
		{updateWriter, another.ID, "already exists", []string{"--milestone", " V1 "}},
		{updateWriter, doc.ID, "--concept", []string{"--concept", "bad value"}},
		{updateWriter, doc.ID, "--preserve-row-order", []string{"--preserve-row-order", "--stdin"}},
		{updateReviewer, doc.ID, "supplied together", []string{"--review-comment", "incomplete"}},
		{updateReviewer, doc.ID, "cannot be combined", []string{"--clear-review-comment", "--review-comment", "x", "--review-status", "pass"}},
	} {
		_, diagnostic := cli.run(2, tc.actor, "replacement", append([]string{"doc", "update", tc.id}, tc.flags...)...)
		if !strings.Contains(diagnostic, tc.message) {
			t.Fatalf("expected %q in %q", tc.message, diagnostic)
		}
	}
	invalid, _ := cli.run(1, updateWriter, `{"title":`, "doc", "update", another.ID, "--stdin", "--description", "bad draft")
	if invalid != "Invalid JSON: unexpected end of input.\n" {
		t.Fatalf("malformed replacement: %q", invalid)
	}
	after, err := os.ReadFile(catPath)
	if err != nil || !bytes.Equal(before, after) || cli.readRevision(updateWriter, doc.ID) != base {
		t.Fatalf("rejected update changed stored state: %v", err)
	}

	result := cli.update(updateWriter, doc.ID, "new", "--stdin", "--expect", base.Revision, "--milestone", "", "--concept", "Cursor/Pagination")
	if result.FormattingChanged == nil || !*result.FormattingChanged || result.Milestone != "" || result.Concept != "Cursor/Pagination" {
		t.Fatalf("replacement output: %+v", result)
	}
	_, diagnostic := cli.run(2, updateReviewer, "", "doc", "update", doc.ID, "--json", "--expect", base.Revision, "--status", "approved")
	if !strings.Contains(diagnostic, `"code":"revision_mismatch"`) {
		t.Fatalf("wrong stale revision diagnostic: %q", diagnostic)
	}
	saved := cli.readRevision(updateWriter, doc.ID)
	if saved.Content != "new\n" {
		t.Fatalf("content: %q", saved.Content)
	}
	status := cli.update(updateReviewer, doc.ID, "", "--expect", saved.Revision, "--status", "review")
	if status.Status != "review" || status.FormattingChanged != nil {
		t.Fatalf("metadata-only update: %+v", status)
	}
}

func TestReviewHandoffAndSupervision(t *testing.T) {
	cli, _ := updateFixture(t)
	handoff := cli.create(updateWriter, "implement the cursor", updateScope, "handoff", "md", "--milestone", "v1 launch/Specs")
	selfNote := cli.create(updateWriter, "implementation details", updateWriter, "note", "md")
	attachment := cli.create(updateReviewer, "test results", updateReviewer, "results", "md")
	if note, _ := cli.run(0, updateWriter, "", "doc", "get", selfNote.ID); note != "implementation details\n" {
		t.Fatalf("self note: %q", note)
	}

	base := cli.readRevision(updateReviewer, handoff.ID)
	cli.run(2, updateReviewer, "", "doc", "update", handoff.ID, "--review-comment", "Private note", "--review-status", "pass", "--review-locator", selfNote.ID)
	summary, _ := cli.run(0, updateReviewer, "", "doc", "update", handoff.ID, "--expect", base.Revision, "--status", "approved", "--review-comment", "Tests pass", "--review-status", "pass", "--review-locator", attachment.ID)
	if summary != "Updated "+handoff.ID+". Status: approved. Review by agent:test: pass.\n" {
		t.Fatalf("review output: %q", summary)
	}

	hidden := cli.update(updateWriter, handoff.ID, "", "--description", "progress", "--assignee", updateReviewer)
	if hidden.Assignee != updateReviewer {
		t.Fatalf("assignment not saved: %+v", hidden)
	}
	cli.run(2, updateWriter, "", "doc", "update", selfNote.ID, "--assignee", updateReviewer)
	clearedAssignment := cli.update(updateWriter, handoff.ID, "", "--assignee", "")
	if clearedAssignment.Assignee != "" {
		t.Fatalf("assignment not cleared: %+v", clearedAssignment)
	}
	if review := hidden.Reviews[updateReviewer]; review.Locator != "" || review.Comment != "Tests pass" {
		t.Fatalf("redacted review: %+v", review)
	}
	adminView := cli.update(updateAdmin, handoff.ID, "", "--review-comment", "Follow up", "--review-status", "warning")
	if adminView.Reviews[updateReviewer].Locator != attachment.ID {
		t.Fatalf("lost stored locator: %+v", adminView.Reviews)
	}
	cli.run(0, updateAdmin, "", "access", "grant", updateReviewer, "--to", updateWriter, "--permission", "read")
	search, _ := cli.run(0, updateWriter, "", "doc", "search", "--status", "approved", "--json")
	if !strings.Contains(search, attachment.ID) {
		t.Fatalf("read grant did not expose locator: %q", search)
	}
	cli.run(0, updateAdmin, "", "access", "revoke", updateReviewer, "--from", updateWriter, "--permission", "read")
	search, _ = cli.run(0, updateWriter, "", "doc", "search", "--status", "approved", "--json")
	if strings.Contains(search, attachment.ID) || !strings.Contains(search, "Tests pass") {
		t.Fatalf("redaction after revoke: %q", search)
	}

	cli.run(0, updateAdmin, "", "doc", "delete", attachment.ID)
	afterDelete := cli.update(updateAdmin, handoff.ID, "", "--status", "approved")
	if review := afterDelete.Reviews[updateReviewer]; review.Locator != "" || review.Comment != "Tests pass" {
		t.Fatalf("deleted locator: %+v", review)
	}

	changed := cli.update(updateReviewer, handoff.ID, "", "--status", "blocked", "--review-comment", "New result", "--review-status", "warning")
	if review := changed.Reviews[updateReviewer]; review.Comment != "New result" || review.Locator != "" {
		t.Fatalf("review replacement: %+v", review)
	}
	for range 2 {
		cleared := cli.update(updateReviewer, handoff.ID, "", "--clear-review-comment")
		if len(cleared.Reviews) != 1 || cleared.Reviews[updateAdmin].Comment != "Follow up" {
			t.Fatalf("clear changed another author's review: %+v", cleared.Reviews)
		}
	}
	out, _ := cli.run(0, updateAdmin, "", "doc", "search", "--milestone", "v1 launch/*", "--status", "blocked", "--format", "table")
	if !strings.Contains(out, handoff.ID) || strings.Contains(out, selfNote.ID) {
		t.Fatalf("supervision search: %q", out)
	}
}

func TestUpdateFormattingAndArchiveStatus(t *testing.T) {
	cli, root := updateFixture(t)
	doc := cli.create(updateAdmin, "id,value\na,1\nb,2\n", updateScope, "data", "csv", "--status", "archived")
	file := filepath.Join(root, ".gh-corkboard", "topics", "specs", doc.ID+"--data.csv")
	before, err := os.Stat(file)
	if err != nil {
		t.Fatal(err)
	}
	result := cli.update(updateWriter, doc.ID, "id,value\nb,2\na,1\n", "--stdin")
	after, err := os.Stat(file)
	if err != nil || !os.SameFile(before, after) || result.FormattingChanged == nil || !*result.FormattingChanged || result.Status != "archived" {
		t.Fatalf("formatting-only update: %+v, %v", result, err)
	}
	result = cli.update(updateWriter, doc.ID, "id,value\nb,2\na,1\n", "--stdin", "--preserve-row-order")
	if result.FormattingChanged == nil || *result.FormattingChanged || result.Status != "draft" {
		t.Fatalf("changed archived bytes: %+v", result)
	}
	replacement := "id,value\na,3\n"
	result = cli.update(updateAdmin, doc.ID, replacement, "--stdin", "--status", "archived")
	if result.Status != "archived" {
		t.Fatalf("explicit status lost: %+v", result)
	}
	if err := os.WriteFile(file, []byte("id,value\na,4\n"), 0600); err != nil {
		t.Fatal(err)
	}
	result = cli.update(updateWriter, doc.ID, "id,value\na,4\n", "--stdin")
	if result.Status != "draft" {
		t.Fatalf("external edit recorded without reset: %+v", result)
	}
}
