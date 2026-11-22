package board

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
)

type IndexChange struct {
	ID     string `json:"id"`
	Path   string `json:"path"`
	Status string `json:"status"`
}

type ResyncResult struct {
	Added        []IndexChange  `json:"added"`
	Removed      []IndexChange  `json:"removed"`
	Updated      []IndexChange  `json:"updated"`
	ResetToDraft []IndexChange  `json:"resetToDraft"`
	Rebuilt      bool           `json:"rebuilt,omitempty"`
	Findings     []IndexFinding `json:"findings,omitempty"`
}

func newResyncResult() ResyncResult {
	return ResyncResult{Added: []IndexChange{}, Removed: []IndexChange{}, Updated: []IndexChange{}, ResetToDraft: []IndexChange{}}
}

func (doc document) indexChange() IndexChange {
	return IndexChange{doc.ID, doc.Path, doc.Status}
}

func (a *Access) ResyncIndex() (ResyncResult, error) {
	return a.resyncIndex(func(d *Documents, next catalog, data []byte) error {
		return d.publishCatalog(next, data)
	})
}

func (a *Access) resyncIndex(publish func(*Documents, catalog, []byte) error) (ResyncResult, error) {
	if err := a.requireAdminEdit(); err != nil {
		return ResyncResult{}, err
	}
	previous, missing, err := a.readReconcileCatalog()
	if err != nil {
		return ResyncResult{}, fmt.Errorf("preserving invalid index: %w", err)
	}
	files, findings, err := a.inventory()
	if err != nil {
		return ResyncResult{}, err
	}
	plan := planReconcile(previous, files)
	plan.report.Rebuilt = missing
	conflicts := plan.blocking
	for _, finding := range findings {
		if finding.Code == "unsupported_entry" {
			plan.report.Findings = append(plan.report.Findings, finding)
		} else {
			conflicts = append(conflicts, finding)
		}
	}
	if len(conflicts) > 0 {
		sortFindings(conflicts)
		var message strings.Builder
		message.WriteString("index resync cannot reconcile these entries")
		for _, finding := range conflicts {
			fmt.Fprintf(&message, "\n%s", finding.Description())
		}
		return ResyncResult{}, errors.New(message.String())
	}
	if !missing && len(plan.added)+len(plan.report.Removed)+len(plan.report.Updated) == 0 {
		return plan.report, nil
	}

	next := catalog{Version: 1, Documents: append([]document{}, plan.existing...)}
	var renames []diskDocument
	for _, candidate := range plan.added {
		if candidate.ID == "" {
			candidate.ID = newDocumentID()
		}
		candidate.Path = candidate.canonicalPath()
		if len(candidate.Path) > 244 {
			return ResyncResult{}, fmt.Errorf("canonical path for %q exceeds 244 bytes", candidate.source)
		}
		if candidate.Path != candidate.source {
			renames = append(renames, candidate)
		}
		next.Documents = append(next.Documents, candidate.document)
		plan.report.Added = append(plan.report.Added, candidate.document.indexChange())
	}

	docs := &Documents{access: a, catalog: previous}
	encoded, err := docs.prepareCatalog(next)
	if err != nil {
		return ResyncResult{}, err
	}
	for i, entry := range renames {
		if err := a.adoptFilename(entry); err != nil {
			return ResyncResult{}, fmt.Errorf("renamed %d document(s), index unchanged: %w", i, err)
		}
	}
	if err := publish(docs, next, encoded); err != nil {
		return ResyncResult{}, fmt.Errorf("renamed %d document(s), reconciled index was not saved: %w", len(renames), err)
	}
	return plan.report, nil
}

func (a *Access) adoptFilename(entry diskDocument) error {
	folder, err := a.scopeRoot(entry.Scope)
	if err != nil {
		return err
	}
	defer folder.Close()

	source, target := path.Base(entry.source), path.Base(entry.Path)
	digest, info, err := fileDigest(folder, source)
	if err != nil {
		return fmt.Errorf("recheck %q: %w", entry.source, err)
	}
	if !os.SameFile(entry.info, info) || entry.Digest != digest {
		return fmt.Errorf("%q changed during reconciliation", entry.source)
	}
	if err := renameEntryExclusive(folder, source, target); err != nil {
		return fmt.Errorf("rename %q to %q: %w", entry.source, entry.Path, err)
	}
	return nil
}
