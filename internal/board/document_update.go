package board

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/flarebyte/gh-corkboard/internal/docformat"
)

type UpdateOptions struct {
	Content                                                   *[]byte
	Description, Milestone, Concept, Assignee, Status, Expect *string
	ReviewComment, ReviewStatus, ReviewLocator                *string
	ClearReview, PreserveRowOrder                             bool
}

type UpdateResult struct {
	DocumentMetadata
	FormattingChanged *bool `json:"formattingChanged,omitempty"`
}

var reviewStatuses = []string{"pass", "warning", "fail"}
var revisionToken = regexp.MustCompile(`^[0-9a-f]{64}$`)
var ErrRevisionMismatch = errors.New("document changed since it was read. Get the current content and revision before retrying")

func (o UpdateOptions) needsWrite() bool {
	return o.Content != nil || o.Description != nil || o.Milestone != nil || o.Concept != nil || o.Assignee != nil
}

func (o UpdateOptions) editsReview() bool {
	return o.ReviewComment != nil || o.ReviewStatus != nil || o.ReviewLocator != nil
}

func (o UpdateOptions) needsStatus() bool {
	return o.Status != nil || o.editsReview() || o.ClearReview
}

func (o UpdateOptions) validate(format string) error {
	switch {
	case !o.needsWrite() && !o.needsStatus():
		return errors.New("doc update requires at least one change flag")
	case o.PreserveRowOrder && (o.Content == nil || format != "csv"):
		return errors.New("--preserve-row-order requires --stdin for a CSV document")
	case o.Status != nil && !slices.Contains(statuses, *o.Status):
		return errors.New("--status must be a lifecycle status")
	case o.ClearReview && o.editsReview():
		return errors.New("--clear-review-comment cannot be combined with review fields")
	case o.editsReview() && (o.ReviewComment == nil || o.ReviewStatus == nil):
		return errors.New("--review-comment and --review-status must be supplied together")
	case o.ReviewStatus != nil && !slices.Contains(reviewStatuses, *o.ReviewStatus):
		return errors.New("--review-status must be pass, warning, or fail")
	case o.ReviewComment != nil && !validReviewComment(*o.ReviewComment):
		return errors.New("--review-comment must contain nonblank display text of at most 2000 characters")
	}
	if o.Expect != nil {
		return ValidateRevision(*o.Expect)
	}
	return nil
}

func validReviewComment(comment string) bool {
	return utf8.ValidString(comment) && strings.TrimSpace(comment) != "" &&
		utf8.RuneCountInString(comment) <= 2000 && displayText(comment, true)
}

func ValidateRevision(expected string) error {
	if !revisionToken.MatchString(expected) {
		return errors.New("--expect must be a 64-character lowercase hexadecimal revision from doc get --with-revision")
	}
	return nil
}

func checkRevision(expected *string, current []byte) error {
	if expected != nil && *expected != revision(current) {
		return ErrRevisionMismatch
	}
	return nil
}

func (d *Documents) Update(id string, o UpdateOptions) (UpdateResult, error) {
	var result UpdateResult
	if !d.access.edit {
		return result, errors.New("document update requires an exclusive board lock")
	}
	doc, index, err := d.selectDocument(id, "read")
	if err != nil {
		return result, err
	}
	if err := o.validate(doc.Format); err != nil {
		return result, err
	}
	if o.needsWrite() && !d.allowed(doc.Scope, "write") {
		return result, errors.New("scope write permission required")
	}
	if o.needsStatus() && !d.allowed(doc.Scope, "status") {
		return result, errors.New("scope status permission required")
	}

	scope, current, err := d.openUpdateSource(doc, o.Expect, o.Content != nil)
	if err != nil {
		return result, err
	}
	defer scope.Close()

	if err := d.applyUpdate(&doc, o); err != nil {
		return result, err
	}
	var replacement *[]byte
	if o.Content != nil {
		formatted, err := docformat.Format(doc.Format, *o.Content, o.PreserveRowOrder)
		if err != nil {
			return result, err
		}
		formattingChanged := !bytes.Equal(*o.Content, formatted)
		result.FormattingChanged = &formattingChanged
		replacement = doc.trackReplacement(current, formatted, o.Status != nil)
	}
	if err := d.commitUpdate(scope, index, doc, replacement); err != nil {
		return UpdateResult{}, err
	}
	result.DocumentMetadata = doc.visibleMetadata(d.canReadLocator)
	return result, nil
}

