package docformat

import "fmt"

func mergeJSON(base, patch []byte) ([]byte, error) {
	target, err := parseJSON(base)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON in stored document: %w", err)
	}
	changes, err := parseJSON(patch)
	if err != nil {
		return nil, err
	}
	return encodeJSON(mergeValue(target, changes))
}

// mergeValue implements the object merge and whole-value replacement rules of RFC 7396.
func mergeValue(target, patch any) any {
	changes, ok := patch.(map[string]any)
	if !ok {
		return patch
	}
	object, ok := target.(map[string]any)
	if !ok {
		object = make(map[string]any)
	}
	for name, change := range changes {
		if change == nil {
			delete(object, name)
		} else {
			object[name] = mergeValue(object[name], change)
		}
	}
	return object
}
