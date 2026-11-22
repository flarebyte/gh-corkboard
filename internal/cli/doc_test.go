package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/board"
	"github.com/flarebyte/gh-corkboard/internal/testrepo"
)

type docCLI struct{ t *testing.T }

func (c docCLI) run(code int, actor, stdin string, args ...string) (string, string) {
	c.t.Helper()
	var out, errOut bytes.Buffer
	status := Run(append([]string{"--as", actor}, args...), strings.NewReader(stdin), &out, &errOut)
	if status != code || (status == 2 && out.Len() != 0) || (status != 2 && errOut.Len() != 0) {
		c.t.Fatalf("%s %v returned %d, stdout=%q stderr=%q", actor, args, status, &out, &errOut)
	}
	return out.String(), errOut.String()
}

func (c docCLI) create(actor, input, scope, name, format string, args ...string) board.DocumentMetadata {
	c.t.Helper()
	command := append([]string{"doc", "create", "--in", scope, "--name", name, "--format", format, "--json"}, args...)
	out, _ := c.run(0, actor, input, command...)
	var meta board.DocumentMetadata
	if err := json.Unmarshal([]byte(out), &meta); err != nil || meta.ID == "" || strings.Contains(out, `"sha256"`) || strings.Contains(out, `"path"`) {
		c.t.Fatalf("create metadata: %s, %v", out, err)
	}
	return meta
}

func TestDocumentRoundTrip(t *testing.T) {
	root := testrepo.New(t)
	t.Chdir(root)
	cli := docCLI{t}
	const admin = "human:owner"
	cli.run(0, admin, "", "init")
	cli.run(0, admin, "", "access", "topic", "create", "specs")
	doc := cli.create(admin, "paragraph  \r\n    code", "topic:specs", "pagination", "md", "--milestone", " V1/Specs ", "--concept", "Cursor/Pagination")
	expected := "paragraph  \n    code\n"
	content, _ := cli.run(0, admin, "", "doc", "get", doc.ID)
	if content != expected {
		t.Fatalf("raw content %q", content)
	}
	out, _ := cli.run(0, admin, "", "doc", "get", doc.ID, "--with-revision")
	var version board.DocumentContent
	if err := json.Unmarshal([]byte(out), &version); err != nil {
		t.Fatal(err)
	}
	sha := fmt.Sprintf("%x", sha256.Sum256([]byte(expected)))
	if version.Content != expected || version.Revision != sha {
		t.Fatalf("unexpected revision: %+v", version)
	}

	catalog := filepath.Join(root, ".gh-corkboard", "index", "documents.json")
	before, err := os.ReadFile(catalog)
	if err != nil || !bytes.Contains(before, []byte(sha)) {
		t.Fatalf("stored digest: %s, %v", before, err)
	}
	cli.run(2, admin, "{}", "doc", "create", "--in", "topic:specs", "--name", "pagination", "--format", "json", "--milestone", "v1/specs")
	diagnostic, _ := cli.run(1, admin, `{"x":`, "doc", "create", "--in", "topic:specs", "--name", "broken", "--format", "json", "--json")
	if diagnostic != "{\"diagnostics\":[{\"code\":\"invalid_json\",\"message\":\"Unexpected end of input.\"}]}\n" {
		t.Fatalf("diagnostic %q", diagnostic)
	}
	after, err := os.ReadFile(catalog)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("rejected operations changed index: %v", err)
	}
	scope := filepath.Join(root, ".gh-corkboard", "topics", "specs")
	files, err := os.ReadDir(scope)
	if err != nil || len(files) != 1 {
		t.Fatalf("rejected writes left files: %v, %v", files, err)
	}

	jsonDoc := cli.create(admin, `{"z":10.000,"a":true}`, "topic:specs", "pagination", "json", "--schema", "spec")
	csvDoc := cli.create(admin, "id,value\nb,01\na,02", "topic:specs", "priorities", "csv", "--preserve-row-order")
	for _, tc := range []struct{ id, want string }{
		{jsonDoc.ID, "{\n  \"a\": true,\n  \"z\": 10.000\n}\n"},
		{csvDoc.ID, "id,value\nb,01\na,02\n"},
	} {
		out, _ := cli.run(0, admin, "", "doc", "get", tc.id)
		if out != tc.want {
			t.Fatalf("%s: got %q", tc.id, out)
		}
	}
	cli.run(2, admin, "", "doc", "locate", "--in", "topic:specs", "--name", "pagination")
	for _, tc := range []struct{ milestone, id string }{{"", jsonDoc.ID}, {"v1/SPECS", doc.ID}} {
		id, _ := cli.run(0, admin, "", "doc", "locate", "--in", "topic:specs", "--name", "pagination", "--milestone", tc.milestone)
		if id != tc.id+"\n" {
			t.Fatalf("locate returned %q", id)
		}
	}
	for _, id := range []string{doc.ID, jsonDoc.ID, csvDoc.ID} {
		cli.run(0, admin, "", "doc", "delete", id)
	}
	files, err = os.ReadDir(scope)
	if err != nil || len(files) != 0 {
		t.Fatalf("expected retained empty folder: %v, %v", files, err)
	}
}

