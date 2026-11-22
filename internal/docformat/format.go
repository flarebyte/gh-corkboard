package docformat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"unicode/utf8"

	"github.com/flarebyte/gh-corkboard/internal/strictjson"
)

type InputError struct {
	Format string
	Err    error
}

func (e *InputError) Error() string { return e.Err.Error() }
func (e *InputError) Unwrap() error { return e.Err }

func Valid(format string) bool {
	return format == "md" || format == "json" || format == "csv"
}

// Format normalizes valid document content while preserving its values.
func Format(format string, input []byte, preserveRowOrder bool) ([]byte, error) {
	if !Valid(format) {
		return nil, fmt.Errorf("unsupported format %q", format)
	}
	if preserveRowOrder && format != "csv" {
		return nil, fmt.Errorf("--preserve-row-order requires CSV content")
	}
	if !utf8.Valid(input) {
		return nil, &InputError{format, fmt.Errorf("content is not valid UTF-8")}
	}

	switch format {
	case "md":
		return markdown(input), nil
	case "csv":
		table, err := ParseCSV(string(input))
		if err != nil {
			return nil, &InputError{format, err}
		}
		return formatCSV(table, preserveRowOrder), nil
	default:
		value, err := parseJSON(input)
		if err != nil {
			return nil, err
		}
		return encodeJSON(value)
	}
}

func parseJSON(input []byte) (any, error) {
	value, err := strictjson.Decode(input, 10000)
	if err != nil {
		return nil, &InputError{"json", err}
	}
	return value, nil
}

func encodeJSON(value any) ([]byte, error) {
	var output bytes.Buffer
	enc := json.NewEncoder(&output)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func markdown(input []byte) []byte {
	normalized := bytes.ReplaceAll(input, []byte("\r\n"), []byte("\n"))
	normalized = bytes.ReplaceAll(normalized, []byte("\r"), []byte("\n"))
	if len(normalized) != 0 && normalized[len(normalized)-1] != '\n' {
		normalized = append(normalized, '\n')
	}
	return normalized
}
