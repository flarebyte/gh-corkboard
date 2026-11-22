package schema

import (
	"testing"

	"github.com/flarebyte/gh-corkboard/internal/strictjson"
)

func TestOfflineCompiler(t *testing.T) {
	for _, source := range []string{
		`{"$schema":"http://json-schema.org/draft-07/schema#"}`,
		`{"$schema":[]}`,
		`{"$ref":"https://json-schema.org/draft/2020-12/schema"}`,
		`{"$ref":"file:///example.schema.json"}`,
		`{"$ref":"https://example.invalid/schema"}`,
		`{"$ref":"#/$defs/missing"}`,
		`{"$defs":{"inner":{"$id":"child","$schema":"http://json-schema.org/draft-07/schema#"}}}`,
		`{"allOf":[{"$ref":"child"}],"$defs":{"inner":{"$id":"child"}}}`,
		`{"$ref":"#/custom","custom":{"$ref":"https://json-schema.org/draft/2020-12/schema"}}`,
	} {
		data, err := strictjson.Decode([]byte(source), 100)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := Compile("https://example.invalid/root", data); err == nil {
			t.Fatalf("compiled unsupported schema: %s", source)
		}
	}

	source := `{"$schema":"https://json-schema.org/draft/2020-12/schema","$defs":{"email":{"type":"string","format":"email"}},"properties":{"value":{"$ref":"#/$defs/email"},"literal":{"const":{"$ref":"an ordinary string"}}}}`
	data, err := strictjson.Decode([]byte(source), 100)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := Compile("https://example.invalid/root", data)
	if err != nil {
		t.Fatal(err)
	}
	value := map[string]any{"value": "invalid-email", "literal": map[string]any{"$ref": "an ordinary string"}}
	if err := compiled.Validate(value); err != nil {
		t.Fatalf("annotation should pass: %v", err)
	}
}
