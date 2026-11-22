package docformat

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

func patchText(base, patch []byte, format string) ([]byte, error) {
	if !textBytes(base) {
		return nil, errors.New("stored document is not UTF-8 text")
	}
	invalid := func(err error) ([]byte, error) { return nil, &InputError{format, err} }
	if !textBytes(patch) || !bytes.HasSuffix(patch, []byte("\n")) {
		return invalid(errors.New("patch must be UTF-8 text with complete lines"))
	}
	head, rest, _ := bytes.Cut(patch, []byte("\n"))
	next, body, _ := bytes.Cut(rest, []byte("\n"))
	if !bytes.HasPrefix(head, []byte("--- ")) || !bytes.HasPrefix(next, []byte("+++ ")) {
		return invalid(errors.New("expected unified diff headers followed by one or more hunks"))
	}
	files, preamble, err := gitdiff.Parse(bytes.NewReader(patch))
	if err != nil {
		return invalid(err)
	}
	if preamble != "" || len(files) != 1 {
		return invalid(errors.New("expected a single-document unified diff"))
	}
	file := files[0]
	if file.IsNew || file.IsDelete || file.IsRename || file.IsCopy || file.IsBinary ||
		file.OldMode != 0 || file.NewMode != 0 ||
		file.OldName == "/dev/null" || file.NewName == "/dev/null" ||
		len(file.TextFragments) == 0 || !validHunkBody(string(body), file.TextFragments) {
		return invalid(errors.New("expected only unified diff hunks for an existing document"))
	}

	result, err := applyHunks(base, file.TextFragments)
	if err != nil {
		if errors.Is(err, ErrPatchContext) {
			return nil, err
		}
		return invalid(err)
	}
	return result, nil
}

func textBytes(data []byte) bool {
	return utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
}

// The library accepts preambles, trailing material and arbitrary text after a
// backslash marker. Match the parsed hunks against the complete input envelope.
func validHunkBody(body string, hunks []*gitdiff.TextFragment) bool {
	const noNewline = "\\ No newline at end of file"
	for _, hunk := range hunks {
		header, rest, _ := strings.Cut(body, "\n")
		if !strings.HasPrefix(header, "@@ -") {
			return false
		}
		body = rest
		for _, line := range hunk.Lines {
			physical, rest, ok := strings.Cut(body, "\n")
			if !ok {
				return false
			}
			body = rest
			data := physical + "\n"
			if physical != "" {
				data = physical[1:] + "\n"
			}
			marker, rest, _ := strings.Cut(body, "\n")
			if strings.TrimSuffix(marker, "\r") == noNewline {
				data = strings.TrimSuffix(data, "\n")
				body = rest
			}
			if data != line.Line {
				return false
			}
		}
	}
	return body == ""
}

func physicalLines(data []byte) int64 {
	count := int64(bytes.Count(data, []byte("\n")))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		count++
	}
	return count
}

func rangeStart(position, count int64) (int64, error) {
	if position < 0 || count < 0 || (position == 0 && count != 0) {
		return 0, errors.New("invalid unified diff range")
	}
	if count == 0 {
		return position, nil
	}
	return position - 1, nil
}

func applyHunks(base []byte, hunks []*gitdiff.TextFragment) ([]byte, error) {
	var out bytes.Buffer
	applier := gitdiff.NewTextApplier(&out, bytes.NewReader(base))
	sourceLines := physicalLines(base)
	var previousEnd, delta int64
	for n, hunk := range hunks {
		oldStart, err := rangeStart(hunk.OldPosition, hunk.OldLines)
		if err != nil {
			return nil, err
		}
		newStart, err := rangeStart(hunk.NewPosition, hunk.NewLines)
		if err != nil {
			return nil, err
		}
		if oldStart < previousEnd || oldStart > sourceLines || hunk.OldLines > sourceLines-oldStart ||
			newStart != oldStart+delta {
			return nil, fmt.Errorf("%w (hunk %d)", ErrPatchContext, n+1)
		}
		previousEnd = oldStart + hunk.OldLines
		delta += hunk.NewLines - hunk.OldLines

		// Unified diffs number an empty range after its preceding line. The
		// library interprets zero positions as file creation or full deletion.
		fragment := *hunk
		fragment.OldPosition = oldStart + 1
		fragment.NewPosition = newStart + 1
		if err := applier.ApplyFragment(&fragment); err != nil {
			var conflict *gitdiff.Conflict
			if errors.As(err, &conflict) {
				return nil, fmt.Errorf("%w (hunk %d)", ErrPatchContext, n+1)
			}
			return nil, err
		}
	}
	if err := applier.Close(); err != nil {
		return nil, err
	}
	if physicalLines(out.Bytes()) != sourceLines+delta {
		return nil, errors.New("patch produces inconsistent line boundaries")
	}
	return out.Bytes(), nil
}
