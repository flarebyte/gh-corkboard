package board

import (
	"errors"

	"github.com/flarebyte/gh-corkboard/internal/docformat"
)

func (d *Documents) Patch(id, expected string, patch []byte, preserveRowOrder bool) (UpdateResult, error) {
	var result UpdateResult
	if !d.access.edit {
		return result, errors.New("document patch requires an exclusive board lock")
	}
	if err := ValidateRevision(expected); err != nil {
		return result, err
	}
	doc, index, err := d.selectDocument(id, "write")
	if err != nil {
		return result, err
	}
	if preserveRowOrder && doc.Format != "csv" {
		return result, errors.New("--preserve-row-order requires a CSV document")
	}

	scope, current, err := d.openUpdateSource(doc, &expected, true)
	if err != nil {
		return result, err
	}
	defer scope.Close()

	patched, err := docformat.Patch(doc.Format, current, patch, preserveRowOrder)
	if err != nil {
		return result, err
	}
	replacement := doc.trackReplacement(current, patched.Content, false)
	if err := d.commitUpdate(scope, index, doc, replacement); err != nil {
		return result, err
	}
	return UpdateResult{
		DocumentMetadata:  doc.visibleMetadata(d.canReadLocator),
		FormattingChanged: patched.FormattingChanged,
	}, nil
}
