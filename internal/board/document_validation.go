package board

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/flarebyte/gh-corkboard/internal/docformat"
	"github.com/flarebyte/gh-corkboard/internal/schema"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type ValidationResult struct {
	ID          string              `json:"id"`
	Format      string              `json:"format"`
	SchemaID    string              `json:"schemaId,omitempty"`
	State       string              `json:"state"`
	Diagnostics []schema.Diagnostic `json:"diagnostics"`
}

type validationSchema struct {
	compiled *jsonschema.Schema
	finding  *schema.Diagnostic
}

func (d *Documents) Validate(id string, all bool) ([]ValidationResult, error) {
	var docs []document
	if all {
		readable := map[string]bool{}
		for _, doc := range d.catalog.Documents {
			allowed, known := readable[doc.Scope]
			if !known {
				allowed = d.allowed(doc.Scope, "read")
				readable[doc.Scope] = allowed
			}
			if allowed {
				docs = append(docs, doc)
			}
		}
		slices.SortFunc(docs, compareDocuments)
	} else {
		doc, _, err := d.selectDocument(id, "read")
		if err != nil {
			return nil, err
		}
		docs = []document{doc}
	}

	cache := map[SchemaKey]validationSchema{}
	results := make([]ValidationResult, 0, len(docs))
	for _, doc := range docs {
		result, err := d.validateDocument(doc, cache)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (d *Documents) validateDocument(doc document, cache map[SchemaKey]validationSchema) (ValidationResult, error) {
	result := ValidationResult{ID: doc.ID, Format: doc.Format, SchemaID: doc.SchemaID, State: "invalid", Diagnostics: []schema.Diagnostic{}}
	file, err := d.openDocument(doc)
	if os.IsNotExist(err) {
		return result, documentUnavailable
	}
	if err != nil {
		return result, err
	}
	defer file.Close()
	content, err := io.ReadAll(file)
	if err != nil {
		return result, err
	}

	projected, err := docformat.Project(doc.Format, content)
	if err != nil {
		line, col := docformat.ErrorPosition(content, err)
		result.Diagnostics = append(result.Diagnostics, schema.Diagnostic{Code: "invalid_" + doc.Format, Message: err.Error(), Line: line, Column: col})
		return result, nil
	}
	if doc.SchemaID == "" {
		result.State = "schemaless"
		return result, nil
	}

	key := SchemaKey{doc.Format, doc.SchemaID}
	compiled, ok := cache[key]
	if !ok {
		compiled, err = d.access.loadSchemaForValidation(key)
		if err != nil {
			return result, err
		}
		cache[key] = compiled
	}
	if compiled.finding != nil {
		result.Diagnostics = append(result.Diagnostics, *compiled.finding)
	} else if err := compiled.compiled.Validate(projected.Value); err != nil {
		result.Diagnostics = schema.Findings(err, projected.Location(content))
	} else {
		result.State = "valid"
	}
	return result, nil
}

func (a *Access) loadSchemaForValidation(key SchemaKey) (validationSchema, error) {
	file, err := a.openSchema(key)
	if errors.Is(err, os.ErrNotExist) {
		return validationSchema{finding: &schema.Diagnostic{Code: "schema_missing",
			Message: fmt.Sprintf("Schema %s/%s is missing.", key.Format, key.ID)}}, nil
	}
	if err != nil {
		return validationSchema{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return validationSchema{}, err
	}
	compiled, err := compileSchema(key, data)
	if err != nil {
		return validationSchema{finding: &schema.Diagnostic{Code: "schema_invalid", Message: err.Error()}}, nil
	}
	return validationSchema{compiled: compiled}, nil
}
