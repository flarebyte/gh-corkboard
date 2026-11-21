package docformat

import (
	"errors"
	"testing"
)

func TestPatchTextRanges(t *testing.T) {
	for _, tc := range []struct{ name, base, hunk, want string }{
		{"replace", "a\nb\nc\n", "@@ -2 +2 @@\n-b\n+B\n", "a\nB\nc\n"},
		{"insert-first", "b\n", "@@ -0,0 +1 @@\n+a\n", "a\nb\n"},
		{"insert-middle", "a\nb\n", "@@ -1,0 +2 @@\n+x\n", "a\nx\nb\n"},
		{"insert-last", "a\n", "@@ -1,0 +2 @@\n+b\n", "a\nb\n"},
		{"insert-empty", "", "@@ -0,0 +1 @@\n+a\n", "a\n"},
		{"delete-first", "a\nb\n", "@@ -1 +0,0 @@\n-a\n", "b\n"},
		{"delete-all", "a\n", "@@ -1 +0,0 @@\n-a\n", ""},
		{"separate-hunks", "a\nb\nc\n", "@@ -0,0 +1 @@\n+x\n@@ -3 +4 @@\n-c\n+C\n", "x\na\nb\nC\n"},
		{"old-no-newline", "a", "@@ -1 +1 @@\n-a\n\\ No newline at end of file\n+b\n", "b\n"},
		{"new-no-newline", "a\n", "@@ -1 +1 @@\n-a\n+b\n\\ No newline at end of file\n", "b"},
		{"exact-crlf", "a\r\n", "@@ -1 +1 @@\n-a\r\n+b\r\n", "b\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			patch := "--- first file\t2026-11-20 10:10:00 +0000\n+++ second file\t2026-11-21 10:10:00 +0000\n" + tc.hunk
			got, err := patchText([]byte(tc.base), []byte(patch), "md")
			if err != nil || string(got) != tc.want {
				t.Fatalf("got %q, %v, want %q", got, err, tc.want)
			}
		})
	}
}

func TestRejectInvalidPatchShapes(t *testing.T) {
	const head = "--- document\n+++ document\n"
	const hunk = "@@ -1 +1 @@\n-a\n+b\n"
	for _, tc := range []string{
		"", head, "\x00" + head + hunk, head + hunk + "garbage\n", head + hunk + head + hunk,
		"diff --git a/document b/document\n" + head + hunk, "old mode 100644\nnew mode 100755\n" + head + hunk,
		"--- /dev/null\n+++ document\n@@ -0,0 +1 @@\n+b\n",
		"--- document\n+++ /dev/null\n@@ -1 +0,0 @@\n-a\n",
		head + "@@ -1,2 +1 @@\n-a\n+b\n", head + hunk + "\\ not a valid marker\n",
	} {
		_, err := Patch("md", []byte("a\n"), []byte(tc), false)
		var input *InputError
		if !errors.As(err, &input) {
			t.Errorf("expected input error for %q, got %v", tc, err)
		}
	}
}

func TestPatchContextAndLineEndings(t *testing.T) {
	const head = "--- document\n+++ document\n"
	for _, tc := range []struct{ base, hunk string }{
		{"a\nb\n", "@@ -1 +1 @@\n-b\n+B\n"},
		{"a\nb\n", "@@ -1 +2 @@\n-a\n+A\n"},
		{"a\r\n", "@@ -1 +1 @@\n-a\n+A\n"},
		{"a\n", "@@ -2,0 +3 @@\n+x\n"},
		{"a\nb\n", "@@ -2 +2 @@\n-b\n+B\n@@ -1 +1 @@\n-a\n+A\n"},
	} {
		_, err := Patch("md", []byte(tc.base), []byte(head+tc.hunk), false)
		if !errors.Is(err, ErrPatchContext) {
			t.Errorf("expected conflict for %q, got %v", tc.hunk, err)
		}
	}
}

func TestJSONMergePatchExamples(t *testing.T) {
	// Cases exercise the operations in RFC 7396 Appendix A.
	for _, tc := range []struct{ base, patch, want string }{
		{`{"a":"b"}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"b"}`, `{"b":"c"}`, `{"a":"b","b":"c"}`},
		{`{"a":"b","b":"c"}`, `{"a":null}`, `{"b":"c"}`},
		{`{"a":["b"]}`, `{"a":"c"}`, `{"a":"c"}`},
		{`{"a":"c"}`, `{"a":["b"]}`, `{"a":["b"]}`},
		{`{"a":{"b":"c"}}`, `{"a":{"b":"d","c":null}}`, `{"a":{"b":"d"}}`},
		{`{"a":[{"b":"c"}]}`, `{"a":[1]}`, `{"a":[1]}`},
		{`["a","b"]`, `["c","d"]`, `["c","d"]`},
		{`{"a":"b"}`, `["c"]`, `["c"]`},
		{`{"a":"foo"}`, `null`, `null`},
		{`{"a":"foo"}`, `"bar"`, `"bar"`},
		{`{"e":null}`, `{"a":1}`, `{"a":1,"e":null}`},
		{`[1,2]`, `{"a":"b","c":null}`, `{"a":"b"}`},
		{`{}`, `{"a":{"bb":{"ccc":null}}}`, `{"a":{"bb":{}}}`},
		{`{"n":90071992547409931234,"a":[]}`, `{"a":[null,1.00e+01,{"n":null}]}`, `{"a":[null,1.00e+01,{"n":null}],"n":90071992547409931234}`},
	} {
		result, err := Patch("json", []byte(tc.base), []byte(tc.patch), false)
		expected, formatErr := Format("json", []byte(tc.want), false)
		if err != nil || formatErr != nil || string(result.Content) != string(expected) || result.FormattingChanged != nil {
			t.Errorf("%s + %s: %q, %v, want %q", tc.base, tc.patch, result.Content, err, expected)
		}
	}
}

func TestPatchCSVQuotedPhysicalLines(t *testing.T) {
	const base = "id,note\nb,\"line one\nline two\"\n"
	const patch = "--- document\n+++ document\n@@ -2,2 +2,3 @@\n b,\"line one\n-line two\"\n+changed\"\n+a,first\n"
	for _, keep := range []bool{false, true} {
		got, err := Patch("csv", []byte(base), []byte(patch), keep)
		want := "id,note\na,first\nb,\"line one\nchanged\"\n"
		if keep {
			want = "id,note\nb,\"line one\nchanged\"\na,first\n"
		}
		if err != nil || string(got.Content) != want || got.FormattingChanged == nil || *got.FormattingChanged == keep {
			t.Fatalf("keep=%v: got %q, %v, want %q", keep, got.Content, err, want)
		}
	}
}
