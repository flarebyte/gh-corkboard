package schema

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

const draft2020 = "https://json-schema.org/draft/2020-12/schema"

type schemaResource struct {
	value    any
	location string
}

type localChecker struct {
	seen map[string]bool
}

func checkLocal(document any) error {
	checker := localChecker{seen: map[string]bool{}}
	return checker.visit(document, "", schemaResource{value: document})
}

// Visit schema-valued keywords and explicit pointer targets without treating
// arbitrary annotation values or literals as schemas.
func (c localChecker) visit(document any, location string, resource schemaResource) error {
	object, ok := document.(map[string]any)
	if !ok || c.seen[location] {
		return nil
	}
	c.seen[location] = true
	if _, ok := object["$id"].(string); ok {
		resource = schemaResource{document, location}
	}
	if value, exists := object["$schema"]; exists {
		uri, ok := value.(string)
		if !ok || (uri != draft2020 && uri != draft2020+"#") {
			return fmt.Errorf("only JSON Schema Draft 2020-12 is supported")
		}
	}
	if err := c.visitChildren(object, location, resource); err != nil {
		return err
	}
	for _, keyword := range []string{"$ref", "$dynamicRef", "$recursiveRef"} {
		ref, ok := object[keyword].(string)
		if !ok {
			continue
		}
		fragment, ok := strings.CutPrefix(ref, "#")
		if !ok {
			return fmt.Errorf("%s must be a same-resource fragment reference", keyword)
		}
		pointer, err := url.PathUnescape(fragment)
		if err == nil && strings.HasPrefix(pointer, "/") {
			target, loc := pointerTarget(resource, pointer)
			if err := c.visit(target, loc, resource); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c localChecker) visitChildren(object map[string]any, location string, resource schemaResource) error {
	for _, keyword := range []string{
		"items", "contains", "additionalProperties", "unevaluatedProperties", "unevaluatedItems",
		"propertyNames", "if", "then", "else", "not", "contentSchema",
	} {
		if err := c.visit(object[keyword], location+"/"+keyword, resource); err != nil {
			return err
		}
	}
	for _, keyword := range []string{"$defs", "definitions", "properties", "patternProperties", "dependentSchemas"} {
		children, _ := object[keyword].(map[string]any)
		for name, child := range children {
			if err := c.visit(child, location+"/"+keyword+jsonPointer([]string{name}), resource); err != nil {
				return err
			}
		}
	}
	for _, keyword := range []string{"prefixItems", "allOf", "anyOf", "oneOf"} {
		children, _ := object[keyword].([]any)
		for i, child := range children {
			if err := c.visit(child, location+"/"+keyword+"/"+strconv.Itoa(i), resource); err != nil {
				return err
			}
		}
	}
	return nil
}

func pointerTarget(resource schemaResource, pointer string) (any, string) {
	value, location := resource.value, resource.location
	unescape := strings.NewReplacer("~1", "/", "~0", "~")
	for encoded := range strings.SplitSeq(pointer[1:], "/") {
		token := unescape.Replace(encoded)
		switch node := value.(type) {
		case map[string]any:
			value = node[token]
		case []any:
			index, err := strconv.Atoi(token)
			if err != nil || index < 0 || index >= len(node) {
				return nil, ""
			}
			value = node[index]
			token = strconv.Itoa(index)
		default:
			return nil, ""
		}
		location += jsonPointer([]string{token})
	}
	return value, location
}
