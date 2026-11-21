package board

import (
	"cmp"
	"errors"
	"os"
	"slices"
	"strconv"
)

const catalogPath = "index/documents.json"

type IndexFinding struct {
	Code        string `json:"code"`
	Path        string `json:"path"`
	RelatedPath string `json:"relatedPath,omitempty"`
	Message     string `json:"message,omitempty"`
}

func (f IndexFinding) Description() string {
	var description string
	switch f.Code {
	case "missing_index":
		description = "Missing index"
	case "invalid_index":
		description = "Invalid index"
	case "missing_document":
		description = "Missing document"
	case "unindexed_document":
		description = "Unindexed document"
	case "digest_mismatch":
		description = "Document digest changed"
	case "unsupported_entry":
		description = "Unsupported entry"
	case "unsafe_entry":
		description = "Unsafe document entry"
	case "unsafe_scope":
		description = "Unsafe scope"
	case "duplicate_uuid":
		description = "Duplicate document UUID"
	case "uuid_at_different_path":
		description = "Indexed UUID at a different path"
	case "name_collision":
		description = "Conflicting document name"
	default:
		description = "Index finding"
	}
	text := description + ": " + displayIndexPath(f.Path)
	if f.RelatedPath != "" {
		text += " (conflicts with " + displayIndexPath(f.RelatedPath) + ")"
	}
	if f.Message != "" {
		text += ": " + f.Message
	}
	return text
}

func displayIndexPath(path string) string {
	if !displayText(path, false) {
		return strconv.Quote(path)
	}
	return path
}

func (a *Access) AuditIndex() ([]IndexFinding, error) {
	if a.policy.role(a.actor) != "admin" {
		return nil, errors.New("index audit requires a board administrator")
	}
	previous, missing, err := a.readReconcileCatalog()
	if err != nil {
		return []IndexFinding{{Code: "invalid_index", Path: catalogPath, Message: err.Error()}}, nil
	}
	files, findings, err := a.inventory()
	if err != nil {
		return nil, err
	}
	if missing {
		findings = append(findings, IndexFinding{Code: "missing_index", Path: catalogPath})
	}
	plan := planReconcile(previous, files)
	findings = append(findings, plan.findings...)
	sortFindings(findings)
	return findings, nil
}

func (a *Access) readReconcileCatalog() (catalog, bool, error) {
	cat, err := loadCatalog(a.index, a.policy)
	if errors.Is(err, os.ErrNotExist) {
		return catalog{Version: 1, Documents: []document{}}, true, nil
	}
	return cat, false, err
}

func sortFindings(findings []IndexFinding) {
	slices.SortFunc(findings, func(a, b IndexFinding) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Code, b.Code), cmp.Compare(a.RelatedPath, b.RelatedPath))
	})
}

func conflicting(code, path, related string) IndexFinding {
	return IndexFinding{Code: code, Path: path, RelatedPath: related}
}

func comparisonFinding(code, path string) IndexFinding {
	return IndexFinding{Code: code, Path: path}
}

type reconciliation struct {
	existing []document
	added    []diskDocument
	findings []IndexFinding
	blocking []IndexFinding
	report   ResyncResult
}

func planReconcile(previous catalog, files []diskDocument) reconciliation {
	plan := reconciliation{report: newResyncResult()}
	byID := map[string]document{}
	byPath := map[string]document{}
	for _, doc := range previous.Documents {
		byID[doc.ID], byPath[doc.Path] = doc, doc
	}
	ids := map[string]string{}
	names := map[[3]string]string{}
	for _, file := range files {
		if file.ID != "" {
			if other, exists := ids[file.ID]; exists {
				plan.blocking = append(plan.blocking, conflicting("duplicate_uuid", file.source, other))
			}
			ids[file.ID] = file.source
			if indexed, exists := byID[file.ID]; exists && indexed.Path != file.source {
				plan.blocking = append(plan.blocking, conflicting("uuid_at_different_path", file.source, indexed.Path))
			}
		}
		entry := file.document
		if indexed, exists := byPath[file.source]; exists {
			delete(byPath, file.source)
			entry = plan.recordExisting(indexed, file.Digest)
		} else {
			plan.added = append(plan.added, file)
			plan.findings = append(plan.findings, comparisonFinding("unindexed_document", file.source))
		}
		key := entry.nameKey()
		if other, exists := names[key]; exists {
			plan.blocking = append(plan.blocking, conflicting("name_collision", file.source, other))
		}
		names[key] = file.source
	}
	for _, doc := range previous.Documents {
		if _, missing := byPath[doc.Path]; missing {
			plan.findings = append(plan.findings, comparisonFinding("missing_document", doc.Path))
			plan.report.Removed = append(plan.report.Removed, doc.indexChange())
		}
	}
	plan.findings = append(plan.findings, plan.blocking...)
	return plan
}

func (p *reconciliation) recordExisting(doc document, digest string) document {
	if doc.Digest != digest {
		status := doc.Status
		doc.recordDigest(digest, true, false)
		p.findings = append(p.findings, comparisonFinding("digest_mismatch", doc.Path))
		p.report.Updated = append(p.report.Updated, doc.indexChange())
		if status != doc.Status {
			p.report.ResetToDraft = append(p.report.ResetToDraft, doc.indexChange())
		}
	}
	p.existing = append(p.existing, doc)
	return doc
}
