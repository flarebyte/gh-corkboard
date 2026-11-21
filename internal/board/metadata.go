package board

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/flarebyte/gh-corkboard/internal/schema"
	"github.com/flarebyte/gh-corkboard/internal/strictjson"
)

//go:embed spec/*.schema.json
var specs embed.FS

const accessLimit = 4 << 20
const catalogLimit = 32 << 20

func metadata[T any](root *os.Root, name string, limit int) (T, error) {
	var result T
	file, err := openRegular(root, name+".json", os.O_RDONLY)
	if err != nil {
		return result, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return result, err
	}
	return decodeMetadata[T](name, data, limit)
}

func decodeMetadata[T any](name string, data []byte, limit int) (T, error) {
	var result T
	if len(data) > limit {
		return result, fmt.Errorf("%s exceeds %d bytes", name, limit)
	}
	document, err := strictjson.Decode(data, 8)
	if err != nil {
		return result, err
	}
	definition, err := specs.ReadFile("spec/" + name + ".schema.json")
	if err != nil {
		return result, err
	}
	var source any
	if err := json.Unmarshal(definition, &source); err != nil {
		return result, err
	}
	validator, err := schema.Compile("https://gh-corkboard.invalid/"+name, source)
	if err != nil {
		return result, err
	}
	if err := validator.Validate(document); err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)
	return result, err
}
