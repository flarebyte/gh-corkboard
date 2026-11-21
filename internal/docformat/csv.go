package docformat

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

type CSV struct {
	Columns []string
	Rows    [][]string
}

// ParseCSV preserves cell strings, including line endings inside quoted fields.
func ParseCSV(input string) (CSV, error) {
	return parseCSV(input, nil)
}

func parseCSV(input string, offsets *[][]int) (CSV, error) {
	table := CSV{Rows: [][]string{}}
	length := len(input)
	if !utf8.ValidString(input) {
		return table, errors.New("content is not valid UTF-8")
	}
	for input != "" {
		if rest, blank := cutRecordEnd(input); blank {
			input = rest
			continue
		}
		start := length - len(input)
		var fields *[]int
		if offsets != nil {
			*offsets = append(*offsets, nil)
			fields = &(*offsets)[len(*offsets)-1]
		}
		row, rest, err := csvRecord(input, start, fields)
		if err != nil {
			return table, &csvError{start, fmt.Errorf("CSV record %d: %w", len(table.Rows)+recordOffset(table), err)}
		}
		input = rest
		if table.Columns == nil {
			if err := checkColumns(row); err != nil {
				return table, &csvError{start, err}
			}
			table.Columns = row
			continue
		}
		if len(row) != len(table.Columns) {
			return table, &csvError{start, fmt.Errorf("CSV record %d has %d fields, expected %d", len(table.Rows)+2, len(row), len(table.Columns))}
		}
		table.Rows = append(table.Rows, row)
	}
	if table.Columns == nil {
		return table, errors.New("CSV requires a header")
	}
	return table, nil
}

func recordOffset(table CSV) int {
	if table.Columns == nil {
		return 1
	}
	return 2
}

func checkColumns(columns []string) error {
	seen := make(map[string]bool, len(columns))
	for _, name := range columns {
		if name == "" || seen[name] {
			return fmt.Errorf("CSV header names must be nonempty and unique (found %q)", name)
		}
		seen[name] = true
	}
	return nil
}

type csvError struct {
	offset int
	err    error
}

func (e *csvError) Error() string { return e.err.Error() }
func (e *csvError) Unwrap() error { return e.err }

func csvRecord(input string, start int, offsets *[]int) ([]string, string, error) {
	var cells []string
	end := start + len(input)
	for {
		if offsets != nil {
			*offsets = append(*offsets, end-len(input))
		}
		cell, rest, err := csvField(input)
		if err != nil {
			return nil, "", err
		}
		cells = append(cells, cell)
		if rest == "" {
			return cells, "", nil
		}
		if after, end := cutRecordEnd(rest); end {
			return cells, after, nil
		}
		if rest[0] != ',' {
			return nil, "", errors.New("expected comma or record end after field")
		}
		input = rest[1:]
	}
}

func cutRecordEnd(input string) (string, bool) {
	if rest, ok := strings.CutPrefix(input, "\r\n"); ok {
		return rest, true
	}
	return strings.CutPrefix(input, "\n")
}

func csvField(input string) (string, string, error) {
	if after, quoted := strings.CutPrefix(input, "\""); quoted {
		var out strings.Builder
		for {
			before, rest, found := strings.Cut(after, "\"")
			if !found {
				return "", "", errors.New("unterminated quoted field")
			}
			out.WriteString(before)
			after, found = strings.CutPrefix(rest, "\"")
			if !found {
				return out.String(), rest, nil
			}
			out.WriteByte('"')
		}
	}
	end := strings.IndexAny(input, ",\"\n")
	if end < 0 {
		return input, "", nil
	}
	if input[end] == '"' {
		return "", "", errors.New("unexpected quote in unquoted field")
	}
	if input[end] == '\n' && end > 0 && input[end-1] == '\r' {
		end--
	}
	return input[:end], input[end:], nil
}

func formatCSV(table CSV, preserveRowOrder bool) []byte {
	first := table.Columns[0]
	if !preserveRowOrder && (strings.EqualFold(first, "id") || strings.EqualFold(first, "name")) {
		slices.SortStableFunc(table.Rows, func(a, b []string) int { return cmp.Compare(a[0], b[0]) })
	}
	var out strings.Builder
	writeCSVRecord(&out, table.Columns)
	for _, row := range table.Rows {
		writeCSVRecord(&out, row)
	}
	return []byte(out.String())
}

func writeCSVRecord(out *strings.Builder, row []string) {
	for i, cell := range row {
		if i != 0 {
			out.WriteByte(',')
		}
		if (cell == "" && len(row) == 1) || strings.ContainsAny(cell, ",\"\r\n") {
			out.WriteByte('"')
			out.WriteString(strings.ReplaceAll(cell, "\"", "\"\""))
			out.WriteByte('"')
		} else {
			out.WriteString(cell)
		}
	}
	out.WriteByte('\n')
}
