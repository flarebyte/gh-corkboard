package strictjson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"unicode/utf8"
)

// OffsetError retains the byte offset of a native JSON parse failure.
type OffsetError struct {
	Offset int
	Err    error
}

func (e *OffsetError) Error() string { return e.Err.Error() }
func (e *OffsetError) Unwrap() error { return e.Err }

// Decode preserves number spellings and rejects ambiguous or lossy JSON.
func Decode(data []byte, maxDepth int) (any, error) {
	if !utf8.Valid(data) || !validSurrogates(data) {
		return nil, errors.New("invalid Unicode in JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	value, err := readValue(decoder, maxDepth)
	if err != nil {
		return nil, &OffsetError{int(decoder.InputOffset()), err}
	}
	offset := int(decoder.InputOffset())
	if _, err := decoder.Token(); err != io.EOF {
		return nil, &OffsetError{offset, errors.New("expected exactly one JSON value")}
	}
	return value, nil
}

func validSurrogates(data []byte) bool {
	for {
		slash := bytes.IndexByte(data, '\\')
		if slash < 0 || slash+1 >= len(data) {
			return true
		}
		data = data[slash:]
		code, ok := unicodeEscape(data)
		if !ok {
			data = data[2:]
			continue
		}
		data = data[6:]
		if code >= 0xdc00 && code <= 0xdfff {
			return false
		}
		if code >= 0xd800 && code <= 0xdbff {
			low, ok := unicodeEscape(data)
			if !ok || low < 0xdc00 || low > 0xdfff {
				return false
			}
			data = data[6:]
		}
	}
}

func unicodeEscape(data []byte) (uint64, bool) {
	if len(data) < 6 || data[0] != '\\' || data[1] != 'u' {
		return 0, false
	}
	code, err := strconv.ParseUint(string(data[2:6]), 16, 16)
	return code, err == nil
}

func readValue(decoder *json.Decoder, depth int) (any, error) {
	token, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	delim, container := token.(json.Delim)
	if !container {
		return token, nil
	}
	if depth == 0 {
		return nil, errors.New("JSON nesting limit exceeded")
	}
	if delim == '{' {
		obj := make(map[string]any)
		for decoder.More() {
			key, err := decoder.Token()
			if err != nil {
				return nil, err
			}
			name, ok := key.(string)
			if !ok {
				return nil, errors.New("expected an object key")
			}
			if _, exists := obj[name]; exists {
				return nil, fmt.Errorf("duplicate JSON key %q", name)
			}
			child, err := readValue(decoder, depth-1)
			if err != nil {
				return nil, err
			}
			obj[name] = child
		}
		_, err = decoder.Token()
		return obj, err
	}
	if delim != '[' {
		return nil, errors.New("unexpected JSON delimiter")
	}
	arr := []any{}
	for decoder.More() {
		child, err := readValue(decoder, depth-1)
		if err != nil {
			return nil, err
		}
		arr = append(arr, child)
	}
	_, err = decoder.Token()
	return arr, err
}
