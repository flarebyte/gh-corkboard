package project

import (
	_ "embed"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed main.project.yaml
var manifest []byte

// Version is the version embedded from main.project.yaml at build time.
var Version = readVersion()

func readVersion() string {
	var config struct {
		Version string `yaml:"version"`
	}
	if err := yaml.Unmarshal(manifest, &config); err != nil {
		return "unknown"
	}
	if version := strings.TrimSpace(config.Version); version != "" {
		return version
	}
	return "unknown"
}
