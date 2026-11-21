package board

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

type SearchOptions struct {
	Scope, Assignee, Milestone, Concept, Query string
	Statuses                                   []string
}

func (o SearchOptions) Validate() error {
	if o.Scope != "" && !validScope(o.Scope) {
		return fmt.Errorf("invalid --in scope")
	}
	if o.Assignee != "" {
		if _, _, ok := principalParts(o.Assignee); !ok {
			return fmt.Errorf("invalid --assignee")
		}
	}
	for _, status := range o.Statuses {
		if !slices.Contains(statuses, status) {
			return fmt.Errorf("invalid --status %q", status)
		}
	}
	if o.Milestone != "" && !validFilter(o.Milestone, false) {
		return fmt.Errorf("invalid --milestone filter")
	}
	if o.Concept != "" && !validFilter(o.Concept, true) {
		return fmt.Errorf("invalid --concept filter")
	}
	return nil
}

func validLabel(label string, concept bool) bool {
	if concept {
		return len(label) <= 48 && conceptPattern.MatchString(label)
	}
	if len(label) > 64 || !milestonePattern.MatchString(label) {
		return false
	}
	for segment := range strings.SplitSeq(label, "/") {
		if strings.HasSuffix(segment, " ") {
			return false
		}
	}
	return true
}

func validFilter(filter string, concept bool) bool {
	filter = strings.TrimSpace(filter)
	prefix, wild := strings.CutSuffix(filter, "*")
	if wild && prefix == "" {
		return true
	}
	if wild && (strings.HasSuffix(prefix, "/") || strings.HasSuffix(prefix, " ")) {
		return validLabel(prefix+"a", concept)
	}
	return validLabel(prefix, concept)
}

func matchFilter(filter, value string) bool {
	if filter == "" {
		return true
	}
	if value == "" {
		return false
	}
	term, prefix := strings.CutSuffix(strings.ToLower(strings.TrimSpace(filter)), "*")
	if prefix {
		return strings.HasPrefix(strings.ToLower(value), term)
	}
	return strings.EqualFold(value, term)
}

func (d *Documents) Search(o SearchOptions) ([]DocumentMetadata, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	readable := map[string]bool{}
	var matched []document
	targets := make(map[string]document, len(d.catalog.Documents))
	for _, doc := range d.catalog.Documents {
		targets[doc.ID] = doc
		canRead, cached := readable[doc.Scope]
		if !cached {
			canRead = d.allowed(doc.Scope, "read")
			readable[doc.Scope] = canRead
		}
		if canRead && o.matches(doc) {
			matched = append(matched, doc)
		}
	}
	slices.SortFunc(matched, compareDocuments)
	results := make([]DocumentMetadata, 0, len(matched))
	availability := map[string]bool{}
	for _, doc := range matched {
		item := doc.metadata()
		item.Reviews = make(map[string]Review, len(doc.Reviews))
		for author, review := range doc.Reviews {
			if review.Locator != "" && !locatorAvailable(d, targets[review.Locator], readable, availability) {
				review.Locator = ""
			}
			item.Reviews[author] = review
		}
		results = append(results, item)
	}
	return results, nil
}

func locatorAvailable(d *Documents, target document, readable, cache map[string]bool) bool {
	if !readable[target.Scope] {
		return false
	}
	if exists, cached := cache[target.ID]; cached {
		return exists
	}
	file, err := d.openDocument(target)
	if err == nil {
		file.Close()
	}
	cache[target.ID] = err == nil
	return err == nil
}

func compareDocuments(a, b document) int {
	return cmp.Or(cmp.Compare(a.Scope, b.Scope), cmp.Compare(a.Name, b.Name),
		cmp.Compare(strings.ToLower(a.Milestone), strings.ToLower(b.Milestone)), cmp.Compare(a.ID, b.ID))
}

func (o SearchOptions) matches(doc document) bool {
	if (o.Scope != "" && o.Scope != doc.Scope) || (o.Assignee != "" && o.Assignee != doc.Assignee) ||
		(len(o.Statuses) > 0 && !slices.Contains(o.Statuses, doc.Status)) ||
		!matchFilter(o.Milestone, doc.Milestone) || !matchFilter(o.Concept, doc.Concept) {
		return false
	}
	query := strings.ToLower(o.Query)
	for _, text := range []string{doc.Name, doc.Description, doc.Milestone, doc.Concept} {
		if strings.Contains(strings.ToLower(text), query) {
			return true
		}
	}
	return false
}
