package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/board"
)

func TestSchemaRegistry(t *testing.T) {
	cli, root := updateFixture(t)
	const definition = "{\n  \"type\":\"object\"\n}"
	result, _ := cli.run(0, updateAdmin, definition, "schema", "create", "json", "spec", "--json")
	if result != "{\"created\":true,\"format\":\"json\",\"schemaId\":\"spec\"}\n" {
		t.Fatalf("create output: %q", result)
	}
	cli.run(0, updateAdmin, "true", "schema", "create", "csv", "rows")
	cli.run(0, updateAdmin, "true", "schema", "create", "json", "alphabet")
	cli.run(2, updateWriter, "true", "schema", "update", "json", "spec")
	cli.run(2, updateAdmin, "true", "schema", "create", "json", "spec")
	cli.run(2, updateAdmin, "true", "schema", "update", "json", "absent")
	cli.run(1, updateAdmin, `{"$ref":"other.json"}`, "schema", "update", "json", "spec")
	content, _ := cli.run(0, updateWriter, "", "schema", "get", "json", "spec")
	if content != definition {
		t.Fatalf("stored bytes: %q", content)
	}
	list, _ := cli.run(0, updateReviewer, "", "schema", "list", "--json")
	want := "{\"schemas\":[{\"format\":\"csv\",\"schemaId\":\"rows\"},{\"format\":\"json\",\"schemaId\":\"alphabet\"},{\"format\":\"json\",\"schemaId\":\"spec\"}]}\n"
	if list != want {
		t.Fatalf("schema list: %q", list)
	}
	out, _ := cli.run(0, updateReviewer, "", "schema", "list", "--format", "csv")
	if out != "csv  rows\n" {
		t.Fatalf("format listing: %q", out)
	}

	indexed := cli.create(updateWriter, `{}`, updateScope, "spec", "json", "--schema", "spec")
	cli.run(2, updateAdmin, "", "schema", "delete", "json", "spec")
	cli.run(0, updateAdmin, "", "doc", "delete", indexed.ID)
	scope := filepath.Join(root, ".gh-corkboard", "topics", "specs")
	for _, name := range []string{"local.spec.json", "29b4dd81-af57-4c98-b3f0-ff0bf66da7bb--local.spec.json"} {
		file := filepath.Join(scope, name)
		if err := os.WriteFile(file, []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
		cli.run(2, updateAdmin, "", "schema", "delete", "json", "spec")
		if err := os.Remove(file); err != nil {
			t.Fatal(err)
		}
	}
	cli.run(0, updateAdmin, "", "schema", "delete", "json", "spec")
}

func TestDocumentValidationReportsAndPermissions(t *testing.T) {
	cli, root := updateFixture(t)
	cli.run(0, updateAdmin, `{"type":"object","required":["title"]}`, "schema", "create", "json", "spec")
	bad := cli.create(updateWriter, `{}`, updateScope, "b-json", "json", "--schema", "spec")
	text, _ := cli.run(1, updateWriter, "", "doc", "validate", bad.ID)
	if text != bad.ID+"  invalid\n  Required property \"title\" is missing.\n" {
		t.Fatalf("required report: %q", text)
	}
	precise := cli.create(updateWriter, `{"n":900719925474099312345}`, updateScope, "a-json", "json", "--schema", "spec")
	cli.run(0, updateAdmin, `{"properties":{"n":{"const":900719925474099312345}},"required":["n"]}`, "schema", "update", "json", "spec")
	schemaLess := cli.create(updateWriter, "# Heading", updateScope, "c-md", "md")
	missing := cli.create(updateWriter, "id\n1", updateScope, "d-csv", "csv", "--schema", "missing")
	hidden := cli.create(updateReviewer, "hidden", updateReviewer, "hidden", "md")
	cli.run(2, updateWriter, "", "doc", "validate", hidden.ID)
	out, _ := cli.run(1, updateWriter, "", "doc", "validate", "--all", "--json")
	var report struct{ Results []board.ValidationResult }
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 4 {
		t.Fatalf("readable reports: %q", out)
	}
	for i, want := range []struct{ id, state string }{
		{precise.ID, "valid"}, {bad.ID, "invalid"}, {schemaLess.ID, "schemaless"}, {missing.ID, "invalid"},
	} {
		got := report.Results[i]
		if got.ID != want.id || got.State != want.state || got.Diagnostics == nil {
			t.Fatalf("result %d: %+v", i, got)
		}
	}
	if report.Results[3].Diagnostics[0].Code != "schema_missing" {
		t.Fatalf("missing schema report: %q", out)
	}

	file := filepath.Join(root, ".gh-corkboard", "topics", "specs", schemaLess.ID+"--c-md.md")
	if err := os.WriteFile(file, []byte{0xff}, 0600); err != nil {
		t.Fatal(err)
	}
	out, _ = cli.run(1, updateWriter, "", "doc", "validate", schemaLess.ID, "--json")
	if !strings.Contains(out, `"code":"invalid_md"`) {
		t.Fatalf("malformed external document: %q", out)
	}
}

func TestValidationProjectionCoordinates(t *testing.T) {
	cli, _ := updateFixture(t)
	cli.run(0, updateAdmin, `{"properties":{"rows":{"items":{"properties":{"priority":{"const":"01"}}}}}}`, "schema", "create", "csv", "rows")
	cli.run(0, updateAdmin, `{"properties":{"headings":{"prefixItems":[{"properties":{"text":{"const":"Required heading"}}}]}}}`, "schema", "create", "md", "headings")
	for _, tc := range []struct{ format, schema, content, diagnostic string }{
		{"csv", "rows", "id,priority\nb,02\n", `"instancePath":"/rows/0/priority","line":2,"column":3`},
		{"md", "headings", "prose\n\n## Other **heading**\n", `"instancePath":"/headings/0/text","line":3`},
	} {
		doc := cli.create(updateWriter, tc.content, updateScope, "example-"+tc.format, tc.format, "--schema", tc.schema)
		text, _ := cli.run(1, updateWriter, "", "doc", "validate", doc.ID, "--json")
		if !strings.Contains(text, tc.diagnostic) {
			t.Fatalf("missing source location %q: %s", tc.diagnostic, text)
		}
	}
}
