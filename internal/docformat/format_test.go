package docformat

import (
	"errors"
	"testing"
)

func TestFormat(t *testing.T) {
	for _, tc := range []struct {
		format, input, want string
		keep                bool
	}{
		{"json", `{"z":1.00e+01,"a":[0,{"z":3,"a":9007199254740993}]}`, "{\n  \"a\": [\n    0,\n    {\n      \"a\": 9007199254740993,\n      \"z\": 3\n    }\n  ],\n  \"z\": 1.00e+01\n}\n", false},
		{"md", "  prose  \r\n    code\r\n- a\r- b", "  prose  \n    code\n- a\n- b\n", false},
		{"csv", "ID,value\r\nb,01\r\na,\"line\r\nnext\"\r\na,\"quote\"\"a\"\r\n", "ID,value\na,\"line\r\nnext\"\na,\"quote\"\"a\"\nb,01\n", false},
		{"csv", "name,value\nb, 01 \na,02", "name,value\nb, 01 \na,02\n", true},
		{"csv", "key,value\nb,01\na,02", "key,value\nb,01\na,02\n", false},
		{"csv", "id\n\n\"\"\n", "id\n\"\"\n", false},
	} {
		formatted, err := Format(tc.format, []byte(tc.input), tc.keep)
		if err != nil || string(formatted) != tc.want {
			t.Fatalf("%s %q: %q, %v", tc.format, tc.input, formatted, err)
		}
		again, err := Format(tc.format, formatted, tc.keep)
		if err != nil || string(again) != tc.want {
			t.Fatalf("not idempotent: %q, %v", again, err)
		}
	}
}

func TestMalformedContent(t *testing.T) {
	for _, tc := range []struct{ format, input string }{
		{"json", `{"x":1,"x":2}`}, {"json", `{"x":`}, {"md", "\xff"},
		{"csv", ""}, {"csv", "id,id\na,b"}, {"csv", "id,\na,b"}, {"csv", "id\na,b"}, {"csv", "id\n\"a"}, {"csv", "id\n\"a\"b"},
	} {
		_, err := Format(tc.format, []byte(tc.input), false)
		var input *InputError
		if !errors.As(err, &input) {
			t.Fatalf("%s %q: expected input diagnostic, got %v", tc.format, tc.input, err)
		}
	}
}
