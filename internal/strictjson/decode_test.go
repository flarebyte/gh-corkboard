package strictjson

import (
	"encoding/json"
	"testing"
)

func TestDecode(t *testing.T) {
	for _, input := range []string{`{"a": {"x":1,"\u0078":2}}`, `["\uD800"]`, `["\uDC00"]`, `["\uD800 \uDC00"]`, `null 0`, `[[[]]]`, "{\"x\":\"\xff\"}"} {
		if _, err := Decode([]byte(input), 2); err == nil {
			t.Errorf("accepted %q", input)
		}
	}
	got, err := Decode([]byte(`["\uD83D\uDE00","\\uD800",12345678901234567890]`), 2)
	if err != nil {
		t.Fatal(err)
	}
	values := got.([]any)
	if values[0] != "😀" || values[1] != `\uD800` || values[2] != json.Number("12345678901234567890") {
		t.Fatalf("decoded %#v", got)
	}
}
