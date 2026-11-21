package board

import (
	"cmp"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/flarebyte/gh-corkboard/internal/docformat"
)

type Documents struct {
	access  *Access
	catalog catalog
}

func WithDocuments(dir, actor string, edit bool, operation func(*Documents) error) error {
	return withBoard(dir, actor, edit, func(a *Access) error {
		cat, err := loadCatalog(a.index, a.policy)
		if err != nil {
			return err
		}
		return operation(&Documents{access: a, catalog: cat})
	})
}

type DocumentMetadata struct {
	ID          string            `json:"id"`
	Scope       string            `json:"scope"`
	Name        string            `json:"name"`
	Format      string            `json:"format"`
	SchemaID    string            `json:"schemaId,omitempty"`
	Description string            `json:"description"`
	Milestone   string            `json:"milestone,omitempty"`
	Concept     string            `json:"concept,omitempty"`
	Assignee    string            `json:"assignee,omitempty"`
	Status      string            `json:"status"`
	Reviews     map[string]Review `json:"reviewComments"`
}

type Review = review

func (d document) metadata() DocumentMetadata {
	return DocumentMetadata{
		ID: d.ID, Scope: d.Scope, Name: d.Name, Format: d.Format, SchemaID: d.SchemaID, Description: d.Description,
		Milestone: d.Milestone, Concept: d.Concept, Assignee: d.Assignee, Status: d.Status, Reviews: d.Reviews,
	}
}

type CreateOptions struct {
	Scope, Name, Format, SchemaID, Description, Milestone, Concept, Assignee string
	Status                                                                   *string
	PreserveRowOrder                                                         bool
}