func TestDocumentDiscoveryAndPermissions(t *testing.T) {
	t.Chdir(testrepo.New(t))
	cli := docCLI{t}
	const admin = "human:owner"
	const agent = "agent:writer"
	cli.run(0, admin, "", "init")
	cli.run(0, admin, "", "access", "principal", "add", agent)
	cli.run(0, admin, "", "access", "principal", "add", "skill:review")
	cli.run(0, admin, "", "access", "topic", "create", "specs")
	cli.run(0, admin, "", "access", "grant", "topic:specs", "--to", agent, "--permission", "write")
	hidden := cli.create(admin, "secret", "skill:review", "hidden", "md")
	cli.run(2, agent, "body", "doc", "create", "--in", "topic:specs", "--name", "blocked", "--format", "md", "--status", "draft")
	cli.run(2, agent, "body", "doc", "create", "--in", "topic:specs", "--name", "blocked", "--format", "md", "--assignee", "skill:review")
	_, forbidden := cli.run(2, agent, "", "doc", "get", hidden.ID)
	_, missing := cli.run(2, agent, "", "doc", "get", "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb")
	if missing != forbidden {
		t.Fatalf("distinct diagnostics: %q / %q", forbidden, missing)
	}
	private := cli.create(agent, "mine", agent, "private", "md")
	cli.run(2, agent, "", "doc", "delete", private.ID)
	base := []string{"--description", strings.Repeat("é", 61), "--milestone", "v1/Specs", "--concept", "Cursor/Pagination", "--assignee", agent}
	doc := cli.create(agent, "body", "topic:specs", "pagination", "md", base...)
	out, _ := cli.run(0, agent, "", "doc", "search", "--status", "draft", "--status", "draft", "--assignee", agent, "--milestone", "V1/*", "--concept", "cursor/pagination", "--query", "CURSOR", "--format", "csv")
	want := "id,name,scope,description,milestone\n" + doc.ID + ",pagination,topic:specs," + strings.Repeat("é", 57) + "...,v1/Specs\n"
	if out != want {
		t.Fatalf("compact search: %q, want %q", out, want)
	}
	out, _ = cli.run(0, agent, "", "doc", "search", "--in", "skill:review", "--format", "json")
	if out != "{\"documents\":[]}\n" {
		t.Fatalf("hidden result: %q", out)
	}
	cli.run(2, agent, "", "doc", "search", "--concept", "Cur*sor")
	cli.run(2, agent, "", "doc", "search", "--format", "table", "--json")
	cli.run(2, agent, "", "doc", "delete", doc.ID)
	cli.run(0, admin, "", "access", "grant", "topic:specs", "--to", agent, "--permission", "delete")
	cli.run(0, agent, "", "doc", "delete", doc.ID)
}
