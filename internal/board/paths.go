package board

import (
	"path"
	"strings"

	"github.com/flarebyte/gh-corkboard/internal/docformat"
)

func (d document) canonicalPath() string {
	filename := d.ID + "--" + d.Name
	if d.SchemaID != "" {
		filename += "." + d.SchemaID
	}
	return path.Join(scopePath(d.Scope), filename+"."+d.Format)
}

// documentFromFilename accepts canonical and shorthand names in an already
// anchored scope. The caller supplies the logical scope, never a disk path.
func documentFromFilename(scope, filename string) (document, bool) {
	fields := strings.Split(filename, ".")
	if len(fields) < 2 || len(fields) > 3 {
		return document{}, false
	}
	format := fields[len(fields)-1]
	if !docformat.Valid(format) {
		return document{}, false
	}
	doc := document{Scope: scope, Name: fields[0], Format: format, Status: "draft", Reviews: map[string]review{}}
	if len(fields) == 3 {
		doc.SchemaID = fields[1]
		if !validSlug(doc.SchemaID) {
			return document{}, false
		}
	}
	if id, name, ok := strings.Cut(doc.Name, "--"); ok && uuidV4.MatchString(id) {
		doc.ID, doc.Name = id, name
	}
	return doc, validSlug(doc.Name)
}

func scopePath(scope string) string {
	kind, name, _ := strings.Cut(scope, ":")
	return path.Join(kind+"s", name)
}
