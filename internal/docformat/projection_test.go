package docformat

import (
	"encoding/json"
	"testing"
)

func TestMarkdownProjection(t *testing.T) {
	source := []byte("# A **bold** [link](https://example.invalid) ![alt *text*](image) `code &amp;`\n\n> # Hidden\n\n- # Nested\n\nLine one\nline two &amp; \\&amp; <em>tagged</em>\n---\n\n## Café\n")
	p, err := Project("md", source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(p.Value)
	want := `{"headings":[{"level":1,"text":"A bold link alt text code \u0026amp;"},{"level":2,"text":"Line one line two \u0026 \u0026amp; tagged"},{"level":2,"text":"Café"}]}`
	if err != nil || string(got) != want {
		t.Fatalf("headings: %s, %v", got, err)
	}
	line, _ := p.Location(source)([]string{"headings", "2", "text"})
	if line != 11 {
		t.Fatalf("heading line %d", line)
	}
}

func TestCSVProjectionLocations(t *testing.T) {
	source := []byte("id,priority,note\r\n\r\na,01,\"line\r\nnext\"\r\nb, 02 ,\"x,y\"\r\n")
	p, err := Project("csv", source)
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(p.Value)
	want := `{"columns":["id","priority","note"],"rows":[{"id":"a","note":"line\r\nnext","priority":"01"},{"id":"b","note":"x,y","priority":" 02 "}]}`
	if err != nil || string(got) != want {
		t.Fatalf("CSV projection: %s, %v", got, err)
	}
	if line, col := p.Location(source)([]string{"rows", "1", "priority"}); line != 5 || col != 3 {
		t.Fatalf("CSV position: %d:%d", line, col)
	}
}
