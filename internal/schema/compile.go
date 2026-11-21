package schema

import (
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

type bundledOnly struct{}

func (bundledOnly) Load(url string) (any, error) {
	return nil, fmt.Errorf("schema resource is unavailable: %q", url)
}

func Compile(id string, document any) (*jsonschema.Schema, error) {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.UseLoader(bundledOnly{})
	if err := compiler.AddResource(id, document); err != nil {
		return nil, err
	}
	return compiler.Compile(id)
}