var (
	uuidV4              = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	milestonePattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*(/[A-Za-z0-9][A-Za-z0-9 ._-]*)*$`)
	conceptPattern      = regexp.MustCompile(`^[A-Za-z]+(/[A-Za-z]+)*$`)
	documentUnavailable = errors.New("document unavailable")
)

var statuses = []string{"draft", "active", "blocked", "review", "approved", "rejected", "resolved", "obsolete", "archived"}

func (o CreateOptions) validate() error {
	switch {
	case !validScope(o.Scope):
		return errors.New("--in must name a topic, agent, or skill scope")
	case !validSlug(o.Name):
		return errors.New("--name must be a valid slug")
	case !docformat.Valid(o.Format):
		return errors.New("--format must be md, json, or csv")
	case o.SchemaID != "" && !validSlug(o.SchemaID):
		return errors.New("--schema must be a valid slug")
	case o.PreserveRowOrder && o.Format != "csv":
		return errors.New("--preserve-row-order requires CSV content")
	case !utf8.ValidString(o.Description) || !displayText(o.Description, false) || utf8.RuneCountInString(o.Description) > 500:
		return errors.New("--description must be single-line display text of at most 500 characters")
	case o.Milestone != "" && !validLabel(o.Milestone, false):
		return errors.New("--milestone must be a valid slash-separated label of at most 64 characters")
	case o.Concept != "" && !validLabel(o.Concept, true):
		return errors.New("--concept must be 1–48 characters in slash-separated segments of ASCII letters")
	case o.Status != nil && !slices.Contains(statuses, *o.Status):
		return errors.New("--status must be a lifecycle status")
	}
	return nil
}

func (d *Documents) Create(o CreateOptions, input io.Reader) (DocumentMetadata, error) {
	var empty DocumentMetadata
	o.Milestone, o.Concept = strings.TrimSpace(o.Milestone), strings.TrimSpace(o.Concept)
	if err := o.validate(); err != nil {
		return empty, err
	}
	if !d.access.edit || !d.allowed(o.Scope, "write") {
		return empty, errors.New("scope write permission required")
	}
	status := "draft"
	if o.Status != nil {
		if !d.allowed(o.Scope, "status") {
			return empty, errors.New("scope status permission required")
		}
		status = *o.Status
	}
	if o.Assignee != "" && !slices.Contains(d.access.policy.permissions(o.Assignee, o.Scope), "read") {
		return empty, errors.New("--assignee must be a registered principal with scope read access")
	}
	for _, entry := range d.catalog.Documents {
		if entry.Scope == o.Scope && entry.Name == o.Name && strings.EqualFold(entry.Milestone, o.Milestone) {
			return empty, fmt.Errorf("%s already exists in %s for this milestone", o.Name, o.Scope)
		}
	}

	doc := document{
		ID: newDocumentID(), Scope: o.Scope, Name: o.Name, Format: o.Format, SchemaID: o.SchemaID,
		Description: o.Description, Milestone: o.Milestone, Concept: o.Concept, Assignee: o.Assignee,
		Status: status, Reviews: map[string]review{},
	}
	doc.Path = doc.canonicalPath()
	if len(doc.Path) > 244 {
		return empty, errors.New("--in, --name, and --schema produce a path longer than 244 bytes")
	}

	data, err := io.ReadAll(input)
	if err != nil {
		return empty, err
	}
	formatted, err := docformat.Format(o.Format, data, o.PreserveRowOrder)
	if err != nil {
		return empty, err
	}
	doc.Digest = revision(formatted)
	next := d.catalog
	next.Documents = append(slices.Clone(next.Documents), doc)
	encoded, err := d.prepareCatalog(next)
	if err != nil {
		return empty, err
	}

	scope, err := d.scopeRoot(doc.Scope)
	if err != nil {
		return empty, err
	}
	defer scope.Close()
	if err := createFile(scope, path.Base(doc.Path), formatted); err != nil {
		return empty, err
	}
	if err := d.publishCatalog(next, encoded); err != nil {
		return empty, fmt.Errorf("document %s was written but its index entry was not saved: %w", doc.ID, err)
	}
	return doc.metadata(), nil
}

func newDocumentID() string {
	var b [16]byte
	rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func revision(data []byte) string { return fmt.Sprintf("%x", sha256.Sum256(data)) }

func (d *Documents) prepareCatalog(next catalog) ([]byte, error) {
	slices.SortFunc(next.Documents, func(a, b document) int { return cmp.Compare(a.Path, b.Path) })
	if err := next.validate(d.access.policy); err != nil {
		return nil, err
	}
	encoded, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return nil, err
	}
	encoded = append(encoded, '\n')
	if _, err := decodeMetadata[catalog]("documents", encoded, catalogLimit); err != nil {
		return nil, err
	}
	return encoded, nil
}

func (d *Documents) publishCatalog(next catalog, data []byte) error {
	if err := replaceFile(d.access.index, "documents.json", data, 0600); err != nil {
		return err
	}
	d.catalog = next
	return nil
}

func (d *Documents) allowed(scope, permission string) bool {
	return slices.Contains(d.access.policy.permissions(d.access.actor, scope), permission)
}

func validScope(scope string) bool {
	kind, name, _ := strings.Cut(scope, ":")
	return (kind == "topic" || kind == "agent" || kind == "skill") && validSlug(name)
}

func (d *Documents) scopeRoot(scope string) (*os.Root, error) {
	kind, name, _ := strings.Cut(scope, ":")
	container, err := openDirectory(d.access.root, kind+"s")
	if err != nil {
		return nil, err
	}
	defer container.Close()
	return openDirectory(container, name)
}

func (d *Documents) openDocument(doc document) (*os.File, error) {
	scope, err := d.scopeRoot(doc.Scope)
	if err != nil {
		return nil, err
	}
	defer scope.Close()
	return openRegular(scope, path.Base(doc.Path), os.O_RDONLY)
}

func (d *Documents) documentIndex(id string) int {
	return slices.IndexFunc(d.catalog.Documents, func(doc document) bool { return doc.ID == id })
}

func (d *Documents) selectDocument(id, permission string) (document, int, error) {
	if !uuidV4.MatchString(id) {
		return document{}, -1, errors.New("document ID must be a canonical UUIDv4")
	}
	i := d.documentIndex(id)
	if i < 0 || !d.allowed(d.catalog.Documents[i].Scope, permission) {
		return document{}, -1, documentUnavailable
	}
	return d.catalog.Documents[i], i, nil
}

type DocumentContent struct {
	Content  string `json:"content"`
	Revision string `json:"revision"`
}

func (d *Documents) Get(id string, output io.Writer, withRevision bool) (DocumentContent, error) {
	doc, _, err := d.selectDocument(id, "read")
	if err != nil {
		return DocumentContent{}, err
	}
	file, err := d.openDocument(doc)
	if os.IsNotExist(err) {
		return DocumentContent{}, documentUnavailable
	}
	if err != nil {
		return DocumentContent{}, err
	}
	defer file.Close()
	if !withRevision {
		_, err := io.Copy(output, file)
		return DocumentContent{}, err
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return DocumentContent{}, err
	}
	if !utf8.Valid(data) {
		return DocumentContent{}, errors.New("stored document is not valid UTF-8")
	}
	return DocumentContent{Content: string(data), Revision: revision(data)}, nil
}

func (d *Documents) Locate(scope, name string, milestone *string) (string, error) {
	if !validScope(scope) || !validSlug(name) {
		return "", errors.New("--in and --name must identify a valid scope and name")
	}
	label := ""
	if milestone != nil {
		label = strings.TrimSpace(*milestone)
		if label != "" && !validLabel(label, false) {
			return "", errors.New("invalid --milestone")
		}
	}
	if !d.allowed(scope, "read") {
		return "", documentUnavailable
	}
	var found *document
	for _, doc := range d.catalog.Documents {
		if doc.Scope != scope || doc.Name != name || (milestone != nil && !strings.EqualFold(doc.Milestone, label)) {
			continue
		}
		if found != nil {
			return "", fmt.Errorf("%s matches multiple documents in %s. Supply --milestone", name, scope)
		}
		found = &doc
	}
	if found == nil {
		return "", documentUnavailable
	}
	file, err := d.openDocument(*found)
	if os.IsNotExist(err) {
		return "", documentUnavailable
	}
	if err != nil {
		return "", err
	}
	return found.ID, file.Close()
}

func (d *Documents) Delete(id string) error {
	if !d.access.edit {
		return errors.New("document deletion requires an exclusive board lock")
	}
	doc, i, err := d.selectDocument(id, "delete")
	if err != nil {
		return err
	}
	next := d.catalog
	next.Documents = slices.Delete(slices.Clone(next.Documents), i, i+1)
	encoded, err := d.prepareCatalog(next)
	if err != nil {
		return err
	}

	scope, err := d.scopeRoot(doc.Scope)
	if os.IsNotExist(err) {
		return documentUnavailable
	}
	if err != nil {
		return err
	}
	defer scope.Close()
	name := path.Base(doc.Path)
	info, err := scope.Lstat(name)
	if os.IsNotExist(err) {
		return documentUnavailable
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("document %s is not a regular file", id)
	}

	if err := removeDocumentFile(scope, name); err != nil {
		return err
	}
	if err := d.publishCatalog(next, encoded); err != nil {
		return fmt.Errorf("document %s was removed but its index entry was not removed: %w", id, err)
	}
	return nil
}
