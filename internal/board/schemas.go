package board

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/flarebyte/gh-corkboard/internal/docformat"
	"github.com/flarebyte/gh-corkboard/internal/schema"
	"github.com/flarebyte/gh-corkboard/internal/strictjson"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type SchemaKey struct {
	Format string `json:"format"`
	ID     string `json:"schemaId"`
}

var schemaFormats = []string{"csv", "json", "md"}

func (key SchemaKey) filename() string { return key.ID + ".schema.json" }

func (key SchemaKey) validate() error {
	if !docformat.Valid(key.Format) {
		return fmt.Errorf("schema format must be csv, json, or md")
	}
	if !validSlug(key.ID) {
		return fmt.Errorf("schema ID must be a valid slug")
	}
	return nil
}

func (a *Access) schemaRoot(format string) (*os.Root, error) {
	registry, err := openDirectory(a.root, "schemas")
	if err != nil {
		return nil, err
	}
	defer registry.Close()
	return openDirectory(registry, format)
}

func (a *Access) openSchema(key SchemaKey) (*os.File, error) {
	if err := key.validate(); err != nil {
		return nil, err
	}
	format, err := a.schemaRoot(key.Format)
	if err != nil {
		return nil, err
	}
	defer format.Close()
	return openRegular(format, key.filename(), os.O_RDONLY)
}

func (a *Access) GetSchema(key SchemaKey, out io.Writer) error {
	file, err := a.openSchema(key)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(out, file)
	return err
}

func (a *Access) ListSchemas(filter string) ([]SchemaKey, error) {
	formats := schemaFormats
	if filter != "" {
		if !docformat.Valid(filter) {
			return nil, fmt.Errorf("--format must be csv, json, or md")
		}
		formats = []string{filter}
	}
	keys := []SchemaKey{}
	for _, name := range formats {
		folder, err := a.schemaRoot(name)
		if err != nil {
			return nil, err
		}
		entries, err := readDirectory(folder)
		folder.Close()
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			id, ok := strings.CutSuffix(entry.Name(), ".schema.json")
			if !ok || !validSlug(id) {
				continue
			}
			if !entry.Type().IsRegular() {
				return nil, fmt.Errorf("schema %s/%s is not a regular file", name, id)
			}
			keys = append(keys, SchemaKey{name, id})
		}
	}
	return keys, nil
}

func compileSchema(key SchemaKey, data []byte) (*jsonschema.Schema, error) {
	definition, err := strictjson.Decode(data, 10000)
	if err != nil {
		return nil, err
	}
	return schema.Compile("https://gh-corkboard.invalid/schemas/"+key.Format+"/"+key.filename(), definition)
}

func (a *Access) WriteSchema(key SchemaKey, data []byte, create bool) error {
	if err := a.requireAdminEdit(); err != nil {
		return err
	}
	if err := key.validate(); err != nil {
		return err
	}
	if _, err := compileSchema(key, data); err != nil {
		return &docformat.InputError{Format: "schema", Err: err}
	}
	root, err := a.schemaRoot(key.Format)
	if err != nil {
		return err
	}
	defer root.Close()
	name := key.filename()
	if create {
		return createFile(root, name, data)
	}
	file, err := openRegular(root, name, os.O_RDONLY)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return replaceFile(root, name, data, 0600)
}

func (a *Access) DeleteSchema(key SchemaKey) error {
	if err := a.requireAdminEdit(); err != nil {
		return err
	}
	if err := key.validate(); err != nil {
		return err
	}
	root, err := a.schemaRoot(key.Format)
	if err != nil {
		return err
	}
	defer root.Close()
	name := key.filename()
	file, err := openRegular(root, name, os.O_RDONLY)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := a.checkSchemaUnused(key); err != nil {
		return err
	}
	return removeDocumentFile(root, name)
}

func (a *Access) checkSchemaUnused(key SchemaKey) error {
	catalog, err := loadCatalog(a.index, a.policy)
	if err != nil {
		return err
	}
	bound := fmt.Errorf("schema %s/%s is bound to a document", key.Format, key.ID)
	for _, doc := range catalog.Documents {
		if doc.Format == key.Format && doc.SchemaID == key.ID {
			return bound
		}
	}

	for _, kind := range []string{"agents", "skills", "topics"} {
		found, err := a.findBinding(kind, key)
		if err != nil {
			return err
		}
		if found {
			return bound
		}
	}
	return nil
}

func (a *Access) findBinding(kind string, key SchemaKey) (bool, error) {
	container, err := openDirectory(a.root, kind)
	if err != nil {
		return false, err
	}
	defer container.Close()
	scopes, err := readDirectory(container)
	if err != nil {
		return false, err
	}
	for _, scope := range scopes {
		if !validSlug(scope.Name()) || !scope.IsDir() {
			continue
		}
		folder, err := openDirectory(container, scope.Name())
		if err != nil {
			return false, err
		}
		entries, err := readDirectory(folder)
		folder.Close()
		if err != nil {
			return false, err
		}
		for _, entry := range entries {
			doc, ok := documentFromFilename("", entry.Name())
			if ok && doc.Format == key.Format && doc.SchemaID == key.ID {
				return true, nil
			}
		}
	}
	return false, nil
}

func (a *Access) requireAdminEdit() error {
	if !a.edit || a.policy.role(a.actor) != "admin" {
		return errors.New("operation requires a board administrator and an exclusive board lock")
	}
	return nil
}
