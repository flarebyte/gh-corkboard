package docformat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/flarebyte/gh-corkboard/internal/strictjson"
)

type Projection struct {
	Value  any
	offset func([]string) int
}

// Project parses document content and builds the v1 validation input.
func Project(format string, content []byte) (Projection, error) {
	if !utf8.Valid(content) {
		return Projection{}, errors.New("content is not valid UTF-8")
	}
	switch format {
	case "json":
		value, err := parseJSON(content)
		return Projection{Value: value}, err
	case "csv":
		return projectCSV(string(content))
	case "md":
		return projectMarkdown(content), nil
	}
	return Projection{}, fmt.Errorf("unsupported format %q", format)
}

func projectCSV(content string) (Projection, error) {
	var offsets [][]int
	table, err := parseCSV(content, &offsets)
	if err != nil {
		return Projection{}, err
	}
	columns := make([]any, len(table.Columns))
	for i, column := range table.Columns {
		columns[i] = column
	}
	rows := make([]any, 0, len(table.Rows))
	for _, row := range table.Rows {
		value := make(map[string]any, len(row))
		for i, cell := range row {
			value[table.Columns[i]] = cell
		}
		rows = append(rows, value)
	}
	return Projection{
		Value: map[string]any{"columns": columns, "rows": rows},
		offset: func(path []string) int {
			if len(path) < 2 {
				return -1
			}
			index, err := strconv.Atoi(path[1])
			if err != nil || index < 0 {
				return -1
			}
			switch path[0] {
			case "columns":
				if index < len(offsets[0]) {
					return offsets[0][index]
				}
			case "rows":
				if index >= len(rows) {
					return -1
				}
				col := 0
				if len(path) > 2 {
					col = slices.Index(table.Columns, path[2])
				}
				if col >= 0 {
					return offsets[index+1][col]
				}
			}
			return -1
		},
	}, nil
}

// Location maps projected paths back to source positions when a format records them.
func (p Projection) Location(source []byte) func([]string) (int, int) {
	if p.offset == nil {
		return func([]string) (int, int) { return 0, 0 }
	}
	starts := []int{0}
	for i, b := range source {
		if b == '\n' {
			starts = append(starts, i+1)
		}
	}
	return func(path []string) (int, int) {
		offset := p.offset(path)
		if offset < 0 || offset > len(source) {
			return 0, 0
		}
		line, found := slices.BinarySearch(starts, offset)
		if !found {
			line--
		}
		return line + 1, utf8.RuneCount(source[starts[line]:offset]) + 1
	}
}

func ErrorPosition(source []byte, err error) (int, int) {
	var native *json.SyntaxError
	if errors.As(err, &native) && native.Offset > 0 {
		return positionAt(source, int(native.Offset)-1)
	}
	var offset *strictjson.OffsetError
	if errors.As(err, &offset) {
		return positionAt(source, offset.Offset)
	}
	var csv *csvError
	if errors.As(err, &csv) {
		return positionAt(source, csv.offset)
	}
	return 0, 0
}

func positionAt(source []byte, offset int) (int, int) {
	if offset < 0 || offset > len(source) {
		return 0, 0
	}
	line := bytes.Count(source[:offset], []byte("\n")) + 1
	start := bytes.LastIndexByte(source[:offset], '\n') + 1
	return line, utf8.RuneCount(source[start:offset]) + 1
}
