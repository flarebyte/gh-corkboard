package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/board"
)

func (c docCLI) findOne(actor string, filters ...string) string {
	c.t.Helper()
	out, _ := c.run(0, actor, "", append([]string{"doc", "search", "--json"}, filters...)...)
	var result struct{ Documents []board.DocumentMetadata }
	if err := json.Unmarshal([]byte(out), &result); err != nil || len(result.Documents) != 1 {
		c.t.Fatalf("expected one search result: %q, %v", out, err)
	}
	return result.Documents[0].ID
}

func TestSharedHandoffWorkflow(t *testing.T) {
	cli, _ := updateFixture(t)
	const specs = "agent:specs"
	cli.run(0, updateAdmin, "", "access", "principal", "add", specs)
	cli.run(0, updateAdmin, "", "access", "grant", updateScope, "--to", specs, "--permission", "write")
	cli.run(0, updateAdmin, "", "access", "grant", updateScope, "--to", specs, "--permission", "status")
	spec := cli.create(specs, "# Pagination", updateScope, "pagination", "md")
	handoff := "Implement the spec with UUID " + spec.ID
	cli.create(specs, handoff, updateScope, "pagination-handoff", "md", "--milestone", "v1 launch/Specs",
		"--concept", "Cursor/Handoff", "--assignee", updateWriter, "--status", "active")
	id := cli.findOne(updateWriter, "--in", updateScope, "--milestone", "v1 launch/Specs", "--concept", "Cursor/Handoff")
	note, _ := cli.run(0, updateWriter, "", "doc", "get", id)
	linked, ok := strings.CutPrefix(strings.TrimSpace(note), "Implement the spec with UUID ")
	if !ok || linked != spec.ID {
		t.Fatalf("handoff text: %q", note)
	}
	content, _ := cli.run(0, updateWriter, "", "doc", "get", linked)
	if content != "# Pagination\n" {
		t.Fatalf("handoff target: %q", content)
	}
}

func TestSelfNoteWorkflow(t *testing.T) {
	cli, _ := updateFixture(t)
	note := cli.create(updateWriter, "Retain the local context", updateWriter, "follow-up", "md")
	id, _ := cli.run(0, updateWriter, "", "doc", "locate", "--in", updateWriter, "--name", "follow-up")
	content, _ := cli.run(0, updateWriter, "", "doc", "get", strings.TrimSpace(id))
	if strings.TrimSpace(id) != note.ID || content != "Retain the local context\n" {
		t.Fatalf("self-note: %q, %q", id, content)
	}
}

func TestSkillReferenceWorkflow(t *testing.T) {
	cli, _ := updateFixture(t)
	const skill = "skill:review"
	cli.run(0, updateAdmin, "", "access", "principal", "add", skill)
	guide := cli.create(skill, "# Review checklist", skill, "guide", "md", "--description", "review-guide")
	cli.run(2, updateWriter, "", "doc", "get", guide.ID)
	cli.run(0, updateAdmin, "", "access", "grant", skill, "--to", updateWriter, "--permission", "read")
	id := cli.findOne(updateWriter, "--in", skill, "--query", "review-guide")
	content, _ := cli.run(0, updateWriter, "", "doc", "get", id)
	if content != "# Review checklist\n" {
		t.Fatalf("reference: %q", content)
	}
}

func TestFindWorkWorkflow(t *testing.T) {
	cli, _ := updateFixture(t)
	wanted := cli.create(updateWriter, "work", updateScope, "find-work", "md", "--concept", "Cursor/Pagination", "--milestone", "v1 launch/Specs")
	other := cli.create(updateWriter, "later", updateScope, "find-later", "md", "--concept", "Cursor/Pagination", "--milestone", "v2/Specs")
	out, _ := cli.run(0, updateWriter, "", "doc", "search", "--milestone", "v1 launch/*", "--concept", "Cursor/*", "--format", "table")
	if !strings.Contains(out, wanted.ID) || strings.Contains(out, other.ID) {
		t.Fatalf("find work: %q", out)
	}
}

func TestSuperviseProgressWorkflow(t *testing.T) {
	cli, _ := updateFixture(t)
	cli.run(0, updateAdmin, "", "access", "grant", updateWriter, "--to", updateWriter, "--permission", "status")
	note := cli.create(updateWriter, "working", updateWriter, "pagination-implementation", "md", "--assignee", updateWriter,
		"--milestone", "v1 launch/Specs", "--concept", "Cursor/Implementation", "--status", "active")
	cli.update(updateWriter, note.ID, "Blocked on cursor shape", "--stdin", "--status", "blocked")
	out, _ := cli.run(0, updateAdmin, "", "doc", "search", "--milestone", "v1 launch/*", "--status", "active", "--status", "blocked", "--status", "review", "--format", "table")
	if !strings.Contains(out, note.ID) || !strings.Contains(out, "blocked") || !strings.Contains(out, updateWriter) {
		t.Fatalf("supervision table: %q", out)
	}
	content, _ := cli.run(0, updateAdmin, "", "doc", "get", note.ID)
	if content != "Blocked on cursor shape\n" {
		t.Fatalf("progress note: %q", content)
	}
}
