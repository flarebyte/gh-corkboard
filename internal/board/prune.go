package board

import (
	"cmp"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
)

type PruneCandidate struct {
	ID   string `json:"id"`
	Path string `json:"path"`
}

type PruneSkip struct {
	PruneCandidate
	Reason string `json:"reason"`
}

type PrunePlan struct {
	ID         string           `json:"planId"`
	Candidates []PruneCandidate `json:"candidates"`
	Skipped    []PruneSkip      `json:"skipped"`
	documents  []document
}

type PruneResult struct {
	PlanID         string           `json:"planId"`
	Deleted        []string         `json:"deleted"`
	Remaining      []string         `json:"remaining"`
	DeletedEntries []PruneCandidate `json:"-"`
}

type PartialPruneError struct {
	Result PruneResult
	Err    error
}

func (e *PartialPruneError) Error() string {
	return fmt.Sprintf("%s\nCompleted IDs: %s\nRemaining IDs: %s", e.Err,
		pruneIDs(e.Result.Deleted), pruneIDs(e.Result.Remaining))
}

func (e *PartialPruneError) Unwrap() error { return e.Err }

func pruneIDs(ids []string) string {
	if len(ids) == 0 {
		return "(none)"
	}
	return strings.Join(ids, ", ")
}

var ErrPrunePlanMismatch = errors.New("prune plan changed. Run index prune again to review the current candidates")

func ValidatePlanID(id string) error {
	if !revisionToken.MatchString(id) {
		return errors.New("--expect must be a 64-character lowercase hexadecimal plan ID from index prune")
	}
	return nil
}

func (a *Access) PreviewPrune() (PrunePlan, error) {
	if a.policy.role(a.actor) != "admin" {
		return PrunePlan{}, errors.New("index prune requires a board administrator")
	}
	current, err := loadCatalog(a.index, a.policy)
	if err != nil {
		return PrunePlan{}, err
	}
	return a.prunePlan(current)
}

func (a *Access) prunePlan(current catalog) (PrunePlan, error) {
	var archived []document
	for _, doc := range current.Documents {
		if doc.Status == "archived" {
			archived = append(archived, doc)
		}
	}
	slices.SortFunc(archived, func(a, b document) int { return cmp.Compare(a.ID, b.ID) })
	plan := PrunePlan{Candidates: []PruneCandidate{}, Skipped: []PruneSkip{}}
	hash := sha256.New()
	for _, doc := range archived {
		candidate := PruneCandidate{doc.ID, doc.Path}
		reason, err := a.pruneEligibility(doc)
		if err != nil {
			return PrunePlan{}, fmt.Errorf("check prune candidate %q: %w", doc.Path, err)
		}
		if reason != "" {
			plan.Skipped = append(plan.Skipped, PruneSkip{candidate, reason})
			continue
		}
		plan.Candidates = append(plan.Candidates, candidate)
		plan.documents = append(plan.documents, doc)
		// These validated fields cannot contain LF. A fixed field order and UUID
		// sort give the same identity to the same candidate set across platforms.
		fmt.Fprintf(hash, "%s\n%s\n%s\n", doc.ID, doc.Path, doc.Digest)
	}
	plan.ID = fmt.Sprintf("%x", hash.Sum(nil))
	return plan, nil
}

func (a *Access) pruneEligibility(doc document) (string, error) {
	folder, err := a.scopeRoot(doc.Scope)
	if os.IsNotExist(err) {
		return "missing", nil
	}
	if err != nil {
		return "", err
	}
	defer folder.Close()
	reason, _, err := comparePruneFile(folder, doc)
	return reason, err
}

func comparePruneFile(folder *os.Root, doc document) (string, os.FileInfo, error) {
	digest, info, err := fileDigest(folder, path.Base(doc.Path))
	if os.IsNotExist(err) {
		return "missing", nil, nil
	}
	if err != nil {
		return "", nil, err
	}
	if digest != doc.Digest {
		return "changed", nil, nil
	}
	return "", info, nil
}

func (a *Access) ApplyPrune(expected string) (PruneResult, error) {
	return a.applyPrune(expected, removeDocumentFile, (*Documents).publishCatalog)
}

func (a *Access) applyPrune(expected string, remove func(*os.Root, string) error, publish func(*Documents, catalog, []byte) error) (PruneResult, error) {
	if err := a.requireAdminEdit(); err != nil {
		return PruneResult{}, err
	}
	if err := ValidatePlanID(expected); err != nil {
		return PruneResult{}, err
	}
	current, err := loadCatalog(a.index, a.policy)
	if err != nil {
		return PruneResult{}, err
	}
	plan, err := a.prunePlan(current)
	if err != nil {
		return PruneResult{}, err
	}
	if plan.ID != expected {
		return PruneResult{}, ErrPrunePlanMismatch
	}

	result := PruneResult{PlanID: plan.ID, Deleted: []string{}, Remaining: []string{}}
	var failure error
	for i, doc := range plan.documents {
		if err := a.unlinkCandidate(doc, remove); err != nil {
			failure = fmt.Errorf("prune stopped at %q: %w", doc.Path, err)
			for _, remaining := range plan.documents[i:] {
				result.Remaining = append(result.Remaining, remaining.ID)
			}
			break
		}
		result.Deleted = append(result.Deleted, doc.ID)
		result.DeletedEntries = append(result.DeletedEntries, plan.Candidates[i])
	}
	if len(result.Deleted) > 0 {
		if err := a.reconcilePruned(current, result.Deleted, publish); err != nil {
			failure = errors.Join(failure, fmt.Errorf("prune could not save the reconciled index. Run index audit to inspect drift: %w", err))
		}
	}
	if failure != nil {
		return result, &PartialPruneError{Result: result, Err: failure}
	}
	return result, nil
}

func (a *Access) unlinkCandidate(doc document, remove func(*os.Root, string) error) error {
	folder, err := a.scopeRoot(doc.Scope)
	if err != nil {
		return err
	}
	defer folder.Close()
	reason, info, err := comparePruneFile(folder, doc)
	if err != nil {
		return err
	}
	if reason != "" {
		return fmt.Errorf("document became %s", reason)
	}
	name := path.Base(doc.Path)
	current, err := folder.Lstat(name)
	if err != nil {
		return err
	}
	if !current.Mode().IsRegular() || !os.SameFile(info, current) {
		return errors.New("document changed before unlink")
	}
	return remove(folder, name)
}

func (a *Access) reconcilePruned(current catalog, deleted []string, publish func(*Documents, catalog, []byte) error) error {
	removed := make(map[string]bool, len(deleted))
	for _, id := range deleted {
		removed[id] = true
	}
	next := current
	next.Documents = slices.DeleteFunc(slices.Clone(current.Documents), func(doc document) bool { return removed[doc.ID] })
	d := &Documents{access: a, catalog: current}
	encoded, err := d.prepareCatalog(next)
	if err != nil {
		return err
	}
	return publish(d, next, encoded)
}
