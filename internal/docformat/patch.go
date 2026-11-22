package docformat

import (
	"bytes"
	"errors"
	"fmt"
)

var ErrPatchContext = errors.New("patch hunk does not match the current content and position")

type PatchResult struct {
	Content           []byte
	FormattingChanged *bool
}

func Patch(format string, base, patch []byte, preserveRowOrder bool) (PatchResult, error) {
	if !Valid(format) {
		return PatchResult{}, fmt.Errorf("unsupported format %q", format)
	}
	if preserveRowOrder && format != "csv" {
		return PatchResult{}, errors.New("--preserve-row-order requires a CSV document")
	}
	if format == "json" {
		content, err := mergeJSON(base, patch)
		return PatchResult{Content: content}, err
	}
	modified, err := patchText(base, patch, format)
	if err != nil {
		return PatchResult{}, err
	}
	content, err := Format(format, modified, preserveRowOrder)
	if err != nil {
		return PatchResult{}, err
	}
	changed := !bytes.Equal(modified, content)
	return PatchResult{Content: content, FormattingChanged: &changed}, nil
}