func (d *Documents) applyUpdate(doc *document, o UpdateOptions) error {
	if o.Description != nil {
		doc.Description = *o.Description
	}
	if o.Milestone != nil {
		doc.Milestone = strings.TrimSpace(*o.Milestone)
	}
	if o.Concept != nil {
		doc.Concept = strings.TrimSpace(*o.Concept)
	}
	if err := validateDocumentLabels(doc.Description, doc.Milestone, doc.Concept); err != nil {
		return err
	}
	if o.Milestone != nil && d.nameTaken(doc.Scope, doc.Name, doc.Milestone, doc.ID) {
		return fmt.Errorf("%s already exists in %s for this milestone", doc.Name, doc.Scope)
	}
	if o.Assignee != nil {
		if err := d.access.policy.validateAssignee(*o.Assignee, doc.Scope); err != nil {
			return err
		}
		doc.Assignee = *o.Assignee
	}
	if o.Status != nil {
		doc.Status = *o.Status
	}
	if o.ClearReview || o.editsReview() {
		doc.Reviews = maps.Clone(doc.Reviews)
		if o.ClearReview {
			delete(doc.Reviews, d.access.actor)
			return nil
		}
		entry := review{Comment: *o.ReviewComment, Status: *o.ReviewStatus}
		if o.ReviewLocator != nil {
			if !uuidV4.MatchString(*o.ReviewLocator) {
				return errors.New("--review-locator must be a canonical UUIDv4")
			}
			if !d.canReadLocator(*o.ReviewLocator) {
				return documentUnavailable
			}
			entry.Locator = *o.ReviewLocator
		}
		doc.Reviews[d.access.actor] = entry
	}
	return nil
}

func (d *Documents) openUpdateSource(doc document, expected *string, readBytes bool) (*os.Root, []byte, error) {
	scope, err := d.scopeRoot(doc.Scope)
	if os.IsNotExist(err) {
		return nil, nil, documentUnavailable
	}
	if err != nil {
		return nil, nil, err
	}
	current, err := currentDocument(scope, doc, readBytes || expected != nil)
	if err == nil {
		err = checkRevision(expected, current)
	}
	if err != nil {
		scope.Close()
		return nil, nil, err
	}
	return scope, current, nil
}

func (doc *document) trackReplacement(current, formatted []byte, explicitStatus bool) *[]byte {
	changed := !bytes.Equal(current, formatted)
	digest := revision(formatted)
	if !explicitStatus && doc.Status == "archived" && (digest != doc.Digest || changed) {
		doc.Status = "draft"
	}
	doc.Digest = digest
	if changed {
		return &formatted
	}
	return nil
}

func currentDocument(scope *os.Root, doc document, readBytes bool) ([]byte, error) {
	file, err := openRegular(scope, path.Base(doc.Path), os.O_RDONLY)
	if os.IsNotExist(err) {
		return nil, documentUnavailable
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if !readBytes {
		return nil, nil
	}
	return io.ReadAll(file)
}

func (d *Documents) commitUpdate(scope *os.Root, index int, doc document, replacement *[]byte) error {
	next := d.catalog
	next.Documents = slices.Clone(next.Documents)
	next.Documents[index] = doc
	encoded, err := d.prepareCatalog(next)
	if err != nil {
		return err
	}
	if replacement != nil {
		if err := replaceFile(scope, path.Base(doc.Path), *replacement, 0600); err != nil {
			return err
		}
	}
	if err := d.publishCatalog(next, encoded); err != nil {
		if replacement != nil {
			return fmt.Errorf("document %s was written but its index entry was not saved: %w", doc.ID, err)
		}
		return err
	}
	return nil
}
