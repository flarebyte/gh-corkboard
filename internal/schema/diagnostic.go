package schema

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Diagnostic struct {
	Code         string  `json:"code"`
	Message      string  `json:"message"`
	InstancePath *string `json:"instancePath,omitempty"`
	Line         int     `json:"line,omitempty"`
	Column       int     `json:"column,omitempty"`
}

func Findings(err error, location func([]string) (int, int)) []Diagnostic {
	var root *jsonschema.ValidationError
	if !errors.As(err, &root) {
		return []Diagnostic{{Code: "schema_validation", Message: err.Error()}}
	}
	diagnostics := []Diagnostic{}
	english := message.NewPrinter(language.English)
	var visit func(*jsonschema.ValidationError)
	visit = func(current *jsonschema.ValidationError) {
		if len(current.Causes) > 0 {
			for _, cause := range current.Causes {
				visit(cause)
			}
			return
		}

		pointer := jsonPointer(current.InstanceLocation)
		line, col := location(current.InstanceLocation)
		if required, ok := current.ErrorKind.(*kind.Required); ok {
			for _, name := range required.Missing {
				diagnostics = append(diagnostics, Diagnostic{"schema_required", fmt.Sprintf("Required property %q is missing.", name), &pointer, line, col})
			}
			return
		}
		code := "schema_validation"
		if parts := current.ErrorKind.KeywordPath(); len(parts) > 0 {
			code = "schema_" + strings.TrimPrefix(parts[0], "$")
		}
		msg := current.ErrorKind.LocalizedString(english)
		diagnostics = append(diagnostics, Diagnostic{code, msg, &pointer, line, col})
	}
	visit(root)
	slices.SortFunc(diagnostics, func(a, b Diagnostic) int {
		return cmp.Or(cmp.Compare(*a.InstancePath, *b.InstancePath), cmp.Compare(a.Code, b.Code), cmp.Compare(a.Message, b.Message))
	})
	return slices.CompactFunc(diagnostics, func(a, b Diagnostic) bool {
		return *a.InstancePath == *b.InstancePath && a.Code == b.Code && a.Message == b.Message
	})
}

func jsonPointer(tokens []string) string {
	var out strings.Builder
	escape := strings.NewReplacer("~", "~0", "/", "~1")
	for _, token := range tokens {
		out.WriteByte('/')
		out.WriteString(escape.Replace(token))
	}
	return out.String()
}
