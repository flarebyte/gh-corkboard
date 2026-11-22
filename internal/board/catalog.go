package board

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

type catalog struct {
	Version   int        `json:"version"`
	Documents []document `json:"documents"`
}

type document struct {
	ID          string            `json:"id"`
	Scope       string            `json:"scope"`
	Name        string            `json:"name"`
	Format      string            `json:"format"`
	SchemaID    string            `json:"schemaId,omitempty"`
	Path        string            `json:"path"`
	Description string            `json:"description"`
	Milestone   string            `json:"milestone,omitempty"`
	Concept     string            `json:"concept,omitempty"`
	Assignee    string            `json:"assignee,omitempty"`
	Status      string            `json:"status"`
	Digest      string            `json:"sha256"`
	Reviews     map[string]review `json:"reviewComments"`
}

type review struct {
	Comment string `json:"comment"`
	Status  string `json:"status"`
	Locator string `json:"locator,omitempty"`
}

func loadCatalog(index *os.Root, policy accessPolicy) (catalog, error) {
	cat, err := metadata[catalog](index, "documents", catalogLimit)
	if err != nil {
		return cat, fmt.Errorf("invalid document index: %w", err)
	}
	return cat, cat.validate(policy)
}

func (cat catalog) validate(policy accessPolicy) error {
	ids := map[string]bool{}
	names := map[[3]string]bool{}
	previous := ""
	for _, doc := range cat.Documents {
		if err := doc.validate(policy); err != nil {
			return err
		}
		key := doc.nameKey()
		if ids[doc.ID] || names[key] || doc.Path <= previous {
			return fmt.Errorf("duplicate or unordered catalog entry at %q", doc.Path)
		}
		ids[doc.ID], names[key], previous = true, true, doc.Path
	}
	return nil
}

func (d document) validate(policy accessPolicy) error {
	if !policy.hasScope(d.Scope) || !validSlug(d.Name) || (d.SchemaID != "" && !validSlug(d.SchemaID)) || d.Path != d.canonicalPath() {
		return fmt.Errorf("invalid scope or derived path for document %q", d.ID)
	}
	if !displayText(d.Description, false) {
		return fmt.Errorf("invalid description for document %q", d.ID)
	}
	for segment := range strings.SplitSeq(d.Milestone, "/") {
		if strings.HasSuffix(segment, " ") {
			return fmt.Errorf("invalid milestone for document %q", d.ID)
		}
	}
	if d.Assignee != "" && policy.role(d.Assignee) == "" {
		return fmt.Errorf("unregistered assignee for document %q", d.ID)
	}
	for author, r := range d.Reviews {
		if policy.role(author) == "" || strings.TrimSpace(r.Comment) == "" || !displayText(r.Comment, true) {
			return fmt.Errorf("invalid review for document %q by %q", d.ID, author)
		}
	}
	return nil
}

func displayText(value string, multiline bool) bool {
	for _, r := range value {
		if multiline && (r == '\n' || r == '\t') {
			continue
		}
		if unicode.IsControl(r) || unicode.In(r, unicode.Bidi_Control) || r == '\u2028' || r == '\u2029' {
			return false
		}
	}
	return true
}

func (d document) nameKey() [3]string {
	return [3]string{d.Scope, strings.ToLower(d.Milestone), d.Name}
}
