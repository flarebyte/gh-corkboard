package board

import (
	"path"
	"strings"
)

func (d document) canonicalPath() string {
	kind, scope, _ := strings.Cut(d.Scope, ":")
	filename := d.ID + "--" + d.Name
	if d.SchemaID != "" {
		filename += "." + d.SchemaID
	}
	return path.Join(kind+"s", scope, filename+"."+d.Format)
}
